package main

import (
	"cheops.com/api"
	"cheops.com/endpoint"
	"cheops.com/utils"
	"fmt"
	"os"
	"time"
	//"cheops.com/client"
	"cheops.com/operation"
)

var app = "k8s"

func main() {


	var conf = utils.Conf

	// https://chriswiegman.com/2019/01/ensuring-the-file-path-is-present-to-create-a-file-in-golang/
	arango_file := "/root/arango"
	if _, err := os.Stat(arango_file); os.IsNotExist(err) {
		fmt.Printf("Using credentials: mdp=%s, pwd=%s\n",
			conf.Database.DBUser, conf.Database.DBPassword)
		utils.PrepareForExecution()
		os.MkdirAll(arango_file, 0700)
	}

	test_file := "/root/test"
	if _, err := os.Stat(test_file); os.IsNotExist(err) {
/*		for _, site := range conf.Sites.Site {
			prepSite := endpoint.Site{SiteName: site.name}
		}*/
		endpoint.CreateSite("Site1", "172.16.96.11")
        endpoint.CreateSite("Site2", "172.16.96.12")
        endpoint.CreateSite("Site3", "172.16.96.13")

		col := utils.ConnectionToCorrectCollection("replications")

		col.EnsurePersistentIndex(nil, []string{"MetaID", "IsLeader"}, nil)
		doca := operation.Replicant{
			MetaID: "42",
			Replicas: []operation.Replica{
				operation.Replica{Site: endpoint.Site{"Paris", "127.0.0.1"}, ID: "65"},
				operation.Replica{Site: endpoint.Site{"Nantes", "192.168.0.1"}, ID: "42"}},
			Leader: "Paris",
			Logs:  []operation.Log {
				operation.Log{Operation: "incredible operation",
					Date: (time.Now())}}}
		utils.CreateResource("replications", doca)
		coli := utils.ConnectionToCorrectCollection("sites")
		coli.EnsurePersistentIndex(nil, []string{"Site", "Address"}, nil)
		os.MkdirAll(test_file, 0700)


	}

	api.Routing()
}
