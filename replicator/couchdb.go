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
