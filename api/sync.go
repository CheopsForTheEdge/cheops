package api

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cheops.com/backends"
	"cheops.com/env"
	"cheops.com/replicator"
	"github.com/gorilla/mux"
)

func Sync(port int, d replicator.Doer) {

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.RequestURI()
		header := r.Header

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

		sites, err := backends.SitesFor(method, path, header, body)
		if err != nil {
			log.Printf("Error parsing sites: %v\n", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		log.Printf("Found sites to apply: %v\n", sites)

		for _, site := range sites {
			header := fmt.Sprintf("X-status-%s", site)
			w.Header().Add("Trailer", header)
		}

		if len(sites) == 0 {
			proxy(r.Context(), "127.0.0.1:8283", w, r.Method, path, r.Header, body)
			return
		}

		randBytes, err := io.ReadAll(&io.LimitedReader{R: rand.Reader, N: 64})
		if err != nil {
			return
		}

		resourceId, err := backends.ResourceIdFor(method, path, header, body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			log.Printf("Bad request, no resourceId can be found: %s\n", err)
			return
		}

		req := replicator.Payload{
			Method:     method,
			ResourceId: resourceId,
			Header:     r.Header,
			Path:       path,
			Body:       string(body),
			RequestId:  base32.StdEncoding.EncodeToString(randBytes),
			Site:       env.Myfqdn,
		}

		reply, err := d.Do(r.Context(), sites, req)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		for key, vals := range reply.Header {
			for _, val := range vals {
				w.Header().Add(key, val)
			}
		}
		io.Copy(w, strings.NewReader(reply.Body))
	})

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
