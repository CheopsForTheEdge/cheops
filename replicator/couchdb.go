package replicator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"cheops.com/env"
	"cheops.com/model"
)

// Not working yet
/*
func (r *Replicator) Get(id string) (model.ResourceDocument, error) {
	u := fmt.Sprintf("http://localhost:5984/cheops/%s", id)
	resp, err := http.Get(u)
	if err != nil {
		return model.ResourceDocument{}, fmt.Errorf("Couldn't get %s: %v", id, err)
	}
	defer resp.Body.Close()

	var d model.ResourceDocument
	err = json.NewDecoder(resp.Body).Decode(&d)
	if err != nil {
		return model.ResourceDocument{}, fmt.Errorf("Couldn't get %s: %v", id, err)
	}
	return d, nil
}
*/

func (r *Replicator) postDocument(v interface{}) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("Couldn't marshal reply: %v\n", err)
	}
	newresp, err := http.Post("http://localhost:5984/cheops", "application/json", bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("Couldn't send reply: %v\n", err)
	}
	defer newresp.Body.Close()

	if newresp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Couldn't send reply: %v\n", newresp.Status)
	}

	return nil
}

func (r *Replicator) putDocument(v interface{}, id string) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("Couldn't marshal document: %v\n", err)
	}
	url := fmt.Sprintf("http://localhost:5984/cheops/%s", id)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("Couldn't create document request: %v\n", err)
	}
	newresp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Couldn't send document: %v\n", err)
	}
	defer newresp.Body.Close()

	if newresp.StatusCode != http.StatusCreated && newresp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Couldn't send document: %v\n", newresp.Status)
	}

	return nil
}

// Count returns the number of resources known by this node
func (r *Replicator) CountResources() (int, error) {
	byResourceResp, err := http.Get("http://admin:password@localhost:5984/cheops/_design/cheops/_view/all-by-resourceid?group_level=1")
	if err != nil {
		return 0, fmt.Errorf("Error running by-resource view: %v\n", err)
	}
	defer byResourceResp.Body.Close()

	if byResourceResp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Error running by-resource view: status is %v\n", byResourceResp.Status)
	}

	type byResource struct {
		Rows []struct{} `json:"rows"`
	}

	var resp byResource
	err = json.NewDecoder(byResourceResp.Body).Decode(&resp)
	return len(resp.Rows), err
}

func (r *Replicator) GetOrderedReplies(id string) (map[string][]model.ReplyDocument, error) {
	var docs []json.RawMessage
	docs, err := r.getDocsForView("last-reply", env.Myfqdn, id)
	if err != nil {
		return nil, fmt.Errorf("Error running last-reply view: %v\n", err)
	}

	m := make(map[string][]model.ReplyDocument)
	for _, doc := range docs {
		var d model.ReplyDocument
		err := json.Unmarshal(doc, &d)
		if err != nil {
			return nil, fmt.Errorf("Invalid resource document: %v\n", err)
		}
		if _, ok := m[d.ResourceId]; !ok {
			m[d.ResourceId] = make([]model.ReplyDocument, 0)
		}
		m[d.ResourceId] = append(m[d.ResourceId], d)
	}

	return m, nil
}
