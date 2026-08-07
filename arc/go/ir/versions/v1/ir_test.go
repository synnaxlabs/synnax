// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"encoding/json"

	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	v1 "github.com/synnaxlabs/arc/ir/versions/v1"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("IR", func() {
	Describe("IsZero", func() {
		DescribeTable(
			"Classification",
			func(program v1.IR, expected bool) {
				Expect(program.IsZero()).To(Equal(expected))
			},
			Entry("an empty IR", v1.IR{}, true),
			Entry("a function",
				v1.IR{Functions: v1.Functions{{Key: "add"}}}, false),
			Entry("a node",
				v1.IR{Nodes: v1.Nodes{{Key: "n1"}}}, false),
			Entry("an edge",
				v1.IR{Edges: v1.Edges{{Kind: v1.EdgeKindContinuous}}}, false),
			Entry("a non-zero root",
				v1.IR{Root: v1.Scope{Key: "root"}}, false),
			Entry("a root with members",
				v1.IR{Root: v1.Scope{
					Mode:     v1.ScopeModeParallel,
					Liveness: v1.LivenessAlways,
					Strata:   []v1.Members{{v1.NodeMember("n1")}},
				}}, false),
			Entry("symbols",
				v1.IR{Symbols: &symbol.Symbol{}}, false),
			Entry("a type map",
				v1.IR{TypeMap: map[antlr.ParserRuleContext]types.Type{}}, false),
		)
	})

	Describe("JSON Marshaling", func() {
		It("Should marshal and unmarshal a complete IR structure", func() {
			inputs := types.Params{
				{Name: "a", Type: types.I64()},
				{Name: "b", Type: types.I64()},
			}
			outputs := types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}

			original := &v1.IR{
				Functions: v1.Functions{{Key: "add", Inputs: inputs, Outputs: outputs}},
				Nodes: v1.Nodes{
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
				Edges: v1.Edges{
					{
						Source: v1.Handle{
							Node:  "input_a",
							Param: ir.DefaultOutputParam,
						},
						Target: v1.Handle{Node: "node1", Param: "a"},
					},
					{
						Source: v1.Handle{Node: "input_b", Param: "value"},
						Target: v1.Handle{Node: "node1", Param: "b"},
					},
				},
				Root: v1.Scope{
					Mode:     v1.ScopeModeParallel,
					Liveness: v1.LivenessAlways,
					Strata: []v1.Members{
						{v1.NodeMember("input_a"), v1.NodeMember("input_b")},
						{v1.NodeMember("node1")},
					},
				},
			}

			data := MustSucceed(json.Marshal(original))
			Expect(data).ToNot(BeEmpty())

			var restored v1.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Functions).To(HaveLen(1))
			Expect(restored.Functions[0].Key).To(Equal("add"))
			Expect(restored.Nodes).To(HaveLen(1))
			Expect(restored.Edges).To(HaveLen(2))
			Expect(restored.Root.Mode).To(Equal(v1.ScopeModeParallel))
			Expect(restored.Root.Strata).To(HaveLen(2))
			Expect(restored.Root.Strata[0]).To(HaveLen(2))
		})

		It("Should handle empty IR", func() {
			original := &v1.IR{
				Functions: v1.Functions{},
				Nodes:     v1.Nodes{},
				Edges:     v1.Edges{},
				Root: v1.Scope{
					Mode:     v1.ScopeModeParallel,
					Liveness: v1.LivenessAlways,
				},
			}
			data := MustSucceed(json.Marshal(original))
			var restored v1.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Functions).To(BeEmpty())
			Expect(restored.Root.Strata).To(BeEmpty())
		})

		It("Should exclude Symbols and TypeMap from JSON (json:\"-\" tag)", func() {
			original := &v1.IR{Symbols: symbol.NewRoot(nil, nil)}
			data := MustSucceed(json.Marshal(original))
			jsonStr := string(data)
			Expect(jsonStr).ToNot(ContainSubstring("\"symbols\""))
			Expect(jsonStr).ToNot(ContainSubstring("\"TypeMap\""))
		})

		It("Should round-trip a sequential scope with transitions", func() {
			stepKey := "run"
			original := &v1.IR{
				Root: v1.Scope{
					Mode:     v1.ScopeModeParallel,
					Liveness: v1.LivenessAlways,
					Strata: []v1.Members{{
						{Scope: &v1.Scope{
							Key:      "main",
							Mode:     v1.ScopeModeSequential,
							Liveness: v1.LivenessGated,
							Steps: v1.Members{
								v1.NodeMember("init"),
								v1.NodeMember("run"),
							},
							Transitions: []v1.Transition{
								{
									On:        v1.Handle{Node: "init", Param: "done"},
									TargetKey: &stepKey,
								},
								{
									On:        v1.Handle{Node: "run", Param: "done"},
									TargetKey: nil,
								},
							},
						}},
					}},
				},
			}

			data := MustSucceed(json.Marshal(original))
			var restored v1.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			main := restored.Root.Strata[0][0].Scope
			Expect(main).ToNot(BeNil())
			Expect(main.Mode).To(Equal(v1.ScopeModeSequential))
			Expect(main.Steps).To(HaveLen(2))
			Expect(main.Transitions).To(HaveLen(2))
			Expect(main.Transitions[0].TargetKey).ToNot(BeNil())
			Expect(*main.Transitions[0].TargetKey).To(Equal("run"))
			Expect(main.Transitions[1].TargetKey).To(BeNil())
		})
	})

	Describe("String", func() {
		It("Should render an empty IR as an empty string", func() {
			program := &v1.IR{}
			Expect(program.String()).To(BeEmpty())
		})

		It("Should render Functions, Nodes, Edges, and Root sections", func() {
			program := &v1.IR{
				Functions: v1.Functions{
					{
						Key:    "add",
						Inputs: types.Params{{Name: "a", Type: types.I64()}},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
					},
				},
				Nodes: v1.Nodes{
					{
						Key:  "node1",
						Type: "add",
						Inputs: types.Params{
							{Name: "k", Type: types.I64(), Value: int64(1)},
							{Name: "a", Type: types.I64()},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
					},
				},
				Edges: v1.Edges{{
					Source: v1.Handle{Node: "src", Param: ir.DefaultOutputParam},
					Target: v1.Handle{Node: "node1", Param: "a"},
					Kind:   v1.EdgeKindContinuous,
				}},
				Root: v1.Scope{
					Mode:     v1.ScopeModeParallel,
					Liveness: v1.LivenessAlways,
					Strata:   []v1.Members{{v1.NodeMember("node1")}},
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
			program := &v1.IR{
				Nodes: v1.Nodes{{Key: "only", Type: "input"}},
			}
			out := program.String()
			Expect(out).To(ContainSubstring("Nodes (1)"))
			Expect(out).ToNot(ContainSubstring("Functions"))
			Expect(out).ToNot(ContainSubstring("Edges"))
			Expect(out).ToNot(ContainSubstring("Root"))
		})

		It(
			"Should render multiple functions, nodes, and edges with tree indentation",
			func() {
				program := &v1.IR{
					Functions: v1.Functions{
						{Key: "f1"},
						{Key: "f2"},
					},
					Nodes: v1.Nodes{
						{Key: "n1", Type: "f1"},
						{Key: "n2", Type: "f2"},
					},
					Edges: v1.Edges{
						{
							Source: v1.Handle{Node: "n1", Param: ir.DefaultOutputParam},
							Target: v1.Handle{Node: "n2", Param: ir.DefaultInputParam},
							Kind:   v1.EdgeKindContinuous,
						},
						{
							Source: v1.Handle{Node: "n2", Param: ir.DefaultOutputParam},
							Target: v1.Handle{Node: "n1", Param: ir.DefaultInputParam},
							Kind:   v1.EdgeKindConditional,
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
			},
		)

		It("Should render exact tree glyphs for every section", func() {
			program := v1.IR{
				Functions: v1.Functions{{Key: "add"}},
				Nodes:     v1.Nodes{{Key: "n1", Type: "add"}},
				Edges: v1.Edges{{
					Source: v1.Handle{Node: "n1", Param: "out"},
					Target: v1.Handle{Node: "n2", Param: "in"},
					Kind:   v1.EdgeKindContinuous,
				}},
				Root: v1.Scope{
					Key:      "root",
					Mode:     v1.ScopeModeParallel,
					Liveness: v1.LivenessAlways,
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
			program := v1.IR{Nodes: v1.Nodes{{Key: "n1", Type: "add"}}}
			Expect(program.String()).To(HavePrefix("└── Nodes (1)\n"))
		})
	})
})
