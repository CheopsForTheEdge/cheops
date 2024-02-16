package api

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cheops.com/env"
	"cheops.com/model"
	"cheops.com/replicator"
)

func Run(port int, repl *replicator.Replicator) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(1024 * 1024)
		if err != nil {
			log.Printf("Error parsing multipart form: %v\n", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		configFiles, ok := r.MultipartForm.File["config.json"]
		if !ok || len(configFiles) != 1 {
			log.Println("Missing config.json file")
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		configFile, err := configFiles[0].Open()
		if err != nil {
			log.Printf("Error with config.json file: %v\n", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		defer configFile.Close()

		var config model.ResourceConfig
		err = json.NewDecoder(configFile).Decode(&config)
		if err != nil {
			log.Printf("Error with config.json file: %v\n", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		id := config.Id
		if id == "" {
			log.Println("Missing id")
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// The site where the user wants the resource to exist
		sites, ok := r.MultipartForm.Value["sites"]
		if !ok {
			log.Println("Missing sites")
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		desiredSites := make([]string, 0)
		for _, group := range sites {
			for _, val := range strings.Split(group, "&") {
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

		commands, ok := r.MultipartForm.Value["command"]
		if !ok || len(commands) != 1 {
			log.Println("Missing command")
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		req := model.CrdtUnit{
			Body:      strings.TrimSpace(string(commands[0])),
			RequestId: base32.StdEncoding.EncodeToString(randBytes),
			Time:      time.Now(),
		}

		log.Printf("id: %#v\n", id)
		log.Printf("desiredSites: %#v\n", desiredSites)
		log.Printf("req: %#v\n", req)

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

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		for reply := range replies {
			json.NewEncoder(w).Encode(reply)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}

	})

	err := http.ListenAndServe(":"+strconv.Itoa(port), mux)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
