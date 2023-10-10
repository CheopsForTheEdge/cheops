package api

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	mathrand "math/rand"
	"net/http"
	"strconv"

	"cheops.com/env"
	"cheops.com/replicator"
	"github.com/gorilla/mux"
)

func Run(port int, repl *replicator.Replicator) {

	router := mux.NewRouter()
	router.PathPrefix("/{id}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			http.Error(w, "bad request: missing id", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		err = r.ParseForm()
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// The site where the user wants the resource to exist
		desiredSites := r.Header.Values("X-Cheops-Location")

		var local bool
		for _, site := range desiredSites {
			if site == env.Myfqdn {
				local = true
			}
		}

		if !local {
			// Send the request to any site that is related to the request
			targetSiteIdx := mathrand.Intn(len(desiredSites))
			targetSite := desiredSites[targetSiteIdx]
			u := r.URL
			u.Host = targetSite
			http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
			return
		}

		randBytes, err := io.ReadAll(&io.LimitedReader{R: rand.Reader, N: 64})
		if err != nil {
			return
		}

		req := replicator.CrdtUnit{
			Body:      string(body),
			RequestId: base32.StdEncoding.EncodeToString(randBytes),
		}

		replies, err := repl.Do(r.Context(), desiredSites, id, req)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(replies)

	})

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
