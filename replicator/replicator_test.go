package replicator

import (
	"strings"
	"testing"

	"cheops.com/model"
)

type mergeTestVector struct {
	main      model.ResourceDocument
	conflicts []model.ResourceDocument
	expected  model.ResourceDocument
}

var counterConfig model.ResourceConfig = model.ResourceConfig{
	RelationshipMatrix: []model.Relationship{
		{
			Before: model.OperationType("set"),
			After:  model.OperationType("inc"),
			Result: []int{1},
		}, {
			Before: model.OperationType("inc"),
			After:  model.OperationType("set"),
			Result: []int{2},
		},
		{
			Before: model.OperationType("set"),
			After:  model.OperationType("dec"),
			Result: []int{1},
		}, {
			Before: model.OperationType("dec"),
			After:  model.OperationType("set"),
			Result: []int{2},
		}, {
			Before: model.OperationType("set"),
			After:  model.OperationType("set"),
			Result: []int{2},
		},
	},
}

func TestMerge(t *testing.T) {
	vectors := []mergeTestVector{
		{
			main: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("inc"),
						RequestId: "inc",
					},
				},
			},
			conflicts: []model.ResourceDocument{
				{
					Operations: []model.Operation{
						{
							Type:      model.OperationType("dec"),
							RequestId: "dec",
						},
					},
				},
			},
			expected: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("dec"),
						RequestId: "dec",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc",
					},
				},
			},
		},
		{
			main: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("set"),
						RequestId: "set",
					},
				},
				Config: counterConfig,
			},
			conflicts: []model.ResourceDocument{
				{
					Operations: []model.Operation{
						{
							Type:      model.OperationType("dec"),
							RequestId: "dec",
						},
					},
				},
			},
			expected: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("set"),
						RequestId: "set",
					},
				},
			},
		}, {
			main: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("set"),
						RequestId: "set-main",
					},
				},
				Config: counterConfig,
			},
			conflicts: []model.ResourceDocument{
				{
					Operations: []model.Operation{
						{
							Type:      model.OperationType("set"),
							RequestId: "set-conflict",
						},
					},
				},
			},
			expected: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("set"),
						RequestId: "set-main",
					},
				},
			},
		}, {
			main: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("set"),
						RequestId: "set",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc1",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc-main",
					},
				},
				Config: counterConfig,
			},
			conflicts: []model.ResourceDocument{
				{
					Operations: []model.Operation{
						{
							Type:      model.OperationType("set"),
							RequestId: "set",
						}, {
							Type:      model.OperationType("inc"),
							RequestId: "inc1",
						}, {
							Type:      model.OperationType("inc"),
							RequestId: "inc-conflict",
						},
					},
				},
			},
			expected: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("set"),
						RequestId: "set",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc1",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc-main",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc-conflict",
					},
				},
			},
		}, {
			main: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("set"),
						RequestId: "set-0",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc1",
					},
				},
			},
			conflicts: []model.ResourceDocument{
				{
					Operations: []model.Operation{
						{
							Type:      model.OperationType("set"),
							RequestId: "set-1",
						}, {
							Type:      model.OperationType("inc"),
							RequestId: "inc2",
						},
					},
					Config: counterConfig,
				},
			},
			expected: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("set"),
						RequestId: "set-1",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc1",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc2",
					},
				},
			},
		},
	}

	for _, v := range vectors[4:] {
		resolved, err := resolveMerge(v.main, v.conflicts)
		if err != nil {
			t.Fatalf("got err: %v", err)
		}

		if len(resolved.Operations) != len(v.expected.Operations) {
			t.Fatalf("error in ops: got %s want %s", logops(resolved.Operations), logops(v.expected.Operations))
		}
		for i := range resolved.Operations {
			if resolved.Operations[i].RequestId != v.expected.Operations[i].RequestId {
				t.Fatalf("error in ops: got %s want %s", logops(resolved.Operations), logops(v.expected.Operations))
			}
		}

	}
}

type findTestVector struct {
	ops      []model.Operation
	replies  []model.ReplyDocument
	expected []model.Operation
}

func TestFindOperations(t *testing.T) {
	vectors := []findTestVector{
		{
			ops: []model.Operation{
				{RequestId: "set"},
			},
			replies: []model.ReplyDocument{},
			expected: []model.Operation{
				{RequestId: "set"},
			},
		},
		{
			ops: []model.Operation{
				{RequestId: "set"},
				{RequestId: "inc1"},
			},
			replies: []model.ReplyDocument{
				{RequestId: "set"},
				{RequestId: "inc1"},
			},
			expected: []model.Operation{},
		}, {
			ops: []model.Operation{
				{RequestId: "set"},
				{RequestId: "inc1"},
				{RequestId: "inc2"},
			},
			replies: []model.ReplyDocument{
				{RequestId: "set"},
				{RequestId: "inc2"},
			},
			expected: []model.Operation{
				{RequestId: "set"},
				{RequestId: "inc1"},
				{RequestId: "inc2"},
			},
		},
	}

	for vi, vector := range vectors {
		torun := findOperationsToRun(vector.ops, vector.replies)

		if len(torun) != len(vector.expected) {
			t.Fatalf("vector %d: got %d elem want %d", vi, len(torun), len(vector.expected))
		}

		for i := range torun {
			if torun[i].RequestId != vector.expected[i].RequestId {
				t.Fatalf("vector %d: got %#v at %d, want %#v", vi, torun[i].RequestId, i, vector.expected[i].RequestId)
			}
		}
	}
}

func logops(ops []model.Operation) string {
	str := make([]string, len(ops))
	for i := range ops {
		str[i] = ops[i].RequestId
	}

	return "[" + strings.Join(str, ",") + "]"
}
