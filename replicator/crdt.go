package replicator

import (
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
)

type Replicator struct {
	w *watches
}

func NewReplicator(port int) *Replicator {
	w := newWatches(context.Background())

	r := &Replicator{w}
	r.ensureCouch()
	r.ensureIndex()
	r.replicate()
	r.watchRequests()
	r.listenDump(port)
	return r
}

// ensureCouch makes sure the databases exist and are correctly populated
func (r *Replicator) ensureCouch() {

	reqs := []struct {
		Method        string
		URL           string
		ExpectedCodes []int
		Body          string
	}{
		{
			Method:        "DELETE",
			URL:           "http://admin:password@localhost:5984/cheops",
			ExpectedCodes: []int{http.StatusOK, http.StatusNotFound},
			Body:          "",
		}, {
			Method:        "PUT",
			URL:           "http://admin:password@localhost:5984/cheops",
			ExpectedCodes: []int{http.StatusCreated, http.StatusPreconditionFailed},
			Body:          "",
		}, {
			Method:        "PUT",
			URL:           "http://admin:password@localhost:5984/cheops/_security",
			ExpectedCodes: []int{http.StatusOK},
			Body:          `{"members":{"roles":[]},"admins":{"roles":["_admin"]}}`,
		},
	}

	for _, req := range reqs {
		func() {
			httpReq, err := http.NewRequest(req.Method, req.URL, strings.NewReader(req.Body))
			if err != nil {
				log.Fatal(err)
			}

			if len(req.Body) > 0 {
				httpReq.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				log.Fatal(err)
			}

			for _, expectedCode := range req.ExpectedCodes {
				if resp.StatusCode == expectedCode {
					return
				}
			}

			log.Fatalf("Couldn't init database: method=%v body=[%v] url=%v err=%v\n", req.Method, req.Body, req.URL, fmt.Errorf(resp.Status))
		}()
	}
}

// ensureIndex makes sure that the _find call remains fast enough
// by indexing on the Locations field
func (r *Replicator) ensureIndex() {
	idx, err := http.Post("http://admin:password@localhost:5984/cheops/_index", "application/json", strings.NewReader(`{"index": {"fields": ["Locations"]}}`))
	if err != nil {
		log.Fatal(err)
	}
	if idx.StatusCode != http.StatusCreated && idx.StatusCode != http.StatusOK {
		log.Fatalf("Can't create index: %s\n", idx.Status)
	}
}

func (r *Replicator) Do(ctx context.Context, sites []string, id string, request CrdtUnit) (replies []ReplyDocument, err error) {

	// Prepare replies gathering before running the request
	// It's all asynchronous
	repliesChan := make(chan ReplyDocument)
	repliesCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.watchReplies(repliesCtx, request.RequestId, repliesChan)

	// Get current revision
	url := fmt.Sprintf("http://localhost:5984/cheops/%s", id)
	doc, err := http.Get(url)
	if err != nil {
		return replies, err
	}
	defer doc.Body.Close()

	var d ResourceDocument

	if doc.StatusCode == http.StatusNotFound {
		d = ResourceDocument{
			Id:        id,
			Locations: sites,
			Units:     make([]CrdtUnit, 0),
		}
	} else {
		err = json.NewDecoder(doc.Body).Decode(&d)
		if err != nil {
			return replies, err
		}
	}

	// Add our unit
	request.Generation = uint64(len(d.Units) + 1)
	d.Units = append(d.Units, request)
	sortUnits(d.Units)

	// Send the newly formatted document
	// We of course assume that the revision hasn't changed since the last Get, so this might fail.
	// In this case the user has to retry
	buf, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest("PUT", fmt.Sprintf("http://localhost:5984/cheops/%s", d.Id), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("Couldn't send request %#v to couchdb: %s", string(buf), resp.Status)
	}

	// Gather replies sent on the channel created at the beginning
	// of this function
	replies = make([]ReplyDocument, 0, len(sites))
wait:
	for i := 0; i < len(sites); i++ {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return nil, err
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
		// Hide locations for the reply, they're not useful to the caller
		for _, rep := range replies {
			rep.Locations = nil
		}
		return replies, nil
	}
	return nil, fmt.Errorf("No replies")
}

func (r *Replicator) watchRequests() {
	r.w.watch(func(j json.RawMessage) {
		var d ResourceDocument
		err := json.Unmarshal(j, &d)
		if err != nil {
			log.Printf("Couldn't decode %s", err)
			return
		}

		if len(d.Locations) == 0 {
			// CouchDB status message, discard
			return
		}
		if len(d.Units) == 0 {
			return
		}

		r.run(context.Background(), d)
	})
}

func (r *Replicator) watchReplies(ctx context.Context, requestId string, repliesChan chan ReplyDocument) {
	r.w.watch(func(j json.RawMessage) {
		var d ReplyDocument
		err := json.Unmarshal(j, &d)
		if err != nil {
			log.Printf("Couldn't decode: %s", err)
			return
		}

		if d.RequestId != requestId {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
			repliesChan <- d
		}
	})

}

func (r *Replicator) run(ctx context.Context, d ResourceDocument) {
	docs, err := r.getRepliesForId(d.Id)
	if err != nil {
		log.Printf("Couldn't get docs for id: %v\n", err)
		return
	}

	// index by requestid
	indexedReplies := make(map[string]struct{})
	for _, doc := range docs {
		indexedReplies[doc.RequestId] = struct{}{}
	}

	// find units to run
	// we run the first one for which we have no reply, and then all subsequent ones
	var firstToKeep int
	for i, unit := range d.Units {
		if _, ok := indexedReplies[unit.RequestId]; !ok {
			firstToKeep = i
			break
		}
	}
	bodies := make([]string, 0)
	for _, unit := range d.Units[firstToKeep:] {
		bodies = append(bodies, unit.Body)
	}

	log.Printf("applying %s\n", bodies)
	replies, err := backends.Handle(ctx, bodies)

	status := "OK"
	if err != nil {
		status = "KO"
	}

	cmds := make([]Cmd, 0)
	for i := range bodies {
		cmd := Cmd{
			Input:  bodies[i],
			Output: replies[i],
		}
		cmds = append(cmds, cmd)
	}

	firstUnitToRun := d.Units[firstToKeep]

	// Post document for replication
	newDoc := ReplyDocument{
		Locations:  d.Locations,
		Site:       env.Myfqdn,
		RequestId:  firstUnitToRun.RequestId,
		ResourceId: d.Id,
		Status:     status,
		Cmds:       cmds,
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

	if newresp.StatusCode != http.StatusCreated {
		log.Printf("Couldn't send reply: %v\n", newresp.Status)
		return
	}

	log.Printf("Ran %s %s\n", firstUnitToRun.RequestId, env.Myfqdn)
}

/*
func (r *Replicator) delete(ctx context.Context, p Payload) {
	log.Printf("Deleting %v\n", p.ResourceId)
	err := backends.DeleteKubernetes(ctx, []byte(p.Body))
	if err != nil {
		log.Printf("Couldn't delete %v: %v\n", p.ResourceId, err)
		return
	}

	// Find related reply
	selector := fmt.Sprintf(`{"Payload.ResourceId": "%s", "Payload.Method": "", "Payload.Site": "%s"}`, p.ResourceId, env.Myfqdn)
	docsToDelete, err := r.getDocsForSelector(selector)
	if err != nil {
		log.Printf("Couldn't find reply to delete for %v: %v\n", p.ResourceId, err)
		return
	}

	if len(docsToDelete) != 1 {
		log.Printf("Multiple replies to delete: %#v\n", docsToDelete)
		return
	}

	// Mark related reply as deleted
	docToDelete := docsToDelete[0]
	docToDelete.Deleted = true

	b, err := json.Marshal(docToDelete)
	if err != nil {
		log.Printf("Couldn't marshal: %v\n", err)
		return
	}

	url := fmt.Sprintf("http://localhost:5984/cheops/%s?rev=%s", docToDelete.Id, docToDelete.Rev)
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(b))
	if err != nil {
		log.Printf("Couldn't build request to delete %v: %v\n", docToDelete.Id, err)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Printf("Couldn't send request to delete %v: %v\n", docToDelete.Id, err)
		return
	}
	defer resp.Body.Close()

	var fail bool
	for _, expectedCode := range []int{http.StatusCreated, http.StatusAccepted} {
		if resp.StatusCode == expectedCode {
			fail = true
		}
	}
	if fail {
		log.Printf("Couldn't delete %v: %v\n", docToDelete.Id, resp.Status)
		return
	}
}
*/

func (r *Replicator) getRepliesForId(resourceId string) ([]ReplyDocument, error) {
	selector := fmt.Sprintf(`{"ResourceId": "%s"}`, resourceId)

	docs, err := r.getDocsForSelector(selector)
	if err != nil {
		return nil, err
	}
	out := make([]ReplyDocument, 0)
	for _, doc := range docs {
		var rep ReplyDocument
		err := json.Unmarshal(doc, &rep)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, nil
}

func (r *Replicator) getDocsForSelector(selector string) ([]json.RawMessage, error) {
	docs := make([]json.RawMessage, 0)
	var bookmark string

	for {
		selector := fmt.Sprintf(`{"bookmark": "%s", "selector": %s}`, bookmark, selector)

		current, err := http.Post("http://localhost:5984/cheops/_find", "application/json", strings.NewReader(selector))
		if err != nil {
			return nil, err
		}
		if current.StatusCode != 200 {
			return nil, fmt.Errorf("Post %s: %s", current.Request.URL.String(), current.Status)
		}

		var cr struct {
			Bookmark string            `json:"bookmark"`
			Docs     []json.RawMessage `json:"docs"`
		}

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

// replicate watches the _changes feed and makes sure the replication jobs
// are in place
func (r *Replicator) replicate() {

	r.w.watch(func(j json.RawMessage) {
		existingJobs := r.getExistingJobs()

		var d ResourceDocument
		err := json.Unmarshal(j, &d)
		if err != nil {
			log.Printf("Couldn't decode: %s", err)
			return
		}

		for _, location := range d.Locations {
			if location == env.Myfqdn {
				continue
			}

			// Replication of cheops -> cheops
			cheopsSite := fmt.Sprintf("http://%s:5984/cheops", location)
			if _, ok := existingJobs[cheopsSite]; !ok {
				body := fmt.Sprintf(`{"continuous": true, "source": "http://localhost:5984/cheops", "target": "http://%s:5984/cheops", "selector": {"Locations": {"$elemMatch": {"$eq": "%s"}}}}`, location, location)
				resp, err := http.Post("http://admin:password@localhost:5984/_replicate", "application/json", strings.NewReader(body))
				if err != nil {
					log.Printf("Couldn't add replication: %s\n", err)
				}
				if resp.StatusCode != 202 {
					log.Printf("Couldn't add replication: %s\n", resp.Status)
				}
			}
		}
	})
}

func createReplication(db, otherSite string) {
	body := fmt.Sprintf(`{"continuous": true, "source": "http://localhost:5984/%s", "target": "http://%s:5984/%s"}`, db, otherSite, db)
	resp, err := http.Post("http://admin:password@localhost:5984/_replicate", "application/json", strings.NewReader(body))
	if err != nil {
		log.Printf("Couldn't add replication: %s\n", err)
	}
	if resp.StatusCode != 202 {
		log.Printf("Couldn't add replication: %s\n", resp.Status)
	}
}

type DocChange struct {
	Seq string          `json:"seq"`
	Id  string          `json:"id"`
	Doc json.RawMessage `json:"doc"`
}

func (r *Replicator) getExistingJobs() map[string]struct{} {
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
