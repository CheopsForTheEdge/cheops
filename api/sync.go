package api

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mathrand "math/rand"
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

		// The site where the user wants the resource to exist
		desiredSites := header.Values("X-Cheops-Location")

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
			http.Redirect(w, r, fmt.Sprintf("http://%s:%d", targetSite, port), http.StatusTemporaryRedirect)
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

		reply, err := d.Do(r.Context(), desiredSites, req)
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
