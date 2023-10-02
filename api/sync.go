package api

import (
	"encoding/json"
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

		log.Printf("method=%v path=%v body=%d\n", method, path, len(body))

		sitesAsSlice := make([]string, 0)
		for site := range sites {
			site := site
			host := strings.Split(site, ":")[0]
			header := fmt.Sprintf("X-status-%s", host)
			w.Header().Add("Trailer", header)
			sitesAsSlice = append(sitesAsSlice, site)
		}

		req := Request{
			Method: method,
			Path:   path,
			Body:   body,
		}
		buf, err := json.Marshal(req)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		err = Save(r.Context(), sitesAsSlice, buf)
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
