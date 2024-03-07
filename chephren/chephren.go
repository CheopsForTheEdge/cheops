package chephren

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cheops.com/env"
	"cheops.com/replicator"
	"github.com/gorilla/mux"
)

func Run(port int, repl *replicator.Replicator) {

	router := mux.NewRouter().SkipClean(true)
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := url.Parse(r.Header.Get("Origin"))
			if err == nil && strings.Contains(u.Host, ".grid5000.fr") {
				w.Header().Set("Access-Control-Allow-Origin", u.String())
			}

			next.ServeHTTP(w, r)
		})
	}
	router.Use(corsMiddleware)

	apiRouter := router.PathPrefix("//api").Subrouter()
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
			State:          "ONLINE",
			ResourcesCount: count,
		}
		json.NewEncoder(w).Encode(resp)

	})

	apiRouter.HandleFunc("/resources", func(w http.ResponseWriter, r *http.Request) {
		resources, err := repl.GetResources()
		if err != nil {
			log.Printf("Error with getResources: %v\n", err)
			http.Error(w, "Internal error with getResources", http.StatusInternalServerError)
			return
		}

		type resourceSummaryReply struct {
			Id            string    `json:"id"`
			Name          string    `json:"name"`
			LastUpdate    time.Time `json:"lastUpdate"`
			CommandsCount int       `json:"commandsCount"`
		}

		resp := make([]resourceSummaryReply, 0)
		for _, resource := range resources {
			rr := resourceSummaryReply{
				Id:            resource.Id,
				Name:          resource.Id,
				LastUpdate:    resource.Operations[len(resource.Operations)-1].Time,
				CommandsCount: len(resource.Operations),
			}
			resp = append(resp, rr)
		}
		json.NewEncoder(w).Encode(resp)
	}).Methods("GET")

	apiRouter.HandleFunc("/resource/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		d, err := repl.Get(id)
		if err != nil {
			log.Printf("Error with getResource %s: %v\n", id, err)
			http.Error(w, "Internal error with getResource", http.StatusInternalServerError)
			return
		}

		type commandReply struct {
			Command string    `json:"command"`
			Date    time.Time `json:"date"`
		}
		type resourceReply struct {
			Id         string         `json:"id"`
			Name       string         `json:"name"`
			LastUpdate time.Time      `json:"lastUpdate"`
			Commands   []commandReply `json:"commands"`
		}

		commands := make([]commandReply, 0)
		for _, unit := range d.Operations {
			commands = append(commands, commandReply{
				Command: unit.Command.Command,
				Date:    unit.Time,
			})
		}
		resp := resourceReply{
			Id:         d.Id,
			Name:       d.Id,
			LastUpdate: d.Operations[len(d.Operations)-1].Time,
			Commands:   commands,
		}

		json.NewEncoder(w).Encode(resp)
	}).Methods("GET")

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
