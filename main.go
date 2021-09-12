package main

import (
	"os"
	"fmt"
	"time"
	"cheops/database"
	"cheops/replication"
	//	"cheops/api"
)


func main() {
	//	api.Routing()
	// https://chriswiegman.com/2019/01/ensuring-the-file-path-is-present-to-create-a-file-in-golang/
	check_file := "/root/arango"
	if _, err := os.Stat(check_file); os.IsNotExist(err) {
		database.PrepareForExecution("cheops", "cheopsmodel")
		os.MkdirAll(check_file, 0700)
	}
	c := database.Connection()
	db := database.ConnectToDatabase(c, "cheops")
	col := database.ConnectToCollection(db , "cheopsmodel")
	doc := replication.Replicant{
		MetaID: "42",
		Replicas: []replication.Replica{
			replication.Replica{Site: "Paris", ID: "65"},
			replication.Replica{Site: "Nantes", ID: "42"}},
		IsLeader: true,
		Logs:  []replication.Log {
			replication.Log{Operation: "incredible operation",
				Date: (time.Now())}}}
	key := database.CreateResource(col, doc)
	doci := replication.Replicant{}
	database.ReadResource(col, key, &doci)
	fmt.Println(doci)
}
