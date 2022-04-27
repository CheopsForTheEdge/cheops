package main

import (
	"cheops.com/api"
	"cheops.com/config"
	"cheops.com/database"
	"cheops.com/endpoint"
	"fmt"
	"os"
	"time"
	//"cheops.com/client"
	"cheops.com/operation"
)

var app = "k8s"

func main() {

	var conf = config.Conf

	// https://chriswiegman.com/2019/01/ensuring-the-file-path-is-present-to-create-a-file-in-golang/
	arango_file := "/root/arango"
	if _, err := os.Stat(arango_file); os.IsNotExist(err) {
		fmt.Printf("Using credentials: mdp=%s, pwd=%s\n",
			conf.Database.DBUser, conf.Database.DBPassword)
		database.PrepareForExecution()
		os.MkdirAll(arango_file, 0700)
	}

	test_file := "/root/test"
	if _, err := os.Stat(test_file); os.IsNotExist(err) {
/*		for _, site := range conf.Sites.Site {
			prepSite := endpoint.Site{SiteName: site.name}
		}*/
		endpoint.CreateSite("Site1", "amqp://guest:guest@172.16.96.17:5672/")
		endpoint.CreateSite("Site2", "amqp://guest:guest@172.16.96.18:5672/")
		endpoint.CreateSite("Site3", "amqp://guest:guest@172.16.96.19:5672/")
		col := database.ConnectionToCorrectCollection("replications")
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
		database.CreateResource("replications", doca)
		coli := database.ConnectionToCorrectCollection("sites")
		coli.EnsurePersistentIndex(nil, []string{"Site", "Address"}, nil)
		os.MkdirAll(test_file, 0700)


	}

	api.Routing()
}
