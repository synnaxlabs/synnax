// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stage_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/stage"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var ctx = context.Background()

var _ = Describe("Stage", func() {
	Describe("NewFactory", func() {
		It("Should create factory", func() {
			factory := stage.NewFactory()
			Expect(factory).ToNot(BeNil())
		})
	})

	Describe("Factory.Create", func() {
		var factory *stage.Factory
		var s *state.State

		BeforeEach(func() {
			factory = stage.NewFactory()
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{Key: "stage_entry_1", Type: "stage_entry"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "stage_entry_1", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "source",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
					{
						Key:    "stage_entry",
						Inputs: types.Params{{Name: ir.DefaultInputParam, Type: types.U8()}},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stage.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})

		It("Should create node for stage_entry type", func() {
			cfg := node.Config{
				Node:  ir.Node{Key: "stage_entry_1", Type: "stage_entry"},
				State: s.Node("stage_entry_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})

		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Key: "unknown", Type: "unknown"},
				State: s.Node("stage_entry_1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
		})
	})

	Describe("StageEntry.Next", func() {
		var factory *stage.Factory
		var s *state.State
		var activatedNodes []string

		BeforeEach(func() {
			factory = stage.NewFactory()
			activatedNodes = []string{}
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{Key: "test_seq_test_stage_entry", Type: "stage_entry"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "test_seq_test_stage_entry", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "source",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
					{
						Key:    "stage_entry",
						Inputs: types.Params{{Name: ir.DefaultInputParam, Type: types.U8()}},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stage.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})

		It("Should call ActivateStage when receiving activation signal (1)", func() {
			cfg := node.Config{
				Node:  ir.Node{Key: "test_seq_test_stage_entry", Type: "stage_entry"},
				State: s.Node("test_seq_test_stage_entry"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Set source output to activation signal (1)
			sourceNode := s.Node("source")
			*sourceNode.Output(0) = telem.NewSeriesV[uint8](1)
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())

			// Execute stage entry node with context that tracks activations
			nodeCtx := node.Context{
				Context:     ctx,
				MarkChanged: func(string) {},
				ActivateStage: func(nodeKey string) {
					activatedNodes = append(activatedNodes, nodeKey)
				},
			}
			n.Next(nodeCtx)

			// Should have called ActivateStage with the node key
			Expect(activatedNodes).To(HaveLen(1))
			Expect(activatedNodes[0]).To(Equal("test_seq_test_stage_entry"))
		})

		It("Should not call ActivateStage when receiving non-activation signal (0)", func() {
			cfg := node.Config{
				Node:  ir.Node{Key: "test_seq_test_stage_entry", Type: "stage_entry"},
				State: s.Node("test_seq_test_stage_entry"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Set source output to non-activation signal (0)
			sourceNode := s.Node("source")
			*sourceNode.Output(0) = telem.NewSeriesV[uint8](0)
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())

			// Execute stage entry node
			nodeCtx := node.Context{
				Context:     ctx,
				MarkChanged: func(string) {},
				ActivateStage: func(nodeKey string) {
					activatedNodes = append(activatedNodes, nodeKey)
				},
			}
			n.Next(nodeCtx)

			// Should not have called ActivateStage
			Expect(activatedNodes).To(BeEmpty())
		})

		It("Should not call ActivateStage when input is empty", func() {
			cfg := node.Config{
				Node:  ir.Node{Key: "test_seq_test_stage_entry", Type: "stage_entry"},
				State: s.Node("test_seq_test_stage_entry"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Execute without setting any input
			nodeCtx := node.Context{
				Context:     ctx,
				MarkChanged: func(string) {},
				ActivateStage: func(nodeKey string) {
					activatedNodes = append(activatedNodes, nodeKey)
				},
			}
			n.Next(nodeCtx)

			// Should not have called ActivateStage
			Expect(activatedNodes).To(BeEmpty())
		})
	})

	Describe("SymbolResolver", func() {
		It("Should resolve stage_entry symbol", func() {
			sym, ok := stage.SymbolResolver["stage_entry"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("stage_entry"))
		})

		It("Should have correct input type", func() {
			sym := stage.SymbolResolver["stage_entry"]
			fnType := sym.Type
			Expect(fnType.Kind).To(Equal(types.KindFunction))
			Expect(fnType.Inputs).To(HaveLen(1))
			Expect(fnType.Inputs[0].Type).To(Equal(types.U8()))
		})
	})
})
