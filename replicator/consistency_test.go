package replicator

import (
	"testing"

	"cheops.com/model"
)

type testVector struct {
	name    string
	docs    []model.ResourceDocument
	replies []model.ReplyDocument

	// For each site, the list of operations to run
	operations [][]string
}

func TestFindUnitsToRun(t *testing.T) {
	tvs := []testVector{
		{
			name: "simple",
			docs: []model.ResourceDocument{
				{
					Locations: []string{"S1", "S2"},
					Site:      "S1",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d1-1",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 0,
							},
						},
					},
				},
			},
			replies: []model.ReplyDocument{
				{Site: "S1", RequestId: "d1-1"},
			},
			operations: [][]string{
				{},
				{"d1-1"},
			},
		},
		{
			name: "noTypeC",
			docs: []model.ResourceDocument{
				{
					Locations: []string{"S1", "S2"},
					Site:      "S1",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d1-1",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 0,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d1-2",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 1,
							},
						},
					},
				},
				{
					Locations: []string{"S1", "S2"},
					Site:      "S2",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeCommutative,
							RequestId: "d2-1",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 1,
							},
						}, {
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d2-2",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 2,
							},
						}, {
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d2-3",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 3,
							},
						},
					},
				},
			},
			replies: []model.ReplyDocument{
				{Site: "S1", RequestId: "d1-1"},
				{Site: "S2", RequestId: "d1-1"},
				{Site: "S2", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d2-2"},
			},
			operations: [][]string{
				{"d1-2", "d2-3", "d2-2"},
				{"d2-3", "d1-2"},
			},
		}, {
			name: "withTypeC",
			docs: []model.ResourceDocument{
				{
					Locations: []string{"S1", "S2"},
					Site:      "S1",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d1-1",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 0,
							},
						}, {
							Type:      model.OperationTypeIdempotent,
							RequestId: "d1-2",
							KnownState: map[string]int{
								"S1": 2,
								"S2": 1,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d1-3",
							KnownState: map[string]int{
								"S1": 3,
								"S2": 1,
							},
						},
					},
				},
				{
					Locations: []string{"S1", "S2"},
					Site:      "S2",
					Operations: []model.Operation{
						{
							// Type C but dead, should be skipped
							Type:      model.OperationTypeIdempotent,
							RequestId: "d2-1",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 1,
							},
						}, {
							Type:      model.OperationTypeIdempotent,
							RequestId: "d2-2",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 2,
							},
						}, {
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d2-3",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 3,
							},
						},
					},
				},
			},
			replies: []model.ReplyDocument{
				{Site: "S1", RequestId: "d1-1"},
				{Site: "S1", RequestId: "d1-2"},
				{Site: "S1", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d2-2"},
			},
			operations: [][]string{
				{"d2-2", "d2-3"},
				{"d2-3"},
			},
		}, {
			name: "withTypeC2",
			docs: []model.ResourceDocument{
				{
					Locations: []string{"S1", "S2"},
					Site:      "S1",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeIdempotent,
							RequestId: "d1-1",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 0,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d1-2",
							KnownState: map[string]int{
								"S1": 2,
								"S2": 1,
							},
						},
					},
				},
				{
					Locations: []string{"S1", "S2"},
					Site:      "S2",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeCommutative,
							RequestId: "d2-1",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 1,
							},
						}, {
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d2-2",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 2,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d2-3",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 3,
							},
						},
					},
				},
			},
			replies: []model.ReplyDocument{
				{Site: "S1", RequestId: "d1-1"},
				{Site: "S1", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d1-1"},
				{Site: "S2", RequestId: "d2-1"},
				{Site: "S2", RequestId: "d2-2"},
			},
			operations: [][]string{
				{"d1-2", "d2-3", "d2-2"},
				{"d1-2", "d2-3"},
			},
		}, {
			name: "withTypeC3",
			docs: []model.ResourceDocument{
				{
					Locations: []string{"S1", "S2", "S3"},
					Site:      "S1",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeIdempotent,
							RequestId: "d1-1",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 0,
								"S3": 0,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d1-2",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 0,
								"S3": 2,
							},
						},

						{
							Type:      model.OperationTypeIdempotent,
							RequestId: "d1-3",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 0,
								"S3": 2,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d1-4",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 0,
								"S3": 2,
							},
						},
					},
				},
				{
					Locations: []string{"S1", "S2", "S3"},
					Site:      "S2",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeCommutative,
							RequestId: "d2-1",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 1,
								"S3": 0,
							},
						}, {
							Type:      model.OperationTypeIdempotent,
							RequestId: "d2-2",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 2,
								"S3": 0,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d2-3",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 3,
								"S3": 0,
							},
						},
					},
				},
				{
					Locations: []string{"S1", "S2", "S3"},
					Site:      "S3",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeCommutative,
							RequestId: "d3-1",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 0,
								"S3": 1,
							},
						}, {
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d3-2",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 0,
								"S3": 2,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d3-3",
							KnownState: map[string]int{
								"S1": 0,
								"S2": 0,
								"S3": 3,
							},
						},
					},
				},
			},
			replies: []model.ReplyDocument{
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
			operations: [][]string{
				{"d2-2", "d2-3"},
				{"d2-2", "d2-3"},
				{"d2-3"},
			},
		}, {
			name: "withTypeC4",
			docs: []model.ResourceDocument{
				{
					Locations: []string{"S1", "S2", "S3"},
					Site:      "S1",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeIdempotent,
							RequestId: "d1-1",
							KnownState: map[string]int{
								"S1": 1,
								"S2": 0,
								"S3": 2,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d1-2",
							KnownState: map[string]int{
								"S1": 2,
								"S2": 0,
								"S3": 2,
							},
						},

						{
							Type:      model.OperationTypeIdempotent,
							RequestId: "d1-3",
							KnownState: map[string]int{
								"S1": 3,
								"S2": 0,
								"S3": 2,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d1-4",
							KnownState: map[string]int{
								"S1": 4,
								"S2": 0,
								"S3": 2,
							},
						},
					},
				},
				{
					Locations: []string{"S1", "S2", "S3"},
					Site:      "S3",
					Operations: []model.Operation{
						{
							Type:      model.OperationTypeCommutative,
							RequestId: "d3-1",
							KnownState: map[string]int{
								"S1": 3,
								"S2": 0,
								"S3": 1,
							},
						}, {
							// dead operation
							Type:      model.OperationTypeCommutativeIdempotent,
							RequestId: "d3-2",
							KnownState: map[string]int{
								"S1": 4,
								"S2": 0,
								"S3": 2,
							},
						}, {
							Type:      model.OperationTypeCommutative,
							RequestId: "d3-3",
							KnownState: map[string]int{
								"S1": 4,
								"S2": 0,
								"S3": 3,
							},
						},
					},
				},
			},
			replies: []model.ReplyDocument{},
			operations: [][]string{
				{"d1-3", "d1-4", "d3-3", "d3-2"},
				{"d1-3", "d1-4", "d3-3", "d3-2"},
				{"d1-3", "d1-4", "d3-3", "d3-2"},
			},
		},
	}

	for _, tv := range tvs {
		allSites := tv.docs[0].Locations
		for idx, site := range allSites {
			expectedOperations := tv.operations[idx]

			actualOps := findOperationsToRun(site, tv.docs, tv.replies)
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
