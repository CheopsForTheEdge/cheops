package chephren

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"cheops.com/env"
	"cheops.com/replicator"
	"github.com/gorilla/mux"
)

func Run(port int, repl *replicator.Replicator) {

	router := mux.NewRouter()
	router.PathPrefix("/node").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type nodeReply struct {
			Name           string `json:"name"`
			State          string `json:"state"`
			ResourcesCount int    `json:"resourcesCount"`
			Address        string `json:"address"`
		}

		count, err := repl.Count()
		if err != nil {
			log.Printf("Error with count: %v\n", err)
			http.Error(w, "Internal error with counting", http.StatusInternalServerError)
			return
		}

		resp := nodeReply{
			Name:           env.Myfqdn,
			Address:        fmt.Sprintf("http://%s:%d", env.Myfqdn, port),
			State:          "UP",
			ResourcesCount: count,
		}
		json.NewEncoder(w).Encode(resp)

	})

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
