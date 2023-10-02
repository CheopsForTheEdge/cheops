package replicator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"cheops.com/env"
)

// replicate watches the _changes feed and makes sure the replication jobs
// are in place
func (c *CRDT) replicate() {
	existingJobs := getExistingJobs()

	for {
		feed, err := http.Get("http://localhost:5984/crdt-log/_changes?include_docs=true&feed=continuous")
		if err != nil {
			log.Fatal(err)
		}
		if feed.StatusCode != 200 {
			log.Fatal(fmt.Errorf("Can't get _changes feed: %s", feed.Status))
		}

		defer feed.Body.Close()

		copy, err := os.Create("/tmp/log/changes.log")
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(io.TeeReader(feed.Body, copy))
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if s == "" {
				continue
			}

			var d DocChange
			err := json.NewDecoder(strings.NewReader(s)).Decode(&d)
			if err != nil {
				log.Printf("Couldn't decode: %s", err)
				continue
			}
			for _, location := range d.Doc.Locations {
				if location == env.Myfqdn {
					continue
				}
				if _, ok := existingJobs[location]; !ok {
					body := fmt.Sprintf(`{"continuous": true, "source": "http://localhost:5984/crdt-log", "target": "http://%s:5984/crdt-log"}`, location)
					resp, err := http.Post("http://admin:password@localhost:5984/_replicate", "application/json", strings.NewReader(body))
					if err != nil {
						log.Printf("Couldn't add replication: %s\n", err)
					}
					if resp.StatusCode != 202 {
						log.Printf("Couldn't add replication: %s\n", resp.Status)
					}
					existingJobs[location] = struct{}{}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}

type DocChange struct {
	Doc crdtDocument `json:"doc"`
}

func getExistingJobs() map[string]struct{} {
	existingJobs, err := http.Get("http://admin:password@localhost:5984/_scheduler/jobs")
	if err != nil {
		log.Fatal(err)
	}
	defer existingJobs.Body.Close()

	if existingJobs.StatusCode != 200 {
		log.Fatal(fmt.Errorf("Can't get existing replication jobs: %s", existingJobs.Status))
	}

	var js Jobs
	err = json.NewDecoder(existingJobs.Body).Decode(&js)
	if err != nil {
		log.Fatalf("Couldn't decode: %s", err)
	}

	ret := make(map[string]struct{})
	for _, j := range js.Jobs {
		ret[j.Target] = struct{}{}
	}

	return ret
}

type Jobs struct {
	Jobs []Job `json:"jobs"`
}

type Job struct {
	Target string `json:"target"`
}
