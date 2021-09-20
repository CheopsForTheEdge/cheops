package main

import (
	"os"
	// "fmt"
	// "time"
	"cheops.com/database"
	// "cheops.com/endpoint"
	"cheops.com/api"
	// "cheops.com/operation"
)


func main() {
	// https://chriswiegman.com/2019/01/ensuring-the-file-path-is-present-to-create-a-file-in-golang/
	check_file := "/root/arango"
	if _, err := os.Stat(check_file); os.IsNotExist(err) {
		database.PrepareForExecution("cheops", "replication")
		os.MkdirAll(check_file, 0700)
	}
	// c := database.Connection()
	// db := database.ConnectToDatabase(c)
	// col := database.ConnectionToCorrectCollection("replication")
	// col.EnsurePersistentIndex(nil, []string{"MetaID", "IsLeader"}, nil)
	// doca := operation.Replicant{
	// 	MetaID: "42",
	// 	Replicas: []operation.Replica{
	// 		operation.Replica{Site: "Paris", ID: "65"},
	// 		operation.Replica{Site: "Nantes", ID: "42"}},
	// 	IsLeader: true,
	// 	Logs:  []operation.Log {
	// 		operation.Log{Operation: "incredible operation",
	// 			Date: (time.Now())}}}
	// key := database.CreateResource("replication", doca)
	// fmt.Println(key)
	// doci := operation.Replicant{}
	// database.ReadResource("operation", key, &doci)
	// doc := operation.CreateReplicant()
	// // log = operation.Log{Operation: "incredible operation", Date: (time.Now())}
	// // database.UpdateReplicant(doc)
	// doci := operation.Replicant{}
	// database.ReadResource("operation",  doc, &doci)
	// fmt.Println(doci)
	// fmt.Println(doci.Logs)
	// operation.DeleteReplicantWithKey(doc)
	// coli := database.CreateCollection(db, "endpoint")
	// coli.EnsurePersistentIndex(nil, []string{"Service", "Address"}, nil)
	// endpoint.CreateEndpoint("site3", "localhost:8080/endpoint/getaddress/site3")
	// endpoint.CreateEndpoint("site4", "localhost:8080/endpoint/getaddress/site4")
	// query := "FOR end IN endpoint FILTER end.Site == @name RETURN end"
	// bindvars := map[string]interface{}{ "name": "sitea", }
	// cursor, _ := db.Query(nil, query, bindvars)
	// fmt.Println(sitea)
	// fmt.Println(siteb)
	// // fmt.Println(cursor)
	// result := endpoint.Endpoint{}
	// cursor.ReadDocument(nil, &result)
	// // fmt.Println(database
	// fmt.Println(result)
	// defer cursor.Close()
	// add := endpoint.GetAddress("sitea")
	// fmt.Println(add)
	// col := database.ConnectionToCorrectCollection("replication")
	// col.EnsurePersistentIndex(nil, []string{"MetaID", "IsLeader"}, nil)
	// database.CreateCollection(db, "operation")
	api.Routing()
}
