package replicator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cheops.com/backends"
	"cheops.com/env"
	jp "github.com/evanphx/json-patch"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Crdt struct {
}

func newCrdt(port int) *Crdt {
	c := &Crdt{}
	c.ensureIndex()
	c.replicate()
	c.watchRequests()
	c.listenDump(port)
	return c
}

// ensureIndex makes sure that the _find call remains fast enough
// by indexing on the Locations field
func (c *Crdt) ensureIndex() {
	idx, err := http.Post("http://admin:password@localhost:5984/cheops/_index", "application/json", strings.NewReader(`{"index": {"fields": ["Locations"]}}`))
	if err != nil {
		log.Fatal(err)
	}
	if idx.StatusCode != http.StatusCreated && idx.StatusCode != http.StatusOK {
		log.Fatalf("Can't create index: %s\n", idx.Status)
	}
}

func (c *Crdt) listenDump(port int) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), "GET", "http://localhost:5984/cheops/_all_docs?include_docs=true", nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		type AllDocs struct {
			Rows []struct {
				Doc crdtDocument
			}
		}

		var ad AllDocs
		err = json.NewDecoder(resp.Body).Decode(&ad)

		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		requests := make([]crdtDocument, 0)
		for _, row := range ad.Rows {
			if row.Doc.Payload.IsRequest() {
				requests = append(requests, row.Doc)
			}
		}
		sortDocuments(requests)

		for _, request := range requests {
			for _, doc := range ad.Rows {
				if doc.Doc.Payload.RequestId == request.Payload.RequestId && !doc.Doc.Payload.IsRequest() {
					fmt.Fprintf(w, "Ran %s %s\n", request.Payload.RequestId, doc.Doc.Payload.Site)
				}
			}
		}

	})
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
}

func (c *Crdt) Do(ctx context.Context, sites []string, operation Payload) (reply Payload, err error) {

	// Prepare replies gathering before running the request
	// It's all asynchronous
	repliesChan := make(chan Payload)
	repliesCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.watchReplies(repliesCtx, operation.RequestId, repliesChan)

	// find highest generation
	docs, err := c.getDocsForId(operation.ResourceId)
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

	// Generate diff with locally current config
	currentConfig := backends.CurrentConfig(ctx, []byte(operation.Body))
	asjson, err := yaml.Parse(string(operation.Body))
	if err != nil {
		return reply, err
	}
	asjsonbin, err := json.Marshal(asjson)
	if err != nil {
		return reply, err
	}
	patch, err := jp.CreateMergePatch([]byte(currentConfig), asjsonbin)
	if err != nil {
		return reply, err
	}

	log.Printf("current config: %s\n", currentConfig)
	log.Printf("patch: %s\n", patch)

	// Post document for replication
	newDoc := crdtDocument{
		Locations:  sites,
		Generation: max + 1,
		Payload: Payload{
			RequestId:  operation.RequestId,
			ResourceId: operation.ResourceId,
			Method:     operation.Method,
			Path:       operation.Path,
			Header:     operation.Header,
			Body:       string(patch),
		},
	}
	buf, err := json.Marshal(newDoc)
	if err != nil {
		return reply, err
	}
	newresp, err := http.Post("http://localhost:5984/cheops", "application/json", bytes.NewReader(buf))
	if err != nil {
		return reply, err
	}

	if newresp.StatusCode != 201 {
		type CreatedResp struct {
			Id string `json:"id"`
		}
		var cr CreatedResp
		json.NewDecoder(newresp.Body).Decode(&cr)
		err = fmt.Errorf("Couldn't send request %s to couchdb: %s\n", cr.Id, newresp.Status)
		return
	}

	// Gather replies sent on the channel created at the beginning
	// of this function
	replies := make([]Payload, 0, len(sites))
wait:
	for i := 0; i < len(sites); i++ {
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
		// We only merge bodies, headers are actually not useful

		type multiReply struct {
			Site string

			// "OK" or "KO"
			Status string
			Body   string
		}

		bodies := make([]multiReply, 0)
		for _, rep := range replies {
			bodies = append(bodies, multiReply{
				Site:   rep.Site,
				Status: rep.Status,
				Body:   rep.Body,
			})
		}
		body, err := json.Marshal(bodies)
		if err != nil {
			log.Printf("Tried and failed to marshall %#v\n", bodies)
			return reply, fmt.Errorf("Couldn't marshall bodies: %w", err)
		}
		reply = Payload{
			RequestId:  operation.RequestId,
			Body:       string(body),
			ResourceId: operation.ResourceId,
		}
		return reply, nil
	}
	return reply, fmt.Errorf("No replies")
}

func (c *Crdt) watchRequests() {
	c.watch(context.Background(), func(d DocChange) {
		if len(d.Doc.Locations) == 0 {
			// CouchDB status message, discard
			return
		}
		if !d.Doc.Payload.IsRequest() {
			return
		}

		c.run(context.Background(), d.Doc.Locations, d.Doc.Payload)
	})
}

func (c *Crdt) watchReplies(ctx context.Context, requestId string, repliesChan chan Payload) {
	c.watch(ctx, func(d DocChange) {
		if d.Doc.Payload.RequestId != requestId {
			return
		}
		if d.Doc.Payload.IsRequest() {
			return
		}

		repliesChan <- d.Doc.Payload
	})

}

// watch watches the _changes feed and runs a function when a new document is seen
// The execution of the function blocks the loop; it is good to not have it run too long
func (c *Crdt) watch(ctx context.Context, onNewDoc func(d DocChange)) {
	ready := make(chan struct{})

	go func() {
		since := ""
		for {
			u := "http://localhost:5984/cheops/_changes?include_docs=true&feed=continuous"
			if since != "" {
				u += fmt.Sprintf("&since=%s", since)
			}
			req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
			if err != nil {
				log.Printf("Couldn't create request with context: %v\n", err)
				break
			}
			feed, err := http.DefaultClient.Do(req)
			if err != nil || feed.StatusCode != 200 {
				// When the database is deleted, we are here. Hopefully it will be recreated and we can continue
				log.Printf("No _changes feed, retrying in 10s")
				<-time.After(10 * time.Second)
				continue
			}

			defer feed.Body.Close()

			// close the channel to signal readiness, or if already close, continue
			select {
			case <-ready:
				// Do nothing
			default:
				close(ready)
			}

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

				onNewDoc(d)
				since = d.Seq
			}

			select {
			case <-ctx.Done():
				return
			default:
				continue
			}

		}
	}()
}

func (c *Crdt) run(ctx context.Context, sites []string, p Payload) {
	docs, err := c.getDocsForId(p.ResourceId)
	if err != nil {
		log.Printf("Couldn't get docs for id: %v\n", err)
		return
	}

	requests := make([]crdtDocument, 0)
	requestIdsInReplies := make(map[string]struct{})
	for _, doc := range docs {
		if doc.Payload.IsRequest() {
			requests = append(requests, doc)
		} else if doc.Payload.Site == env.Myfqdn {
			requestIdsInReplies[doc.Payload.RequestId] = struct{}{}
		}
	}
	sortDocuments(requests)
	body := mergePatches(requests)

	if _, ok := requestIdsInReplies[p.RequestId]; ok {
		// We already have a reply from this site, don't run it
		return
	}

	log.Printf("applying %s\n", body)
	headerOut, bodyOut, err := backends.HandleKubernetes(ctx, p.Method, p.Path, p.Header, body)

	status := "OK"
	if err != nil {
		status = "KO"
	}

	// Post document for replication
	newDoc := crdtDocument{
		Locations:  sites,
		Generation: 0,
		Payload: Payload{
			RequestId:  p.RequestId,
			ResourceId: p.ResourceId,
			Header:     headerOut,
			Body:       string(bodyOut),
			Status:     status,
			Site:       env.Myfqdn,
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
	defer newresp.Body.Close()

	if newresp.StatusCode != 201 {
		log.Printf("Couldn't send reply: %v\n", newresp.Status)
		return
	}

	log.Printf("Ran %s %s\n", p.RequestId, env.Myfqdn)
}

func (c *Crdt) getDocsForId(resourceId string) ([]crdtDocument, error) {
	docs := make([]crdtDocument, 0)
	var bookmark string

	for {
		selector := fmt.Sprintf(`{"bookmark": "%s", "selector": {"Payload.ResourceId": "%s"}}`, bookmark, resourceId)

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

		bookmark = cr.Bookmark
		docs = append(docs, cr.Docs...)

		if len(cr.Docs) == 0 {
			break
		}
	}

	return docs, nil
}

type CouchResp struct {
	Bookmark string         `json:"bookmark"`
	Docs     []crdtDocument `json:"docs"`
}

// replicate watches the _changes feed and makes sure the replication jobs
// are in place
func (c *Crdt) replicate() {
	existingJobs := c.getExistingJobs()
	c.watch(context.Background(), func(d DocChange) {
		if len(d.Doc.Locations) == 0 {
			return
		}

		for _, location := range d.Doc.Locations {
			if location == env.Myfqdn {
				continue
			}
			if _, ok := existingJobs[location]; !ok {
				body := fmt.Sprintf(`{"continuous": true, "source": "http://localhost:5984/cheops", "target": "http://%s:5984/cheops"}`, location)
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
	})
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
