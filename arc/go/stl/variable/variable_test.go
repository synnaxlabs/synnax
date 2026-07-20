// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package variable_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/variable"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// registerState builds a variable node "v" with a seeded value param and a
// feeder node "f" edged into its second param.
func registerState(ctx SpecContext) *node.ProgramState {
	g := graph.Graph{
		Functions: []graph.Function{
			{
				Key:     "feeder",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			},
			{
				Key: "variable",
				Inputs: types.Params{
					{Name: "value", Type: types.I64(), Value: int64(42)},
					{Name: "f1", Type: types.I64()},
				},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			},
		},
		Nodes: []graph.Node{{Key: "f"}, {Key: "v"}},
		Inputs: map[string]msgpack.EncodedJSON{
			"f": {"type": "feeder"},
			"v": {"type": "variable"},
		},
		Edges: graph.Edges{{Edge: ir.Edge{
			Source: ir.Handle{Node: "f", Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "v", Param: "f1"},
		}}},
	}
	analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	return node.New(analyzed)
}

// exprReadState builds a variable node "v" whose params are all edge-fed: two
// derivation feeders d0 and d1 plus the sel switch input.
func exprReadState(ctx SpecContext) *node.ProgramState {
	g := graph.Graph{
		Functions: []graph.Function{
			{
				Key:     "d0",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			},
			{
				Key:     "d1",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			},
			// The real lowering feeds sel from the register, a variable node, so
			// sel keeps its fires-on-fresh rearm rule and does not replay on Reset.
			{
				Key:     "stateful_variable",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U32()}},
			},
			{
				Key: "variable",
				Inputs: types.Params{
					{Name: "f0", Type: types.I64()},
					{Name: "f1", Type: types.I64()},
					{Name: "sel", Type: types.U32()},
				},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			},
		},
		Nodes: []graph.Node{{Key: "d0"}, {Key: "d1"}, {Key: "selsrc"}, {Key: "v"}},
		Inputs: map[string]msgpack.EncodedJSON{
			"d0":     {"type": "d0"},
			"d1":     {"type": "d1"},
			"selsrc": {"type": "stateful_variable"},
			"v":      {"type": "variable"},
		},
		Edges: graph.Edges{
			{Edge: ir.Edge{
				Source: ir.Handle{Node: "d0", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "v", Param: "f0"},
			}},
			{Edge: ir.Edge{
				Source: ir.Handle{Node: "d1", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "v", Param: "f1"},
			}},
			{Edge: ir.Edge{
				Source: ir.Handle{Node: "selsrc", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "v", Param: "sel"},
			}},
		},
	}
	analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	return node.New(analyzed)
}

// emit writes data and a second-precision timestamp to a node's first output.
func emit[T telem.Sample](n *node.State, value T, seconds telem.TimeStamp) {
	*n.Output(0) = telem.NewSeriesV[T](value)
	*n.OutputTime(0) = telem.NewSeriesSecondsTSV(seconds)
}

var _ = Describe("Variable", func() {
	var (
		factory *variable.Host
		marked  []int
		nodeCtx node.Context
	)
	BeforeEach(func(ctx SpecContext) {
		factory = variable.NewHost()
		marked = nil
		nodeCtx = node.Context{Context: ctx, MarkChanged: func(i int) {
			marked = append(marked, i)
		}}
	})

	Describe("NewSymbols", func() {
		DescribeTable("internal flow symbol shape",
			func(name string) {
				var sym *symbol.Symbol
				for _, s := range variable.NewSymbols() {
					if s.Name == name {
						sym = s
					}
				}
				Expect(sym).ToNot(BeNil())
				Expect(sym.Kind).To(Equal(symbol.KindFunction))
				Expect(sym.Internal).To(BeTrue())
				Expect(sym.Exec).To(Equal(symbol.ExecFlow))
				Expect(sym.Trigger).To(Equal(symbol.TriggerOnly))
				Expect(sym.Type.Inputs).To(HaveLen(1))
				Expect(sym.Type.Inputs[0].Name).To(Equal("value"))
				Expect(sym.Type.Outputs).To(HaveLen(1))
				Expect(sym.Type.Outputs[0].Name).To(Equal(ir.DefaultOutputParam))
			},
			Entry("variable", "variable"),
			Entry("stateful_variable", "stateful_variable"),
		)

		It("Should return exactly the two variable symbols", func() {
			Expect(variable.NewSymbols()).To(HaveLen(2))
		})
	})

	Describe("Create", func() {
		var s *node.ProgramState
		BeforeEach(func(ctx SpecContext) { s = registerState(ctx) })

		It("Should return NotFound for an unknown node type", func(ctx SpecContext) {
			cfg := node.Config{Node: ir.Node{Type: "unknown"}, State: s.Node("v")}
			Expect(factory.Create(ctx, cfg)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should create a node for a seeded variable", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "variable",
					Inputs: types.Params{{Name: "value", Type: types.I64(), Value: int64(1)}},
				},
				State: s.Node("v"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})

		It("Should create a node for a seeded stateful_variable", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "stateful_variable",
					Inputs: types.Params{{Name: "value", Type: types.I64(), Value: int64(1)}},
				},
				State: s.Node("v"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})

		It("Should create a node for an edge-fed variable", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "variable",
					Inputs: types.Params{{Name: "f0", Type: types.I64()}},
				},
				State: s.Node("v"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})

		It("Should create a node for a variable with no inputs", func(ctx SpecContext) {
			cfg := node.Config{Node: ir.Node{Type: "variable"}, State: s.Node("v")}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})

		It("Should create a node for an edge-fed stateful_variable", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "stateful_variable",
					Inputs: types.Params{{Name: "f0", Type: types.I64()}},
				},
				State: s.Node("v"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
	})

	Describe("Register", func() {
		var (
			s *node.ProgramState
			v *node.State
			f *node.State
		)
		mk := func(ctx SpecContext, nodeType string) node.Node {
			cfg := node.Config{
				Node: ir.Node{
					Type:   nodeType,
					Inputs: types.Params{{Name: "value", Type: types.I64(), Value: int64(42)}},
				},
				State: v,
			}
			return MustSucceed(factory.Create(ctx, cfg))
		}
		BeforeEach(func(ctx SpecContext) {
			s = registerState(ctx)
			v, f = s.Node("v"), s.Node("f")
		})

		It("Should emit its pending seed on first Next", func(ctx SpecContext) {
			n := mk(ctx, "variable")
			n.Next(nodeCtx)
			Expect(marked).To(ConsistOf(0))
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](42))
			Expect(v.OutputTime(0).Len()).To(Equal(int64(1)))
		})

		It("Should not re-emit without new data", func(ctx SpecContext) {
			n := mk(ctx, "variable")
			n.Next(nodeCtx)
			n.Next(nodeCtx)
			Expect(marked).To(HaveLen(1))
		})

		It("Should emit a feeder's value", func(ctx SpecContext) {
			n := mk(ctx, "variable")
			n.Next(nodeCtx)
			emit(f, int64(7), 10)
			n.Next(nodeCtx)
			Expect(marked).To(HaveLen(2))
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](7))
		})

		It("Should take the newest value when multiple are pending", func(ctx SpecContext) {
			n := mk(ctx, "variable")
			emit(f, int64(7), 10)
			n.Next(nodeCtx)
			Expect(marked).To(ConsistOf(0))
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](7))
		})

		It("Should not alias the feeder's output buffer", func(ctx SpecContext) {
			n := mk(ctx, "variable")
			n.Next(nodeCtx)
			emit(f, int64(7), 10)
			n.Next(nodeCtx)
			f.Output(0).Data[0] = 9
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](7))
		})

		Context("with a := variable", func() {
			It("Should seed the output on Reset", func(ctx SpecContext) {
				n := mk(ctx, "variable")
				n.Reset()
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](42))
				Expect(v.OutputTime(0).Len()).To(Equal(int64(1)))
			})

			It("Should not double-emit the seed on Next after Reset", func(ctx SpecContext) {
				n := mk(ctx, "variable")
				n.Reset()
				n.Next(nodeCtx)
				Expect(marked).To(BeEmpty())
			})

			It("Should supersede a pending feeder value on Reset", func(ctx SpecContext) {
				n := mk(ctx, "variable")
				emit(f, int64(99), 10)
				n.Reset()
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](42))
				n.Next(nodeCtx)
				Expect(marked).To(BeEmpty())
			})

			It("Should re-seed on scope re-entry", func(ctx SpecContext) {
				n := mk(ctx, "variable")
				n.Reset()
				emit(f, int64(7), 10)
				n.Next(nodeCtx)
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](7))
				n.Reset()
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](42))
			})

			It("Should not alias the seed on Reset", func(ctx SpecContext) {
				n := mk(ctx, "variable")
				n.Reset()
				v.Output(0).Data[0] = 9
				n.Reset()
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](42))
			})
		})

		Context("with a $= variable", func() {
			It("Should emit its seed on first Next", func(ctx SpecContext) {
				n := mk(ctx, "stateful_variable")
				n.Next(nodeCtx)
				Expect(marked).To(ConsistOf(0))
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](42))
			})

			It("Should persist its value across Reset", func(ctx SpecContext) {
				n := mk(ctx, "stateful_variable")
				emit(f, int64(7), 10)
				n.Next(nodeCtx)
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](7))
				n.Reset()
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](7))
			})

			It("Should leave pending feeder values consumable after Reset", func(ctx SpecContext) {
				n := mk(ctx, "stateful_variable")
				emit(f, int64(7), 10)
				n.Reset()
				n.Next(nodeCtx)
				Expect(marked).To(ConsistOf(0))
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](7))
			})
		})

		Context("with a string variable", func() {
			var (
				ss *node.ProgramState
				sv *node.State
				sf *node.State
			)
			BeforeEach(func(ctx SpecContext) {
				g := graph.Graph{
					Functions: []graph.Function{
						{
							Key: "feeder",
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.String()},
							},
						},
						{
							Key: "variable",
							Inputs: types.Params{
								{Name: "value", Type: types.String(), Value: "hello"},
								{Name: "f1", Type: types.String()},
							},
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.String()},
							},
						},
					},
					Nodes: []graph.Node{{Key: "f"}, {Key: "v"}},
					Inputs: map[string]msgpack.EncodedJSON{
						"f": {"type": "feeder"},
						"v": {"type": "variable"},
					},
					Edges: graph.Edges{{Edge: ir.Edge{
						Source: ir.Handle{Node: "f", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "v", Param: "f1"},
					}}},
				}
				analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				ss = node.New(analyzed)
				sv, sf = ss.Node("v"), ss.Node("f")
			})

			It("Should seed and emit string values", func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "variable",
						Inputs: types.Params{
							{Name: "value", Type: types.String(), Value: "hello"},
						},
					},
					State: sv,
				}
				n := MustSucceed(factory.Create(ctx, cfg))
				n.Reset()
				Expect(*sv.Output(0)).To(telem.MatchSeriesDataV("hello"))
				emit(sf, "world", 10)
				n.Next(nodeCtx)
				Expect(marked).To(ConsistOf(0))
				Expect(*sv.Output(0)).To(telem.MatchSeriesDataV("world"))
				n.Reset()
				Expect(*sv.Output(0)).To(telem.MatchSeriesDataV("hello"))
			})
		})
	})

	Describe("ExprRead", func() {
		var (
			s      *node.ProgramState
			v      *node.State
			d0     *node.State
			d1     *node.State
			selsrc *node.State
			n      node.Node
		)
		BeforeEach(func(ctx SpecContext) {
			s = exprReadState(ctx)
			v, d0, d1, selsrc = s.Node("v"), s.Node("d0"), s.Node("d1"), s.Node("selsrc")
			cfg := node.Config{
				Node: ir.Node{
					Type:   "variable",
					Inputs: types.Params{{Name: "f0", Type: types.I64()}},
				},
				State: v,
			}
			n = MustSucceed(factory.Create(ctx, cfg))
		})

		It("Should emit the active derivation's value", func(ctx SpecContext) {
			emit(d0, int64(5), 10)
			n.Next(nodeCtx)
			Expect(marked).To(ConsistOf(0))
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](5))
		})

		It("Should coalesce an unchanged recompute", func(ctx SpecContext) {
			emit(d0, int64(5), 10)
			n.Next(nodeCtx)
			emit(d0, int64(5), 20)
			n.Next(nodeCtx)
			Expect(marked).To(HaveLen(1))
			emit(d0, int64(6), 30)
			n.Next(nodeCtx)
			Expect(marked).To(HaveLen(2))
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](6))
		})

		It("Should drain an inactive feeder without emitting", func(ctx SpecContext) {
			emit(d1, int64(9), 10)
			n.Next(nodeCtx)
			Expect(marked).To(BeEmpty())
			Expect(v.Output(0).Len()).To(BeZero())
		})

		It("Should swallow values pending at a re-point", func(ctx SpecContext) {
			emit(d1, int64(9), 10)
			emit(selsrc, uint32(1), 10)
			n.Next(nodeCtx)
			Expect(marked).To(BeEmpty())
			emit(d1, int64(11), 20)
			n.Next(nodeCtx)
			Expect(marked).To(ConsistOf(0))
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](11))
		})

		It("Should stop emitting a feeder de-activated by a re-point", func(ctx SpecContext) {
			emit(selsrc, uint32(1), 10)
			n.Next(nodeCtx)
			emit(d0, int64(5), 20)
			n.Next(nodeCtx)
			Expect(marked).To(BeEmpty())
		})

		It("Should re-activate a feeder when sel points back", func(ctx SpecContext) {
			emit(selsrc, uint32(1), 10)
			n.Next(nodeCtx)
			emit(d1, int64(11), 20)
			n.Next(nodeCtx)
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](11))
			emit(selsrc, uint32(0), 30)
			n.Next(nodeCtx)
			emit(d0, int64(5), 40)
			n.Next(nodeCtx)
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](5))
			Expect(marked).To(HaveLen(2))
		})

		It("Should drain silently when sel is out of range", func(ctx SpecContext) {
			emit(selsrc, uint32(9), 10)
			n.Next(nodeCtx)
			emit(d0, int64(5), 20)
			emit(d1, int64(6), 20)
			n.Next(nodeCtx)
			Expect(marked).To(BeEmpty())
			Expect(v.Output(0).Len()).To(BeZero())
		})

		It("Should not alias the derivation's output buffer", func(ctx SpecContext) {
			emit(d0, int64(5), 10)
			n.Next(nodeCtx)
			d0.Output(0).Data[0] = 9
			Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](5))
		})

		Describe("Reset", func() {
			It("Should coalesce values replayed by Reset", func(ctx SpecContext) {
				emit(d0, int64(5), 10)
				n.Next(nodeCtx)
				Expect(marked).To(ConsistOf(0))
				n.Reset()
				n.Next(nodeCtx)
				Expect(marked).To(HaveLen(1))
				emit(d0, int64(6), 20)
				n.Next(nodeCtx)
				Expect(marked).To(HaveLen(2))
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](6))
			})

			It("Should keep sel consumed across Reset", func(ctx SpecContext) {
				emit(selsrc, uint32(1), 10)
				n.Next(nodeCtx)
				n.Reset()
				emit(d1, int64(11), 20)
				// A replayed sel would count as a re-point and swallow the 11.
				n.Next(nodeCtx)
				Expect(marked).To(ConsistOf(0))
				Expect(*v.Output(0)).To(telem.MatchSeriesDataV[int64](11))
			})
		})

		It("Should emit the sole derivation when no sel input exists", func(ctx SpecContext) {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key:     "d0",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
					},
					{
						Key:     "variable",
						Inputs:  types.Params{{Name: "f0", Type: types.I64()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
					},
				},
				Nodes: []graph.Node{{Key: "d0"}, {Key: "v"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"d0": {"type": "d0"},
					"v":  {"type": "variable"},
				},
				Edges: graph.Edges{{Edge: ir.Edge{
					Source: ir.Handle{Node: "d0", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "v", Param: "f0"},
				}}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			ps := node.New(analyzed)
			vNode, feeder := ps.Node("v"), ps.Node("d0")
			cfg := node.Config{
				Node: ir.Node{
					Type:   "variable",
					Inputs: types.Params{{Name: "f0", Type: types.I64()}},
				},
				State: vNode,
			}
			soleN := MustSucceed(factory.Create(ctx, cfg))
			emit(feeder, int64(5), 10)
			soleN.Next(nodeCtx)
			Expect(marked).To(ConsistOf(0))
			Expect(*vNode.Output(0)).To(telem.MatchSeriesDataV[int64](5))
		})
	})
})
