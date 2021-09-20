package api

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"cheops.com/operation"
	"cheops.com/request"
	"cheops.com/endpoint"
)

/*
This function creates a router with no care for trailing slash.
It adds handlers and routes to the router. Finally,
it runs the listening on defined port with those routes.
*/
func Routing() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeLink)
	commonHandlers := alice.New(CheckRequestFilledHandler)
	// Replication
	router.Handle("/replication", commonHandlers.ThenFunc(operation.CreateLeaderFromOperationAPI)).Methods("POST")
	router.HandleFunc("/replicant/{metaID}", operation.GetReplicant).Methods("GET")
	router.HandleFunc("/replicant/{metaID}", operation.AddReplica).Methods("PUT")
	router.HandleFunc("/replicant/{metaID}", operation.DeleteReplicant).Methods("DELETE")
	router.Handle("/replicants", commonHandlers.ThenFunc(operation.
		GetAllReplicants)).Methods("GET")
	// Endpoint
	router.HandleFunc("/endpoint/getaddress/{Site}", endpoint.GetAddressAPI).Methods("GET")
	// Database
	// Operation
	// Broker - Driver
	router.HandleFunc("/scope",request.ExtractScope).Methods("GET")
	router.HandleFunc("/scope/forward",request.RedirectRequest).Methods("POST")
	router.HandleFunc("/Appb/{flexible:.*}", request.Appb).Methods("GET")
	router.HandleFunc("/SendRemote", request.SendRemote).Methods("GET")
	router.HandleFunc("/RegisterRemoteSite", request.RegisterRemoteSite).Methods("POST")
	router.HandleFunc("/GetRemoteSite/{site}", request.GetRemoteSite).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", router))
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
