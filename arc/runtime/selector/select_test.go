// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package selector_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/selector"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Select", func() {
	Describe("NewFactory", func() {
		It("Should create factory", func() {
			factory := selector.NewFactory()
			Expect(factory).ToNot(BeNil())
		})
	})
	Describe("Factory.Create", func() {
		var factory node.Factory
		var s *state.State
		BeforeEach(func() {
			factory = selector.NewFactory()
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{Key: "select", Type: "select"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "select", Param: ir.DefaultInputParam},
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
						Key: "select",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.U8()},
						},
						Outputs: types.Params{
							{Name: "true", Type: types.U8()},
							{Name: "false", Type: types.U8()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, selector.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should create node for select type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node(ctx, "select"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
		})
	})
	Describe("select.Init", func() {
		It("Should not emit output on Init", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{Key: "select", Type: "select"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "select", Param: ir.DefaultInputParam},
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
						Key: "select",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.U8()},
						},
						Outputs: types.Params{
							{Name: "true", Type: types.U8()},
							{Name: "false", Type: types.U8()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, selector.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			factory := selector.NewFactory()
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			n, _ := factory.Create(ctx, cfg)
			outputs := []string{}
			n.Init(node.Context{Context: ctx, MarkChanged: func(output string) {
				outputs = append(outputs, output)
			}})
			Expect(outputs).To(BeEmpty())
		})
	})
	Describe("select.Next", func() {
		var s *state.State
		var factory node.Factory
		BeforeEach(func() {
			factory = selector.NewFactory()
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{Key: "select", Type: "select"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "select", Param: ir.DefaultInputParam},
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
						Key: "select",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.U8()},
						},
						Outputs: types.Params{
							{Name: "true", Type: types.U8()},
							{Name: "false", Type: types.U8()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, selector.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should handle empty input", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8]()
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV()
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { outputs.Add(output) }})
			Expect(outputs.Contains("true")).To(BeFalse())
			Expect(outputs.Contains("false")).To(BeFalse())
		})
		It("Should split all true values", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](1, 1, 1)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { outputs.Add(output) }})
			Expect(outputs.Contains("true")).To(BeTrue())
			Expect(outputs.Contains("false")).To(BeFalse())
			selectNode := s.Node(ctx, "select")
			trueOut := selectNode.Output(0)
			Expect(trueOut.Len()).To(Equal(int64(3)))
			trueVals := telem.UnmarshalSeries[uint8](*trueOut)
			Expect(trueVals).To(Equal([]uint8{1, 1, 1}))
		})
		It("Should split all false values", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](0, 0, 0, 0)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(10, 20, 30, 40)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { outputs.Add(output) }})
			Expect(outputs.Contains("true")).To(BeFalse())
			Expect(outputs.Contains("false")).To(BeTrue())
			selectNode := s.Node(ctx, "select")
			falseOut := selectNode.Output(1)
			Expect(falseOut.Len()).To(Equal(int64(4)))
			falseVals := telem.UnmarshalSeries[uint8](*falseOut)
			Expect(falseVals).To(Equal([]uint8{0, 0, 0, 0}))
		})
		It("Should split mixed true and false values", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](1, 0, 1, 0, 1)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { outputs.Add(output) }})
			Expect(outputs.Contains("true")).To(BeTrue())
			Expect(outputs.Contains("false")).To(BeTrue())
			selectNode := s.Node(ctx, "select")
			trueOut := selectNode.Output(0)
			falseOut := selectNode.Output(1)
			Expect(trueOut.Len()).To(Equal(int64(3)))
			Expect(falseOut.Len()).To(Equal(int64(2)))
			trueVals := telem.UnmarshalSeries[uint8](*trueOut)
			falseVals := telem.UnmarshalSeries[uint8](*falseOut)
			Expect(trueVals).To(Equal([]uint8{1, 1, 1}))
			Expect(falseVals).To(Equal([]uint8{0, 0}))
		})
		It("Should correctly copy timestamps for true values", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](1, 0, 1, 0, 1)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(10, 20, 30, 40, 50)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			selectNode := s.Node(ctx, "select")
			trueTime := selectNode.OutputTime(0)
			trueTimes := telem.UnmarshalSeries[telem.TimeStamp](*trueTime)
			Expect(trueTimes).To(Equal([]telem.TimeStamp{
				telem.SecondTS * 10,
				telem.SecondTS * 30,
				telem.SecondTS * 50,
			}))
		})
		It("Should correctly copy timestamps for false values", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](1, 0, 1, 0, 1)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(10, 20, 30, 40, 50)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			selectNode := s.Node(ctx, "select")
			falseTime := selectNode.OutputTime(1)
			falseTimes := telem.UnmarshalSeries[telem.TimeStamp](*falseTime)
			Expect(falseTimes).To(Equal([]telem.TimeStamp{
				telem.SecondTS * 20,
				telem.SecondTS * 40,
			}))
		})
		It("Should handle single true value", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](1)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(100)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { outputs.Add(output) }})
			Expect(outputs.Contains("true")).To(BeTrue())
			Expect(outputs.Contains("false")).To(BeFalse())
			selectNode := s.Node(ctx, "select")
			trueOut := selectNode.Output(0)
			Expect(trueOut.Len()).To(Equal(int64(1)))
		})
		It("Should handle single false value", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](0)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(100)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { outputs.Add(output) }})
			Expect(outputs.Contains("true")).To(BeFalse())
			Expect(outputs.Contains("false")).To(BeTrue())
			selectNode := s.Node(ctx, "select")
			falseOut := selectNode.Output(1)
			Expect(falseOut.Len()).To(Equal(int64(1)))
		})
		It("Should handle values other than 0 and 1 as false", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](1, 2, 3, 1, 0)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { outputs.Add(output) }})
			Expect(outputs.Contains("true")).To(BeTrue())
			Expect(outputs.Contains("false")).To(BeTrue())
			selectNode := s.Node(ctx, "select")
			trueOut := selectNode.Output(0)
			falseOut := selectNode.Output(1)
			Expect(trueOut.Len()).To(Equal(int64(2)))
			Expect(falseOut.Len()).To(Equal(int64(3)))
		})
		It("Should handle long series", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			data := make([]uint8, 1000)
			times := make([]telem.TimeStamp, 1000)
			for i := range data {
				data[i] = uint8(i % 2)
				times[i] = telem.TimeStamp(i) * telem.SecondTS
			}
			*source.Output(0) = telem.NewSeriesV(data...)
			*source.OutputTime(0) = telem.NewSeriesV(times...)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			selectNode := s.Node(ctx, "select")
			trueOut := selectNode.Output(0)
			falseOut := selectNode.Output(1)
			Expect(trueOut.Len()).To(Equal(int64(500)))
			Expect(falseOut.Len()).To(Equal(int64(500)))
		})
		It("Should handle consecutive true values", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](0, 0, 1, 1, 1, 0)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			selectNode := s.Node(ctx, "select")
			trueOut := selectNode.Output(0)
			trueTime := selectNode.OutputTime(0)
			Expect(trueOut.Len()).To(Equal(int64(3)))
			trueTimes := telem.UnmarshalSeries[telem.TimeStamp](*trueTime)
			Expect(trueTimes).To(Equal([]telem.TimeStamp{
				telem.SecondTS * 3,
				telem.SecondTS * 4,
				telem.SecondTS * 5,
			}))
		})
		It("Should handle consecutive false values", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "select"},
				State: s.Node(ctx, "select"),
			}
			source := s.Node(ctx, "source")
			*source.Output(0) = telem.NewSeriesV[uint8](1, 1, 0, 0, 0, 1)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			selectNode := s.Node(ctx, "select")
			falseOut := selectNode.Output(1)
			falseTime := selectNode.OutputTime(1)
			Expect(falseOut.Len()).To(Equal(int64(3)))
			falseTimes := telem.UnmarshalSeries[telem.TimeStamp](*falseTime)
			Expect(falseTimes).To(Equal([]telem.TimeStamp{
				telem.SecondTS * 3,
				telem.SecondTS * 4,
				telem.SecondTS * 5,
			}))
		})
	})
	Describe("SymbolResolver", func() {
		It("Should resolve select symbol", func() {
			sym, ok := selector.SymbolResolver["select"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("select"))
		})
	})
})
