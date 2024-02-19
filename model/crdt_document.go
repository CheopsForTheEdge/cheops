package model

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"cheops.com/backends"
)

type ResourceDocument struct {
	// Couchdb internal structs
	Id        string   `json:"_id,omitempty"`
	Rev       string   `json:"_rev,omitempty"`
	Conflicts []string `json:"_conflicts,omitempty"`
	Deleted   bool     `json:"_deleted,omitempty"`

	Locations []string
	Units     []CrdtUnit

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

type CrdtUnit struct {
	Generation uint64
	RequestId  string
	Command    backends.ShellCommand
	Time       time.Time
}

func ResolveConflicts(d ResourceDocument) (ResourceDocument, error) {
	conflicts := make([]ResourceDocument, 0)
	for _, rev := range d.Conflicts {
		url := fmt.Sprintf("http://localhost:5984/cheops/%s?rev=%s", d.Id, rev)
		conflictDocResp, err := http.Get(url)
		if err != nil {
			return ResourceDocument{}, fmt.Errorf("Couldn't get id=%s rev=%s: %v", d.Id, rev, err)
		}
		defer conflictDocResp.Body.Close()

		if conflictDocResp.StatusCode != http.StatusOK {
			return ResourceDocument{}, fmt.Errorf("Couldn't get id=%s rev=%s: %v", d.Id, rev, conflictDocResp.Status)
		}

		var conflictDoc ResourceDocument
		err = json.NewDecoder(conflictDocResp.Body).Decode(&conflictDoc)
		if err != nil {
			return ResourceDocument{}, fmt.Errorf("Couldn't get id=%s rev=%s: %v", d.Id, rev, err)
		}
		conflicts = append(conflicts, conflictDoc)
	}

	return resolveConflictsWithDocs(d, conflicts), nil
}

func resolveConflictsWithDocs(winner ResourceDocument, conflicts []ResourceDocument) ResourceDocument {
	uniqUnits := make(map[string]CrdtUnit)
	for _, unit := range winner.Units {
		uniqUnits[unit.RequestId] = unit
	}
	for _, doc := range conflicts {
		for _, unit := range doc.Units {
			uniqUnits[unit.RequestId] = unit
		}
	}

	list := make([]CrdtUnit, 0)
	for _, unit := range uniqUnits {
		list = append(list, unit)
	}

	SortUnits(list)

	winner.Conflicts = []string{}
	winner.Units = list

	return winner
}

func SortUnits(list []CrdtUnit) {
	sort.Slice(list, func(i, j int) bool {
		if list[i].Generation < list[j].Generation {
			return true
		} else if list[i].Generation > list[j].Generation {
			return false
		} else {
			return strings.Compare(list[i].RequestId, list[j].RequestId) <= 0
		}
	})
}
