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
	ResolutionMatrix: []model.Resolution{
		{
			Before: model.OperationType("set"),
			After:  model.OperationType("inc"),
			Result: model.TakeBothKeepOrder,
		}, {
			Before: model.OperationType("inc"),
			After:  model.OperationType("set"),
			Result: model.TakeBothReverseOrder,
		},
		{
			Before: model.OperationType("set"),
			After:  model.OperationType("dec"),
			Result: model.TakeBothKeepOrder,
		}, {
			Before: model.OperationType("dec"),
			After:  model.OperationType("set"),
			Result: model.TakeBothReverseOrder,
		}, {
			Before: model.OperationType("set"),
			After:  model.OperationType("set"),
			Result: model.TakeOne,
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
						Type:      model.OperationType("inc"),
						RequestId: "inc",
					}, {
						Type:      model.OperationType("dec"),
						RequestId: "dec",
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
					{
						Type:      model.OperationType("dec"),
						RequestId: "dec",
					},
				},
			},
		}, {
			main: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("dec"),
						RequestId: "dec",
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
					{
						Type:      model.OperationType("dec"),
						RequestId: "dec",
					},
				},
			},
		},
		{
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
						RequestId: "set-0",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc1",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc2",
					},
				},
			},
		}, {
			main: model.ResourceDocument{
				Operations: []model.Operation{
					{
						Type:      model.OperationType("inc"),
						RequestId: "inc-0",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc-1",
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
							RequestId: "inc-2",
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
						RequestId: "inc-0",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc-1",
					}, {
						Type:      model.OperationType("inc"),
						RequestId: "inc-2",
					},
				},
			},
		},
	}

	for _, v := range vectors {
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
	config   model.ResourceConfig
}

func TestFindOperations(t *testing.T) {
	vectors := []findTestVector{
		{
			ops: []model.Operation{
				{RequestId: "set", Type: "set"},
			},
			replies: []model.ReplyDocument{},
			expected: []model.Operation{
				{RequestId: "set"},
			},
			config: model.ResourceConfig{
				ResolutionMatrix: []model.Resolution{{
					Before: "inc", After: "inc", Result: model.TakeBothAnyOrder,
				}, {
					Before: "set", After: "inc", Result: model.TakeBothKeepOrder,
				}},
			},
		},
		{
			ops: []model.Operation{
				{RequestId: "set", Type: "set"},
				{RequestId: "inc1", Type: "inc"},
			},
			replies: []model.ReplyDocument{
				{RequestId: "set"},
				{RequestId: "inc1"},
			},
			expected: []model.Operation{},
			config: model.ResourceConfig{
				ResolutionMatrix: []model.Resolution{{
					Before: "inc", After: "inc", Result: model.TakeBothAnyOrder,
				}, {
					Before: "set", After: "inc", Result: model.TakeBothKeepOrder,
				}},
			},
		}, {
			ops: []model.Operation{
				{RequestId: "set", Type: "set"},
				{RequestId: "inc1", Type: "inc"},
				{RequestId: "inc2", Type: "inc"},
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
			config: model.ResourceConfig{
				ResolutionMatrix: []model.Resolution{{
					Before: "inc", After: "inc", Result: model.TakeBothAnyOrder,
				}, {
					Before: "set", After: "inc", Result: model.TakeBothKeepOrder,
				}},
			},
		}, {
			ops: []model.Operation{
				{RequestId: "set", Type: "set"},
				{RequestId: "inc1", Type: "inc"},
				{RequestId: "inc2", Type: "inc"},
			},
			replies: []model.ReplyDocument{},
			expected: []model.Operation{
				{RequestId: "set"},
				{RequestId: "inc1"},
				{RequestId: "inc2"},
			},
			config: model.ResourceConfig{
				ResolutionMatrix: []model.Resolution{{
					Before: "inc", After: "inc", Result: model.TakeBothAnyOrder,
				}, {
					Before: "set", After: "inc", Result: model.TakeBothKeepOrder,
				}},
			},
		}, {
			ops: []model.Operation{
				{RequestId: "set", Type: "set"},
				{RequestId: "inc1", Type: "inc"},
				{RequestId: "inc2", Type: "inc"},
			},
			replies: []model.ReplyDocument{
				{RequestId: "set"},
				{RequestId: "inc1"}},
			expected: []model.Operation{
				{RequestId: "inc2"},
			},
			config: model.ResourceConfig{
				ResolutionMatrix: []model.Resolution{{
					Before: "inc", After: "inc", Result: model.TakeBothAnyOrder,
				}, {
					Before: "set", After: "inc", Result: model.TakeBothKeepOrder,
				}},
			},
		}, {
			ops: []model.Operation{
				{RequestId: "inc", Type: "inc"},
				{RequestId: "inc1", Type: "inc"},
				{RequestId: "inc2", Type: "inc"},
			},
			replies: []model.ReplyDocument{
				{RequestId: "inc1"},
				{RequestId: "inc2"}},
			expected: []model.Operation{
				{RequestId: "inc"},
			},
			config: model.ResourceConfig{
				ResolutionMatrix: []model.Resolution{{
					Before: "inc", After: "inc", Result: model.TakeBothAnyOrder,
				}},
			},
		},
	}

	for vi, vector := range vectors {
		torun := findOperationsToRun(vector.ops, vector.replies, vector.config)

		if len(torun) != len(vector.expected) {
			t.Fatalf("vector %d: got %v want %v", vi, logops(torun), logops(vector.expected))
		}

		for i := range torun {
			if torun[i].RequestId != vector.expected[i].RequestId {
				t.Fatalf("vector %d: got %#v at %d, want %#v", vi, torun[i].RequestId, i, vector.expected[i].RequestId)
			}
		}
	}
}

type decideTestVector struct {
	existing []model.Operation
	new      model.Operation
	expected []model.Operation
}

func TestDecideOperations(t *testing.T) {
	vectors := []decideTestVector{
		{
			existing: []model.Operation{
				{
					Type:      "set",
					RequestId: "set-init",
				},
			},
			new: model.Operation{
				Type:      "set",
				RequestId: "set-new",
			},
			expected: []model.Operation{
				{
					Type:      "set",
					RequestId: "set-new",
				},
			},
		},
		{
			existing: []model.Operation{
				{
					Type:      "set",
					RequestId: "set",
				},
			},
			new: model.Operation{
				Type:      "inc",
				RequestId: "inc",
			},
			expected: []model.Operation{
				{
					Type:      "set",
					RequestId: "set",
				}, {
					Type:      "inc",
					RequestId: "inc",
				},
			},
		}, {
			existing: []model.Operation{
				{
					Type:      "dec",
					RequestId: "dec",
				},
			},
			new: model.Operation{
				Type:      "set",
				RequestId: "set",
			},
			expected: []model.Operation{
				{
					Type:      "set",
					RequestId: "set",
				},
			},
		}, {
			existing: []model.Operation{
				{
					Type:      "dec",
					RequestId: "dec",
				},
			},
			new: model.Operation{
				Type:      "inc",
				RequestId: "inc",
			},
			expected: []model.Operation{
				{
					Type:      "dec",
					RequestId: "dec",
				}, {
					Type:      "inc",
					RequestId: "inc",
				},
			},
		},
	}

	for _, vector := range vectors {
		actual := decideOperationsToKeep(counterConfig, vector.existing, vector.new)
		if len(actual) != len(vector.expected) {
			t.Fatalf("Invalid operations to keep: got %s want %s\n", logops(actual), logops(vector.expected))
		}
		for i := range actual {
			if actual[i].RequestId != vector.expected[i].RequestId {
				t.Fatalf("Invalid operations to keep: got %s want %s\n", logops(actual), logops(vector.expected))
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
