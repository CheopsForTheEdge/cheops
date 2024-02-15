package replicator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"cheops.com/model"
)

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

// Count returns the number of resources known by this node
func (r *Replicator) Count() (int, error) {
	byResourceResp, err := http.Get("http://admin:password@localhost:5984/cheops/_design/cheops/_view/by-resource")
	if err != nil {
		return 0, fmt.Errorf("Error running by-resource view: %v\n", err)
	}
	defer byResourceResp.Body.Close()

	if byResourceResp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Error running by-resource view: status is %v\n", byResourceResp.Status)
	}

	type byResource struct {
		Rows []struct {
			Value int `json:"value"`
		} `json:"rows"`
	}

	var resp byResource
	err = json.NewDecoder(byResourceResp.Body).Decode(&resp)
	if len(resp.Rows) == 0 {
		return 0, nil
	}
	if len(resp.Rows) != 1 {
		return 0, fmt.Errorf("Bad reply: %#v\n", resp)
	}
	return resp.Rows[0].Value, err
}

func (r *Replicator) GetResources() ([]model.ResourceDocument, error) {
	byResourceResp, err := http.Get("http://admin:password@localhost:5984/cheops/_design/cheops/_view/by-resource?reduce=false&include_docs=true")
	if err != nil {
		return nil, fmt.Errorf("Error running by-resource view: %v\n", err)
	}
	defer byResourceResp.Body.Close()

	if byResourceResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error running by-resource view: status is %v\n", byResourceResp.Status)
	}

	type byResource struct {
		Rows []struct {
			Doc model.ResourceDocument `json:"doc"`
		} `json:"rows"`
	}

	var resp byResource
	err = json.NewDecoder(byResourceResp.Body).Decode(&resp)
	if err != nil {
		return nil, fmt.Errorf("Error running by-resource view: %v\n", err)
	}

	resources := make([]model.ResourceDocument, 0)
	for _, row := range resp.Rows {
		resources = append(resources, row.Doc)
	}
	return resources, err

}
