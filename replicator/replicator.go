// package replicator is responsible for managing the replication of operations
// such that they are run as desired everywhere they are supposed to.
//
// To understand the ideas, please see CONSISTENCY.md at the root of the project

package replicator

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	_ "golang.org/x/crypto/blake2b"

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
      "map": "function (doc) {\n  if(doc.Type === 'RESOURCE') {emit(null, null); return}\n  if (doc.Type === 'REPLY') {emit(null, null);return} }",
      "reduce": "_count"
    },
    "last-reply": {
      "map": "function (doc) {\n  if (doc.Type != 'REPLY') return;\n  emit([doc.Site, doc.Id], {Time: doc.ExecutionTime, RequestId: doc.RequestId, Sites: doc.Locations});\n}",
      "reduce": "function (keys, values, rereduce) {\n  let sorted = values.sort((a, b) => {\n    return a.Time.localeCompare(b.Time)\n  })\n  return sorted[sorted.length - 1]\n}"
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
	doc, err := r.getResourceDocFor(id)
	if err != nil {
		return replies, err
	}

	if doc.Id == "" {
		if len(request.Command.Command) == 0 {
			return nil, ErrInvalidRequest("will not create a document with an empty body")
		}

		doc = model.ResourceDocument{
			Locations:  sites,
			Operations: make([]model.Operation, 0),
			Type:       "RESOURCE",
			Id:         id,
		}
	}

	doc.Operations = append(doc.Operations, request)
	log.Printf("New request: resourceId=%v requestId=%v\n", doc.Id, request.RequestId)

	// Send the newly formatted document
	// We of course assume that the revision hasn't changed since the last Get, so this might fail.
	// In this case the user has to retry
	buf, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("http://localhost:5984/cheops/%s", id)
	httpReq, err := http.NewRequest("PUT", url, bytes.NewReader(buf))
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
	for _, location := range doc.Locations {
		expected[location] = struct{}{}
	}
	ret := make(chan model.ReplyDocument)

	go func() {
		defer func() {
			done()
			close(ret)
		}()

		for len(expected) > 0 {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Printf("Error with runing %s: %s\n", request.RequestId, err)
					return
				}
			case reply := <-repliesChan:
				ret <- reply
				delete(expected, reply.Site)
			case <-time.After(20 * time.Second):
				// timeout
				for remaining := range expected {
					ret <- model.ReplyDocument{
						Locations:  doc.Locations,
						Site:       remaining,
						RequestId:  request.RequestId,
						ResourceId: doc.Id,
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

		if !r.merge(context.Background(), d.Id) {
			r.run(context.Background(), d)
		}
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

// merge will merge conflicts for resource "id"
// It retuns true if something was merged
func (r *Replicator) merge(ctx context.Context, id string) bool {

	hasmerged := false

loop:
	for {
		url := fmt.Sprintf("http://localhost:5984/cheops/%s?conflicts=true", id)
		res, err := http.Get(url)
		if err != nil {
			log.Printf("Couldn't get doc with conflicts for %s: %v\n", id, err)
			<-time.After(10 * time.Second)
			continue
		}
		var d model.ResourceDocument
		err = json.NewDecoder(res.Body).Decode(&d)
		res.Body.Close()

		if err != nil {
			log.Printf("Couldn't get doc with conflicts for %s: %v\n", id, err)
			<-time.After(10 * time.Second)
			continue
		}

		if len(d.Conflicts) == 0 {
			return false
		}

		conflicts := make([]model.ResourceDocument, 0)
		for _, rev := range d.Conflicts {
			url := fmt.Sprintf("http://localhost:5984/cheops/%s?rev=%s", id, rev)
			res, err := http.Get(url)
			if err != nil {
				log.Printf("Couldn't get doc=%s rev=%s: %v\n", id, rev, err)
				<-time.After(10 * time.Second)
				continue loop
			}
			var d model.ResourceDocument
			err = json.NewDecoder(res.Body).Decode(&d)
			if err != nil {
				log.Printf("Bad json document: doc=%s rev=%s %s\n", id, rev, err)
				<-time.After(10 * time.Second)
				continue loop
			}
			res.Body.Close()
			conflicts = append(conflicts, d)
		}

		resolved, err := resolveMerge(d, conflicts)
		if err != nil {
			log.Printf("Couldn't merge conflicts for %s: %v\n", id, err)
			<-time.After(10 * time.Second)
			continue
		}

		for _, rev := range d.Conflicts {
			r.deleteDocument(resolved.Id, rev)
		}

		err = r.putDocument(resolved, resolved.Id)
		if err != nil {
			log.Printf("Couldn't put resolution document for %s: %v\n", id, err)
			<-time.After(10 * time.Second)
			continue
		}

		hasmerged = true
		break
	}

	return hasmerged
}

func resolveMerge(main model.ResourceDocument, conflicts []model.ResourceDocument) (resolved model.ResourceDocument, err error) {

	// Find winning config, we take the higher one
	hash := func(c model.Config) []byte {
		h := crypto.BLAKE2b_256.New()
		json.NewEncoder(h).Encode(c)
		return h.Sum(nil)
	}

	c := main.Config
	h := hash(c)
	for _, conflict := range conflicts {
		cc := conflict.Config
		if len(c.RelationshipMatrix) == 0 {
			c = cc
			continue
		}
		if len(cc.RelationshipMatrix) == 0 {
			continue
		}
		hh := hash(cc)
		if bytes.Compare(hh, h) > 0 {
			c = cc
			h = hh
		}
	}

	main.Config = c

	ops := main.Operations
	for _, conflict := range conflicts {
		// TODO: how to determine the main Config ?
		// For now it's the one that couchdb gives us

		// Find first op
		// We take the associated config as well
		hasRelationship := false
		for _, relationship := range main.Config.RelationshipMatrix {
			if relationship.Before == ops[0].Type && relationship.After == conflict.Operations[0].Type {
				hasRelationship = true
				if slicesEqual(relationship.Result, []int{1}) {
					ops[0] = ops[0] // duh
				} else if slicesEqual(relationship.Result, []int{2}) {
					if ops[0].Type == conflict.Operations[0].Type {
						// We actually have a conflict of the same operation. We don't just take the
						// second one but the highest to be deterministic
						if strings.Compare(ops[0].RequestId, conflict.Operations[0].RequestId) < 0 {
							ops[0] = conflict.Operations[0]
						}
					} else {
						ops[0] = conflict.Operations[0]
					}
				} else {
					ops = append(ops, model.Operation{})
					copy(ops[2:], ops[1:])
					ops[1] = conflict.Operations[0]
					break
				}
			}
		}
		if !hasRelationship {
			// no relationship: it's all commutative
			// Take them all and sort them (to make it deterministic)
			ops = append(ops, conflict.Operations[0])
			sort.Slice(ops[:2], func(i, j int) bool {
				return strings.Compare(ops[:2][i].RequestId, ops[:2][j].RequestId) <= 0
			})
		}

		// Add rest of ops
		for _, op := range conflict.Operations[1:] {
			hasop := false
			for _, existingop := range ops {
				if op.RequestId == existingop.RequestId {
					hasop = true
					break
				}
			}
			if !hasop {
				ops = append(ops, op)
			}
		}
	}
	main.Operations = ops
	return main, nil
}

func slicesEqual(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (r *Replicator) run(ctx context.Context, d model.ResourceDocument) {

	if len(d.Operations) == 0 {
		log.Printf("WARN: Resource %v has been inserted with no operations\n", d.Id)
		return
	}

	allDocs, err := r.getAllDocsFor(d.Id)
	if err != nil {
		log.Printf("Couldn't get docs for id: %v\n", err)
		return
	}

	var resourceDocument model.ResourceDocument
	replies := make([]model.ReplyDocument, 0)
	for _, doc := range allDocs {
		var reply model.ReplyDocument
		err := json.Unmarshal(doc, &reply)
		if err == nil && reply.Type == "REPLY" {
			replies = append(replies, reply)
		} else {
			err = json.Unmarshal(doc, &resourceDocument)
			if err != nil {
				log.Printf("Invalid doc: %s\n", doc)
				return
			}
		}
	}

	commands := make([]backends.ShellCommand, 0)
	commandsToResource := make(map[int]int)
	commandsInt := 0

operations:
	for i, operation := range resourceDocument.Operations {
		for _, reply := range replies {
			if operation.RequestId == reply.RequestId && reply.Site == env.Myfqdn {
				continue operations
			}
		}
		commands = append(commands, operation.Command)
		commandsToResource[commandsInt] = i
		commandsInt = commandsInt + 1
		log.Printf("will run %s\n", operation.RequestId)
	}

	executionReplies, err := backends.Handle(ctx, commands)

	status := "OK"
	if err != nil {
		status = "KO"
	}

	// Post reply for replication
	for i, command := range commands {

		cmd := model.Cmd{
			Input:  command.Command,
			Output: executionReplies[i],
		}

		err = r.postDocument(model.ReplyDocument{
			Locations:     d.Locations,
			Site:          env.Myfqdn,
			RequestId:     resourceDocument.Operations[commandsToResource[i]].RequestId,
			ResourceId:    d.Id,
			Status:        status,
			Cmd:           cmd,
			Type:          "REPLY",
			ExecutionTime: time.Now(),
		})
		if err != nil {
			log.Println(err)
		}
	}
}

// getResourceDocFor gets the document for the given resource
// It will wait until conflicts are resolved. Conflict resolution is expected to happen
// in another goroutine
func (r *Replicator) getResourceDocFor(resourceId string) (model.ResourceDocument, error) {
	tries := 10
	for {
		if tries == 0 {
			return model.ResourceDocument{}, fmt.Errorf("Waited too long for %s to resolve merge, aborting\n", resourceId)
		}
		url := fmt.Sprintf("http://localhost:5984/%s?conflicts=true", resourceId)
		res, err := http.Get(url)
		if err != nil {
			tries = tries - 1
			<-time.After(1 * time.Second)
			continue
		}
		defer res.Body.Close()

		var doc model.ResourceDocument
		err = json.NewDecoder(res.Body).Decode(&doc)

		if len(doc.Conflicts) > 0 {
			log.Printf("%v has conflict, waiting 1s for resolution\n", resourceId)
			tries = tries - 1
			<-time.After(1 * time.Second)
			continue
		}
		return doc, nil
	}
}

func (r *Replicator) getAllDocsFor(resourceId string) ([]json.RawMessage, error) {
	return r.getDocsForView("all-by-resourceid", resourceId)
}

func (r *Replicator) getDocsForView(viewname string, keyArgs ...string) ([]json.RawMessage, error) {
	startkey := make([]string, 0)
	endkey := make([]string, 0)
	for _, arg := range keyArgs {
		if arg != "" {
			startkey = append(startkey, arg)
			endkey = append(endkey, arg)
		}
	}
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
