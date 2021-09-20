package main

import (
	replication "cheops.com/operation"
	"fmt"
	"os"
	"time"
	"cheops.com/database"
	"cheops.com/endpoint"
	//	"cheops/api"
)


func main() {
	//	api.Routing()
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
	doca := replication.Replicant{
		MetaID: "42",
		Replicas: []replication.Replica{
			replication.Replica{Site: "Paris", ID: "65"},
			replication.Replica{Site: "Nantes", ID: "42"}},
		IsLeader: true,
		Logs:  []replication.Log {
			replication.Log{Operation: "incredible operation",
				Date: (time.Now())}}}
	key := database.CreateResource("replication", doca)
	doci := replication.Replicant{}
	database.ReadResource("replication", key, &doci)
	// fmt.Println(key)
	// doc := replication.CreateReplicant()
	// // log = replication.Log{Operation: "incredible operation", Date: (time.Now())}
	// // database.UpdateReplicant(doc)
	// doci := replication.Replicant{}
	// database.ReadResource("replication",  doc, &doci)
	// fmt.Println(doci)
	// fmt.Println(doci.Logs)
	// replication.DeleteReplicantWithKey(doc)
	// coli := database.CreateCollection(db, "endpoint")
	// coli.EnsurePersistentIndex(nil, []string{"Service", "Address"}, nil)
	// sitea := endpoint.CreateEndpoint("sitea", "0.0.0.0")
	// siteb := endpoint.CreateEndpoint("siteb", "0.0.0.1")
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
	add := endpoint.GetAddress("sitea")
	fmt.Println(add)
}
