// replicator manages taking a request and running it accross the network
package replicator

import (
	"context"
	"net/http"
)

func NewDoer() Doer {
	return newCrdt(7071)
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

	ResourceId string

	// Only filled when this is a request
	Method string
	Path   string

	Header http.Header
	Body   string

	// The site where this payload comes from
	Site string
}

func (p Payload) IsRequest() bool {
	return p.Method != ""
}

type LogDump struct {
	Request Payload
	Replies []Payload
}
