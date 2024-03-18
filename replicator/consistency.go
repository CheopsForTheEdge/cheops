package replicator

import (
	"strings"

	"cheops.com/model"
)

// findOperationsToRun outputs the operations to run for a resource
// based on the consistency model and the known states
//
// site is the site to run operations on
//
// documents is the list of documents for the resource, one by site
//
// See CONSISTENCY.md at the top of the project to understand what is happening here
func findOperationsToRun(site string, documents []model.ResourceDocument, replies []model.ReplyDocument) []model.Operation {
	if len(documents) == 0 {
		return nil
	}

	// Step 1: on each site, find the best operation of type 3 that is
	//         causally concurrent or after the last local operation
	// Step 2: pick all the operations causally after the winner

	var localLast model.Operation
	for _, document := range documents {
		if document.Site == site {
			localLast = document.Operations[len(document.Operations)-1]
		}
	}
	if localLast.RequestId == "" {
		// No document for local site, make a fake operation that is before
		// everything else
		localLast.KnownState = make(map[string]int)
		for _, location := range documents[0].Locations {
			localLast.KnownState[string(location)] = -1
		}
	}

	firstToRun := localLast
	for _, document := range documents {

	document:
		for i := range document.Operations {
			idx := len(document.Operations) - 1 - i

			op := document.Operations[idx]
			if op.Type == model.OperationTypeIdempotent {
				if isAfter(op, firstToRun) {
					firstToRun = op
					break document
				} else if firstToRun.Type != model.OperationTypeIdempotent {
					// True iff firstToRun is localLast
					firstToRun = op
					break document
				} else if isConcurrent(op, firstToRun) && strings.Compare(op.RequestId, firstToRun.RequestId) > 0 {
					firstToRun = op
					break document
				}
			}
		}
	}

	alreadyRan := func(op model.Operation) bool {
		for _, reply := range replies {
			if reply.RequestId == op.RequestId && reply.Site == site {
				return true
			}
		}
		return false
	}

	// Now gather all operations to run
	// It will only be type 2, so no need to care about the order
	operationsToRun := make([]model.Operation, 0)
	if !alreadyRan(firstToRun) && firstToRun.RequestId != "" {
		operationsToRun = append(operationsToRun, firstToRun)
	}

	for _, document := range documents {
		for i := range document.Operations {
			idx := len(document.Operations) - 1 - i
			op := document.Operations[idx]
			if op.Type == model.OperationTypeIdempotent {
				break
			}
			if firstToRun.Type == model.OperationTypeIdempotent {
				// Type 3 first, take every type 2 that is causally after
				if isAfter(op, firstToRun) && !alreadyRan(op) {
					operationsToRun = append(operationsToRun, op)
				}
			} else if firstToRun.Type != model.OperationTypeIdempotent {
				// Type 2 first, take every type 2 that is concurrent or after
				if (isAfter(op, firstToRun) || isConcurrent(op, firstToRun)) && !alreadyRan(op) {
					operationsToRun = append(operationsToRun, op)
				}
			}
		}
	}

	return operationsToRun
}

// isAfter returns true if a is strictly after b, false if it is concurrent or before.
// An operation a is after operation b if all known state of a is
// higher than the known state of b
func isAfter(a, b model.Operation) bool {
	atLeastOneAfter := false
	atLeastOneBefore := false
	for site := range a.KnownState {
		if a.KnownState[site] > b.KnownState[site] {
			atLeastOneAfter = true
		} else if a.KnownState[site] < b.KnownState[site] {
			atLeastOneBefore = true
		}
	}

	return atLeastOneAfter && !atLeastOneBefore
}

// isConcurrent returns true if a and b are concurrent, false otherwise.
// An operation a is concurrent to operation b if it happened after or at the same
// time on one site, and before or at the same time on another.
func isConcurrent(a, b model.Operation) bool {
	atLeastOneAfter := false
	atLeastOneBefore := false
	atLeastOneEqual := false
	for site := range a.KnownState {
		if a.KnownState[site] < b.KnownState[site] {
			atLeastOneBefore = true
		} else if a.KnownState[site] == b.KnownState[site] {
			atLeastOneEqual = true
		} else {
			atLeastOneAfter = true
		}
	}

	if atLeastOneAfter && atLeastOneBefore ||
		atLeastOneBefore && atLeastOneEqual ||
		atLeastOneAfter && atLeastOneEqual {
		return true
	}

	return false
}
