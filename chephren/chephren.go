package chephren

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"cheops.com/env"
	"cheops.com/replicator"
	"github.com/gorilla/mux"
)

func Run(port int, repl *replicator.Replicator) {

	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/node", func(w http.ResponseWriter, r *http.Request) {
		count, err := repl.Count()
		if err != nil {
			log.Printf("Error with count: %v\n", err)
			http.Error(w, "Internal error with counting", http.StatusInternalServerError)
			return
		}

		type nodeReply struct {
			Name           string `json:"name"`
			State          string `json:"state"`
			ResourcesCount int    `json:"resourcesCount"`
			Address        string `json:"address"`
		}

		resp := nodeReply{
			Name:           env.Myfqdn,
			Address:        fmt.Sprintf("http://%s:%d", env.Myfqdn, port),
			State:          "UP",
			ResourcesCount: count,
		}
		json.NewEncoder(w).Encode(resp)

	})

	resourcesRouter := apiRouter.PathPrefix("/resources").Subrouter()
	resourcesRouter.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		resources, err := repl.GetResources()
		if err != nil {
			log.Printf("Error with getResources: %v\n", err)
			http.Error(w, "Internal error with getResources", http.StatusInternalServerError)
			return
		}

		type resourceReply struct {
			Id            string
			Name          string
			LastUpdate    time.Time
			CommandsCount int
		}

		resp := make([]resourceReply, 0)
		for _, resource := range resources {
			rr := resourceReply{
				Id:            resource.Id,
				Name:          resource.Id,
				LastUpdate:    resource.Units[len(resource.Units)-1].Time,
				CommandsCount: len(resource.Units),
			}
			resp = append(resp, rr)
		}
		json.NewEncoder(w).Encode(resp)
	}).Methods("GET")

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
