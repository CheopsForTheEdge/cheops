package replicator

import (
	"strings"

	"cheops.com/model"
)

// findOperationsToRun outputs the operations to run for a resource
// based on the consistency model and the operations that have
// already been ran.
//
// site is the site to run operations on
//
// documents is the list of documents for the resource, one by site
//
// replies is all the replies for all operations for this resource
//
// See CONSISTENCY.md at the top of the project to understand what is happening here
func findOperationsToRun(site string, documents []model.ResourceDocument, replies []model.ReplyDocument) []model.Operation {
	// First step: for each site, find the last operation of type 3 that is alive
	// and put it with the following type 2 operations in a bag
	// Second step: find the winning type 3 operation
	// Third step: apply the winning type 3 operation and the following type 2
	//				operations on all sites except the one that wins
	//
	// Second step bis: if there are no type 3 operations, it's all type 2:
	// play it all

	// Index replies to easily find dead and already ran operations
	// requestId -> site
	repliesByOperationAndSite := make(map[string]map[string]struct{})
	for _, reply := range replies {
		if _, ok := repliesByOperationAndSite[reply.RequestId]; !ok {
			repliesByOperationAndSite[reply.RequestId] = make(map[string]struct{})
		}
		repliesByOperationAndSite[reply.RequestId][reply.Site] = struct{}{}
	}

	deadOperations := make(map[string]struct{})
	for requestId, replies := range repliesByOperationAndSite {
	findOp:
		for _, document := range documents {
			for _, operation := range document.Operations {
				if operation.RequestId == requestId {
					if len(replies) == len(document.Locations) {
						deadOperations[requestId] = struct{}{}
						break findOp
					}
				}
			}
		}
	}

	// Gather "one type 3 and the rest of type 2" for each document
	suits := make([][]model.Operation, 0)
	hasTypeC := false
	for _, document := range documents {
		suitStart := -1
		for i := range document.Operations {
			idx := len(document.Operations) - 1 - i
			operation := document.Operations[idx]
			if _, dead := deadOperations[operation.RequestId]; dead {
				break
			}
			suitStart = idx
			if operation.Type == model.OperationTypeIdempotent {
				hasTypeC = true
				break
			}
		}
		if suitStart == -1 {
			suits = append(suits, []model.Operation{})
		} else {
			suit := document.Operations[suitStart:]
			suits = append(suits, suit)
		}
	}

	// If there is no type 3, return the rest
	if !hasTypeC {
		ret := make([]model.Operation, 0)
		for _, suit := range suits {
			for _, operation := range suit {
				if _, alreadyRanAny := repliesByOperationAndSite[operation.RequestId]; alreadyRanAny {

					if _, alreadyRan := repliesByOperationAndSite[operation.RequestId][site]; alreadyRan {
						continue
					}
				}
				ret = append(ret, operation)
			}
		}

		return ret
	}

	// If there is a type C, check the first operation if it is a type C, get the highest by comparing
	// requestid: that's the winner
	idx := -1
	var max string
	for i, suit := range suits {
		if len(suit) == 0 {
			continue
		}
		if suit[0].Type == model.OperationTypeIdempotent {
			if strings.Compare(suit[0].RequestId, max) > 0 {
				max = suit[0].RequestId
				idx = i
			}
		}
	}

	for i, doc := range documents {
		if doc.Site != site {
			continue
		}

		if i == idx {
			return []model.Operation{}
		} else {
			// As an optimization, if the beginning of the winning suit matches the end of the document
			// and the matching operations have already been played, then we can only run the rest.
			// Let's make it simple for now and just blindly run everything, it's still correct
			return suits[idx]
		}
	}

	return nil
}
