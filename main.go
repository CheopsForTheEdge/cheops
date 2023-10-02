// Package main executes Cheops.
package main

import (
	"cheops.com/api"

	// "cheops.com/client"
	"context"

	"cheops.com/kubernetes"
)

var app = "k8s"

func main() {

	kubernetes.Run(context.Background())
	go api.Admin(8081)
	go api.BestEffort(8080)
	go api.Sync(8079)
	go api.Raft(7070)
	select {}
}
