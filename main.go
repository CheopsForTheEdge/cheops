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

	// We want something layer by layer, like
	//
	// back := backends.Kubernetes()
	//
	// repl := replicator.Raft(7070, back.SitesExtractor, back.Executor)
	// // or
	// repl := replicator.Raft(7070, back)
	//
	// go api.Sync(8079, repl)

	backends.Kubernetes(context.Background())
	var repl Doer
	if mode == raftMode {
		repl = replicator.RaftDoer(7070)
	} else if mode == crdtMode {
		repl = replicator.CrdtDoer()
	}
	go api.Sync(8079, repl)
	select {}
}
