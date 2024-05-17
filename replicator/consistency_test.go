package replicator

import (
	"encoding/json"
	"testing"

	"cheops.com/model"
)

type testVector struct {
	name       string
	sites      []string
	operations []model.Operation
	parentage  [][]string
	replies    []model.Reply

	// For each site, the list of operations to run
	torun map[string][]string
}

func TestFindUnitsToRun(t *testing.T) {
	tvs := []testVector{
		{
			name:  "simple",
			sites: []string{"S1", "S2"},
			operations: []model.Operation{
				{
					Site:      "S1",
					Type:      model.OperationTypeCommutativeIdempotent,
					RequestId: "d1-1",
				},
			},
			parentage: [][]string{
				nil,
			},
			replies: []model.Reply{
				{Site: "S1", RequestId: "d1-1"},
			},
			torun: map[string][]string{
				"S1": []string{},
				"S2": []string{"d1-1"},
			},
		},
		{
			name:  "noTypeC",
			sites: []string{"S1", "S2"},
			operations: []model.Operation{
				{
					Site:      "S1",
					Type:      model.OperationTypeCommutativeIdempotent,
					RequestId: "d1-1",
				}, {
					Site:      "S1",
					Type:      model.OperationTypeCommutative,
					RequestId: "d1-2",
				},
				{
					Site:      "S2",
					Type:      model.OperationTypeCommutative,
					RequestId: "d2-1",
				}, {
					Site:      "S2",
					Type:      model.OperationTypeCommutativeIdempotent,
					RequestId: "d2-2",
				}, {
					Site:      "S2",
					Type:      model.OperationTypeCommutativeIdempotent,
					RequestId: "d2-3",
				},
			},
			parentage: [][]string{
				nil,
				{"d1-1"},
				{"d1-1"},
				{"d2-1"},
				{"d2-2"},
			},
			replies: []model.Reply{
				{Site: "S1", RequestId: "d1-1"},
				{Site: "S2", RequestId: "d1-1"},
				{Site: "S2", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d2-2"},
			},
			torun: map[string][]string{
				"S1": []string{"d1-2", "d2-3", "d2-2", "d2-1"},
				"S2": []string{"d1-2", "d2-3"},
			},
		}, {
			name:  "withTypeC",
			sites: []string{"S1", "S2"},
			operations: []model.Operation{
				{
					Type:      model.OperationTypeCommutativeIdempotent,
					Site:      "S1",
					RequestId: "d1-1",
				}, {
					Type:      model.OperationTypeIdempotent,
					Site:      "S1",
					RequestId: "d1-2",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S1",
					RequestId: "d1-3",
				}, {
					// Type C but dead, should be skipped
					Type:      model.OperationTypeIdempotent,
					Site:      "S2",
					RequestId: "d2-1",
				}, {
					Type:      model.OperationTypeIdempotent,
					Site:      "S2",
					RequestId: "d2-2",
				}, {
					Type:      model.OperationTypeCommutativeIdempotent,
					Site:      "S2",
					RequestId: "d2-3",
				},
			},
			parentage: [][]string{
				nil,
				{"d1-1", "d2-1"},
				{"d1-2"},
				nil,
				{"d2-1"},
				{"d2-2"},
			},
			replies: []model.Reply{
				{Site: "S1", RequestId: "d1-1"},
				{Site: "S1", RequestId: "d1-2"},
				{Site: "S1", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d2-2"},
			},
			torun: map[string][]string{
				"S1": []string{"d2-2", "d2-3"},
				"S2": []string{"d2-3"},
			},
		}, {
			name: "withTypeC2",
			operations: []model.Operation{
				{
					Type:      model.OperationTypeIdempotent,
					Site:      "S1",
					RequestId: "d1-1",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S1",
					RequestId: "d1-2",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S2",
					RequestId: "d2-1",
				}, {
					Type:      model.OperationTypeCommutativeIdempotent,
					Site:      "S2",
					RequestId: "d2-2",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S2",
					RequestId: "d2-3",
				},
			},
			parentage: [][]string{
				nil,
				{"d1-1", "d2-1"},
				nil,
				{"d1-1", "d2-1"},
				{"d2-2"},
			},
			replies: []model.Reply{
				{Site: "S1", RequestId: "d1-1"},
				{Site: "S1", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d1-1"},
				{Site: "S2", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d2-2"},
			},
			torun: map[string][]string{
				"S1": []string{"d1-2", "d2-3", "d2-2"},
				"s2": []string{"d1-2", "d2-3"},
			},
		}, {
			name:  "withTypeC3",
			sites: []string{"S1", "S2", "S3"},
			operations: []model.Operation{
				{
					Type:      model.OperationTypeIdempotent,
					Site:      "S1",
					RequestId: "d1-1",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S1",
					RequestId: "d1-2",
				}, {
					Type:      model.OperationTypeIdempotent,
					Site:      "S1",
					RequestId: "d1-3",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S1",
					RequestId: "d1-4",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S2",
					RequestId: "d2-1",
				}, {
					Type:      model.OperationTypeIdempotent,
					Site:      "S2",
					RequestId: "d2-2",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S2",
					RequestId: "d2-3",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S3",
					RequestId: "d3-1",
				}, {
					Type:      model.OperationTypeCommutativeIdempotent,
					Site:      "S3",
					RequestId: "d3-2",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S3",
					RequestId: "d3-3",
				},
			},
			parentage: [][]string{
				nil,
				{"d1-1"},
				{"d1-2"},
				{"d1-3"},
				nil,
				{"d2-1"},
				{"d2-2"},
				nil,
				{"d3-1"},
				{"d3-2"},
			},
			replies: []model.Reply{
				{Site: "S1", RequestId: "d1-1"},
				{Site: "S1", RequestId: "d3-1"},
				{Site: "S1", RequestId: "d3-2"},

				{Site: "S2", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d3-1"},
				{Site: "S2", RequestId: "d3-2"},

				{Site: "S3", RequestId: "d2-1"},
				{Site: "S3", RequestId: "d2-2"},
				{Site: "S3", RequestId: "d3-1"},
				{Site: "S3", RequestId: "d3-2"},
			},
			torun: map[string][]string{
				"S1": []string{"d2-2", "d2-3"},
				"S2": []string{"d2-2", "d2-3"},
				"S3": []string{"d2-3"},
			},
		}, {
			name:  "withTypeC4",
			sites: []string{"S1", "S2", "S3"},
			operations: []model.Operation{
				{
					Type:      model.OperationTypeIdempotent,
					Site:      "S1",
					RequestId: "d1-1",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S1",
					RequestId: "d1-2",
				},
				{
					Type:      model.OperationTypeIdempotent,
					Site:      "S1",
					RequestId: "d1-3",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S1",
					RequestId: "d1-4",
				},
				{
					Type:      model.OperationTypeCommutative,
					Site:      "S3",
					RequestId: "d3-1",
				}, {
					// dead operation
					Type:      model.OperationTypeCommutativeIdempotent,
					Site:      "S3",
					RequestId: "d3-2",
				}, {
					Type:      model.OperationTypeCommutative,
					Site:      "S3",
					RequestId: "d3-3",
				},
			},
			parentage: [][]string{
				nil,
				{"d1-1", "d3-1"},
				{"d1-2"},
				{"d1-3", "d3-2"},
				nil,
				{"d1-3", "d3-1"},
				{"d3-2"},
			},
			replies: []model.Reply{
				{Site: "S1", RequestId: "d1-3"},
			},
			torun: map[string][]string{
				"S1": []string{"d1-4", "d3-2", "d3-3"},
				"S2": []string{"d1-3", "d1-4", "d3-2", "d3-3"},
				"S3": []string{"d1-3", "d1-4", "d3-2", "d3-3"},
			},
		},
	}

	for _, tv := range tvs {
		for _, site := range tv.sites {
			docs := make([]model.PayloadDocument, 0)
			for idx := range tv.operations {
				raw, _ := json.Marshal(tv.operations[idx])
				doc := model.PayloadDocument{
					Parents: tv.parentage[idx],
					Type:    "OPERATION",
					Payload: json.RawMessage(raw),
				}
				docs = append(docs, doc)
			}
			for _, reply := range tv.replies {
				raw, _ := json.Marshal(reply)
				doc := model.PayloadDocument{
					Type:    "REPLY",
					Payload: json.RawMessage(raw),
				}
				docs = append(docs, doc)
			}

			tree, existingReplies := makeTreeWithReplies(site, docs)

			expectedOperations := tv.torun[site]

			actualOps := findOperationsToRun(tree, existingReplies)
			if len(actualOps) != len(expectedOperations) {
				t.Fatalf("Wrong operations at %s site %s: got %s want %s\n", tv.name, site, mapOpsToRequestId(actualOps), expectedOperations)
			}
			for i, op := range actualOps {
				if op.RequestId != expectedOperations[i] {
					t.Fatalf("Wrong operations at %s site %s: got %s want %s\n", tv.name, site, mapOpsToRequestId(actualOps), expectedOperations)

				}
			}
		}
	}
}

func mapOpsToRequestId(ops []model.Operation) []string {
	v := make([]string, 0)
	for _, op := range ops {
		v = append(v, op.RequestId)
	}
	return v
}
