// package replicator is responsible for managing the replication of operations
// such that they are run as desired everywhere they are supposed to.
//
// To understand the ideas, please see CONSISTENCY.md at the root of the project

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
    "all-by-resourceid": {
      "map": "function (doc) {\n  if(doc.Type != 'RESOURCE' && doc.Type != 'REPLY') return;\n  emit([doc.ResourceId, doc.Type], null);\n}",
      "reduce": "_count"
    },
    "by-resource": {
      "map": "function (doc) {\n  if(doc.Type != 'RESOURCE') return;\n  emit(null, null);\n}",
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
func (r *Replicator) Do(ctx context.Context, sites []string, id string, request model.Operation) (replies chan model.ReplyDocument, err error) {

	repliesChan := make(chan model.ReplyDocument)
	done := func() {
		close(repliesChan)
	}

	if len(request.Command.Command) > 0 {
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
	docs, err := r.getResourceDocsFor(id)
	if err != nil {
		return replies, err
	}
	var localDoc model.ResourceDocument
	for _, doc := range docs {
		if doc.Site == env.Myfqdn {
			localDoc = doc
			break
		}
	}

	if localDoc.ResourceId == "" {
		if len(request.Command.Command) == 0 {
			return nil, ErrInvalidRequest("will not create a document with an empty body")
		}

		localDoc = model.ResourceDocument{
			Locations:  sites,
			Operations: make([]model.Operation, 0),
			Type:       "RESOURCE",
			Site:       env.Myfqdn,
			ResourceId: id,
		}
	}

	// Add our operation if needed
	request.KnownState = make(map[string]int)
	for _, location := range sites {
		height := 0
		for _, document := range docs {
			if document.Site == location {
				height = len(document.Operations)
			}
		}
		if location == env.Myfqdn {
			height = height + 1
		}
		request.KnownState[location] = height
	}
	localDoc.Operations = append(localDoc.Operations, request)
	log.Printf("New request: resourceId=%v requestId=%v\n", localDoc.ResourceId, request.RequestId)

	// Send the newly formatted document
	// We of course assume that the revision hasn't changed since the last Get, so this might fail.
	// In this case the user has to retry
	buf, err := json.Marshal(localDoc)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest("POST", "http://localhost:5984/cheops", bytes.NewReader(buf))
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

	// location -> struct{}{}
	expected := make(map[string]struct{})
	for _, location := range localDoc.Locations {
		expected[location] = struct{}{}
	}
	ret := make(chan model.ReplyDocument)

	go func() {
		defer func() {
			done()
			close(ret)
		}()

		for i := 0; i < len(expected); i++ {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Printf("Error with runing %s: %s\n", request.RequestId, err)
					return
				}
			case reply := <-repliesChan:
				delete(expected, reply.Site)
				ret <- reply
			case <-time.After(20 * time.Second):
				// timeout
				for remaining := range expected {
					ret <- model.ReplyDocument{
						Locations:  localDoc.Locations,
						Site:       remaining,
						RequestId:  request.RequestId,
						ResourceId: localDoc.ResourceId,
						Status:     "TIMEOUT",
						Cmd: model.Cmd{
							Input: request.Command.Command,
						},
						Type: "REPLY",
					}
				}
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
	if len(d.Operations) == 0 {
		log.Printf("WARN: Resource %v has been inserted with no operations\n", d.ResourceId)
		return
	}

	allDocs, err := r.getAllDocsFor(d.ResourceId)
	if err != nil {
		log.Printf("Couldn't get docs for id: %v\n", err)
		return
	}

	resourceDocuments := make([]model.ResourceDocument, 0)
	replies := make([]model.ReplyDocument, 0)
	for _, doc := range allDocs {
		var reply model.ReplyDocument
		err := json.Unmarshal(doc, &reply)
		if err == nil && reply.Type == "REPLY" {
			replies = append(replies, reply)
		} else {
			var resource model.ResourceDocument
			err = json.Unmarshal(doc, &resource)
			if err == nil && reply.Type == "RESOURCE" {
				resourceDocuments = append(resourceDocuments, resource)
			} else {
				log.Printf("Invalid doc: %s\n", doc)
				return
			}
		}
	}

	operationsToRun := findOperationsToRun(env.Myfqdn, resourceDocuments, replies)
	if len(operationsToRun) == 0 {
		return
	}

	commands := make([]backends.ShellCommand, 0)
	for _, operation := range operationsToRun {
		commands = append(commands, operation.Command)
		log.Printf("will run %s\n", operation.RequestId)
	}

	executionReplies, err := backends.Handle(ctx, commands)

	status := "OK"
	if err != nil {
		status = "KO"
	}

	// Post reply for replication
	for i, operation := range operationsToRun {
		log.Printf("Ran %s\n", operation.RequestId)

		cmd := model.Cmd{
			Input:  commands[i].Command,
			Output: executionReplies[i],
		}

		err = r.postDocument(model.ReplyDocument{
			Locations:  d.Locations,
			Site:       env.Myfqdn,
			RequestId:  operation.RequestId,
			ResourceId: d.ResourceId,
			Status:     status,
			Cmd:        cmd,
			Type:       "REPLY",
		})
		if err != nil {
			log.Println(err)
		}
	}
}

func (r *Replicator) getResourceDocsFor(resourceId string) ([]model.ResourceDocument, error) {
	docs, err := r.getDocsForView("all-by-resourceid", resourceId, "RESOURCE")
	resourceDocuments := make([]model.ResourceDocument, 0)
	for _, d := range docs {
		var doc model.ResourceDocument
		err := json.Unmarshal(d, &doc)
		if err == nil && doc.Type == "RESOURCE" {
			resourceDocuments = append(resourceDocuments, doc)
		}
	}

	return resourceDocuments, err
}

func (r *Replicator) getAllDocsFor(resourceId string) ([]json.RawMessage, error) {
	return r.getDocsForView("all-by-resourceid", resourceId)
}

func (r *Replicator) getDocsForView(viewname string, keyArgs ...string) ([]json.RawMessage, error) {
	endkey := make([]string, len(keyArgs))
	copy(endkey, keyArgs)
	endkey = append(endkey, "\uffff")

	query := struct {
		StartKey []string `json:"start_key"`
		EndKey   []string `json:"end_key"`
	}{
		StartKey: keyArgs,
		EndKey:   endkey,
	}
	b, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:5984/cheops/_design/cheops/_view/%s?reduce=false&include_docs=true", viewname)
	res, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Post %s: %s", res.Request.URL.String(), res.Status)
	}

	var cr struct {
		Rows []struct {
			Doc json.RawMessage `json:"doc"`
		} `json:"rows"`
	}

	err = json.NewDecoder(res.Body).Decode(&cr)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	docs := make([]json.RawMessage, 0)
	for _, row := range cr.Rows {
		docs = append(docs, row.Doc)
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
