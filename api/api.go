// Package api offers the API entrypoints for the different packages
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	_ "cheops.com/kubernetes"
	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
)

/*
This function creates a router with no care for trailing slash.
It adds handlers and routes to the router. Finally,
it runs the listening on defined port with those routes.
*/
func Routing() {
	/*
		router.HandleFunc("/", homeLink)
		//commonHandlers := alice.New(CheckRequestFilledHandler)
		// Replication
		// router.Handle("/replication", commonHandlers.ThenFunc(operation.CreateLeaderFromOperationAPI)).Methods("POST")
		// router.HandleFunc("/replicationLeader",
		//operation.CreateLeaderFromOperationAPI).Methods("POST")
		router.HandleFunc("/replication", operation.CreateReplicantAPI).Methods(
			"POST")
		router.HandleFunc("/replicant/{metaID}", operation.GetReplicantAPI).Methods("GET")
		router.HandleFunc("/replicant/{metaID}", operation.AddReplica).Methods("PUT")
		router.HandleFunc("/replicant/{metaID}", operation.DeleteReplicant).Methods("DELETE")
		//router.Handle("/replicants", commonHandlers.ThenFunc(operation.GetAllReplicantsAPI)).Methods("GET")
		// Endpoint
		router.HandleFunc("/endpoint", endpoint.CreateEndpointAPI).Methods("POST")
		router.HandleFunc("/endpoint/createsite/{Site}/{Address}", endpoint.CreateSiteAPI).Methods("POST")
		router.HandleFunc("/endpoint/getaddress/{Site}",
			endpoint.GetSiteAddressAPI).Methods("GET")
		// Database
		// Operation
		router.HandleFunc("/operation", operation.CreateOperationAPI).Methods("POST")
		router.HandleFunc("/operation/execute", operation.ExecuteOperationAPI).Methods("POST")
		router.HandleFunc("/operation/localrequest",
			operation.ExecRequestLocallyAPI).
			Methods("POST")
		// Broker - Driver
		router.HandleFunc("/scope", request.ExtractScope).Methods("GET")
		router.HandleFunc("/scope/forward", request.RedirectRequest).Methods("POST")
		router.HandleFunc("/Appb/{flexible:.*}", request.Appb).Methods("GET")
		router.HandleFunc("/SendRemote", request.SendRemote).Methods("GET")
		router.HandleFunc("/RegisterRemoteSite", request.RegisterRemoteSite).Methods("POST")
		router.HandleFunc("/GetRemoteSite/{site}", request.GetRemoteSite).Methods("GET")
		// Client
		router.HandleFunc("/get", cli.GetHandler)
		router.HandleFunc("/deploy", cli.DeployHandler)
		// router.HandleFunc("/cross/", cli.CrossHandler)
		// router.HandleFunc("/replica/", cli.ReplicaHandler)
		router.HandleFunc("/sendoperation", cli.SendOperationToSites)
	*/
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		index := 0
		g, ctx := errgroup.WithContext(r.Context())

		for site := range sites {
			site := site
			if index == 0 {
				w.Header().Add("Trailer", fmt.Sprintf("X-status-%s", site))
				g.Go(func() error {
					return proxy(ctx, site, w, r)
				})
			}
			g.Go(func() error {
				e := emptyResponseWriter{}
				err := proxy(ctx, site, e, r)
				w.Header().Set(fmt.Sprintf("X-status-%s", site), string(e.statusCode))
				return err
			})
			index++
		}

		err := g.Wait()
		if err != nil {
			log.Println(err)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", router))
}

// emptyResponseWriter stores the status code and discards everything else
type emptyResponseWriter struct {
	statusCode int
}

func (e emptyResponseWriter) Header() http.Header {
	return http.Header(make(map[string][]string))
}
func (e emptyResponseWriter) WriteHeader(statusCode int) {
	e.statusCode = statusCode
}
func (e emptyResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func proxy(ctx context.Context, host string, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()
	reqbuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read request body", http.StatusInternalServerError)
		return err
	}

	u := r.URL
	u.Host = host

	// Not filled by default
	u.Scheme = "http"

	newreq, err := http.NewRequestWithContext(ctx, r.Method, u.String(), bytes.NewReader(reqbuf))
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "can't build proxy request", http.StatusInternalServerError)
		return err
	}

	for key, vals := range r.Header {
		for _, val := range vals {
			newreq.Header.Add(key, val)
		}
	}

	timeout, _ := time.ParseDuration("3s")
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(newreq)
	if err != nil {
		http.Error(w, "can't send to backend", http.StatusInternalServerError)
		log.Println(err)

		// Not a blocking error
		return nil
	}
	defer resp.Body.Close()

	respbuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "can't read reply from backend", http.StatusInternalServerError)
		// Not a blocking error
		return nil
	}

	for key, vals := range resp.Header {
		for _, val := range vals {
			w.Header().Add(key, val)
		}
	}

	_, err = io.Copy(w, bytes.NewReader(respbuf))
	if err != nil {
		http.Error(w, "can't write reply", http.StatusInternalServerError)
		log.Println(err)

		// Not a blocking error
		return nil
	}

	indent := func(b []byte) string {
		var obj map[string]interface{}
		json.Unmarshal(b, &obj)
		indented, _ := json.MarshalIndent(obj, "", "\t")
		return string(indented)
	}
	headers := func(h http.Header) string {
		var asstring string
		for key, val := range h {
			asstring += fmt.Sprintf("%s=%s\n", key, val)
		}
		return asstring
	}

	log.Printf(`->[%s] %s %s
-> [%s] %s
<- [%s] %s
`, host, r.Method, r.URL.String(), host, indent(reqbuf), headers(resp.Header), host, indent(respbuf))
	return nil

}

// Checks if the request has data
func CheckRequestFilledHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		_, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Kindly enter data", r.Method, r.URL.String())

			fmt.Fprintf(w, "Kindly enter data")
			return
		} else {
			next.ServeHTTP(w, r)
		}
	}
	return http.HandlerFunc(fn)
}

// Default route
func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome home!")
}
