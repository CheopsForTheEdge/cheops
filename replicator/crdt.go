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
	"cheops.com/model"
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
//
// The output is a chan of each individual reply as they arrive. After a timeout or all replies are sent, the chan is closed
func (r *Replicator) Do(ctx context.Context, sites []string, id string, request model.CrdtUnit) (replies chan model.ReplyDocument, err error) {

	repliesChan := make(chan model.ReplyDocument)
	done := func() {
		close(repliesChan)
	}

	if len(request.Body) > 0 {
		// Prepare replies gathering before running the request
		// It's all asynchronous
		var repliesCtx context.Context
		repliesCtx, cancel := context.WithCancel(ctx)
		done = func() {
			cancel()
			close(repliesChan)
		}
		r.watchReplies(repliesCtx, request.RequestId, repliesChan)
	}

	// Get current revision
	url := fmt.Sprintf("http://localhost:5984/cheops/%s", id)
	doc, err := http.Get(url)
	if err != nil {
		return replies, err
	}
	defer doc.Body.Close()

	var d model.ResourceDocument

	// Filled in case of migration
	var deletedSites []string
	var newSites []string
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

		d = model.ResourceDocument{
			Id:        id,
			Locations: sites,
			Units:     make([]model.CrdtUnit, 0),
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

			for _, new := range sites {
				remains := false
				for _, old := range d.Locations {
					if old == new {
						remains = true
						break
					}
				}
				if !remains {
					newSites = append(newSites, new)
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
		model.SortUnits(d.Units)
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
		r.postDocument(model.DeleteDocument{
			ResourceId:  id,
			ResourceRev: currentRev,
			Locations:   []string{oldSite},
			Type:        "DELETE",
		})
	}

	var expected int
	if len(request.Body) > 0 {
		expected = len(d.Locations)
	} else {
		expected = len(newSites)
	}

	ret := make(chan model.ReplyDocument)

	go func() {
		defer func() {
			done()
			close(ret)
		}()

		for i := 0; i < expected; i++ {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Printf("Error with runing %s: %s\n", request.RequestId, err)
					return
				}
			case reply := <-repliesChan:
				ret <- reply
			case <-time.After(20 * time.Second):
				// timeout
				//
				// Because there are multiple cases, let's leave it like that,
				// some goroutines will wait for nothing, that's alright
				return
			}
		}
	}()

	return ret, nil
}

func (r *Replicator) watchRequests() {
	r.w.watch(func(j json.RawMessage) {
		var d model.ResourceDocument
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

func (r *Replicator) watchReplies(ctx context.Context, requestId string, repliesChan chan model.ReplyDocument) {
	r.w.watch(func(j json.RawMessage) {
		var d model.ReplyDocument
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

func (r *Replicator) run(ctx context.Context, d model.ResourceDocument) {
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

	type UnitToRun struct {
		model.CrdtUnit
		alreadyRan bool
	}

	// find units to run
	// we run the first one for which we have no reply,
	// and then all subsequent ones to be sure that everything is run in the same order
	//
	// We also need to remember all units that we run that weren't before, so that we can
	// mark them as ran
	unitsToRun := make([]UnitToRun, 0)
	for _, unit := range d.Units {
		toAdd := UnitToRun{
			CrdtUnit: unit,
		}
		_, ok := indexedReplies[unit.RequestId]
		toAdd.alreadyRan = ok

		// If the list is empty and the unit wasn't already ran, add it
		// Otherwise if there already is something, add them all
		if !toAdd.alreadyRan && len(unitsToRun) == 0 || len(unitsToRun) > 0 {
			unitsToRun = append(unitsToRun, toAdd)
		}

	}
	bodies := make([]string, 0)
	for _, unit := range unitsToRun {
		bodies = append(bodies, unit.Body)
		log.Printf("will apply [%s]\n", unit.Body)
	}

	replies, err := backends.Handle(ctx, bodies)

	status := "OK"
	if err != nil {
		status = "KO"
	}

	// Post reply for replication
	for i, unit := range unitsToRun {
		log.Printf("Ran %s %s\n", unit.RequestId, env.Myfqdn)

		if unit.alreadyRan {
			continue
		}
		cmd := model.Cmd{
			Input:  bodies[i],
			Output: replies[i],
		}

		err = r.postDocument(model.ReplyDocument{
			Locations:  d.Locations,
			Site:       env.Myfqdn,
			RequestId:  unit.RequestId,
			ResourceId: d.Id,
			Status:     status,
			Cmd:        cmd,
			Type:       "REPLY",
		})
		if err != nil {
			log.Println(err)
		}
	}
}

func (r *Replicator) getRepliesForId(resourceId string) ([]model.ReplyDocument, error) {
	selector := fmt.Sprintf(`{"Type": "REPLY", "ResourceId": "%s", "Site": "%s"}`, resourceId, env.Myfqdn)

	docs, err := r.getDocsForSelector(selector)
	if err != nil {
		return nil, err
	}
	out := make([]model.ReplyDocument, 0)
	for _, doc := range docs {
		var rep model.ReplyDocument
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

	// Use a function so we can defer
	manageReplications := func(locations []string) {
		existingJobs := r.getExistingReplications()
		for _, location := range locations {
			if location == env.Myfqdn {
				continue
			}

			cheopsSite := fmt.Sprintf("http://%s:5984/cheops/", location)
			if _, ok := existingJobs[cheopsSite]; ok {
				continue
			}

			// Replication doesn't exist, create it
			body := fmt.Sprintf(`{"continuous": true, "source": "http://localhost:5984/cheops/", "target": "%s", "selector": {"Locations": {"$elemMatch": {"$eq": "%s"}}}}`, cheopsSite, location)

			resp, err := http.Post("http://admin:password@localhost:5984/_replicator", "application/json", strings.NewReader(body))
			defer resp.Body.Close()

			if err != nil {
				log.Printf("Couldn't add replication: %s\n", err)
			}
			if resp.StatusCode != 201 {
				log.Printf("Couldn't add replication: %s\n", resp.Status)

			}
		}
	}

	// Install replication if it's new
	r.w.watch(func(j json.RawMessage) {
		var d model.ResourceDocument
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

func (r *Replicator) getExistingReplications() map[string]struct{} {
	existingReplications, err := http.Get("http://admin:password@localhost:5984/_scheduler/docs")
	if err != nil {
		log.Fatal(err)
	}
	defer existingReplications.Body.Close()

	if existingReplications.StatusCode != 200 {
		log.Fatal(fmt.Errorf("Can't get existing replication docs: %s", existingReplications.Status))
	}

	var js Replications
	err = json.NewDecoder(existingReplications.Body).Decode(&js)
	if err != nil {
		log.Fatalf("Couldn't decode: %s", err)
	}

	ret := make(map[string]struct{})
	for _, j := range js.Replications {
		ret[j.Target] = struct{}{}
	}

	return ret
}

type Replications struct {
	Replications []Replication `json:"docs"`
}

type Replication struct {
	Target string `json:"target"`
}
