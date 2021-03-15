package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"

	"../replication"
)


func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome home!")
}


func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeLink)
	fmt.Printf("%v replicants", replication.Replicants)
	router.HandleFunc("/replication", replication.CreateReplicant).Methods("POST")
	router.HandleFunc("/replicant/{metaID}", replication.GetReplicant).Methods("GET")
	router.HandleFunc("/replicant/{metaID}", replication.AddReplica).Methods("PUT")
	router.HandleFunc("/replicant/{metaID}", replication.DeleteReplicant).Methods("DELETE")
	router.HandleFunc("/replicants", replication.GetAllReplicants).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", router))
}