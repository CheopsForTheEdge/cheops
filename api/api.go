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

	"cheops.com/backends"
	"cheops.com/env"
	"cheops.com/model"
	"cheops.com/replicator"
	"github.com/gorilla/mux"
)

func Run(port int, repl *replicator.Replicator) {
	m := mux.NewRouter()
	m.HandleFunc("/direct", func(w http.ResponseWriter, r *http.Request) {
		id, command, _, _, files, ok := parseRequest(w, r)
		if !ok {
			return
		}
		log.Printf("id=%v command=[%v]\n", id, command)
		log.Printf("files=%#v\n", files)
		commands := []backends.ShellCommand{{
			Command: string(command),
			Files:   files,
		}}
		replies, err := backends.Handle(r.Context(), commands)
		status := "OK"
		if err != nil {
			status = "KO"
		}

		if len(replies) != 1 {
			log.Printf("Error running command id=%v command=[%s]: invalid number of replies, got %d\n", id, command, len(replies))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return

		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")

		type reply struct {
			Site   string
			Status string
		}
		json.NewEncoder(w).Encode(reply{
			Site:   env.Myfqdn,
			Status: status,
		})
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		id, command, _, sites, files, ok := parseRequest(w, r)
		if !ok {
			return
		}

		randBytes, err := io.ReadAll(&io.LimitedReader{R: rand.Reader, N: 64})
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		cmd := backends.ShellCommand{
			Command: command,
			Files:   files,
		}
		req := model.CrdtUnit{
			Command:   cmd,
			RequestId: base32.StdEncoding.EncodeToString(randBytes),
			Time:      time.Now(),
		}

		replies, err := repl.Do(r.Context(), sites, id, req)
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

	err := http.ListenAndServe(":"+strconv.Itoa(port), m)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func parseRequest(w http.ResponseWriter, r *http.Request) (id, command string, config model.ResourceConfig, sites []string, files map[string][]byte, ok bool) {
	ok = false
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

	err = json.NewDecoder(configFile).Decode(&config)
	if err != nil {
		log.Printf("Error with config.json file: %v\n", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	id = config.Id
	if id == "" {
		log.Println("Missing id")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// The site where the user wants the resource to exist
	sitesString, ok := r.MultipartForm.Value["sites"]
	if !ok {
		log.Println("Missing sites")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sites = make([]string, 0)
	for _, group := range sitesString {
		for _, val := range strings.Split(group, "&") {
			sites = append(sites, strings.TrimSpace(val))
		}
	}

	if len(sites) > 0 {
		forMe := false
		for _, desiredSite := range sites {
			if desiredSite == env.Myfqdn {
				forMe = true
			}
		}
		if !forMe {
			http.Error(w, "Site is not in locations", http.StatusBadRequest)
			return
		}
	}

	commands, ok := r.MultipartForm.Value["command"]
	if !ok || len(commands) != 1 {
		log.Println("Missing command")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	command = strings.TrimSpace(commands[0])
	if command == "" {
		log.Println("Missing command")
		http.Error(w, "bad request", http.StatusBadRequest)
		return

	}

	ok = true
	return
}
