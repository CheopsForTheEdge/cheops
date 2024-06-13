package model

import (
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

	Operations []Operation
	Config     Config

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
	ExecutionTime time.Time

	// Always REPLY
	Type string
}

type Cmd struct {
	Input  string
	Output string
}

type RelationType string

const (
	RelationTypeIterative RelationType = "1"

	RelationTypeCommutative RelationType = "2"

	RelationTypeExclusive RelationType = "3"
)

type OperationType string

type Operation struct {
	Type      OperationType
	RequestId string
	Command   backends.ShellCommand
	Time      time.Time
}

type Config struct {
	RelationshipMatrix []Relationship
}

type Relationship struct {
	Before OperationType
	After  OperationType
	Result []int
}
