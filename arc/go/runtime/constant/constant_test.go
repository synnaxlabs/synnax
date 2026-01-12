// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constant_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Constant", func() {
	Describe("NewFactory", func() {
		It("Should create factory", func() {
			factory := constant.NewFactory()
			Expect(factory).ToNot(BeNil())
		})
	})

	Describe("Factory.Create", func() {
		var (
			factory node.Factory
			s       *state.State
		)
		BeforeEach(func() {
			factory = constant.NewFactory()
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "const", Type: "constant"}},
				Functions: []graph.Function{{
					Key: "constant",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, constant.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should create node for constant type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I64(), Value: 42}}},
				State: s.Node("const"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("const"),
			}
			Expect(factory.Create(ctx, cfg)).Error().To(MatchError(query.NotFound))
		})
		It("Should handle float64 value", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Config: types.Params{{
						Name:  "value",
						Type:  types.F64(),
						Value: 3.14,
					}},
				},
				State: s.Node("const"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
		It("Should handle int value", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Config: types.Params{{
						Name:  "value",
						Type:  types.I64(),
						Value: 100,
					}}},
				State: s.Node("const"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
		It("Should handle uint8 value", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Config: types.Params{{
						Name:  "value",
						Type:  types.U8(),
						Value: uint8(255),
					}}},
				State: s.Node("const"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
	})

	Describe("Next", func() {
		var (
			s       *state.State
			factory node.Factory
			outputs []string
		)
		BeforeEach(func() {
			factory = constant.NewFactory()
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "const", Type: "constant"}},
				Functions: []graph.Function{{
					Key: "constant",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, constant.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
			outputs = []string{}
		})

		It("Should emit output on Next with int value", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I64(), Value: 42}}},
				State: s.Node("const"),
			}
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				outputs = append(outputs, output)
			}})
			Expect(outputs).To(HaveLen(1))
			Expect(outputs[0]).To(Equal(ir.DefaultOutputParam))
		})

		It("Should set output data on Next", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I64(), Value: int64(100)}}},
				State: s.Node("const"),
			}
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			out := s.Node("const").Output(0)
			Expect(out.Len()).To(Equal(int64(1)))
		})

		It("Should set output time on Next", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "constant",
					Config: types.Params{{Name: "value", Type: types.F64(), Value: 3.14}},
				},
				State: s.Node("const"),
			}
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			outTime := s.Node("const").OutputTime(0)
			Expect(outTime.Len()).To(Equal(int64(1)))
			times := telem.UnmarshalSeries[telem.TimeStamp](*outTime)
			Expect(times[0]).To(BeNumerically(">", int64(0)))
		})

		It("Should handle float64 constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.F64(), Value: 2.718}}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[float64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[float64](*out)
			Expect(vals[0]).To(Equal(2.718))
		})

		It("Should handle int32 constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I32(), Value: int32(42)}}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int32](0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int32](*out)
			Expect(vals[0]).To(Equal(int32(42)))
		})

		It("Should handle uint8 constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.U8(), Value: uint8(255)}}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[uint8](0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[uint8](*out)
			Expect(vals[0]).To(Equal(uint8(255)))
		})

		It("Should allow downstream nodes to read constant", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "const", Type: "constant"},
					{Key: "sink", Type: "sink"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "const", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "constant",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
					},
					{
						Key: "sink",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.I64()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, constant.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I64(), Value: int64(999)}}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			sink := s.Node("sink")
			recalc := sink.RefreshInputs()
			Expect(recalc).To(BeTrue())
			input := sink.Input(0)
			vals := telem.UnmarshalSeries[int64](input)
			Expect(vals[0]).To(Equal(int64(999)))
		})

		It("Should handle zero value constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I64(), Value: 0}}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int64](*out)
			Expect(vals[0]).To(Equal(int64(0)))
		})

		It("Should handle negative value constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I64(), Value: -42}}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int64](*out)
			Expect(vals[0]).To(Equal(int64(-42)))
		})

		It("Should only emit once across multiple Next calls", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I64(), Value: int64(42)}}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				outputs = append(outputs, output)
			}})
			Expect(outputs).To(HaveLen(1))
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				outputs = append(outputs, output)
			}})
			Expect(outputs).To(HaveLen(1))
		})

		It("Should emit again after Reset is called", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", Config: types.Params{{Name: "value", Type: types.I64(), Value: int64(42)}}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				outputs = append(outputs, output)
			}})
			Expect(outputs).To(HaveLen(1))
			n.Reset()
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				outputs = append(outputs, output)
			}})
			Expect(outputs).To(HaveLen(2))
		})
	})

	Describe("SymbolResolver", func() {
		It("Should resolve constant symbol", func() {
			sym, ok := constant.SymbolResolver["constant"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("constant"))
		})
	})
})
