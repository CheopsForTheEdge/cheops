package main

import (
	"cheops.com/api"
	"cheops.com/database"
	"cheops.com/endpoint"
	"fmt"
	"os"
	"time"
	//"cheops.com/client"
	"cheops.com/operation"
)


func main() {
	// https://chriswiegman.com/2019/01/ensuring-the-file-path-is-present-to-create-a-file-in-golang/
	arango_file := "/root/arango"
	if _, err := os.Stat(arango_file); os.IsNotExist(err) {
		database.PrepareForExecution("cheops", "replication")
		os.MkdirAll(arango_file, 0700)
	}

	test_file := "/root/test"
	if _, err := os.Stat(test_file); os.IsNotExist(err) {
		c := database.Connection()
		db := database.ConnectToDatabase(c)
		database.CreateCollection(db, "replication")
		col := database.ConnectionToCorrectCollection("replication")
		col.EnsurePersistentIndex(nil, []string{"MetaID", "IsLeader"}, nil)
		doca := operation.Replicant{
			MetaID: "42",
			Replicas: []operation.Replica{
				operation.Replica{Site: "Paris", ID: "65"},
				operation.Replica{Site: "Nantes", ID: "42"}},
			IsLeader: true,
			Logs:  []operation.Log {
				operation.Log{Operation: "incredible operation",
					Date: (time.Now())}}}
		key := database.CreateResource("replication", doca)
		fmt.Println(key)
		coli := database.CreateCollection(db, "endpoint")
		coli.EnsurePersistentIndex(nil, []string{"Site", "Address"}, nil)
		endpoint.CreateEndpoint("site3", "localhost:8080/endpoint/getaddress/site3")
		endpoint.CreateEndpoint("site4", "localhost:8080/endpoint/getaddress/site4")
		database.CreateCollection(db, "operation")
		os.MkdirAll(test_file, 0700)
	}

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
	// col = database.ConnectionToCorrectCollection("replication")
	// col.EnsurePersistentIndex(nil, []string{"MetaID", "IsLeader"}, nil)
	// endpoint.CreateEndpoint("site1", "10.12.1.8")
	// endpoint.CreateEndpoint("site2", "10.12.65.7")
	// doca := operation.Operation{
	// 	Operation: "&",
	// 	Sites: []string{"site1", "site2"},
	// 	Platform: "openstack",
	// 	Resource: "image",
	// 	PlatformOperation: "create",
	// 	ExtraArgs: []string{"lol"},
	// 	Request: "openstack image create lol"}
	// key := database.CreateResource("replication", doca)
	// fmt.Println(key)
	api.Routing()
}
