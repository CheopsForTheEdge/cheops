// Package main executes Cheops.
package main

import (
	"cheops.com/api"
	"cheops.com/replicator"

	"context"

	"cheops.com/backends"
)

var app = "k8s"

func main() {
	backends.Kubernetes(context.Background())
	repl := replicator.NewDoer()
	go api.Sync(8079, repl)
	select {}
}
