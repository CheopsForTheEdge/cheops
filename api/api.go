package api

import (
	"log"
	"os"
)

var mode syncMode

type syncMode int

const (
	raftMode syncMode = iota
	crdtMode
)

func init() {
	m, ok := os.LookupEnv("MODE")
	if !ok {
		log.Fatal("My FQDN must be given with the MYFQDN environment variable !")
	}
	switch m {
	case "raft":
		mode = raftMode
	case "crdt":
		mode = crdtMode
	default:
		log.Fatalf("Invalid MODE, want 'raft' or 'crdt', got [%v]\n", m)
	}
}
