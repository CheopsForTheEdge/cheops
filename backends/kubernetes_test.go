package backends

import (
	"context"
	"testing"
)

func TestHandle(t *testing.T) {
	bodies := []string{"echo -n foo"}
	replies, err := Handle(context.Background(), bodies)
	if err != nil {
		t.Fatalf("Error when executing cmd: %v\n", err)
	}

	if len(replies) != 1 {
		t.Fatalf("Incorrect number of responses, got %v, expected 1\n", len(replies))
	}

	if replies[0] != "foo" {
		t.Fatalf("Invalid reply, got [%v], expected [foo]\n", replies[0])
	}
}
