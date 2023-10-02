package replicator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cheops.com/backends"
	"cheops.com/env"
)

type Crdt struct {
}

var crdt *Crdt = newCrdt()

func newCrdt() *Crdt {
	c := &Crdt{}
	go c.replicate()
	go c.watchRequests()
	return c
}

func (c *Crdt) Do(ctx context.Context, sites []string, operation Payload) (reply Payload, err error) {

	// find highest generation
	docs, err := c.getDocsForSites(sites)
	if err != nil {
		return reply, err
	}
	max := uint64(0)
	for _, d := range docs {
		if !d.Payload.IsRequest() {
			continue
		}
		if d.Generation >= max {
			max = d.Generation
		}
	}

	// Post document for replication
	newDoc := crdtDocument{
		Locations:  sites,
		Generation: max + 1,
		Payload:    operation,
	}
	buf, err := json.Marshal(newDoc)
	if err != nil {
		return reply, err
	}
	newresp, err := http.Post("http://localhost:5984/cheops", "application/json", bytes.NewReader(buf))
	if err != nil {
		return reply, err
	}

	replicatedOperationId := ""
	if newresp.StatusCode == 201 {
		type CreatedResp struct {
			Id string `json:"id"`
		}
		var cr CreatedResp
		json.NewDecoder(newresp.Body).Decode(&cr)
		replicatedOperationId = cr.Id
	}

	repliesChan := make(chan Payload)

	feedCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		since := ""
		for {
			u := "http://localhost:5984/cheops/_changes?include_docs=true&feed=continuous"
			if since != "" {
				u += fmt.Sprintf("&since=%s", since)
			}
			req, err := http.NewRequestWithContext(feedCtx, "GET", u, nil)
			if err != nil {
				log.Printf("Couldn't create request with context: %v\n", err)
				break
			}
			feed, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("Couldn't get feed for replies: %v\n", err)
			}
			if feed.StatusCode != 200 {
				log.Fatal(fmt.Errorf("Can't get _changes feed: %s", feed.Status))
			}

			defer feed.Body.Close()

			scanner := bufio.NewScanner(feed.Body)
			for scanner.Scan() {
				s := strings.TrimSpace(scanner.Text())
				if s == "" {
					continue
				}

				var d DocChange
				err := json.NewDecoder(strings.NewReader(s)).Decode(&d)
				if err != nil {
					log.Printf("Couldn't decode: %s", err)
					continue
				}

				if d.Doc.Payload.RequestId != replicatedOperationId {
					continue
				}
				if d.Doc.Payload.IsRequest() {
					continue
				}

				repliesChan <- d.Doc.Payload
				since = d.Seq
			}

			select {
			case <-feedCtx.Done():
				return
			default:
				continue
			}

		}
	}()

	replies := make([]Payload, 0, len(sites))

wait:
	for i := 0; i < len(sites); i++ {
		log.Printf("Waiting for reply %d\n", i)
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return reply, err
			}
		case reply := <-repliesChan:
			replies = append(replies, reply)
		case <-time.After(20 * time.Second):
			// timeout
			// happens when the node didn't do the request locally
			// but request comes from replication group (could be optimized by creating
			// the channel here only, so we know)
			// or when the reply doesn't arrive locally.
			//
			// Because there are multiple cases, let's leave it like that,
			// some goroutines will wait for nothing, that's alright
			break wait
		}
	}
	if len(replies) > 0 {
		// No particular reason, there is no one good response that can fit
		// while being a merge of the N replies
		// TODO: add a status header for other replies still
		return replies[0], nil
	}
	return reply, fmt.Errorf("No replies")
}

func (c *Crdt) watchRequests() {

	go func() {
		since := ""
		for {
			u := "http://localhost:5984/cheops/_changes?include_docs=true&feed=continuous"
			if since != "" {
				u += fmt.Sprintf("&since=%s", since)
			}
			feed, err := http.Get(u)
			if err != nil {
				log.Fatal(err)
			}
			if feed.StatusCode != 200 {
				log.Fatal(fmt.Errorf("Can't get _changes feed: %s", feed.Status))
			}

			defer feed.Body.Close()

			scanner := bufio.NewScanner(feed.Body)
			idx := 0
			for scanner.Scan() {
				idx++
				fmt.Printf("idx: %d\n", idx)
				s := strings.TrimSpace(scanner.Text())
				if s == "" {
					continue
				}

				var d DocChange
				err := json.NewDecoder(strings.NewReader(s)).Decode(&d)
				if err != nil {
					log.Printf("Couldn't decode: %s", err)
					continue
				}

				if len(d.Doc.Locations) == 0 {
					// CouchDB status message, discard
					continue
				}
				if !d.Doc.Payload.IsRequest() {
					continue
				}

				log.Printf("Running from %d at %d on %v\n", os.Getpid(), idx, d.Doc)
				c.run(d.Doc.Locations)
				since = d.Seq
			}
		}
	}()
}

func (c *Crdt) run(sites []string) {
	docs, err := c.getDocsForSites(sites)
	if err != nil {
		log.Printf("Couldn't get docs for sites: %v\n", err)
		return
	}

	requests := make([]crdtDocument, 0)
	requestIdsInReplies := make(map[string]struct{})
	for _, doc := range docs {
		if doc.Payload.IsRequest() {
			requests = append(requests, doc)
		} else {
			requestIdsInReplies[doc.Payload.RequestId] = struct{}{}
			return
		}
	}
	sortDocuments(requests)
	p := requests[len(requests)-1].Payload // Only execute the last one
	if _, ok := requestIdsInReplies[p.RequestId]; ok {
		// A reply already exists, don't run it again
	}

	headerOut, bodyOut, err := backends.HandleKubernetes(p.Method, p.Path, p.Header, p.Body)

	if err != nil {
		log.Printf("Couldn't exec request: %v\n", err)
		return
	}

	// Post document for replication
	newDoc := crdtDocument{
		Locations:  sites,
		Generation: 0,
		Payload: Payload{
			RequestId: p.RequestId,
			Header:    headerOut,
			Body:      bodyOut,
			Site:      env.Myfqdn,
		},
	}
	buf, err := json.Marshal(newDoc)
	if err != nil {
		log.Printf("Couldn't marshal reply: %v\n", err)
		return
	}
	newresp, err := http.Post("http://localhost:5984/cheops", "application/json", bytes.NewReader(buf))
	if err != nil {
		log.Printf("Couldn't send reply: %v\n", err)
		return
	}
	if newresp.StatusCode != 201 {
		log.Printf("Couldn't send reply: %v\n", newresp.Status)
		return
	}
}

func (c *Crdt) getDocsForSites(sites []string) ([]crdtDocument, error) {
	locations := make([]string, 0)
	for _, site := range sites {
		locations = append(locations, fmt.Sprintf(`{"Locations": {"$all": ["%s"]}}`, site))
	}
	selector := fmt.Sprintf(`{"selector": {"$and": [%s]}}`, strings.Join(locations, ","))

	fmt.Println(selector)
	current, err := http.Post("http://localhost:5984/cheops/_find", "application/json", strings.NewReader(selector))
	if err != nil {
		return nil, err
	}
	if current.StatusCode != 200 {
		return nil, fmt.Errorf("Post %s: %s", current.Request.URL.String(), current.Status)
	}

	var cr CouchResp

	err = json.NewDecoder(current.Body).Decode(&cr)
	current.Body.Close()
	if err != nil {
		return nil, err
	}

	return cr.Docs, nil
}

type CouchResp struct {
	Docs []crdtDocument `json:"docs"`
}

// replicate watches the _changes feed and makes sure the replication jobs
// are in place
func (c *Crdt) replicate() {
	existingJobs := c.getExistingJobs()

	since := ""
	for {
		u := "http://localhost:5984/cheops/_changes?include_docs=true&feed=continuous"
		if since != "" {
			u += fmt.Sprintf("&since=%s", since)
		}
		feed, err := http.Get(u)
		if err != nil {
			log.Fatal(err)
		}
		if feed.StatusCode != 200 {
			log.Fatal(fmt.Errorf("Can't get _changes feed: %s", feed.Status))
		}

		defer feed.Body.Close()

		scanner := bufio.NewScanner(feed.Body)
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if s == "" {
				continue
			}

			var d DocChange
			err := json.NewDecoder(strings.NewReader(s)).Decode(&d)
			if err != nil {
				log.Printf("Couldn't decode: %s", err)
				continue
			}

			if len(d.Doc.Locations) == 0 {
				continue
			}

			for _, location := range d.Doc.Locations {
				if location == env.Myfqdn {
					continue
				}
				if _, ok := existingJobs[location]; !ok {
					body := fmt.Sprintf(`{"continuous": true, "source": "http://localhost:5984/cheops", "target": "http://%s:5984/crdt-log"}`, location)
					resp, err := http.Post("http://admin:password@localhost:5984/_replicate", "application/json", strings.NewReader(body))
					if err != nil {
						log.Printf("Couldn't add replication: %s\n", err)
					}
					if resp.StatusCode != 202 {
						log.Printf("Couldn't add replication: %s\n", resp.Status)
					}
					existingJobs[location] = struct{}{}
				}
			}

			since = d.Seq
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}

type DocChange struct {
	Seq string       `json:"seq"`
	Doc crdtDocument `json:"doc"`
}

func (c *Crdt) getExistingJobs() map[string]struct{} {
	existingJobs, err := http.Get("http://admin:password@localhost:5984/_scheduler/jobs")
	if err != nil {
		log.Fatal(err)
	}
	defer existingJobs.Body.Close()

	if existingJobs.StatusCode != 200 {
		log.Fatal(fmt.Errorf("Can't get existing replication jobs: %s", existingJobs.Status))
	}

	var js Jobs
	err = json.NewDecoder(existingJobs.Body).Decode(&js)
	if err != nil {
		log.Fatalf("Couldn't decode: %s", err)
	}

	ret := make(map[string]struct{})
	for _, j := range js.Jobs {
		ret[j.Target] = struct{}{}
	}

	return ret
}

type Jobs struct {
	Jobs []Job `json:"jobs"`
}

type Job struct {
	Target string `json:"target"`
}
