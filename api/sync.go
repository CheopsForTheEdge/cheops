package api

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func Sync(port int) {

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.EscapedPath

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		log.Printf("method=%v path=%v body=%d\n", method, path, len(body))
		w.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(":"+strconv.Itoa(port), router)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

type request struct {
	method string
	path   string
	body   []byte
}
