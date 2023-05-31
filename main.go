// Package main executes Cheops.
package main

import (
	"cheops.com/api"
	"cheops.com/endpoint"
	"cheops.com/utils"
	"fmt"
	"os"
	"time"
	// "cheops.com/client"
	"cheops.com/kubernetes"
	"cheops.com/operation"
	"context"
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

		col := utils.ConnectionToCorrectCollection("replications")

		col.EnsurePersistentIndex(nil, []string{"MetaID", "IsLeader"}, nil)
		date := (time.Now())
		doca := operation.Replicant{
			MetaID: "42",
			Replicas: []operation.Replica{
				operation.Replica{Site: endpoint.Site{"Paris", "127.0.0.1",
					0}, ID: "65",
					Logs: []operation.Log{operation.Log{Operation: "incredible operation",
						Date: date}}},
				operation.Replica{Site: endpoint.Site{"Nantes",
					"192.168.0.1", 0}, ID: "42",
					Logs: []operation.Log{operation.Log{Operation: "incredible operation",
						Date: date}}},
			},
			Leader: "Paris",
		}
		utils.CreateResource("replications", doca)
		coli := utils.ConnectionToCorrectCollection("sites")
		coli.EnsurePersistentIndex(nil, []string{"Site", "Address"}, nil)
		os.MkdirAll(test_file, 0700)

	}

	kubernetes.Run(context.Background())
	api.Routing()
}
