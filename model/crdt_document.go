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

type Operation struct {
	Type      OperationType
	RequestId string
	Command   backends.ShellCommand
	Time      time.Time
}
