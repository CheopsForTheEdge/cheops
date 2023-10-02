package api

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"cheops.com/backends"
	"github.com/gorilla/mux"
)

func Sync(port int) {

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.EscapedPath()
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

		log.Printf("method=%v path=%v body=%s\n", method, path, string(body))

		sites, err := backends.SitesFor(method, path, header, body)
		if err != nil {
			log.Printf("Error parsing sites: %v\n", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		for _, site := range sites {
			header := fmt.Sprintf("X-status-%s", site)
			w.Header().Add("Trailer", header)
		}

		log.Printf("sites=%v\n", sites)

		if len(sites) == 0 {
			proxy(r.Context(), "127.0.0.1:8283", w, r.Method, path, r.Header, body)
			return
		}

		req := Payload{
			Method: method,
			Header: r.Header,
			Path:   path,
			Body:   body,
		}

		reply, err := Do(r.Context(), sites, req)
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
		w.Write(reply.Body)
	})

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
