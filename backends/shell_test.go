package backends

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestHandleSimple(t *testing.T) {
	commands := []ShellCommand{{Command: "echo -n foo"}}
	replies, err := Handle(context.Background(), commands)
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

func TestHandleRedirect(t *testing.T) {
	defer func() {
		os.RemoveAll("/tmp/cheopstest")
	}()

	bodies := []ShellCommand{{Command: "mkdir /tmp/cheopstest"}, {Command: "echo something > /tmp/cheopstest/file"}}
	replies, err := Handle(context.Background(), bodies)
	if err != nil {
		t.Fatalf("Error when executing cmd: %v\noutputs: %v", err, replies)
	}

	if len(replies) != 2 {
		t.Fatalf("Incorrect number of responses, got %v, expected 1\n", len(replies))
	}

	content, err := os.ReadFile("/tmp/cheopstest/file")
	if err != nil {
		t.Fatalf("Couldn't read test file: %v\n", err)
	}
	if bytes.Compare(content, []byte("something\n")) != 0 {
		t.Fatalf("Invalid content, got [%v], expected [something\\n]\n", content)
	}
}
