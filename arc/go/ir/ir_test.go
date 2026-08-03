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

	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("IR", func() {
	Describe("IsZero", func() {
		DescribeTable(
			"Classification",
			func(program ir.IR, expected bool) {
				Expect(program.IsZero()).To(Equal(expected))
			},
			Entry("an empty IR", ir.IR{}, true),
			Entry("a function",
				ir.IR{Functions: ir.Functions{{Key: "add"}}}, false),
			Entry("a node",
				ir.IR{Nodes: ir.Nodes{{Key: "n1"}}}, false),
			Entry("an edge",
				ir.IR{Edges: ir.Edges{{Kind: ir.EdgeKindContinuous}}}, false),
			Entry("a non-zero root",
				ir.IR{Root: ir.Scope{Key: "root"}}, false),
			Entry("a root with members",
				ir.IR{Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata:   []ir.Members{{ir.NodeMember("n1")}},
				}}, false),
			Entry("symbols",
				ir.IR{Symbols: &symbol.Symbol{}}, false),
			Entry("a type map",
				ir.IR{TypeMap: map[antlr.ParserRuleContext]types.Type{}}, false),
		)
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
						Key:  "node1",
						Type: "add",
						Inputs: types.Params{
							{Name: "multiplier", Type: types.F64(), Value: 2.0},
							{Name: "a", Type: types.I64()},
							{Name: "b", Type: types.I64()},
						},
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
			original := &ir.IR{Symbols: symbol.NewRoot(nil, nil)}
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

	Describe("String", func() {
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
					Key:  "node1",
					Type: "add",
					Inputs: types.Params{
						{Name: "k", Type: types.I64(), Value: int64(1)},
						{Name: "a", Type: types.I64()},
					},
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

		It("Should render exact tree glyphs for every section", func() {
			program := ir.IR{
				Functions: ir.Functions{{Key: "add"}},
				Nodes:     ir.Nodes{{Key: "n1", Type: "add"}},
				Edges: ir.Edges{{
					Source: ir.Handle{Node: "n1", Param: "out"},
					Target: ir.Handle{Node: "n2", Param: "in"},
					Kind:   ir.EdgeKindContinuous,
				}},
				Root: ir.Scope{
					Key:      "root",
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
				},
			}
			out := program.String()
			Expect(out).To(HavePrefix("├── Functions (1)\n"))
			Expect(out).To(ContainSubstring("├── Nodes (1)\n"))
			Expect(out).To(ContainSubstring("n1 (type: add)"))
			Expect(out).To(ContainSubstring("├── Edges (1)\n"))
			Expect(out).To(ContainSubstring("n1.out -> n2.in (EdgeKindContinuous)"))
			Expect(out).To(ContainSubstring("└── Root\n"))
			Expect(out).To(ContainSubstring("root [ScopeModeParallel, LivenessAlways]"))
		})

		It("Should mark the only section as the last tree item", func() {
			program := ir.IR{Nodes: ir.Nodes{{Key: "n1", Type: "add"}}}
			Expect(program.String()).To(HavePrefix("└── Nodes (1)\n"))
		})
	})
})
