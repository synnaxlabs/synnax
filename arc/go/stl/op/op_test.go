// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OP", func() {
	DescribeTable("Outputs", func(
		ctx SpecContext, t string, lhs, lhsTime, rhs, rhsTime, output, outputTime telem.Series) {
		g := graph.Graph{
			Nodes: []graph.Node{
				{Key: "lhs", Type: "lhs"},
				{Key: "rhs", Type: "rhs"},
				{Key: "op", Type: t},
			},
			Edges: []graph.Edge{
				{
					Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
				},
				{
					Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
				},
			},
			Functions: []graph.Function{
				{
					Key: "lhs",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.FromTelem(lhs.DataType)},
					},
				},
				{
					Key: "rhs",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.FromTelem(rhs.DataType)},
					},
				},
			},
		}
		analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
		Expect(diagnostics.Ok()).To(BeTrue())
		s := node.New(analyzed)
		lhsNode := s.Node("lhs")
		rhsNode := s.Node("rhs")
		*lhsNode.Output(0) = lhs
		*lhsNode.OutputTime(0) = lhsTime
		*rhsNode.Output(0) = rhs
		*rhsNode.OutputTime(0) = rhsTime
		c := MustSucceed(op.NewModule().Create(ctx, node.Config{
			Node:  ir.Node{Type: t},
			State: s.Node("op"),
		}))
		changed := make(set.Set[int])
		c.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
		Expect(changed.Contains(0)).To(BeTrue())
		Expect(*s.Node("op").Output(0)).To(telem.MatchSeries(output))
		Expect(*s.Node("op").OutputTime(0)).To(telem.MatchSeries(outputTime))
	},
		Entry("Float32 GE", "ge", telem.NewSeriesV[float32](1, 2, 3), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](0, 1, 5), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Float64 GE", "ge", telem.NewSeriesV[float64](2.5, 3.5, 1.5), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[float64](2.5, 3.0, 2.0), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[uint8](1, 1, 0), telem.NewSeriesSecondsTSV(10, 20, 30)),
		Entry("Int64 GE", "ge", telem.NewSeriesV[int64](10, 20, 30), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int64](5, 20, 35), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](1, 1, 0), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Uint32 GE", "ge", telem.NewSeriesV[uint32](100, 200, 150), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint32](100, 150, 200), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Float32 GT", "gt", telem.NewSeriesV[float32](5, 10, 15), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](4, 10, 16), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int32 GT", "gt", telem.NewSeriesV[int32](50, 60, 70), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[int32](40, 60, 80), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[uint8](1, 0, 0), telem.NewSeriesSecondsTSV(10, 20, 30)),
		Entry("Uint64 GT", "gt", telem.NewSeriesV[uint64](1000, 2000, 3000), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint64](999, 2000, 3001), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](1, 0, 0), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Float64 LE", "le", telem.NewSeriesV[float64](1.5, 2.5, 3.5), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float64](2.0, 2.5, 3.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int16 LE", "le", telem.NewSeriesV[int16](10, 20, 30), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[int16](15, 20, 25), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint16 LE", "le", telem.NewSeriesV[uint16](100, 200, 300), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint16](150, 200, 250), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](1, 1, 0), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Float32 LT", "lt", telem.NewSeriesV[float32](1.0, 2.0, 3.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](2.0, 2.0, 2.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int8 LT", "lt", telem.NewSeriesV[int8](5, 10, 15), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[int8](10, 10, 10), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 LT", "lt", telem.NewSeriesV[uint8](1, 2, 3), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[uint8](2, 2, 2), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[uint8](1, 0, 0), telem.NewSeriesSecondsTSV(10, 20, 30)),
		Entry("Float64 EQ", "eq", telem.NewSeriesV[float64](1.5, 2.5, 3.5), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float64](1.5, 2.0, 3.5), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 0, 1), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int64 EQ", "eq", telem.NewSeriesV[int64](100, 200, 300), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int64](100, 150, 300), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](1, 0, 1), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Uint32 EQ", "eq", telem.NewSeriesV[uint32](50, 60, 70), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint32](50, 65, 70), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 0, 1), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Float32 NE", "ne", telem.NewSeriesV[float32](1.0, 2.0, 3.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](1.0, 2.5, 3.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](0, 1, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int32 NE", "ne", telem.NewSeriesV[int32](10, 20, 30), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int32](10, 25, 30), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](0, 1, 0), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Uint64 NE", "ne", telem.NewSeriesV[uint64](1000, 2000, 3000), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint64](1000, 2500, 3000), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](0, 1, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 OR - all false", "or", telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 OR - all true", "or", telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 OR - mixed", "or", telem.NewSeriesV[uint8](0, 1, 0, 1), telem.NewSeriesSecondsTSV(1, 2, 3, 4), telem.NewSeriesV[uint8](0, 0, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3, 4), telem.NewSeriesV[uint8](0, 1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3, 4)),
		Entry("Uint8 OR - first true", "or", telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Uint8 OR - second true", "or", telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 AND - all false", "and", telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 AND - all true", "and", telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 AND - mixed", "and", telem.NewSeriesV[uint8](0, 1, 0, 1), telem.NewSeriesSecondsTSV(1, 2, 3, 4), telem.NewSeriesV[uint8](0, 0, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3, 4), telem.NewSeriesV[uint8](0, 0, 0, 1), telem.NewSeriesSecondsTSV(1, 2, 3, 4)),
		Entry("Uint8 AND - first false", "and", telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Uint8 AND - second false", "and", telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3)),
	)
	DescribeTable("Unary Outputs", func(
		ctx SpecContext, t string, input, inputTime, output, outputTime telem.Series) {
		g := graph.Graph{
			Nodes: []graph.Node{
				{Key: "input", Type: "input"},
				{Key: "op", Type: t},
			},
			Edges: []graph.Edge{
				{
					Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "op", Param: ir.DefaultInputParam},
				},
			},
			Functions: []graph.Function{
				{
					Key: "input",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.FromTelem(input.DataType)},
					},
				},
			},
		}
		analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
		Expect(diagnostics.Ok()).To(BeTrue())
		s := node.New(analyzed)
		inputNode := s.Node("input")
		*inputNode.Output(0) = input
		*inputNode.OutputTime(0) = inputTime
		c := MustSucceed(op.NewModule().Create(ctx, node.Config{
			Node:  ir.Node{Type: t},
			State: s.Node("op"),
		}))
		changed := make(set.Set[int])
		c.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
		Expect(changed.Contains(0)).To(BeTrue())
		Expect(*s.Node("op").Output(0)).To(telem.MatchSeries(output))
		Expect(*s.Node("op").OutputTime(0)).To(telem.MatchSeries(outputTime))
	},
		Entry("Uint8 NOT - all false", "not", telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](255, 255, 255), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 NOT - all true", "not", telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](254, 254, 254), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 NOT - mixed", "not", telem.NewSeriesV[uint8](0, 1, 0, 1), telem.NewSeriesSecondsTSV(1, 2, 3, 4), telem.NewSeriesV[uint8](255, 254, 255, 254), telem.NewSeriesSecondsTSV(1, 2, 3, 4)),
	)
	Describe("Edge Cases", func() {
		It("Should handle lhs longer than rhs", func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "ge"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "lhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F32()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F32()},
						},
					},
				},
			}
			combinedResolver := symbol.CompoundResolver{op.SymbolResolver, stlmath.SymbolResolver}
			analyzed, diagnostics := graph.Analyze(ctx, g, combinedResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)
			lhsNode := s.Node("lhs")
			rhsNode := s.Node("rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7)
			*rhsNode.Output(0) = telem.NewSeriesV[float32](2, 3, 4)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			c := MustSucceed(op.NewModule().Create(ctx, node.Config{
				Node:  ir.Node{Type: "ge"},
				State: s.Node("op"),
			}))
			changed := make(set.Set[int])
			c.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
			Expect(changed.Contains(0)).To(BeTrue())
			result := *s.Node("op").Output(0)
			Expect(result.Len()).To(Equal(int64(7)))
		})
		It("Should handle rhs longer than lhs", func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "eq"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "lhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I16()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I16()},
						},
					},
				},
			}
			combinedResolver := symbol.CompoundResolver{op.SymbolResolver, stlmath.SymbolResolver}
			analyzed, diagnostics := graph.Analyze(ctx, g, combinedResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)
			lhsNode := s.Node("lhs")
			rhsNode := s.Node("rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[int16](10, 20)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(5, 10)
			*rhsNode.Output(0) = telem.NewSeriesV[int16](10, 20, 30, 40, 50)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(5, 10, 15, 20, 25)
			c := MustSucceed(op.NewModule().Create(ctx, node.Config{
				Node:  ir.Node{Type: "eq"},
				State: s.Node("op"),
			}))
			changed := make(set.Set[int])
			c.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
			Expect(changed.Contains(0)).To(BeTrue())
			result := *s.Node("op").Output(0)
			Expect(result.Len()).To(Equal(int64(5)))
		})

		It("Should handle logical OR with mismatched lengths", func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "or"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "lhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
			}
			combinedResolver := symbol.CompoundResolver{op.SymbolResolver, stlmath.SymbolResolver}
			analyzed, diagnostics := graph.Analyze(ctx, g, combinedResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)
			lhsNode := s.Node("lhs")
			rhsNode := s.Node("rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint8](0, 1, 0, 1, 1)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
			*rhsNode.Output(0) = telem.NewSeriesV[uint8](1, 0)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
			c := MustSucceed(op.NewModule().Create(ctx, node.Config{
				Node:  ir.Node{Type: "or"},
				State: s.Node("op"),
			}))
			changed := make(set.Set[int])
			c.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
			Expect(changed.Contains(0)).To(BeTrue())
			result := *s.Node("op").Output(0)
			Expect(result.Len()).To(Equal(int64(5)))
		})

		It("Should handle logical AND with mismatched lengths", func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "and"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "lhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
			}
			combinedResolver := symbol.CompoundResolver{op.SymbolResolver, stlmath.SymbolResolver}
			analyzed, diagnostics := graph.Analyze(ctx, g, combinedResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)
			lhsNode := s.Node("lhs")
			rhsNode := s.Node("rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint8](1, 1)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
			*rhsNode.Output(0) = telem.NewSeriesV[uint8](1, 0, 1, 1, 0)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
			c := MustSucceed(op.NewModule().Create(ctx, node.Config{
				Node:  ir.Node{Type: "and"},
				State: s.Node("op"),
			}))
			changed := make(set.Set[int])
			c.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
			Expect(changed.Contains(0)).To(BeTrue())
			result := *s.Node("op").Output(0)
			Expect(result.Len()).To(Equal(int64(5)))
		})

		It("Should handle logical OR with single values", func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "or"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "lhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
			}
			combinedResolver := symbol.CompoundResolver{op.SymbolResolver, stlmath.SymbolResolver}
			analyzed, diagnostics := graph.Analyze(ctx, g, combinedResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)
			lhsNode := s.Node("lhs")
			rhsNode := s.Node("rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint8](0)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			*rhsNode.Output(0) = telem.NewSeriesV[uint8](1)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			c := MustSucceed(op.NewModule().Create(ctx, node.Config{
				Node:  ir.Node{Type: "or"},
				State: s.Node("op"),
			}))
			changed := make(set.Set[int])
			c.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
			Expect(changed.Contains(0)).To(BeTrue())
			Expect(*s.Node("op").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[uint8](1)))
		})

		It("Should handle logical AND with single values", func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "and"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "lhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
			}
			combinedResolver := symbol.CompoundResolver{op.SymbolResolver, stlmath.SymbolResolver}
			analyzed, diagnostics := graph.Analyze(ctx, g, combinedResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)
			lhsNode := s.Node("lhs")
			rhsNode := s.Node("rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint8](1)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			*rhsNode.Output(0) = telem.NewSeriesV[uint8](1)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			c := MustSucceed(op.NewModule().Create(ctx, node.Config{
				Node:  ir.Node{Type: "and"},
				State: s.Node("op"),
			}))
			changed := make(set.Set[int])
			c.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
			Expect(changed.Contains(0)).To(BeTrue())
			Expect(*s.Node("op").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[uint8](1)))
		})
	})
})
