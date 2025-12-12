// Copyright 2025 Synnax Labs, Inc.
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
					{
						Key:  "stage_entry_1",
						Type: "stage_entry",
						Config: map[string]any{
							"sequence": "test_seq",
							"stage":    "test_stage",
						},
					},
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
						Key: "stage_entry",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.U8()},
						},
						Config: types.Params{
							{Name: "sequence", Type: types.String()},
							{Name: "stage", Type: types.String()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stage.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should create node for stage_entry type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stage_entry",
					Config: types.Params{
						{Name: "sequence", Type: types.String(), Value: "my_sequence"},
						{Name: "stage", Type: types.String(), Value: "my_stage"},
					},
				},
				State: s.Node("stage_entry_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("stage_entry_1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
		})
		It("Should error when sequence config is missing", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stage_entry",
					Config: types.Params{
						{Name: "stage", Type: types.String(), Value: "my_stage"},
					},
				},
				State: s.Node("stage_entry_1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing sequence or stage config"))
		})
		It("Should error when stage config is missing", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stage_entry",
					Config: types.Params{
						{Name: "sequence", Type: types.String(), Value: "my_sequence"},
					},
				},
				State: s.Node("stage_entry_1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing sequence or stage config"))
		})
	})
	Describe("StageEntry.Next", func() {
		var factory *stage.Factory
		var s *state.State
		var callbackInvocations []struct{ seq, stage string }
		BeforeEach(func() {
			factory = stage.NewFactory()
			callbackInvocations = []struct{ seq, stage string }{}
			factory.SetActivateCallback(func(seq, stageName string) {
				callbackInvocations = append(callbackInvocations, struct{ seq, stage string }{seq, stageName})
			})
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{
						Key:  "stage_entry_1",
						Type: "stage_entry",
						Config: map[string]any{
							"sequence": "test_seq",
							"stage":    "test_stage",
						},
					},
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
						Key: "stage_entry",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.U8()},
						},
						Config: types.Params{
							{Name: "sequence", Type: types.String()},
							{Name: "stage", Type: types.String()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stage.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should invoke callback when receiving activation signal (1)", func() {
			cfg := node.Config{
				Node: ir.Node{
					Key:  "stage_entry_1",
					Type: "stage_entry",
					Config: types.Params{
						{Name: "sequence", Type: types.String(), Value: "seq1"},
						{Name: "stage", Type: types.String(), Value: "stage_a"},
					},
				},
				State: s.Node("stage_entry_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Set source output to activation signal (1)
			sourceNode := s.Node("source")
			*sourceNode.Output(0) = telem.NewSeriesV[uint8](1)
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())

			// Execute stage entry node
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})

			// Should have invoked callback
			Expect(callbackInvocations).To(HaveLen(1))
			Expect(callbackInvocations[0].seq).To(Equal("seq1"))
			Expect(callbackInvocations[0].stage).To(Equal("stage_a"))
		})
		It("Should not invoke callback when receiving non-activation signal (0)", func() {
			cfg := node.Config{
				Node: ir.Node{
					Key:  "stage_entry_1",
					Type: "stage_entry",
					Config: types.Params{
						{Name: "sequence", Type: types.String(), Value: "seq1"},
						{Name: "stage", Type: types.String(), Value: "stage_a"},
					},
				},
				State: s.Node("stage_entry_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Set source output to non-activation signal (0)
			sourceNode := s.Node("source")
			*sourceNode.Output(0) = telem.NewSeriesV[uint8](0)
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())

			// Execute stage entry node
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})

			// Should not have invoked callback
			Expect(callbackInvocations).To(BeEmpty())
		})
		It("Should not invoke callback when input is empty", func() {
			cfg := node.Config{
				Node: ir.Node{
					Key:  "stage_entry_1",
					Type: "stage_entry",
					Config: types.Params{
						{Name: "sequence", Type: types.String(), Value: "seq1"},
						{Name: "stage", Type: types.String(), Value: "stage_a"},
					},
				},
				State: s.Node("stage_entry_1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			// Execute without setting any input
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})

			// Should not have invoked callback
			Expect(callbackInvocations).To(BeEmpty())
		})
	})
	Describe("SetActivateCallback", func() {
		It("Should allow setting callback after factory creation", func() {
			factory := stage.NewFactory()
			callCount := 0
			factory.SetActivateCallback(func(seq, stageName string) {
				callCount++
			})

			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{
						Key:  "stage_entry_1",
						Type: "stage_entry",
						Config: map[string]any{
							"sequence": "seq",
							"stage":    "stg",
						},
					},
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
						Key: "stage_entry",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.U8()},
						},
						Config: types.Params{
							{Name: "sequence", Type: types.String()},
							{Name: "stage", Type: types.String()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stage.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})

			cfg := node.Config{
				Node: ir.Node{
					Key:  "stage_entry_1",
					Type: "stage_entry",
					Config: types.Params{
						{Name: "sequence", Type: types.String(), Value: "seq"},
						{Name: "stage", Type: types.String(), Value: "stg"},
					},
				},
				State: s.Node("stage_entry_1"),
			}
			n, _ := factory.Create(ctx, cfg)

			sourceNode := s.Node("source")
			*sourceNode.Output(0) = telem.NewSeriesV[uint8](1)
			*sourceNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())

			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			Expect(callCount).To(Equal(1))
		})
		It("Should share callback across multiple StageEntry nodes", func() {
			factory := stage.NewFactory()
			callCount := 0
			factory.SetActivateCallback(func(seq, stageName string) {
				callCount++
			})

			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source1", Type: "source"},
					{Key: "source2", Type: "source"},
					{
						Key:  "stage_entry_1",
						Type: "stage_entry",
						Config: map[string]any{
							"sequence": "seq1",
							"stage":    "stg1",
						},
					},
					{
						Key:  "stage_entry_2",
						Type: "stage_entry",
						Config: map[string]any{
							"sequence": "seq2",
							"stage":    "stg2",
						},
					},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "stage_entry_1", Param: ir.DefaultInputParam},
					},
					{
						Source: ir.Handle{Node: "source2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "stage_entry_2", Param: ir.DefaultInputParam},
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
						Key: "stage_entry",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.U8()},
						},
						Config: types.Params{
							{Name: "sequence", Type: types.String()},
							{Name: "stage", Type: types.String()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stage.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})

			cfg1 := node.Config{
				Node: ir.Node{
					Key:  "stage_entry_1",
					Type: "stage_entry",
					Config: types.Params{
						{Name: "sequence", Type: types.String(), Value: "seq1"},
						{Name: "stage", Type: types.String(), Value: "stg1"},
					},
				},
				State: s.Node("stage_entry_1"),
			}
			cfg2 := node.Config{
				Node: ir.Node{
					Key:  "stage_entry_2",
					Type: "stage_entry",
					Config: types.Params{
						{Name: "sequence", Type: types.String(), Value: "seq2"},
						{Name: "stage", Type: types.String(), Value: "stg2"},
					},
				},
				State: s.Node("stage_entry_2"),
			}
			n1, _ := factory.Create(ctx, cfg1)
			n2, _ := factory.Create(ctx, cfg2)

			// Activate both stage entries
			*s.Node("source1").Output(0) = telem.NewSeriesV[uint8](1)
			*s.Node("source1").OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())
			*s.Node("source2").Output(0) = telem.NewSeriesV[uint8](1)
			*s.Node("source2").OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())

			n1.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			n2.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})

			Expect(callCount).To(Equal(2))
		})
	})
	Describe("SymbolResolver", func() {
		It("Should resolve stage_entry symbol", func() {
			sym, ok := stage.SymbolResolver["stage_entry"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("stage_entry"))
		})
	})
})
