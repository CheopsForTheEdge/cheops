package main

import (
	//	"fmt"
	"time"
	"cheops/database"
	"cheops/replication"
	//	"cheops/api"
)


func main() {
	//	api.Routing()
	//	database.PrepareForExecution("cheops", "cheopsmodel")
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
			replication.Log{Operation: (time.Now()).String()}}}
	database.CreateResource(col, doc)
}