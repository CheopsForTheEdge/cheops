package api

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cheops.com/env"
	"cheops.com/model"
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
		desiredSites := make([]string, 0)
		for _, group := range r.Header.Values("X-Cheops-Location") {
			for _, val := range strings.Split(group, ",") {
				desiredSites = append(desiredSites, strings.TrimSpace(val))
			}
		}

		if len(desiredSites) > 0 {
			forMe := false
			for _, desiredSite := range desiredSites {
				if desiredSite == env.Myfqdn {
					forMe = true
				}
			}
			if !forMe {
				http.Error(w, "Site is not in locations", http.StatusBadRequest)
				return
			}
		}

		randBytes, err := io.ReadAll(&io.LimitedReader{R: rand.Reader, N: 64})
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		req := model.CrdtUnit{
			Body:      strings.TrimSpace(string(body)),
			RequestId: base32.StdEncoding.EncodeToString(randBytes),
		}

		replies, err := repl.Do(r.Context(), desiredSites, id, req)
		if err != nil {
			if err == replicator.ErrDoesNotExist {
				log.Printf("resource [%s] does not exist on this site\n", id)
				http.NotFound(w, r)
				return
			}

			if e, ok := err.(replicator.ErrInvalidRequest); ok {
				log.Printf("invalid request for [%s]: %s\n", id, e)
				http.Error(w, e.Error(), http.StatusBadRequest)
				return
			}

			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		for reply := range replies {
			json.NewEncoder(w).Encode(reply)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}

	})

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
