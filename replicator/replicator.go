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

	"cheops.com/env"
	"cheops.com/model"
	"github.com/goombaio/dag"
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
    "sites-for": {
      "map": "function (doc) {\n  if(doc.Type != 'OPERATION') return;\n  emit(doc.Payload.ResourceId, doc.Locations);\n}",
      "reduce": "function (keys, values, rereduce) {\n  return values[0];\n}"
    },
    "all-by-resourceid": {
      "map": "function (doc) {\n  if(doc.Type != 'OPERATION' && doc.Type != 'REPLY') return;\n  emit([doc.Payload.ResourceId, doc.Type], null);\n}",
      "reduce": "_count"
    },
    "all-by-targetid": {
      "map": "function (doc) {emit(doc.TargetId, null);\n}",
      "reduce": "_count"
    },
    "last-reply": {
      "map": "function (doc) {\n  if (doc.Type != 'REPLY') return;\n  emit([doc.Payload.Site, doc.TargetId], {Time: doc.Payload.ExecutionTime, RequestId: doc.TargetId, Sites: doc.Locations});\n}",
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

func (r *Replicator) getDocsForView(viewname string, keyArgs ...string) ([]model.PayloadDocument, error) {
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
			Doc model.PayloadDocument `json:"doc"`
		} `json:"rows"`
	}

	err = json.NewDecoder(res.Body).Decode(&cr)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	docs := make([]model.PayloadDocument, 0)
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
	r.w.watch(func(d model.PayloadDocument) {
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

type ErrorNotFound struct {
	ResourceId string
}

func (e ErrorNotFound) Error() string {
	return fmt.Sprintf("resource %v doesn't exist", e.ResourceId)
}

func (r *Replicator) SitesFor(resourceId string) (sites []string, err error) {
	url := fmt.Sprintf("http://localhost:5984/cheops/_design/cheops/_view/sites-for?key=%s", resourceId)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Get %s: %s", url, res.Status)
	}

	var cr struct {
		Rows []struct {
			Value []string `json:"value"`
		} `json:"rows"`
	}

	err = json.NewDecoder(res.Body).Decode(&cr)
	res.Body.Close()

	if len(cr.Rows) == 0 {
		return nil, err
	}
	return cr.Rows[0].Value, err
}

// Query will find the parents for the given id and replicate the request.
// Don't forget to install request and reply handlers before
// running this
func (r *Replicator) Send(ctx context.Context, sites []string, id string, payload json.RawMessage, typ string) error {
	parents := make([]string, 0)
	url := fmt.Sprintf(`http://localhost:5984/cheops/_design/cheops/_view/all-by-targetid?key="%s"&include_docs=true&reduce=false`, id)
	res, err := http.Get(url)
	if err != nil || res.StatusCode != 200 {
		return fmt.Errorf("Error with view all-by-targetid for %s: %s", id, err)
	}
	defer res.Body.Close()

	var cr struct {
		Rows []struct {
			Doc model.PayloadDocument `json:"doc"`
		} `json:"rows"`
	}

	err = json.NewDecoder(res.Body).Decode(&cr)
	if err != nil {
		return fmt.Errorf("Error with view all-by-targetid for %s: %s", id, err)
	}

	docs := make([]model.PayloadDocument, 0)
	for _, r := range cr.Rows {
		docs = append(docs, r.Doc)
	}
	tree := makeTree(docs)

	sv := tree.SinkVertices()
	for _, v := range sv {
		parents = append(parents, v.ID)
	}

	return r.postDocument(model.PayloadDocument{
		Locations: sites,
		Parents:   parents,
		Payload:   payload,
		TargetId:  id,
		Type:      typ,
	})
}

func makeTree(allDocs []model.PayloadDocument) (tree *dag.DAG) {
	tree = dag.NewDAG()

	for _, doc := range allDocs {
		var op model.Operation
		json.Unmarshal(doc.Payload, &op)
		// First pass: build all vertices only
		tree.AddVertex(dag.NewVertex(op.RequestId, doc.Parents))
	}

	// Second pass: build all edges
	// Every vertex is a sink vertex because there are no edges yet
	for _, vertex := range tree.SinkVertices() {
		parents := vertex.Value.([]string)
		for _, parent := range parents {
			parentVertex, _ := tree.GetVertex(parent)
			tree.AddEdge(parentVertex, vertex)
		}
	}

	return tree
}
