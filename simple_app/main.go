package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

func main() {
	m := mux.NewRouter()
	m.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			log.Println("Bad request: missing id")
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		err := r.ParseForm()
		if err != nil {
			log.Printf("Bad request: invalid params: %v\n", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		t := r.Form.Get("type")
		operation := r.Form.Get("operation")
		value := r.Form.Get("value")

		log.Printf("type=%v operation=%v value=%v\n", t, operation, value)

		switch t {
		case "counter":
			switch r.Method {
			case "POST":
				ok := Counter.Handle(id, operation, value)
				if !ok {
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}
			case "GET":
				io.WriteString(w, Counter.Get(id))
			}
		default:
			log.Printf("Unknown resource type: %v\n", t)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	})

	port := 8080
	if len(os.Args) > 1 {
		p, err := strconv.Atoi(os.Args[1])
		if err != nil {
			log.Println("Bad number for port, defaulting to 8080")
			p = 8080
		}
		port = p
	}
	log.Printf("Listening on :%d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), m)
	if err != nil {
		log.Fatal(err)
	}
}
