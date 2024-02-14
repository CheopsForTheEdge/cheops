package replicator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

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
	if len(resp.Rows) != 1 {
		return 0, fmt.Errorf("Bad reply: %#v\n", resp)
	}
	return resp.Rows[0].Value, err
}
