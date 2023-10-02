package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func Admin(port int) {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Working !")
	})
	router.HandleFunc("/sites", addSite).Methods("POST")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}

var sites map[string]struct{}

func addSite(w http.ResponseWriter, r *http.Request) {
	if sites == nil {
		sites = make(map[string]struct{})
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	site := r.Form.Get("site")
	sites[site] = struct{}{}

	fmt.Fprintf(w, "Registered %s", site)

	log.Printf("Registered %s as a site\n", site)

	siteslog := make([]string, 0)
	for s := range sites {
		siteslog = append(siteslog, s)
	}
	log.Printf("known sites: %v\n", siteslog)
}
