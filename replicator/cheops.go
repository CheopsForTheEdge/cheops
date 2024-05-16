package replicator

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"strings"
	"time"

	"cheops.com/backends"
	"cheops.com/env"
	"cheops.com/model"
	"github.com/goombaio/dag"
)

// Do handles the request such that it is properly replicated and propagated.
// If the resource already exists and the list of sites is not nil or empty, it will be updated with the desired sites.
//
// The output is a chan of each individual reply as they arrive. After a timeout or all replies are sent, the chan is closed
func (r *Replicator) Do(ctx context.Context, sites []string, id string, request model.Operation) (replies chan model.Reply, err error) {
	log.Printf("New request: resourceId=%v requestId=%v\n", id, request.RequestId)

	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	err = r.Send(ctx, sites, id, json.RawMessage(b), "OPERATION")
	if err != nil {
		return nil, err
	}

	ret := make(chan model.Reply)

	go func() {
		repliesChan := r.watchReplies(ctx, request.RequestId)
		defer close(ret)

		// location -> struct{}{}
		expected := make(map[string]struct{})
		for _, location := range sites {
			expected[location] = struct{}{}
		}
		for len(expected) > 0 {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Printf("Error with runing %s: %s\n", request.RequestId, err)
					return
				}
			case reply := <-repliesChan:
				ret <- reply
				delete(expected, reply.Site)
			case <-time.After(20 * time.Second):
				// timeout
				for remaining := range expected {
					ret <- model.Reply{
						Site:       remaining,
						RequestId:  request.RequestId,
						ResourceId: id,
						Status:     "TIMEOUT",
						Cmd: model.Cmd{
							Input: request.Command.Command,
						},
					}
				}
				return
			}
		}

	}()

	return ret, nil
}

func (r *Replicator) watchReplies(ctx context.Context, requestId string) chan model.Reply {
	repliesChan := make(chan model.Reply)
	r.w.watch(func(d model.PayloadDocument) {
		if d.Type != "REPLY" {
			return
		}

		if d.TargetId != requestId {
			return
		}

		select {
		case <-ctx.Done():
			close(repliesChan)
			return
		default:
			var reply model.Reply
			err := json.Unmarshal(d.Payload, &reply)
			if err != nil {
				return
			}
			repliesChan <- reply
		}
	})
	return repliesChan
}

func (r *Replicator) watchRequests() {
	r.w.watch(func(d model.PayloadDocument) {
		if d.Type != "OPERATION" {
			return
		}

		forMe := false
		for _, location := range d.Locations {
			if location == env.Myfqdn {
				forMe = true
			}
		}
		if !forMe {
			return
		}

		var op model.Operation
		err := json.Unmarshal(d.Payload, &op)
		if err != nil {
			log.Printf("Error unmarshalling: %v\n", err)
			return
		}
		replies := r.run(context.Background(), d.Locations, op)
		for _, reply := range replies {
			b, err := json.Marshal(reply)
			if err != nil {
				log.Printf("Error marshaling reply: %v\n", err)
				continue
			}
			r.Send(context.Background(), d.Locations, op.RequestId, json.RawMessage(b), "REPLY")
		}
	})
}

func (r *Replicator) run(ctx context.Context, sites []string, operation model.Operation) []model.Reply {
	allDocs, err := r.getDocsForView("all-by-resourceid", operation.ResourceId)
	if err != nil {
		log.Printf("Couldn't run %v: %v\n", operation.RequestId, err)
		return nil
	}

	tree, existingReplies := makeTreeWithReplies(env.Myfqdn, allDocs)

	operationsToRun := findOperationsToRun(sites, tree, existingReplies)
	if len(operationsToRun) == 0 {
		return nil
	}

	commands := make([]backends.ShellCommand, 0)
	for _, operation := range operationsToRun {
		commands = append(commands, operation.Command)
		log.Printf("will run %s\n", operation.RequestId)
	}

	executionReplies, err := backends.Handle(context.TODO(), commands)

	status := "OK"
	if err != nil {
		status = "KO"
	}

	replies := make([]model.Reply, 0)
	for i, operation := range operationsToRun {
		log.Printf("Ran %s\n", operation.RequestId)
		cmd := model.Cmd{
			Input:  commands[i].Command,
			Output: executionReplies[i],
		}

		reply := model.Reply{
			Site:          env.Myfqdn,
			RequestId:     operation.RequestId,
			ResourceId:    operation.ResourceId,
			Status:        status,
			Cmd:           cmd,
			ExecutionTime: time.Now(),
		}
		replies = append(replies, reply)

	}

	return replies
}

// TODO most operations will happen towards the "end" of the tree,
// ie close to sink vertices. We should go forward rather than backwards
// because the limit (sink vertices instead of source vertices) is
// closer
func isDescendant(tree *dag.DAG, a *dag.Vertex, b *dag.Vertex) bool {
	isDescendant := false
	walkBackwardsFrom(tree, []*dag.Vertex{b}, func(d *dag.Vertex) bool {
		if vid(d) == vid(a) {
			isDescendant = true
			return false
		}
		return true
	})
	return isDescendant
}

func isSibling(tree *dag.DAG, a, b *dag.Vertex) bool {
	return !isDescendant(tree, a, b) && !isDescendant(tree, b, a)
}

func makeTreeWithReplies(site string, allDocs []model.PayloadDocument) (tree *dag.DAG, existingReplies map[string]struct{}) {
	existingReplies = make(map[string]struct{})
	tree = dag.NewDAG()

	for _, doc := range allDocs {
		switch doc.Type {
		case "OPERATION":
			var op model.Operation
			json.Unmarshal(doc.Payload, &op)
			// First pass: build all vertices only
			tree.AddVertex(dag.NewVertex(op.RequestId, dagNode{
				op:      &op,
				parents: doc.Parents,
			}))
		case "REPLY":
			var r model.Reply
			json.Unmarshal(doc.Payload, &r)
			if r.Site == site {
				existingReplies[r.RequestId] = struct{}{}
			}
		}
	}

	// Second pass: build all edges
	// Every vertex is a sink vertex because there are no edges yet
	for _, vertex := range tree.SinkVertices() {
		parents := vertex.Value.(dagNode).parents
		for _, parent := range parents {
			parentVertex, _ := tree.GetVertex(parent)
			tree.AddEdge(parentVertex, vertex)
		}
	}

	return tree, existingReplies
}

func (r *Replicator) RunDirect(ctx context.Context, command string) (string, error) {
	out, err := backends.Handle(ctx, []backends.ShellCommand{{Command: command}})
	return out[0], err
}

// findOperationsToRun outputs the operations to run for a resource
// based on the consistency model and the known states.
// See CONSISTENCY.md at the top of the project to understand what is happening here
func findOperationsToRun(sites []string, tree *dag.DAG, existingReplies map[string]struct{}) []model.Operation {

	// Step 1: on each site, find the last operation of type 3
	// Step 1bis: if there are no type 3 anywhere, the result is all operations since
	//            they are type 2. Return with this
	// Step 2: Find a winner (causally after all of them, or the RequestId is
	//         lexicographically after)
	// Step 3: Get all operations that are causally after

	lastIdempotentPerSite := make(map[string]*dag.Vertex)
	walkBackwards(tree, func(v *dag.Vertex) bool {
		op := v.Value.(dagNode).op
		if op.Type == model.OperationTypeIdempotent {
			existing, ok := lastIdempotentPerSite[op.Site]
			if !ok || isDescendant(tree, existing, v) {
				lastIdempotentPerSite[op.Site] = v
			}
		}

		if len(lastIdempotentPerSite) == len(sites) {
			return false
		}

		return true
	})

	// Everything is type 2, return it all
	if len(lastIdempotentPerSite) == 0 {
		torun := make([]model.Operation, 0)
		walkBackwards(tree, func(v *dag.Vertex) bool {
			op := v.Value.(dagNode).op
			if _, ok := existingReplies[op.RequestId]; !ok {
				torun = append(torun, *op)
			}
			return true
		})

		return torun
	}

	for site, v := range lastIdempotentPerSite {
		log.Printf("%s %s\n", site, vid(v))
	}

	var winning *dag.Vertex
	for _, v := range lastIdempotentPerSite {
		if winning == nil {
			winning = v
			continue
		}

		if isDescendant(tree, v, winning) {
			winning = v
		} else if isSibling(tree, v, winning) && strings.Compare(vid(v), vid(winning)) > 0 {
			winning = v
		}
	}

	torun := make([]model.Operation, 0)

	toCheck := make([]*dag.Vertex, 0)
	toCheck = append(toCheck, winning)
	toCheck = append(toCheck, findDescendants(tree, winning)...)
	for _, v := range toCheck {
		op := v.Value.(dagNode).op
		if _, ok := existingReplies[op.RequestId]; !ok {
			torun = append(torun, *op)
		}
	}
	return torun

}

type dagNode struct {
	op      *model.Operation
	parents []string
}

func walkBackwards(tree *dag.DAG, f func(d *dag.Vertex) bool) {
	walkBackwardsFrom(tree, tree.SinkVertices(), f)
}

func walkBackwardsFrom(tree *dag.DAG, nodes []*dag.Vertex, f func(d *dag.Vertex) bool) {
	doWalk := func(vs []*dag.Vertex) (next []*dag.Vertex) {
		next = make([]*dag.Vertex, 0)
		for _, v := range vs {
			if !f(v) {
				return nil
			}

			parents, err := tree.Predecessors(v)
			if err != nil {
				log.Printf("Error walking at %v: %v\n", v, err)
			}

			next = append(next, parents...)
		}
		return next
	}

	for len(nodes) > 0 {
		nodes = doWalk(nodes)
	}
}

// findDescendants will find all nodes that are descendants of root
// It doesn't include root.
// There is no order in the descendants (ie the erdor is not relevant)
func findDescendants(tree *dag.DAG, root *dag.Vertex) []*dag.Vertex {

	// Store found descendants in a map to avoid duplicates
	uniq := make(map[string]*dag.Vertex)

	doWalk := func(vs []*dag.Vertex) (next []*dag.Vertex) {
		next = make([]*dag.Vertex, 0)
		for _, v := range vs {
			if vid(v) != vid(root) {
				uniq[vid(v)] = v
			}
			children, err := tree.Successors(v)
			if err != nil {
				log.Printf("Error finding descendants at %v: %v\n", v, err)
			}

			next = append(next, children...)
		}
		return next
	}

	nodes := []*dag.Vertex{root}
	for len(nodes) > 0 {
		nodes = doWalk(nodes)
	}

	ret := make([]*dag.Vertex, 0)
	for _, v := range uniq {
		ret = append(ret, v)
	}

	// To make sure we always get the same order, sort them by id
	sort.Slice(ret, func(i, j int) bool {
		return strings.Compare(vid(ret[i]), vid(ret[j])) <= 0
	})
	return ret
}

func vid(v *dag.Vertex) string {
	return v.Value.(dagNode).op.RequestId
}
