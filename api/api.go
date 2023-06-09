// Package api offers the API entrypoints for the different packages
package api

import (
	cli "cheops.com/client"
	"cheops.com/endpoint"
	"cheops.com/operation"
	"cheops.com/request"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

/*
This function creates a router with no care for trailing slash.
It adds handlers and routes to the router. Finally,
it runs the listening on defined port with those routes.
*/
func Routing() {
	router := mux.NewRouter().StrictSlash(true)
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
