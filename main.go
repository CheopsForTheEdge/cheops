package main

import (
	"os"
	"fmt"
	//"time"
	"cheops/database"
	"cheops/replication"
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
	//c := database.Connection()
	//db := database.ConnectToDatabase(c)
	//col := database.ConnectToCollection(db , "cheopsmodel")
	// doc := replication.Replicant{
	// 	MetaID: "42",
	// 	Replicas: []replication.Replica{
	// 		replication.Replica{Site: "Paris", ID: "65"},
	// 		replication.Replica{Site: "Nantes", ID: "42"}},
	// 	IsLeader: true,
	// 	Logs:  []replication.Log {
	// 		replication.Log{Operation: "incredible operation",
	// 			Date: (time.Now())}}}
	// key := database.CreateResource("replication", doc)
	// doci := replication.Replicant{}
	// database.ReadResource("replication", key, &doci)
	// fmt.Println(doci)
	doc := replication.CreateReplicant()
	// log = replication.Log{Operation: "incredible operation", Date: (time.Now())}
	// database.UpdateReplicant(doc)
	doci := replication.Replicant{}
	database.ReadResource("replication",  doc, &doci)
	fmt.Println(doci)
	fmt.Println(doci.Logs)
	replication.DeleteReplicantWithKey(doc)
}
