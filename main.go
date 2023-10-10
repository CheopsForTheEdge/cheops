// Package main executes Cheops.
package main

import (
	"cheops.com/api"
	"cheops.com/replicator"
)

var app = "k8s"

func main() {
	repl := replicator.NewReplicator(7071)
	go api.Sync(8079, repl)
	select {}
}
