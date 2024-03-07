package replicator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cheops.com/model"
)

type onNewDocFunc func(j json.RawMessage)

// watch watches the _changes feed of the cheops database, resolves all conflicts for said document
// and runs a function when a new document is seen.
// The document is sent as a raw json string, to be decoded by the function.
// The execution of the function blocks the loop; it is good to not have it run too long
type watches struct {
	watchers []onNewDocFunc
}

func newWatches(ctx context.Context) *watches {
	w := &watches{
		watchers: make([]onNewDocFunc, 0),
	}
	w.startWatching(ctx)

	return w
}

func (w *watches) watch(f onNewDocFunc) {
	w.watchers = append(w.watchers, f)
}

func (w *watches) startWatching(ctx context.Context) {
	retryTime := 1

	go func() {
		since := ""
		for {
			u := "http://localhost:5984/cheops/_changes?include_docs=true&feed=continuous"
			if since != "" {
				u += fmt.Sprintf("&since=%s", since)
			}
			req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
			if err != nil {
				log.Printf("Couldn't create request with context: %v\n", err)
				break
			}
			feed, err := http.DefaultClient.Do(req)
			if err != nil || feed.StatusCode != 200 {
				// When the database is deleted, we are here. Hopefully it will be recreated and we can continue
				log.Printf("No _changes feed, retrying in %ds", retryTime)
				<-time.After(time.Duration(retryTime) * time.Second)
				retryTime = 2 * retryTime
				continue
			}

			if retryTime > 1 {
				log.Println("Got _changes feed back, let's go")
				retryTime = 1
			}

			defer feed.Body.Close()

			scanner := bufio.NewScanner(feed.Body)
			for scanner.Scan() {
				s := strings.TrimSpace(scanner.Text())
				if s == "" {
					continue
				}

				var change DocChange
				err := json.NewDecoder(strings.NewReader(s)).Decode(&change)
				if err != nil {
					log.Printf("Couldn't decode: %s", err)
					continue
				}
				if len(change.Doc) == 0 {
					continue
				}

				for _, f := range w.watchers {
					f(change.Doc)
				}
				since = change.Seq
			}

			select {
			case <-ctx.Done():
				return
			default:
				continue
			}

		}
	}()
}

func (w *watches) postResolution(docWithoutConflicts model.ResourceDocument, conflicts []string) {
	type bulkDocsRequest struct {
		Docs []model.ResourceDocument `json:"docs"`
	}

	req := bulkDocsRequest{
		Docs: []model.ResourceDocument{docWithoutConflicts},
	}
	for _, conflict := range conflicts {
		req.Docs = append(req.Docs, model.ResourceDocument{
			Id:      docWithoutConflicts.Id,
			Rev:     conflict,
			Deleted: true,
		})
	}

	buf, err := json.Marshal(req)
	if err != nil {
		log.Printf("Marshalling error: %v\n", err)
	}
	resp, err := http.Post("http://localhost:5984/cheops/_bulk_docs", "application/json", bytes.NewReader(buf))
	if err != nil {
		log.Printf("Couldn't POST _bulk_docs for %v: %v\n", string(buf), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Printf("Couldn't send _bulk_docs request %#v to couchdb: %s", string(buf), resp.Status)
	}

	type BulkDocsEachReply struct {
		Ok     bool   `json:"ok"`
		Id     string `json:"id"`
		Rev    string `json:"rev"`
		Error  string `json:"error"`
		Reason string `json:"reason"`
	}

	var bder []BulkDocsEachReply
	err = json.NewDecoder(resp.Body).Decode(&bder)
	if err != nil {
		log.Printf("Couldn't decode Bulk Docks reply: %v", err)
	}

	for _, reply := range bder {
		if !reply.Ok {
			log.Printf("Couldn't post _bulk_docs for %v:%v: %v -- %v\n", reply.Id, reply.Rev, reply.Error, reply.Reason)
		}
	}
}
