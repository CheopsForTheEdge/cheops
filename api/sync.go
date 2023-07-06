package api

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func Sync(port int) {

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.EscapedPath()

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

		sitesAsSlice := make([]string, 0)
		for _, site := range r.Form["sites"] {
			site := site
			host := strings.Split(site, ":")[0]
			header := fmt.Sprintf("X-status-%s", host)
			w.Header().Add("Trailer", header)
			sitesAsSlice = append(sitesAsSlice, site)
		}

		if len(sitesAsSlice) == 0 {
			proxy(r.Context(), "127.0.0.1:8283", w, r.Method, path, r.Header, body)
			return
		}

		req := Payload{
			Method: method,
			Header: r.Header,
			Path:   path,
			Body:   string(body),
		}

		err = Do(r.Context(), sitesAsSlice, req)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
