package api

import (
	"../replication"
	"../request"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
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
	fmt.Printf("%v replicants", replication.Replicants)
	commonHandlers := alice.New(CheckRequestFilledHandler)
	router.Handle("/replication", commonHandlers.ThenFunc(replication.CreateReplicant)).Methods("POST")
	router.HandleFunc("/replicant/{metaID}", replication.GetReplicant).Methods("GET")
	router.HandleFunc("/replicant/{metaID}", replication.AddReplica).Methods("PUT")
	router.HandleFunc("/replicant/{metaID}", replication.DeleteReplicant).Methods("DELETE")
	router.Handle("/replicants", commonHandlers.ThenFunc(replication.
		GetAllReplicants)).Methods("GET")
	router.HandleFunc("/scope",request.ExtractScope).Methods("GET")
	router.HandleFunc("/scope/forward",request.RedirectRequest).Methods("POST")
	router.HandleFunc("/Appb/{flexible:.*}", request.Appb).Methods("GET")
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
