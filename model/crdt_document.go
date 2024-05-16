package model

import (
	"encoding/json"
	"fmt"
	"time"

	"cheops.com/backends"
)

type PayloadDocument struct {
	// Couchdb internal structs
	Id        string   `json:"_id,omitempty"`
	Rev       string   `json:"_rev,omitempty"`
	Conflicts []string `json:"_conflicts,omitempty"`
	Deleted   bool     `json:"_deleted,omitempty"`

	// Desired locations
	Locations []string

	// List of requestIds, empty string if none
	Parents []string
	Payload json.RawMessage

	// Can be a resourceid for a request,
	// or a requestid for a reply
	TargetId string

	// OPERATION or REPLY
	Type string
}

type Cmd struct {
	Input  string
	Output string
}

type OperationType string

const (
	// Idempotent and Commutative (Type A)
	OperationTypeCommutativeIdempotent OperationType = "1"

	// Commutative only (Type B)
	OperationTypeCommutative OperationType = "2"

	// Idempotent only (Type C)
	OperationTypeIdempotent OperationType = "3"

	// Not commutative, not idempotent (Type D)
	OperationTypeNothing OperationType = "4"
)

func OperationTypeFrom(input string) (OperationType, error) {
	op := OperationType(input)
	if op != OperationTypeCommutativeIdempotent &&
		op != OperationTypeCommutative &&
		op != OperationTypeIdempotent &&
		op != OperationTypeNothing {
		return "", fmt.Errorf("Unknown operation type")
	}

	return OperationType(input), nil
}

type Operation struct {
	RequestId string
	Type      OperationType
	Command   backends.ShellCommand
	Time      time.Time

	// Site -> height
	KnownState map[string]int

	ResourceId string
	Site       string
}

type Reply struct {
	Site       string
	RequestId  string
	ResourceId string

	// "OK" or "KO"
	Status string
	Cmd
	ExecutionTime time.Time
}
