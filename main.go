// Package main executes Cheops.
package main

import (
	"cheops.com/api"
	"cheops.com/replicator"
)

var app = "k8s"

func main() {
	repl := replicator.NewDoer()
	go api.Sync(8079, repl)
	select {}
}
