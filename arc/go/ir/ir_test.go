// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("IR", func() {
	Describe("IsZero", func() {
		It("Should return true for zero-value IR", func() {
			program := &ir.IR{}
			Expect(program.IsZero()).To(BeTrue())
		})

		It("Should return false when Functions is non-empty", func() {
			program := &ir.IR{Functions: ir.Functions{{Key: "test"}}}
			Expect(program.IsZero()).To(BeFalse())
		})

		It("Should return false when Root has members", func() {
			program := &ir.IR{
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{
						{ir.NodeMember("n1")},
					},
				},
			}
			Expect(program.IsZero()).To(BeFalse())
		})

		It("Should return false when Nodes is non-empty", func() {
			program := &ir.IR{Nodes: ir.Nodes{{Key: "node1"}}}
			Expect(program.IsZero()).To(BeFalse())
		})

		It("Should return false when Edges is non-empty", func() {
			program := &ir.IR{Edges: ir.Edges{{Kind: ir.EdgeKindContinuous}}}
			Expect(program.IsZero()).To(BeFalse())
		})

		It("Should return false when Symbols is set", func() {
			program := &ir.IR{Symbols: symbol.CreateRootScope(nil)}
			Expect(program.IsZero()).To(BeFalse())
		})
	})

	Describe("Scope IsZero", func() {
		It("Should return true for an uninitialized scope", func() {
			Expect(ir.Scope{}.IsZero()).To(BeTrue())
		})

		It("Should return false when a stratum carries members", func() {
			s := ir.Scope{
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessAlways,
				Strata:   []ir.Members{{ir.NodeMember("n1")}},
			}
			Expect(s.IsZero()).To(BeFalse())
		})

		It("Should return false when a sequential scope carries steps", func() {
			s := ir.Scope{
				Key:      "main",
				Mode:     ir.ScopeModeSequential,
				Liveness: ir.LivenessGated,
				Steps:    ir.Members{ir.NodeMember("n1")},
			}
			Expect(s.IsZero()).To(BeFalse())
		})
	})

	Describe("JSON Marshaling", func() {
		It("Should marshal and unmarshal a complete IR structure", func() {
			inputs := types.Params{
				{Name: "a", Type: types.I64()},
				{Name: "b", Type: types.I64()},
			}
			outputs := types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}

			original := &ir.IR{
				Functions: ir.Functions{{Key: "add", Inputs: inputs, Outputs: outputs}},
				Nodes: ir.Nodes{
					{
						Key:     "node1",
						Type:    "add",
						Config:  types.Params{{Name: "multiplier", Type: types.F64(), Value: 2.0}},
						Inputs:  inputs,
						Outputs: outputs,
					},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "input_a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "node1", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "input_b", Param: "value"},
						Target: ir.Handle{Node: "node1", Param: "b"},
					},
				},
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{
						{ir.NodeMember("input_a"), ir.NodeMember("input_b")},
						{ir.NodeMember("node1")},
					},
				},
			}

			data := MustSucceed(json.Marshal(original))
			Expect(data).ToNot(BeEmpty())

			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Functions).To(HaveLen(1))
			Expect(restored.Functions[0].Key).To(Equal("add"))
			Expect(restored.Nodes).To(HaveLen(1))
			Expect(restored.Edges).To(HaveLen(2))
			Expect(restored.Root.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(restored.Root.Strata).To(HaveLen(2))
			Expect(restored.Root.Strata[0]).To(HaveLen(2))
		})

		It("Should handle empty IR", func() {
			original := &ir.IR{
				Functions: ir.Functions{},
				Nodes:     ir.Nodes{},
				Edges:     ir.Edges{},
				Root:      ir.Scope{Mode: ir.ScopeModeParallel, Liveness: ir.LivenessAlways},
			}
			data := MustSucceed(json.Marshal(original))
			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Functions).To(BeEmpty())
			Expect(restored.Root.Strata).To(BeEmpty())
		})

		It("Should exclude Symbols and TypeMap from JSON (json:\"-\" tag)", func() {
			original := &ir.IR{Symbols: symbol.CreateRootScope(nil)}
			data := MustSucceed(json.Marshal(original))
			jsonStr := string(data)
			Expect(jsonStr).ToNot(ContainSubstring("\"symbols\""))
			Expect(jsonStr).ToNot(ContainSubstring("\"TypeMap\""))
		})

		It("Should round-trip a sequential scope with transitions", func() {
			stepKey := "run"
			original := &ir.IR{
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{{
						{Scope: &ir.Scope{
							Key:      "main",
							Mode:     ir.ScopeModeSequential,
							Liveness: ir.LivenessGated,
							Steps:    ir.Members{ir.NodeMember("init"), ir.NodeMember("run")},
							Transitions: []ir.Transition{
								{
									On:        ir.Handle{Node: "init", Param: "done"},
									TargetKey: &stepKey,
								},
								{
									On:        ir.Handle{Node: "run", Param: "done"},
									TargetKey: nil,
								},
							},
						}},
					}},
				},
			}

			data := MustSucceed(json.Marshal(original))
			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			main := restored.Root.Strata[0][0].Scope
			Expect(main).ToNot(BeNil())
			Expect(main.Mode).To(Equal(ir.ScopeModeSequential))
			Expect(main.Steps).To(HaveLen(2))
			Expect(main.Transitions).To(HaveLen(2))
			Expect(main.Transitions[0].TargetKey).ToNot(BeNil())
			Expect(*main.Transitions[0].TargetKey).To(Equal("run"))
			Expect(main.Transitions[1].TargetKey).To(BeNil())
		})
	})

	Describe("String Formatting", func() {
		Describe("IR String", func() {
			It("Should render an empty IR as an empty string", func() {
				program := &ir.IR{}
				Expect(program.String()).To(BeEmpty())
			})

			It("Should render Functions, Nodes, Edges, and Root sections", func() {
				program := &ir.IR{
					Functions: ir.Functions{{
						Key:     "add",
						Inputs:  types.Params{{Name: "a", Type: types.I64()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
					}},
					Nodes: ir.Nodes{{
						Key:     "node1",
						Type:    "add",
						Config:  types.Params{{Name: "k", Type: types.I64(), Value: int64(1)}},
						Inputs:  types.Params{{Name: "a", Type: types.I64()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
					}},
					Edges: ir.Edges{{
						Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "node1", Param: "a"},
						Kind:   ir.EdgeKindContinuous,
					}},
					Root: ir.Scope{
						Mode:     ir.ScopeModeParallel,
						Liveness: ir.LivenessAlways,
						Strata:   []ir.Members{{ir.NodeMember("node1")}},
					},
				}
				out := program.String()
				Expect(out).To(ContainSubstring("Functions (1)"))
				Expect(out).To(ContainSubstring("Nodes (1)"))
				Expect(out).To(ContainSubstring("Edges (1)"))
				Expect(out).To(ContainSubstring("Root"))
				Expect(out).To(ContainSubstring("add"))
				Expect(out).To(ContainSubstring("node1"))
				Expect(out).To(ContainSubstring("src.output -> node1.a"))
			})

			It("Should render only populated sections", func() {
				program := &ir.IR{
					Nodes: ir.Nodes{{Key: "only", Type: "input"}},
				}
				out := program.String()
				Expect(out).To(ContainSubstring("Nodes (1)"))
				Expect(out).ToNot(ContainSubstring("Functions"))
				Expect(out).ToNot(ContainSubstring("Edges"))
				Expect(out).ToNot(ContainSubstring("Root"))
			})

			It("Should render multiple functions, nodes, and edges with tree indentation", func() {
				program := &ir.IR{
					Functions: ir.Functions{
						{Key: "f1"},
						{Key: "f2"},
					},
					Nodes: ir.Nodes{
						{Key: "n1", Type: "f1"},
						{Key: "n2", Type: "f2"},
					},
					Edges: ir.Edges{
						{
							Source: ir.Handle{Node: "n1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "n2", Param: ir.DefaultInputParam},
							Kind:   ir.EdgeKindContinuous,
						},
						{
							Source: ir.Handle{Node: "n2", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "n1", Param: ir.DefaultInputParam},
							Kind:   ir.EdgeKindConditional,
						},
					},
				}
				out := program.String()
				Expect(out).To(ContainSubstring("Functions (2)"))
				Expect(out).To(ContainSubstring("Nodes (2)"))
				Expect(out).To(ContainSubstring("Edges (2)"))
				Expect(out).To(ContainSubstring("├── "))
				Expect(out).To(ContainSubstring("└── "))
				Expect(out).To(ContainSubstring("n1.output -> n2.input"))
				Expect(out).To(ContainSubstring("n2.output => n1.input"))
			})
		})

		Describe("Scope String", func() {
			It("Should render parallel scope with named stratum entries", func() {
				s := ir.Scope{
					Key:      "root",
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{
						{ir.NodeMember("a"), ir.NodeMember("b")},
						{ir.NodeMember("c")},
					},
				}
				out := s.String()
				Expect(out).To(ContainSubstring("root"))
				Expect(out).To(ContainSubstring("stratum 0"))
				Expect(out).To(ContainSubstring("stratum 1"))
				Expect(out).To(ContainSubstring("a"))
				Expect(out).To(ContainSubstring("b"))
				Expect(out).To(ContainSubstring("c"))
			})

			It("Should render an unnamed scope with the (scope) placeholder", func() {
				s := ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata:   []ir.Members{{ir.NodeMember("x")}},
				}
				Expect(s.String()).To(ContainSubstring("(scope)"))
			})

			It("Should render sequential scope steps and transitions", func() {
				run := "run"
				s := ir.Scope{
					Key:      "main",
					Mode:     ir.ScopeModeSequential,
					Liveness: ir.LivenessGated,
					Steps:    ir.Members{ir.NodeMember("init"), ir.NodeMember("run")},
					Transitions: []ir.Transition{
						{On: ir.Handle{Node: "init", Param: "done"}, TargetKey: &run},
						{On: ir.Handle{Node: "run", Param: "done"}},
					},
				}
				out := s.String()
				Expect(out).To(ContainSubstring("main"))
				Expect(out).To(ContainSubstring("init"))
				Expect(out).To(ContainSubstring("run"))
				Expect(out).To(ContainSubstring("on init/done => run"))
				Expect(out).To(ContainSubstring("on run/done => exit"))
			})

			It("Should render nested scope members", func() {
				inner := ir.Scope{
					Key:      "inner",
					Mode:     ir.ScopeModeSequential,
					Liveness: ir.LivenessGated,
					Steps:    ir.Members{ir.NodeMember("step1")},
				}
				outer := ir.Scope{
					Key:      "outer",
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata:   []ir.Members{{ir.ScopeMember(inner)}},
				}
				out := outer.String()
				Expect(out).To(ContainSubstring("outer"))
				Expect(out).To(ContainSubstring("inner"))
				Expect(out).To(ContainSubstring("step1"))
			})
		})

		Describe("Member", func() {
			It("Should build a node-backed Member via NodeMember", func() {
				m := ir.NodeMember("n1")
				Expect(m.Key()).To(Equal("n1"))
				Expect(m.String()).To(ContainSubstring("n1"))
			})

			It("Should build a scope-backed Member via ScopeMember", func() {
				s := ir.Scope{
					Key:      "sub",
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
				}
				m := ir.ScopeMember(s)
				Expect(m.Key()).To(Equal("sub"))
				Expect(m.String()).To(ContainSubstring("sub"))
			})

			It("Should return an empty key and placeholder string for a zero Member", func() {
				var m ir.Member
				Expect(m.Key()).To(BeEmpty())
				Expect(m.String()).To(Equal("(empty member)\n"))
			})
		})

		Describe("Transition String", func() {
			It("Should render a transition targeting a sibling step", func() {
				target := "next"
				t := ir.Transition{
					On:        ir.Handle{Node: "n", Param: "done"},
					TargetKey: &target,
				}
				Expect(t.String()).To(Equal("on n/done => next"))
			})

			It("Should render an exiting transition when TargetKey is nil", func() {
				t := ir.Transition{On: ir.Handle{Node: "n", Param: "done"}}
				Expect(t.String()).To(Equal("on n/done => exit"))
			})
		})
	})

	Describe("Edge Helpers Against Root Scope", func() {
		It("Should expose dataflow edges independent of the Scope tree", func() {
			program := &ir.IR{
				Functions: ir.Functions{
					{
						Key: "add",
						Inputs: types.Params{
							{Name: ir.LHSInputParam, Type: types.I64()},
							{Name: ir.RHSInputParam, Type: types.I64()},
						},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
					},
				},
				Nodes: ir.Nodes{
					{Key: "input_a", Type: "input"},
					{Key: "input_b", Type: "input"},
					{Key: "add_node", Type: "add"},
					{Key: "output_c", Type: "output"},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "input_a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "add_node", Param: ir.LHSInputParam},
						Kind:   ir.EdgeKindContinuous,
					},
					{
						Source: ir.Handle{Node: "input_b", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "add_node", Param: ir.RHSInputParam},
						Kind:   ir.EdgeKindContinuous,
					},
					{
						Source: ir.Handle{Node: "add_node", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "output_c", Param: ir.DefaultInputParam},
						Kind:   ir.EdgeKindContinuous,
					},
				},
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{
						{ir.NodeMember("input_a"), ir.NodeMember("input_b")},
						{ir.NodeMember("add_node")},
						{ir.NodeMember("output_c")},
					},
				},
			}

			Expect(program.Edges.GetInputs("add_node")).To(HaveLen(2))
			Expect(program.Edges.GetOutputs("add_node")).To(HaveLen(1))
			Expect(program.Edges.GetByKind(ir.EdgeKindContinuous)).To(HaveLen(3))
		})
	})
})
