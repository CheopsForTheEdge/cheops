package replicator

import (
	"context"
)

type CRDT struct {
}

var crdt *CRDT = newCRDT()

func newCRDT() *CRDT {
	c := &CRDT{}
	c.replicate()
	return c
}

func (c *CRDT) Do(ctx context.Context, sites []string, operation Payload) (reply Payload, err error) {
	return Payload{}, nil
}
