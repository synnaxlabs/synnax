// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OP", func() {
	DescribeTable("Outputs", func(
		t string, lhs, lhsTime, rhs, rhsTime, output, outputTime telem.Series) {
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
		s := state.New(state.Config{IR: analyzed})
		lhsNode := s.Node(ctx, "lhs")
		rhsNode := s.Node(ctx, "rhs")
		*lhsNode.Output(0) = lhs
		*lhsNode.OutputTime(0) = lhsTime
		*rhsNode.Output(0) = rhs
		*rhsNode.OutputTime(0) = rhsTime
		c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
			Node:  ir.Node{Type: t},
			State: s.Node(ctx, "op"),
		}))
		changed := make(set.Set[string])
		c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
		Expect(*s.Node(ctx, "op").Output(0)).To(telem.MatchSeries(output))
		Expect(*s.Node(ctx, "op").OutputTime(0)).To(telem.MatchSeries(outputTime))
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
		Entry("Float32 Add", "add", telem.NewSeriesV[float32](1.5, 2.5, 3.5), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](0.5, 1.5, 2.5), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](2.0, 4.0, 6.0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Float64 Add", "add", telem.NewSeriesV[float64](10.5, 20.5, 30.5), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[float64](5.5, 10.5, 15.5), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[float64](16.0, 31.0, 46.0), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Int64 Add", "add", telem.NewSeriesV[int64](100, 200, 300), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[int64](50, 75, 100), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[int64](150, 275, 400), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint32 Add", "add", telem.NewSeriesV[uint32](10, 20, 30), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[uint32](5, 10, 15), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[uint32](15, 30, 45), telem.NewSeriesSecondsTSV(10, 20, 30)),
		Entry("Float32 Subtract", "subtract", telem.NewSeriesV[float32](10.0, 20.0, 30.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](3.0, 5.0, 7.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](7.0, 15.0, 23.0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int32 Subtract", "subtract", telem.NewSeriesV[int32](100, 200, 300), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int32](25, 50, 75), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int32](75, 150, 225), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Uint16 Subtract", "subtract", telem.NewSeriesV[uint16](500, 400, 300), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint16](100, 150, 200), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint16](400, 250, 100), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Float64 Multiply", "multiply", telem.NewSeriesV[float64](2.5, 3.0, 4.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float64](2.0, 3.0, 5.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float64](5.0, 9.0, 20.0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int64 Multiply", "multiply", telem.NewSeriesV[int64](10, 20, 30), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int64](2, 3, 4), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int64](20, 60, 120), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Uint8 Multiply", "multiply", telem.NewSeriesV[uint8](5, 10, 15), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](2, 3, 4), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](10, 30, 60), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Float32 Divide", "divide", telem.NewSeriesV[float32](10.0, 20.0, 30.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](2.0, 4.0, 5.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](5.0, 5.0, 6.0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int64 Divide", "divide", telem.NewSeriesV[int64](100, 200, 300), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int64](10, 20, 30), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int64](10, 10, 10), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Uint32 Divide", "divide", telem.NewSeriesV[uint32](100, 250, 500), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint32](10, 25, 50), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint32](10, 10, 10), telem.NewSeriesSecondsTSV(1, 2, 3)),
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
		t string, input, inputTime, output, outputTime telem.Series) {
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
		s := state.New(state.Config{IR: analyzed})
		inputNode := s.Node(ctx, "input")
		*inputNode.Output(0) = input
		*inputNode.OutputTime(0) = inputTime
		c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
			Node:  ir.Node{Type: t},
			State: s.Node(ctx, "op"),
		}))
		changed := make(set.Set[string])
		c.Next(node.Context{Context: ctx, MarkChanged: func(o string) { changed.Add(o) }})
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
		Expect(*s.Node(ctx, "op").Output(0)).To(telem.MatchSeries(output))
		Expect(*s.Node(ctx, "op").OutputTime(0)).To(telem.MatchSeries(outputTime))
	},
		Entry("Uint8 NOT - all false", "not", telem.NewSeriesV[uint8](0, 0, 0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](255, 255, 255), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 NOT - all true", "not", telem.NewSeriesV[uint8](1, 1, 1), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[uint8](254, 254, 254), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Uint8 NOT - mixed", "not", telem.NewSeriesV[uint8](0, 1, 0, 1), telem.NewSeriesSecondsTSV(1, 2, 3, 4), telem.NewSeriesV[uint8](255, 254, 255, 254), telem.NewSeriesSecondsTSV(1, 2, 3, 4)),
		Entry("Float64 NEG - positive", "neg", telem.NewSeriesV[float64](1.5, 2.5, 3.5), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float64](-1.5, -2.5, -3.5), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Float64 NEG - negative", "neg", telem.NewSeriesV[float64](-10.0, -20.0, -30.0), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[float64](10.0, 20.0, 30.0), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Float64 NEG - mixed", "neg", telem.NewSeriesV[float64](-1.0, 2.0, -3.0, 4.0), telem.NewSeriesSecondsTSV(1, 2, 3, 4), telem.NewSeriesV[float64](1.0, -2.0, 3.0, -4.0), telem.NewSeriesSecondsTSV(1, 2, 3, 4)),
		Entry("Float32 NEG - positive", "neg", telem.NewSeriesV[float32](5.0, 10.0, 15.0), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[float32](-5.0, -10.0, -15.0), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int64 NEG - positive", "neg", telem.NewSeriesV[int64](100, 200, 300), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[int64](-100, -200, -300), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int64 NEG - negative", "neg", telem.NewSeriesV[int64](-50, -75, -100), telem.NewSeriesSecondsTSV(5, 10, 15), telem.NewSeriesV[int64](50, 75, 100), telem.NewSeriesSecondsTSV(5, 10, 15)),
		Entry("Int32 NEG - mixed", "neg", telem.NewSeriesV[int32](10, -20, 30, -40), telem.NewSeriesSecondsTSV(1, 2, 3, 4), telem.NewSeriesV[int32](-10, 20, -30, 40), telem.NewSeriesSecondsTSV(1, 2, 3, 4)),
		Entry("Int16 NEG - positive", "neg", telem.NewSeriesV[int16](5, 10, 15), telem.NewSeriesSecondsTSV(1, 2, 3), telem.NewSeriesV[int16](-5, -10, -15), telem.NewSeriesSecondsTSV(1, 2, 3)),
		Entry("Int8 NEG - negative", "neg", telem.NewSeriesV[int8](-1, -2, -3), telem.NewSeriesSecondsTSV(10, 20, 30), telem.NewSeriesV[int8](1, 2, 3), telem.NewSeriesSecondsTSV(10, 20, 30)),
	)
	Describe("Edge Cases", func() {
		It("Should handle mismatched series lengths by extending shorter series", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "add"},
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
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[float32](1, 2, 3, 4, 5)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
			*rhsNode.Output(0) = telem.NewSeriesV[float32](10, 20)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "add"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node(ctx, "op").Output(0)
			Expect(result.Len()).To(Equal(int64(5)))
		})
		It("Should handle different time bases", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "multiply"},
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
							{Name: ir.DefaultOutputParam, Type: types.I32()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I32()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[int32](2, 3, 4)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(100, 200, 300)
			*rhsNode.Output(0) = telem.NewSeriesV[int32](5, 6, 7)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(50, 150, 250)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "multiply"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			resultTime := *s.Node(ctx, "op").OutputTime(0)
			Expect(resultTime).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(100, 200, 300)))
		})
		It("Should handle repeated calls to Next with no input changes", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "add"},
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
							{Name: ir.DefaultOutputParam, Type: types.F64()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F64()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[float64](1.5, 2.5)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(10, 20)
			*rhsNode.Output(0) = telem.NewSeriesV[float64](3.5, 4.5)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(10, 20)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "add"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			changed = make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeFalse())
		})

		It("Should handle repeated calls to Next with input changes", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "subtract"},
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
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[int64](100)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			*rhsNode.Output(0) = telem.NewSeriesV[int64](30)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "subtract"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			Expect(*s.Node(ctx, "op").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[int64](70)))
			*lhsNode.Output(0) = telem.NewSeriesV[int64](200)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			*rhsNode.Output(0) = telem.NewSeriesV[int64](50)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			changed = make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			Expect(*s.Node(ctx, "op").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[int64](150)))
		})

		It("Should handle single value series", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "lhs", Type: "lhs"},
					{Key: "rhs", Type: "rhs"},
					{Key: "op", Type: "multiply"},
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
							{Name: ir.DefaultOutputParam, Type: types.U32()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U32()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint32](7)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			*rhsNode.Output(0) = telem.NewSeriesV[uint32](8)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "multiply"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			Expect(*s.Node(ctx, "op").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[uint32](56)))
		})
		It("Should handle lhs longer than rhs", func() {
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
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7)
			*rhsNode.Output(0) = telem.NewSeriesV[float32](2, 3, 4)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "ge"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node(ctx, "op").Output(0)
			Expect(result.Len()).To(Equal(int64(7)))
		})
		It("Should handle rhs longer than lhs", func() {
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
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[int16](10, 20)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(5, 10)
			*rhsNode.Output(0) = telem.NewSeriesV[int16](10, 20, 30, 40, 50)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(5, 10, 15, 20, 25)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "eq"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node(ctx, "op").Output(0)
			Expect(result.Len()).To(Equal(int64(5)))
		})

		It("Should handle logical OR with mismatched lengths", func() {
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
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint8](0, 1, 0, 1, 1)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
			*rhsNode.Output(0) = telem.NewSeriesV[uint8](1, 0)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "or"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node(ctx, "op").Output(0)
			Expect(result.Len()).To(Equal(int64(5)))
		})

		It("Should handle logical AND with mismatched lengths", func() {
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
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint8](1, 1)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
			*rhsNode.Output(0) = telem.NewSeriesV[uint8](1, 0, 1, 1, 0)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "and"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node(ctx, "op").Output(0)
			Expect(result.Len()).To(Equal(int64(5)))
		})

		It("Should handle logical OR with single values", func() {
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
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint8](0)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			*rhsNode.Output(0) = telem.NewSeriesV[uint8](1)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "or"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			Expect(*s.Node(ctx, "op").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[uint8](1)))
		})

		It("Should handle logical AND with single values", func() {
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
			analyzed, diagnostics := graph.Analyze(ctx, g, op.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			lhsNode := s.Node(ctx, "lhs")
			rhsNode := s.Node(ctx, "rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[uint8](1)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			*rhsNode.Output(0) = telem.NewSeriesV[uint8](1)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			c := MustSucceed(op.NewFactory().Create(ctx, node.Config{
				Node:  ir.Node{Type: "and"},
				State: s.Node(ctx, "op"),
			}))
			changed := make(set.Set[string])
			c.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			Expect(*s.Node(ctx, "op").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[uint8](1)))
		})
	})
})
