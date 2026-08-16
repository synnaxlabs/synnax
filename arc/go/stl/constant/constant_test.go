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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/constant"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Constant", func() {
	Describe("NewModule", func() {
		It("Should create module", func(ctx SpecContext) {
			factory := constant.NewHost()
			Expect(factory).ToNot(BeNil())
		})
	})

	Describe("Factory.Create", func() {
		var (
			factory node.Factory
			s       *node.ProgramState
		)
		BeforeEach(func(ctx SpecContext) {
			factory = constant.NewHost()
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "const"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"const": {"type": "constant"},
				},
				Functions: []ir.Function{{
					Key: "constant",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		It("Should create constant for constant type", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "constant",
					Inputs: types.Params{{Name: "value", Type: types.I64(), Value: 42}},
				},
				State: s.Node("const"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("const"),
			}
			Expect(factory.Create(ctx, cfg)).Error().To(MatchError(query.ErrNotFound))
		})
		It("Should handle float64 value", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{{
						Name:  "value",
						Type:  types.F64(),
						Value: 3.14,
					}},
				},
				State: s.Node("const"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
		It("Should handle int value", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{{
						Name:  "value",
						Type:  types.I64(),
						Value: 100,
					}},
				},
				State: s.Node("const"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
		It("Should handle uint8 value", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{{
						Name:  "value",
						Type:  types.U8(),
						Value: uint8(255),
					}},
				},
				State: s.Node("const"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
	})

	Describe("Next", func() {
		var (
			s       *node.ProgramState
			factory node.Factory
			marked  []int
		)
		BeforeEach(func(ctx SpecContext) {
			factory = constant.NewHost()
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "const"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"const": {"type": "constant"},
				},
				Functions: []ir.Function{{
					Key: "constant",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.I64()},
					},
				}},
			}
			inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(inter)
			marked = nil
		})

		It("Should emit output on Next with int value", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "constant",
					Inputs: types.Params{{Name: "value", Type: types.I64(), Value: 42}},
				},
				State: s.Node("const"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(i int) {
				marked = append(marked, i)
			}})
			Expect(marked).To(ConsistOf(0))
		})

		It("Should set output data on Next", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{
						{Name: "value", Type: types.I64(), Value: int64(100)},
					},
				},
				State: s.Node("const"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			out := s.Node("const").Output(0)
			Expect(out.Len()).To(Equal(int64(1)))
		})

		It("Should set output time on Next", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{
						{Name: "value", Type: types.F64(), Value: 3.14},
					},
				},
				State: s.Node("const"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			outTime := s.Node("const").OutputTime(0)
			Expect(outTime.Len()).To(Equal(int64(1)))
			times := telem.UnmarshalSeries[telem.TimeStamp](*outTime)
			Expect(times[0]).To(BeNumerically(">", int64(0)))
		})

		It("Should handle float64 constant", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{
						{Name: "value", Type: types.F64(), Value: 2.718},
					},
				},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[float64](0)
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[float64](*out)
			Expect(vals[0]).To(Equal(2.718))
		})

		It("Should handle int32 constant", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{
						{Name: "value", Type: types.I32(), Value: int32(42)},
					},
				},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int32](0)
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int32](*out)
			Expect(vals[0]).To(Equal(int32(42)))
		})

		It("Should handle uint8 constant", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(255)},
					},
				},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[uint8](0)
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[uint8](*out)
			Expect(vals[0]).To(Equal(uint8(255)))
		})

		It("Should allow downstream nodes to read constant", func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "const"},
					{Key: "sink"},
				},
				Inputs: map[string]msgpack.EncodedJSON{
					"const": {"type": "constant"},
					"sink":  {"type": "sink"},
				},
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: ir.Handle{Node: "const", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
					}},
				},
				Functions: []ir.Function{
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
			inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(inter)
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{
						{Name: "value", Type: types.I64(), Value: int64(999)},
					},
				},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			sink := s.Node("sink")
			recalc := sink.RefreshInputs()
			Expect(recalc).To(BeTrue())
			input := sink.Input(0)
			vals := telem.UnmarshalSeries[int64](input)
			Expect(vals[0]).To(Equal(int64(999)))
		})

		It("Should handle zero value constant", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "constant",
					Inputs: types.Params{{Name: "value", Type: types.I64(), Value: 0}},
				},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int64](*out)
			Expect(vals[0]).To(Equal(int64(0)))
		})

		It("Should handle negative value constant", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "constant",
					Inputs: types.Params{
						{Name: "value", Type: types.I64(), Value: -42},
					},
				},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int64](*out)
			Expect(vals[0]).To(Equal(int64(-42)))
		})

		It(
			"Should only emit once across multiple Next calls when node has no incoming edges",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "constant",
						Inputs: types.Params{
							{Name: "value", Type: types.I64(), Value: int64(42)},
						},
					},
					State: s.Node("const"),
				}
				constNode := s.Node("const")
				*constNode.Output(0) = telem.NewSeriesV[int64](0)
				n := MustSucceed(factory.Create(ctx, cfg))
				n.Next(node.Context{Context: ctx, MarkChanged: func(i int) {
					marked = append(marked, i)
				}})
				Expect(marked).To(HaveLen(1))
				n.Next(node.Context{Context: ctx, MarkChanged: func(i int) {
					marked = append(marked, i)
				}})
				Expect(marked).To(HaveLen(1))
			},
		)

		It(
			"Should emit again after Reset is called when node has no incoming edges",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "constant",
						Inputs: types.Params{
							{Name: "value", Type: types.I64(), Value: int64(42)},
						},
					},
					State: s.Node("const"),
				}
				constNode := s.Node("const")
				*constNode.Output(0) = telem.NewSeriesV[int64](0)
				n := MustSucceed(factory.Create(ctx, cfg))
				n.Next(node.Context{Context: ctx, MarkChanged: func(i int) {
					marked = append(marked, i)
				}})
				Expect(marked).To(HaveLen(1))
				n.Reset()
				n.Next(node.Context{Context: ctx, MarkChanged: func(i int) {
					marked = append(marked, i)
				}})
				Expect(marked).To(HaveLen(2))
			},
		)

		It(
			"Should emit on every Next call when node has incoming edges",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Key:  "const",
						Type: "constant",
						Inputs: types.Params{
							{Name: "value", Type: types.I64(), Value: int64(42)},
						},
					},
					State: s.Node("const"),
					Program: program.Program{IR: ir.IR{Edges: ir.Edges{
						{
							Source: ir.Handle{
								Node:  "upstream",
								Param: ir.DefaultOutputParam,
							},
							Target: ir.Handle{
								Node:  "const",
								Param: ir.DefaultInputParam,
							},
						},
					}}},
				}
				constNode := s.Node("const")
				*constNode.Output(0) = telem.NewSeriesV[int64](0)
				n := MustSucceed(factory.Create(ctx, cfg))
				for range 3 {
					n.Next(node.Context{Context: ctx, MarkChanged: func(i int) {
						marked = append(marked, i)
					}})
				}
				Expect(marked).To(HaveLen(3))
			},
		)

		It(
			"Should not require Reset to re-fire when node has incoming edges",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Key:  "const",
						Type: "constant",
						Inputs: types.Params{
							{Name: "value", Type: types.I64(), Value: int64(42)},
						},
					},
					State: s.Node("const"),
					Program: program.Program{IR: ir.IR{Edges: ir.Edges{
						{
							Source: ir.Handle{
								Node:  "upstream",
								Param: ir.DefaultOutputParam,
							},
							Target: ir.Handle{
								Node:  "const",
								Param: ir.DefaultInputParam,
							},
						},
					}}},
				}
				constNode := s.Node("const")
				*constNode.Output(0) = telem.NewSeriesV[int64](0)
				n := MustSucceed(factory.Create(ctx, cfg))
				n.Next(node.Context{Context: ctx, MarkChanged: func(i int) {
					marked = append(marked, i)
				}})
				n.Next(node.Context{Context: ctx, MarkChanged: func(i int) {
					marked = append(marked, i)
				}})
				Expect(marked).To(HaveLen(2))
			},
		)
	})

	Describe("Var-bound value", func() {
		var factory node.Factory
		BeforeEach(func() {
			factory = constant.NewHost()
		})

		// build wires a constant whose value input references variable node "v"
		// and gives it an inbound trigger edge so it re-emits on every Next.
		build := func(valueType types.Type, initial any) (node.Config, *node.ProgramState) {
			v := ir.Node{
				Key:     "v",
				Type:    "variable",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: valueType}},
			}
			n := ir.Node{
				Key:  "n",
				Type: "constant",
				Inputs: types.Params{{
					Name:  "value",
					Type:  types.VarRef(valueType, "v"),
					Value: initial,
				}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: valueType}},
			}
			edges := ir.Edges{{
				Source: ir.Handle{Node: "up", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "n", Param: ir.DefaultInputParam},
			}}
			state := node.New(ir.IR{Nodes: ir.Nodes{v, n}, Edges: edges})
			cfg := node.Config{
				Node:    n,
				State:   state.Node("n"),
				Program: program.Program{IR: ir.IR{Edges: edges}},
			}
			return cfg, state
		}
		next := func(ctx SpecContext, n node.Node) {
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
		}

		It(
			"Should emit the declared initial before any variable write",
			func(ctx SpecContext) {
				cfg, state := build(types.I64(), int64(42))
				n := MustSucceed(factory.Create(ctx, cfg))
				next(ctx, n)
				out := state.Node("n").Output(0)
				Expect(out.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int64](*out, 0)).To(Equal(int64(42)))
			},
		)

		It("Should emit the live value after a variable write", func(ctx SpecContext) {
			cfg, state := build(types.I64(), int64(42))
			n := MustSucceed(factory.Create(ctx, cfg))
			*state.Node("v").Output(0) = telem.NewSeriesV[int64](7)
			next(ctx, n)
			out := state.Node("n").Output(0)
			Expect(out.Len()).To(Equal(int64(1)))
			Expect(telem.ValueAt[int64](*out, 0)).To(Equal(int64(7)))
			Expect(out.DataType).To(Equal(telem.Int64T))
		})

		It("Should track successive variable writes", func(ctx SpecContext) {
			cfg, state := build(types.I64(), int64(42))
			n := MustSucceed(factory.Create(ctx, cfg))
			*state.Node("v").Output(0) = telem.NewSeriesV[int64](7)
			next(ctx, n)
			Expect(telem.ValueAt[int64](*state.Node("n").Output(0), 0)).
				To(Equal(int64(7)))
			*state.Node("v").Output(0) = telem.NewSeriesV[int64](9)
			next(ctx, n)
			Expect(telem.ValueAt[int64](*state.Node("n").Output(0), 0)).
				To(Equal(int64(9)))
		})

		It(
			"Should emit only the latest sample of the variable's series",
			func(ctx SpecContext) {
				cfg, state := build(types.I64(), int64(42))
				n := MustSucceed(factory.Create(ctx, cfg))
				*state.Node("v").Output(0) = telem.NewSeriesV[int64](1, 2, 3)
				next(ctx, n)
				out := state.Node("n").Output(0)
				Expect(out.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int64](*out, 0)).To(Equal(int64(3)))
			},
		)

		It(
			"Should emit the declared string initial before any write",
			func(ctx SpecContext) {
				cfg, state := build(types.String(), "hello")
				n := MustSucceed(factory.Create(ctx, cfg))
				next(ctx, n)
				out := state.Node("n").Output(0)
				Expect(out.Len()).To(Equal(int64(1)))
				Expect(string(out.At(-1))).To(Equal("hello"))
			},
		)

		It("Should emit the live string after a write", func(ctx SpecContext) {
			cfg, state := build(types.String(), "hello")
			n := MustSucceed(factory.Create(ctx, cfg))
			*state.Node("v").Output(0) = telem.NewSeriesV("goodbye")
			next(ctx, n)
			out := state.Node("n").Output(0)
			Expect(out.Len()).To(Equal(int64(1)))
			Expect(string(out.At(-1))).To(Equal("goodbye"))
			Expect(out.DataType).To(Equal(telem.StringT))
		})

		It("Should emit only the latest string sample", func(ctx SpecContext) {
			cfg, state := build(types.String(), "hello")
			n := MustSucceed(factory.Create(ctx, cfg))
			*state.Node("v").Output(0) = telem.NewSeriesV("a", "b")
			next(ctx, n)
			out := state.Node("n").Output(0)
			Expect(out.Len()).To(Equal(int64(1)))
			Expect(string(out.At(-1))).To(Equal("b"))
		})

		It(
			"Should emit a live float value from raw sample bytes",
			func(ctx SpecContext) {
				cfg, state := build(types.F64(), 1.5)
				n := MustSucceed(factory.Create(ctx, cfg))
				*state.Node("v").Output(0) = telem.NewSeriesV(2.5)
				next(ctx, n)
				out := state.Node("n").Output(0)
				Expect(telem.ValueAt[float64](*out, 0)).To(Equal(2.5))
			},
		)

		It(
			"Should re-emit on every trigger, tracking the variable",
			func(ctx SpecContext) {
				cfg, state := build(types.I64(), int64(42))
				n := MustSucceed(factory.Create(ctx, cfg))
				var marked []int
				mark := node.Context{Context: ctx, MarkChanged: func(i int) {
					marked = append(marked, i)
				}}
				n.Next(mark)
				Expect(telem.ValueAt[int64](*state.Node("n").Output(0), 0)).
					To(Equal(int64(42)))
				*state.Node("v").Output(0) = telem.NewSeriesV[int64](7)
				n.Next(mark)
				Expect(telem.ValueAt[int64](*state.Node("n").Output(0), 0)).
					To(Equal(int64(7)))
				Expect(marked).To(HaveLen(2))
			},
		)
	})

	Describe("Symbols", func() {
		It("Should expose constant symbol", func() {
			var sym *symbol.Symbol
			for _, s := range constant.NewSymbols() {
				if s.Name == "constant" {
					sym = s
					break
				}
			}
			Expect(sym).ToNot(BeNil())
			Expect(sym.Name).To(Equal("constant"))
		})
	})
})
