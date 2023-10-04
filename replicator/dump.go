package replicator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func (c *Crdt) listenDump(port int) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		req, err := http.NewRequestWithContext(r.Context(), "GET", "http://localhost:5984/cheops/_all_docs?include_docs=true", nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		type allDocs struct {
			Rows []struct {
				Doc crdtDocument
			}
		}

		var ad allDocs
		err = json.NewDecoder(resp.Body).Decode(&ad)

		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// map keyed by resourceId -> list of maps keyed by requestId -> map keyed by site -> status
		reply := make(map[string][]map[string]map[string]string)

		docsByResource := make(map[string][]crdtDocument)
		for _, row := range ad.Rows {
			if row.Doc.Payload.ResourceId == "" {
				// design docs of couchdb have no Payload
				continue
			}

			if _, ok := docsByResource[row.Doc.Payload.ResourceId]; !ok {
				docsByResource[row.Doc.Payload.ResourceId] = make([]crdtDocument, 0)
			}
			docsByResource[row.Doc.Payload.ResourceId] = append(docsByResource[row.Doc.Payload.ResourceId], row.Doc)
		}

		for resourceId, docsForResource := range docsByResource {
			if _, ok := reply[resourceId]; !ok {
				reply[resourceId] = make([]map[string]map[string]string, 0)
			}
			r := reply[resourceId]

			requests := make([]crdtDocument, 0)
			for _, doc := range docsForResource {
				if doc.Payload.IsRequest() {
					requests = append(requests, doc)
				}
			}

			for _, request := range requests {
				m := make(map[string]map[string]string)
				requestId := request.Payload.RequestId
				if _, ok := m[requestId]; !ok {
					m[requestId] = make(map[string]string)
				}

				for _, doc := range docsForResource {
					if doc.Payload.RequestId == request.Payload.RequestId && !doc.Payload.IsRequest() {
						m[requestId][doc.Payload.Site] = doc.Payload.Status
					}
				}

				r = append(r, m)
			}

			reply[resourceId] = r
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		enc.Encode(reply)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	})
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
}

func dump(ctx context.Context, w http.ResponseWriter) {}
