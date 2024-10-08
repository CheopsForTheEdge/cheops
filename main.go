// Package main executes Cheops.
package main

import (
	"cheops.com/api"
	"cheops.com/env"
	"cheops.com/replicator"
)

var app = "k8s"

func main() {
	env.Set()
	repl := replicator.NewReplicator(7071)
	go api.Run(8079, repl)
	go api.RunChephren(8080, repl)
	select {}
}
