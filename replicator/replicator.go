// replicator manages taking a request and running it accross the network
package replicator

import (
	"context"
	"log"
	"net/http"
	"os"
)

func NewDoer() Doer {
	m, ok := os.LookupEnv("MODE")
	if !ok {
		log.Fatal("My FQDN must be given with the MYFQDN environment variable !")
	}
	switch m {
	case "raft":
		return newRaft(7070)
	case "crdt":
		return newCrdt()
	default:
		log.Fatalf("Invalid MODE, want 'raft' or 'crdt', got [%v]\n", m)
	}

	// unreachable
	return nil
}

// Doer is the interface that replicators must implement
type Doer interface {

	// Do takes a request and a number of sites, replicates the operation on all sites,
	// and waits for all of them to reply.
	// A reply is sent back to the caller
	Do(ctx context.Context, sites []string, operation Payload) (reply Payload, err error)
}

// Payload represents a query to be run on the network
type Payload struct {
	RequestId string

	// Only filled when this is a request
	Method string
	Path   string

	Header http.Header
	Body   []byte

	// The site where this payload comes from
	Site string
}
