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
		dumpCheops(r.Context(), w)
		dumpCheopsAll(r.Context(), w)
	})
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
}

func dumpCheops(ctx context.Context, w http.ResponseWriter) {

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:5984/cheops/_all_docs?include_docs=true", nil)
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

	docsByResource := make(map[string][]crdtDocument)
	for _, row := range ad.Rows {
		if _, ok := docsByResource[row.Doc.Payload.ResourceId]; !ok {
			docsByResource[row.Doc.Payload.ResourceId] = make([]crdtDocument, 0)
		}
		docsByResource[row.Doc.Payload.ResourceId] = append(docsByResource[row.Doc.Payload.ResourceId], row.Doc)
	}

	for resourceId, docsForResource := range docsByResource {
		requests := make([]crdtDocument, 0)
		for _, doc := range docsForResource {
			if doc.Payload.IsRequest() {
				requests = append(requests, doc)
			}
		}

		fmt.Fprintf(w, "%s\n", resourceId)
		for _, request := range requests {
			fmt.Fprintf(w, "\t%s\n", request.Payload.RequestId)
			for _, doc := range docsForResource {
				if doc.Payload.RequestId == request.Payload.RequestId && !doc.Payload.IsRequest() {
					fmt.Fprintf(w, "\t\t%s: %s\n", doc.Payload.Site, doc.Payload.Status)
				}
			}
		}
	}
}

func dumpCheopsAll(ctx context.Context, w http.ResponseWriter) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:5984/cheops-all/_all_docs?include_docs=true", nil)
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
			Doc MetaDocument
		}
	}

	var res allDocs
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	rep := struct {
		Sites     []string
		Resources map[string][]string
	}{
		Sites:     make([]string, 0),
		Resources: make(map[string][]string),
	}

	for _, row := range res.Rows {
		doc := row.Doc
		switch doc.Type {
		case "SITE":
			rep.Sites = append(rep.Sites, doc.Site)
		case "RESOURCE":
			if _, ok := rep.Resources[doc.ResourceId]; !ok {
				rep.Resources[doc.ResourceId] = make([]string, 0)
			}
			rep.Resources[doc.ResourceId] = append(rep.Resources[doc.ResourceId], doc.Site)
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	err = enc.Encode(rep)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
