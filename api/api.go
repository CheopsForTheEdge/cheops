package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
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
	"golang.org/x/sync/errgroup"
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
	m.HandleFunc("/show/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, command, _, _, _, _, ok := parseRequest(w, r)
		if !ok {
			// Error was already sent to http caller
			return
		}

		sites, err := repl.SitesFor(id)
		if err != nil {
			log.Printf("Error retrieving sites for %v: %v\n", id, err)
			if _, ok := err.(replicator.ErrorNotFound); ok {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("show id=%v sites=[%v] command=[%v]\n", id, sites, command)

		type SiteResp struct {
			Status string // OK, KO or TIMEOUT
			Output string
		}

		// Site -> SiteResp
		resp := make(map[string]SiteResp)

		thirtySeconds, _ := time.ParseDuration("30s")
		ctxWithTimeout, cancel := context.WithTimeout(r.Context(), thirtySeconds)
		defer cancel()

		g, ctx := errgroup.WithContext(ctxWithTimeout)

		for _, site := range sites {
			site := site
			g.Go(func() error {

				u := fmt.Sprintf("http://%s:8079/show_local/%s", site, id)

				var b bytes.Buffer
				mw := multipart.NewWriter(&b)
				mw.WriteField("command", command)
				mw.Close()

				req, err := http.NewRequestWithContext(ctx, "POST", u, &b)
				if err != nil {
					log.Printf("Error building request for show_local for %v: %v\n", site, err)
					r := SiteResp{
						Status: "KO",
					}
					resp[site] = r
					return nil
				}
				req.Header.Set("Content-Length", strconv.Itoa(b.Len()))
				req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", mw.Boundary()))
				reply, err := http.DefaultClient.Do(req)
				if err != nil || reply.StatusCode != http.StatusOK {
					var reason string
					if err != nil {
						reason = err.Error()
					} else {
						reason = reply.Status
					}
					log.Printf("Error running show_local at %v: %v\n", site, reason)
					r := SiteResp{
						Status: "KO",
					}
					resp[site] = r
					return nil
				}

				select {
				case <-ctx.Done():
					r := SiteResp{
						Status: "TIMEOUT",
					}
					resp[site] = r
					return nil
				default:
				}

				defer reply.Body.Close()
				var r SiteResp
				err = json.NewDecoder(reply.Body).Decode(&r)
				if err != nil {
					log.Printf("Error json-decoding resp on %v: %v\n", site, err)
					r = SiteResp{
						Status: "KO",
					}
				}

				resp[site] = r

				return nil
			})
		}

		g.Wait()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	})

	m.HandleFunc("/show_local/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, command, _, _, _, _, ok := parseRequest(w, r)
		if !ok {
			return
		}

		command = strings.Replace(command, "__ID__", id, -1)
		res, err := repl.RunDirect(r.Context(), command)
		status := "OK"
		if err != nil {
			log.Printf("Error running command: %v\n", err)
			status = "KO"
		}

		type SiteResp struct {
			Status string // OK, KO or TIMEOUT
			Output string
		}
		jsonRes := SiteResp{
			Status: status,
			Output: res,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jsonRes)

	})

	m.HandleFunc("/exec/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, command, typ, _, sites, files, ok := parseExecRequest(w, r)
		if !ok {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
			return
		}

		if len(sites) == 0 {
			log.Println("Request doesn't have any sites")
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

func parseExecRequest(w http.ResponseWriter, r *http.Request) (id, command string, typ model.OperationType, config model.ResourceConfig, sites []string, files map[string][]byte, ok bool) {
	id, command, typ, config, sites, files, ok = parseRequest(w, r)
	if typ == "" {
		log.Println("Missing type")
		http.Error(w, "bad request", http.StatusBadRequest)
		ok = false
	}
	if len(sites) == 0 {
		log.Println("Missing sites")
		http.Error(w, "bad request", http.StatusBadRequest)
	}
	return
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
	if okk {
		sites = make([]string, 0)
		for _, group := range sitesString {
			for _, val := range strings.Split(group, "&") {
				sites = append(sites, strings.TrimSpace(val))
			}
		}
	}

	commands, okk := r.MultipartForm.Value["command"]
	if !okk || len(commands) != 1 {
		log.Println("Missing command")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	command = strings.TrimSpace(commands[0])

	types := r.MultipartForm.Value["type"]
	if types != nil && len(types) == 1 {
		if t, err := model.OperationTypeFrom(strings.TrimSpace(types[0])); err == nil {
			typ = t
		}
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
