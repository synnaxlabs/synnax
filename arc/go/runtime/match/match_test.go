// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package match_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/match"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var ctx = context.Background()

var _ = Describe("Match", func() {
	Describe("NewFactory", func() {
		It("Should create factory", func() {
			factory := match.NewFactory()
			Expect(factory).ToNot(BeNil())
		})
	})
	Describe("Factory.Create", func() {
		var factory *match.Factory
		var s *state.State
		BeforeEach(func() {
			factory = match.NewFactory()
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{
						Key:  "match_1",
						Type: "match",
						Config: map[string]any{
							"cases": []any{},
						},
					},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "match_1", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "source",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.String()},
						},
					},
					{
						Key: "match",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.String()},
						},
						// Outputs are dynamic and defined per-node, not in the function
						Config: types.Params{
							{Name: "cases", Type: types.String()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, match.SymbolResolver)
			if !diagnostics.Ok() {
				GinkgoWriter.Printf("Diagnostics: %v\n", diagnostics)
			}
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should create node for match type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "match",
					Outputs: types.Params{
						{Name: "case_a", Type: types.U8()},
						{Name: "case_b", Type: types.U8()},
					},
					Config: types.Params{
						{Name: "cases", Type: types.String(), Value: []any{
							map[string]any{"value": "a", "output": "case_a"},
							map[string]any{"value": "b", "output": "case_b"},
						}},
					},
				},
				State: s.Node("match_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("match_1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
		})
	})
	Describe("Match.Next", func() {
		var factory *match.Factory
		var s *state.State
		var changedOutputs []string
		BeforeEach(func() {
			factory = match.NewFactory()
			changedOutputs = []string{}
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{
						Key:  "match_1",
						Type: "match",
						Config: map[string]any{
							"cases": []any{},
						},
					},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "match_1", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "source",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.String()},
						},
					},
					{
						Key: "match",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.String()},
						},
						// Define outputs here for the state to have them
						Outputs: types.Params{
							{Name: "case_a", Type: types.U8()},
							{Name: "case_b", Type: types.U8()},
							{Name: "case_c", Type: types.U8()},
						},
						Config: types.Params{
							{Name: "cases", Type: types.String()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, match.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should route input to matching output", func() {
			cfg := node.Config{
				Node: ir.Node{
					Key:  "match_1",
					Type: "match",
					Outputs: types.Params{
						{Name: "case_a", Type: types.U8()},
						{Name: "case_b", Type: types.U8()},
						{Name: "case_c", Type: types.U8()},
					},
					Config: types.Params{
						{Name: "cases", Type: types.String(), Value: []any{
							map[string]any{"value": "alpha", "output": "case_a"},
							map[string]any{"value": "beta", "output": "case_b"},
							map[string]any{"value": "gamma", "output": "case_c"},
						}},
					},
				},
				State: s.Node("match_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Set source output to "beta"
			sourceNode := s.Node("source")
			*sourceNode.Output(0) = telem.NewSeriesStringsV("beta")
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())

			// Initialize match outputs
			matchNode := s.Node("match_1")
			*matchNode.Output(0) = telem.NewSeriesV[uint8]()
			*matchNode.Output(1) = telem.NewSeriesV[uint8]()
			*matchNode.Output(2) = telem.NewSeriesV[uint8]()

			// Execute match node
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				changedOutputs = append(changedOutputs, output)
			}})

			// Should have marked case_b as changed
			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal("case_b"))

			// case_b output should be 1
			caseBOutput := matchNode.Output(1)
			Expect(caseBOutput.Len()).To(Equal(int64(1)))
			Expect(telem.ValueAt[uint8](*caseBOutput, 0)).To(Equal(uint8(1)))
		})
		It("Should not route when input doesn't match any case", func() {
			cfg := node.Config{
				Node: ir.Node{
					Key:  "match_1",
					Type: "match",
					Outputs: types.Params{
						{Name: "case_a", Type: types.U8()},
						{Name: "case_b", Type: types.U8()},
					},
					Config: types.Params{
						{Name: "cases", Type: types.String(), Value: []any{
							map[string]any{"value": "alpha", "output": "case_a"},
							map[string]any{"value": "beta", "output": "case_b"},
						}},
					},
				},
				State: s.Node("match_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Set source output to "unknown"
			sourceNode := s.Node("source")
			*sourceNode.Output(0) = telem.NewSeriesStringsV("unknown")
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())

			// Execute match node
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				changedOutputs = append(changedOutputs, output)
			}})

			// Should not have marked any output as changed
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should not route when input is empty", func() {
			cfg := node.Config{
				Node: ir.Node{
					Key:  "match_1",
					Type: "match",
					Outputs: types.Params{
						{Name: "case_a", Type: types.U8()},
					},
					Config: types.Params{
						{Name: "cases", Type: types.String(), Value: []any{
							map[string]any{"value": "alpha", "output": "case_a"},
						}},
					},
				},
				State: s.Node("match_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Execute without setting any input
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				changedOutputs = append(changedOutputs, output)
			}})

			// Should not have marked any output as changed
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should handle multiple matches in sequence", func() {
			cfg := node.Config{
				Node: ir.Node{
					Key:  "match_1",
					Type: "match",
					Outputs: types.Params{
						{Name: "case_a", Type: types.U8()},
						{Name: "case_b", Type: types.U8()},
					},
					Config: types.Params{
						{Name: "cases", Type: types.String(), Value: []any{
							map[string]any{"value": "alpha", "output": "case_a"},
							map[string]any{"value": "beta", "output": "case_b"},
						}},
					},
				},
				State: s.Node("match_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			sourceNode := s.Node("source")
			matchNode := s.Node("match_1")
			*matchNode.Output(0) = telem.NewSeriesV[uint8]()
			*matchNode.Output(1) = telem.NewSeriesV[uint8]()

			// First: match "alpha"
			*sourceNode.Output(0) = telem.NewSeriesStringsV("alpha")
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				changedOutputs = append(changedOutputs, output)
			}})
			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal("case_a"))

			// Second: match "beta" (new timestamp to trigger refresh)
			changedOutputs = []string{}
			*sourceNode.Output(0) = telem.NewSeriesStringsV("beta")
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				changedOutputs = append(changedOutputs, output)
			}})
			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal("case_b"))
		})
	})
	Describe("SymbolResolver", func() {
		It("Should resolve match symbol", func() {
			sym, ok := match.SymbolResolver["match"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("match"))
		})
	})
})
