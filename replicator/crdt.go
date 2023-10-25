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

var (
	ErrDoesNotExist error = fmt.Errorf("doesn't exist")
)

type ErrInvalidRequest string

func (e ErrInvalidRequest) Error() string { return string(e) }

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
			Method:        "PUT",
			URL:           "http://admin:password@localhost:5984/cheops",
			ExpectedCodes: []int{http.StatusCreated, http.StatusPreconditionFailed},
			Body:          "",
		}, {
			Method:        "PUT",
			URL:           "http://admin:password@localhost:5984/cheops/_security",
			ExpectedCodes: []int{http.StatusOK},
			Body:          `{"members":{"roles":[]},"admins":{"roles":["_admin"]}}`,
		}, {
			Method:        "PUT",
			URL:           "http://admin:password@localhost:5984/cheops/_design/cheops",
			ExpectedCodes: []int{http.StatusCreated, http.StatusConflict},
			Body: `
{
  "views": {
    "by-location": {
      "map": "function (doc) {\n  for (const location of doc.Locations) {\n    emit(location, null);\n  }\n}",
      "reduce": "_count"
    }
  },
  "language": "javascript"
}`,
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

// Do handles the request such that it is properly replicated and propagated.
// If the resource doesn't exist, it will be created if the list of sites is not nil or empty; if there are no sites, an ErrDoesNotExist is returned.
// If the resource already exists and the list of sites is not nil or empty, it will be updated with the desired sites.
//
// If the request has an empty body, it means sites are expected to change. In that case we don't wait for replies from other sites.
// If the resource doesn't already exist, an ErrInvalidRequest is returned
func (r *Replicator) Do(ctx context.Context, sites []string, id string, request CrdtUnit) (replies []ReplyDocument, err error) {

	var repliesChan chan ReplyDocument

	if len(request.Body) > 0 {
		// Prepare replies gathering before running the request
		// It's all asynchronous
		repliesChan = make(chan ReplyDocument)
		repliesCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		r.watchReplies(repliesCtx, request.RequestId, repliesChan)
	}

	// Get current revision
	url := fmt.Sprintf("http://localhost:5984/cheops/%s", id)
	doc, err := http.Get(url)
	if err != nil {
		return replies, err
	}
	defer doc.Body.Close()

	var d ResourceDocument

	// Filled in case of migration
	var deletedSites []string
	var currentRev string

	if doc.StatusCode == http.StatusNotFound {
		if len(request.Body) == 0 {
			return nil, ErrInvalidRequest("will not create a document with an empty body")
		}

		if len(sites) == 0 {
			// We are asked to create a document but with no sites: this is invalid, the caller needs to specify
			// where the resource is supposed to be
			return nil, ErrDoesNotExist
		}

		d = ResourceDocument{
			Id:        id,
			Locations: sites,
			Units:     make([]CrdtUnit, 0),
			Type:      "RESOURCE",
		}
	} else {
		err = json.NewDecoder(doc.Body).Decode(&d)
		if err != nil {
			return replies, err
		}

		if len(sites) > 0 {
			deletedSites = make([]string, 0)
			for _, old := range d.Locations {
				remains := false
				for _, new := range sites {
					if old == new {
						remains = true
						break
					}
				}
				if !remains {
					deletedSites = append(deletedSites, old)
				}
			}

			d.Locations = sites

			currentRev = d.Rev
		}
	}

	// Add our unit if needed
	if len(request.Body) > 0 {
		request.Generation = uint64(len(d.Units) + 1)
		d.Units = append(d.Units, request)
		sortUnits(d.Units)
	}

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

	// Send information to delete
	for _, oldSite := range deletedSites {
		r.postDocument(DeleteDocument{
			ResourceId:  id,
			ResourceRev: currentRev,
			Locations:   []string{oldSite},
			Type:        "DELETE",
		})
	}

	if len(request.Body) == 0 {
		return nil, nil
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
	log.Printf("No replies for %s\n", request.RequestId)
	return nil, nil
}

func (r *Replicator) watchRequests() {
	r.w.watch(func(j json.RawMessage) {
		var d ResourceDocument
		err := json.Unmarshal(j, &d)
		if err != nil {
			log.Printf("Couldn't decode %s", err)
			return
		}

		if d.Type != "RESOURCE" {
			return
		}

		forMe := false
		for _, location := range d.Locations {
			if location == env.Myfqdn {
				forMe = true
			}
		}
		if !forMe {
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

		if d.Type != "REPLY" {
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
		log.Printf("will apply [%s]\n", unit.Body)
	}

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

	// Post reply for replication
	err = r.postDocument(ReplyDocument{
		Locations:  d.Locations,
		Site:       env.Myfqdn,
		RequestId:  firstUnitToRun.RequestId,
		ResourceId: d.Id,
		Status:     status,
		Cmds:       cmds,
		Type:       "REPLY",
	})
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Ran %s %s\n", firstUnitToRun.RequestId, env.Myfqdn)
}

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

	manageReplications := func(locations []string) {
		existingJobs := r.getExistingJobs()
		for _, location := range locations {
			if location == env.Myfqdn {
				continue
			}

			var body string
			cheopsSite := fmt.Sprintf("http://%s:5984/cheops", location)
			if _, ok := existingJobs[cheopsSite]; !ok {
				// Replication doesn't exist, create it
				body = fmt.Sprintf(`{"continuous": true, "source": "http://localhost:5984/cheops", "target": "http://%s:5984/cheops", "selector": {"Locations": {"$elemMatch": {"$eq": "%s"}}}}`, location, location)
			}

			resp, err := http.Post("http://admin:password@localhost:5984/_replicate", "application/json", strings.NewReader(body))
			if err != nil {
				log.Printf("Couldn't add replication: %s\n", err)
			}
			if resp.StatusCode != 202 {
				log.Printf("Couldn't add replication: %s\n", resp.Status)
			}
		}
	}

	// Re-install replication if it's not there
	// If the target is not accessible couchdb deletes the replication,
	// so we have to sometimes recreate it
	go func() {
		for _ = range time.Tick(5 * time.Minute) {
			// Anonymous function to make sure defer works
			func() {
				allTagsResp, err := http.Get("http://localhost:5984/cheops/_design/cheops/_view/by-location?group=true")
				if err != nil {
					log.Printf("Error getting by-location view: %v\n", err)
					return
				}
				if allTagsResp.StatusCode != http.StatusOK {
					log.Printf("Error getting by-location view: %v\n", allTagsResp.Status)
					return
				}

				type Row struct {
					Key string `json:"key"`

					// We don't care about the value
				}
				var allTags struct {
					Rows []Row `json:"rows"`
				}

				defer allTagsResp.Body.Close()
				err = json.NewDecoder(allTagsResp.Body).Decode(&allTags)
				if err != nil {
					log.Printf("Error getting by-location view: %v\n", err)
					return
				}

				locations := make([]string, 0)
				for _, row := range allTags.Rows {
					locations = append(locations, row.Key)
				}
				manageReplications(locations)
			}()

		}
	}()

	// Install replication if it's new
	r.w.watch(func(j json.RawMessage) {
		var d ResourceDocument
		err := json.Unmarshal(j, &d)
		if err != nil {
			log.Printf("Couldn't decode: %s", err)
			return
		}

		manageReplications(d.Locations)
	})
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
