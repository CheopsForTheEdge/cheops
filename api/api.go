package api

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cheops.com/backends"
	"cheops.com/env"
	"cheops.com/model"
	"cheops.com/replicator"
	"github.com/gorilla/mux"
)

// Run starts an endpoint on '/{id}' on the given port that will do the Cheops magic.
//
// The body must be a multipart/form-data with the following parts:
//
//		Content-Disposition: form-data; name="command"
//
//			Mandatory: the command to run. If the command needs the
//			content of a local file, this file must be wrapped with {} and included as a file
//			(see later)
//
//		Content-Disposition: form-data; name="sites"
//
//			Mandatory: the sites, separated with a '&'
//
//		Content-Disposition: form-data; name="type"
//
//			Mandatory: the consistency class of the operation
//
//		Content-Disposition: form-data; name="config.json"; filename="config.json"
//
//			If present, the resource logic
//
//		Content-Disposition: form-data; name="local-logic"; filename="local-logic"
//
//			If present, the local logic
//
//		Content-Disposition: form-data; name="XXX"; filename="XXX"
//
//			If present, a file that is needed for the command to run

func Run(port int, repl *replicator.Replicator) {
	m := mux.NewRouter()
	m.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, command, typ, _, sites, files, ok := parseRequest(w, r)
		if !ok {
			return
		}

		if len(sites) == 0 {
			log.Printf("Request doesn't have any sites")
			http.Error(w, "bad request", http.StatusInternalServerError)
			return
		}

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

		randBytes, err := io.ReadAll(&io.LimitedReader{R: rand.Reader, N: 64})
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		cmd := backends.ShellCommand{
			Command: command,
			Files:   files,
		}
		req := model.Operation{
			Command:   cmd,
			Type:      model.OperationType(typ),
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

func parseRequest(w http.ResponseWriter, r *http.Request) (id, command string, typ model.OperationType, config model.ResourceConfig, sites []string, files map[string][]byte, ok bool) {
	vars := mux.Vars(r)
	id = vars["id"]
	if id == "" {
		log.Println("Missing id")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if uid, err := url.PathUnescape(id); err != nil || uid != id {
		log.Println("Unsafe id")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err := r.ParseMultipartForm(1024 * 1024)
	if err != nil {
		log.Printf("Error parsing multipart form: %v\n", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// The site where the user wants the resource to exist
	sitesString, okk := r.MultipartForm.Value["sites"]
	if !okk {
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

	commands, okk := r.MultipartForm.Value["command"]
	if !okk || len(commands) != 1 {
		log.Println("Missing command")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	command = strings.TrimSpace(commands[0])

	types, okk := r.MultipartForm.Value["type"]
	if !okk || len(types) != 1 {
		log.Println("Missing type")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	typ, err = model.OperationTypeFrom(strings.TrimSpace(types[0]))
	if err != nil {
		log.Println("Invalid type")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	files = make(map[string][]byte)
	for name, requestFiles := range r.MultipartForm.File {
		if len(requestFiles) != 1 {
			log.Printf("Warning: not exactly one file for name=%s, got %d, taking 1st one only\n", name, len(files))
		}
		f, err := requestFiles[0].Open()
		if err != nil {
			log.Printf("Couldn't open %s: %v\n", name, err)
			continue
		}

		content, err := ioutil.ReadAll(f)
		f.Close()
		if err != nil {
			log.Printf("Couldn't open %s: %v\n", name, err)
			continue
		}

		files[name] = content
	}

	ok = true
	return
}
