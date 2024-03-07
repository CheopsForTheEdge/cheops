// Package main executes Cheops.
package main

import (
	"cheops.com/api"
	"cheops.com/chephren"
	"cheops.com/env"
	"cheops.com/replicator"
)

var app = "k8s"

func main() {
	env.Set()
	repl := replicator.NewReplicator(7071)
	go api.Run(8079, repl)
	go chephren.Run(8080, repl)
	select {}
}
