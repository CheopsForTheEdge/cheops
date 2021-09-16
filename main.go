package main

import (
	"cheops/operation/replication"
	"fmt"
	"os"
	"time"
	"cheops/database"
	"cheops/endpoint"
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
	c := database.Connection()
	db := database.ConnectToDatabase(c)
	col := database.ConnectionToCorrectCollection("replication")
	col.EnsurePersistentIndex(nil, []string{"MetaID", "IsLeader"}, nil)
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
	// doci := replication.Replicant{}
	// database.ReadResource("replication", key, &doci)
	fmt.Println(key)
	// doc := replication.CreateReplicant()
	// // log = replication.Log{Operation: "incredible operation", Date: (time.Now())}
	// // database.UpdateReplicant(doc)
	// doci := replication.Replicant{}
	// database.ReadResource("replication",  doc, &doci)
	// fmt.Println(doci)
	// fmt.Println(doci.Logs)
	// replication.DeleteReplicantWithKey(doc)
	// col := database.CreateCollection(db, "endpoint")
	// col.EnsurePersistentIndex(nil, []string{"Service", "Address"}, nil)
	sitea := endpoint.CreateEndpoint("sitea", "0.0.0.0")
	siteb := endpoint.CreateEndpoint("siteb", "0.0.0.1")
	query := "FOR end IN replication FILTER end.MetaID == '42' RETURN end"
	// bindvars := map[string]interface{}{ "name": "sitea", }
	cursor, err := db.Query(nil, query, nil)
	if err != nil {
		// handle error
	}
	fmt.Println(sitea)
	fmt.Println(siteb)
	// fmt.Println(cursor)
	var result replication.Replicant
	cursor.ReadDocument(nil, result)
	fmt.Println(result)
	defer cursor.Close()
}
