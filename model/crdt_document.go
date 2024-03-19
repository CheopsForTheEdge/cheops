package model

import (
	"fmt"
	"time"

	"cheops.com/backends"
)

type ResourceDocument struct {
	// Couchdb internal structs
	Id        string   `json:"_id,omitempty"`
	Rev       string   `json:"_rev,omitempty"`
	Conflicts []string `json:"_conflicts,omitempty"`
	Deleted   bool     `json:"_deleted,omitempty"`

	// Desired locations
	Locations []string

	ResourceId string
	Site       string
	Operations []Operation

	// Always RESOURCE
	Type string
}

type ReplyDocument struct {
	Locations  []string
	Site       string
	RequestId  string
	ResourceId string

	// "OK" or "KO"
	Status string
	Cmd

	// Always REPLY
	Type string
}

type DeleteDocument struct {
	ResourceId  string
	ResourceRev string

	// Will always be a single string with the site,
	// but we reuse the existing infrastructure that manages replication
	// for a list of locations
	Locations []string

	// Always DELETE
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
	Type      OperationType
	RequestId string
	Command   backends.ShellCommand
	Time      time.Time

	// Site -> height
	KnownState map[string]int
}
