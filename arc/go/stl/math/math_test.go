// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/runtime/node"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func makeMathGraph(nodeType string, dt types.Type) graph.Graph {
	return graph.Graph{
		Nodes: []graph.Node{
			{Key: "input", Type: "input"},
			{Key: "math", Type: nodeType},
		},
		Edges: []graph.Edge{{
			Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "math", Param: ir.DefaultInputParam},
		}},
		Functions: []graph.Function{{
			Key:     "input",
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: dt}},
		}},
	}
}

func makeMathGraphWithReset(nodeType string, dt types.Type) graph.Graph {
	return graph.Graph{
		Nodes: []graph.Node{
			{Key: "input", Type: "input"},
			{Key: "reset_signal", Type: "reset_signal"},
			{Key: "math", Type: nodeType},
		},
		Edges: []graph.Edge{
			{
				Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "math", Param: ir.DefaultInputParam},
			},
			{
				Source: ir.Handle{Node: "reset_signal", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "math", Param: "reset"},
			},
		},
		Functions: []graph.Function{
			{Key: "input", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: dt}}},
			{Key: "reset_signal", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}},
		},
	}
}

type mathSetup struct {
	state     *node.ProgramState
	inputNode *node.State
	n         node.Node
}

func openMath(
	ctx SpecContext,
	nodeType string,
	dt types.Type,
	config types.Params,
) mathSetup {
	g := makeMathGraph(nodeType, dt)
	analyzed, diagnostics := graph.Analyze(ctx, g, stlmath.SymbolResolver)
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	s := node.New(analyzed)
	inputNode := s.Node("input")
	m := MustSucceed(stlmath.NewModule(ctx, nil))
	n := MustSucceed(m.Create(ctx, node.Config{
		Node:    ir.Node{Key: "math", Type: nodeType, Config: config},
		State:   s.Node("math"),
		Program: program.Program{IR: analyzed},
	}))
	return mathSetup{state: s, inputNode: inputNode, n: n}
}

func openMathWithReset(
	ctx SpecContext,
	nodeType string,
	dt types.Type,
	config types.Params,
) mathSetup {
	g := makeMathGraphWithReset(nodeType, dt)
	analyzed, diagnostics := graph.Analyze(ctx, g, stlmath.SymbolResolver)
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	s := node.New(analyzed)
	inputNode := s.Node("input")
	m := MustSucceed(stlmath.NewModule(ctx, nil))
	n := MustSucceed(m.Create(ctx, node.Config{
		Node:    ir.Node{Key: "math", Type: nodeType, Config: config},
		State:   s.Node("math"),
		Program: program.Program{IR: analyzed},
	}))
	return mathSetup{state: s, inputNode: inputNode, n: n}
}

func nextChanged(ctx SpecContext, n node.Node) set.Set[int] {
	changed := make(set.Set[int])
	n.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(i) }})
	return changed
}

func expectOutput[T telem.NumericSample](s *node.ProgramState, values ...T) {
	result := *s.Node("math").Output(0)
	Expect(result.Len()).To(Equal(int64(len(values))))
	vals := telem.UnmarshalSeries[T](result)
	for i, v := range values {
		Expect(vals[i]).To(BeNumerically("~", v, 0.01))
	}
}

func expectOutputTime(s *node.ProgramState, timestamps ...telem.TimeStamp) {
	result := *s.Node("math").OutputTime(0)
	Expect(result.Len()).To(Equal(int64(len(timestamps))))
	vals := telem.UnmarshalSeries[telem.TimeStamp](result)
	for i, ts := range timestamps {
		Expect(vals[i]).To(Equal(ts))
	}
}

var _ = Describe("Math", func() {
	Describe("pow", func() {
		var rt *testutil.Runtime

		BeforeEach(func(ctx SpecContext) {
			rt = testutil.NewRuntime(ctx)
			MustSucceed(stlmath.NewModule(ctx, rt.Underlying()))
			rt.Passthrough(ctx, "math")
		})

		AfterEach(func(ctx SpecContext) {
			Expect(rt.Close(ctx)).To(Succeed())
		})

		It("Should compute i32 power", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_i32", testutil.U32(3), testutil.U32(2))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(9)))
			res = rt.Call(ctx, "math", "pow_i32", testutil.U32(2), testutil.U32(10))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(1024)))
		})

		It("Should compute i32 power with negative base", func(ctx SpecContext) {
			var negThree int32 = -3
			res := rt.Call(ctx, "math", "pow_i32", testutil.I32(negThree), testutil.U32(2))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(9)))
			var negTwo int32 = -2
			var expected int32 = -8
			res = rt.Call(ctx, "math", "pow_i32", testutil.I32(negTwo), testutil.U32(3))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(uint32(expected))))
		})

		It("Should compute u64 power", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_u64", testutil.U64(2), testutil.U64(10))
			Expect(testutil.AsU64(res[0])).To(Equal(uint64(1024)))
		})

		It("Should compute f32 power", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f32", testutil.F32(2.0), testutil.F32(3.0))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", 8.0, 0.001))
		})

		It("Should compute f64 power", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f64", testutil.F64(2.0), testutil.F64(0.5))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", 1.41421356, 0.0001))
		})

		It("Should truncate negative integer exponents to zero", func(ctx SpecContext) {
			negOne := int32(-1)
			res := rt.Call(ctx, "math", "pow_i32", testutil.U32(2), testutil.I32(negOne))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(0)))
		})

		It("Should compute f64 negative exponents", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f64", testutil.F64(2.0), testutil.F64(-1.0))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", 0.5, 0.0001))
		})

		It("Should compute f32 negative exponents", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f32", testutil.F32(4.0), testutil.F32(-0.5))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", 0.5, 0.001))
		})

		It("Should compute f64 with negative base", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f64", testutil.F64(-3.0), testutil.F64(2.0))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", 9.0, 0.0001))
			res = rt.Call(ctx, "math", "pow_f64", testutil.F64(-2.0), testutil.F64(3.0))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", -8.0, 0.0001))
			res = rt.Call(ctx, "math", "pow_f64", testutil.F64(-2.0), testutil.F64(-1.0))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", -0.5, 0.0001))
		})

		It("Should compute f32 with negative base", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f32", testutil.F32(-3.0), testutil.F32(2.0))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", 9.0, 0.001))
			res = rt.Call(ctx, "math", "pow_f32", testutil.F32(-2.0), testutil.F32(3.0))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", -8.0, 0.001))
		})
	})

	Describe("arithmetic WASM", func() {
		var rt *testutil.Runtime

		BeforeEach(func(ctx SpecContext) {
			rt = testutil.NewRuntime(ctx)
			MustSucceed(stlmath.NewModule(ctx, rt.Underlying()))
			rt.Passthrough(ctx, "math")
		})

		AfterEach(func(ctx SpecContext) {
			Expect(rt.Close(ctx)).To(Succeed())
		})

		It("Should negate i32 values", func(ctx SpecContext) {
			var negFive int32 = -5
			res := rt.Call(ctx, "math", "neg_i32", testutil.I32(5))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(uint32(negFive))))
		})
		It("Should negate f64 values", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "neg_f64", testutil.F64(3.5))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", -3.5, 0.001))
		})
		It("Should negate i8 values", func(ctx SpecContext) {
			var negThree int32 = -3
			res := rt.Call(ctx, "math", "neg_i8", testutil.I32(3))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(uint32(negThree))))
		})
		It("Should negate i64 values", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "neg_i64", testutil.U64(42))
			var expected int64 = -42
			Expect(testutil.AsU64(res[0])).To(Equal(uint64(expected)))
		})
		It("Should negate f32 values", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "neg_f32", testutil.F32(3.5))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", -3.5, 0.001))
		})
	})

	Describe("SymbolResolver", func() {
		It("Should resolve bare avg symbol", func(ctx SpecContext) {
			sym := MustSucceed(stlmath.SymbolResolver.Resolve(ctx, "avg"))
			Expect(sym.Name).To(Equal("avg"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})
		It("Should resolve qualified math.avg symbol", func(ctx SpecContext) {
			sym := MustSucceed(stlmath.SymbolResolver.Resolve(ctx, "math.avg"))
			Expect(sym.Name).To(Equal("avg"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})
		It("Should resolve qualified math.min symbol", func(ctx SpecContext) {
			sym := MustSucceed(stlmath.SymbolResolver.Resolve(ctx, "math.min"))
			Expect(sym.Name).To(Equal("min"))
		})
		It("Should resolve qualified math.max symbol", func(ctx SpecContext) {
			sym := MustSucceed(stlmath.SymbolResolver.Resolve(ctx, "math.max"))
			Expect(sym.Name).To(Equal("max"))
		})
		It("Should resolve qualified math.derivative symbol", func(ctx SpecContext) {
			sym := MustSucceed(stlmath.SymbolResolver.Resolve(ctx, "math.derivative"))
			Expect(sym.Name).To(Equal("derivative"))
		})
		It("Should resolve qualified math.pow symbol as internal", func(ctx SpecContext) {
			sym := MustSucceed(stlmath.SymbolResolver.Resolve(ctx, "math.pow"))
			Expect(sym.Name).To(Equal("pow"))
			Expect(sym.Internal).To(BeTrue())
		})
	})

	Describe("Factory", func() {
		It("Should create node for math.avg via CompoundFactory", func(ctx SpecContext) {
			g := makeMathGraph("avg", types.F64())
			analyzed, diagnostics := graph.Analyze(ctx, g, stlmath.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)
			m := MustSucceed(stlmath.NewModule(ctx, nil))
			compound := node.CompoundFactory{m}
			cfg := node.Config{
				Node:    ir.Node{Key: "math", Type: "math.avg"},
				State:   s.Node("math"),
				Program: program.Program{IR: analyzed},
			}
			n := MustSucceed(compound.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should create node for math.derivative via CompoundFactory", func(ctx SpecContext) {
			g := makeMathGraph("derivative", types.F64())
			analyzed, diagnostics := graph.Analyze(ctx, g, stlmath.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)
			m := MustSucceed(stlmath.NewModule(ctx, nil))
			compound := node.CompoundFactory{m}
			cfg := node.Config{
				Node:  ir.Node{Key: "math", Type: "math.derivative"},
				State: s.Node("math"),
			}
			n := MustSucceed(compound.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
	})
})

var _ = Describe("Avg", func() {
	It("Should compute the average of a single batch", func(ctx SpecContext) {
		s := openMath(ctx, "avg", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeTrue())
		expectOutput[float64](s.state, 20.0)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should accumulate a weighted average across batches", func(ctx SpecContext) {
		s := openMath(ctx, "avg", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
		nextChanged(ctx, s.n)
		*s.inputNode.Output(0) = telem.NewSeriesV(40.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(3)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 23.333)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should reset after count threshold", func(ctx SpecContext) {
		s := openMath(ctx, "avg", types.F64(), types.Params{
			{Name: "count", Type: types.I64(), Value: int64(3)},
		})
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 20.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV(40.0, 50.0, 60.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 50.0)
		expectOutputTime(s.state, 6*telem.SecondTS)
	})

	It("Should reset after duration threshold", func(ctx SpecContext) {
		s := openMath(ctx, "avg", types.F64(), types.Params{
			{Name: "duration", Type: types.TimeSpan(), Value: 5 * telem.Second},
		})
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 20.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV(100.0, 200.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(6, 7)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 150.0)
		expectOutputTime(s.state, 7*telem.SecondTS)
	})

	It("Should reset on signal", func(ctx SpecContext) {
		s := openMathWithReset(ctx, "avg", types.F64(), nil)
		resetNode := s.state.Node("reset_signal")
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](0)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 20.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV(100.0, 200.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](1)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 150.0)
		expectOutputTime(s.state, 5*telem.SecondTS)
	})

	It("Should not execute on empty input", func(ctx SpecContext) {
		s := openMath(ctx, "avg", types.F64(), nil)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeFalse())
	})

	It("Should work with int32 type", func(ctx SpecContext) {
		s := openMath(ctx, "avg", types.I32(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](10, 20, 30)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeTrue())
		expectOutput[int32](s.state, 20)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})
})

var _ = Describe("Min", func() {
	It("Should compute the minimum of a single batch", func(ctx SpecContext) {
		s := openMath(ctx, "min", types.I32(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](50, 10, 70)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeTrue())
		expectOutput[int32](s.state, 10)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should maintain minimum across batches", func(ctx SpecContext) {
		s := openMath(ctx, "min", types.I32(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](50, 30)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
		nextChanged(ctx, s.n)

		*s.inputNode.Output(0) = telem.NewSeriesV[int32](40, 60)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(3, 4)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 30)
		expectOutputTime(s.state, 4*telem.SecondTS)
	})

	It("Should not update when new batch has larger values", func(ctx SpecContext) {
		s := openMath(ctx, "min", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(5.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		nextChanged(ctx, s.n)

		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(2, 3)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 5.0)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should reset after duration threshold", func(ctx SpecContext) {
		s := openMath(ctx, "min", types.I32(), types.Params{
			{Name: "duration", Type: types.TimeSpan(), Value: 5 * telem.Second},
		})
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](50, 10, 70)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 10)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV[int32](80, 40, 60)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(6, 7, 8)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 40)
		expectOutputTime(s.state, 8*telem.SecondTS)
	})

	It("Should reset after count threshold", func(ctx SpecContext) {
		s := openMath(ctx, "min", types.F64(), types.Params{
			{Name: "count", Type: types.I64(), Value: int64(3)},
		})
		*s.inputNode.Output(0) = telem.NewSeriesV(5.0, 10.0, 15.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 5.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV(50.0, 40.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 30.0)
		expectOutputTime(s.state, 6*telem.SecondTS)
	})

	It("Should reset on signal", func(ctx SpecContext) {
		s := openMathWithReset(ctx, "min", types.I32(), nil)
		resetNode := s.state.Node("reset_signal")
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](50, 10, 70)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](0)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 10)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV[int32](80, 40, 60)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](1)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 40)
		expectOutputTime(s.state, 6*telem.SecondTS)
	})
})

var _ = Describe("Max", func() {
	It("Should compute the maximum of a single batch", func(ctx SpecContext) {
		s := openMath(ctx, "max", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 50.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeTrue())
		expectOutput[float64](s.state, 50.0)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should maintain maximum across batches", func(ctx SpecContext) {
		s := openMath(ctx, "max", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 50.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
		nextChanged(ctx, s.n)

		*s.inputNode.Output(0) = telem.NewSeriesV(30.0, 20.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(3, 4)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 50.0)
		expectOutputTime(s.state, 4*telem.SecondTS)
	})

	It("Should update when new batch has larger values", func(ctx SpecContext) {
		s := openMath(ctx, "max", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		nextChanged(ctx, s.n)

		*s.inputNode.Output(0) = telem.NewSeriesV(50.0, 100.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(2, 3)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 100.0)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should reset after duration threshold", func(ctx SpecContext) {
		s := openMath(ctx, "max", types.F64(), types.Params{
			{Name: "duration", Type: types.TimeSpan(), Value: 5 * telem.Second},
		})
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 50.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 50.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV(5.0, 15.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(6, 7)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 15.0)
		expectOutputTime(s.state, 7*telem.SecondTS)
	})

	It("Should reset after count threshold", func(ctx SpecContext) {
		s := openMath(ctx, "max", types.I32(), types.Params{
			{Name: "count", Type: types.I64(), Value: int64(2)},
		})
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](10, 50)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 50)
		expectOutputTime(s.state, 2*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV[int32](5, 15)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(3, 4)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 15)
		expectOutputTime(s.state, 4*telem.SecondTS)
	})

	It("Should reset on signal", func(ctx SpecContext) {
		s := openMathWithReset(ctx, "max", types.F64(), nil)
		resetNode := s.state.Node("reset_signal")
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 50.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](0)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 50.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV(25.0, 15.0, 70.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](1)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 70.0)
		expectOutputTime(s.state, 6*telem.SecondTS)
	})

	It("Should work without optional reset signal connected", func(ctx SpecContext) {
		s := openMath(ctx, "max", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 50.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeTrue())
		expectOutput[float64](s.state, 50.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV(25.0, 80.0, 40.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 80.0)
		expectOutputTime(s.state, 6*telem.SecondTS)
	})

	It("Should catch fast reset pulses (1->0 transition)", func(ctx SpecContext) {
		s := openMathWithReset(ctx, "avg", types.I64(), nil)
		resetNode := s.state.Node("reset_signal")

		*s.inputNode.Output(0) = telem.NewSeriesV[int64](10, 20, 30)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](0)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		nextChanged(ctx, s.n)
		expectOutput[int64](s.state, 20)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV[int64](40, 50, 60)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](1, 0)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5)
		nextChanged(ctx, s.n)
		expectOutput[int64](s.state, 50)
		expectOutputTime(s.state, 6*telem.SecondTS)
	})
})

var _ = Describe("Alignment", func() {
	It("Should propagate alignment from input to output", func(ctx SpecContext) {
		s := openMath(ctx, "avg", types.F64(), nil)
		inputSeries := telem.NewSeriesV(10.0, 20.0, 30.0)
		inputSeries.Alignment = 250
		inputSeries.TimeRange = telem.TimeRange{Start: 100 * telem.SecondTS, End: 300 * telem.SecondTS}
		*s.inputNode.Output(0) = inputSeries
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(100, 200, 300)
		nextChanged(ctx, s.n)

		result := *s.state.Node("math").Output(0)
		Expect(result.Alignment).To(Equal(telem.Alignment(250)))
		Expect(result.TimeRange.Start).To(Equal(100 * telem.SecondTS))
		Expect(result.TimeRange.End).To(Equal(300 * telem.SecondTS))

		resultTime := *s.state.Node("math").OutputTime(0)
		Expect(resultTime.Alignment).To(Equal(telem.Alignment(250)))
	})

	It("Should sum alignments when reset signal is connected", func(ctx SpecContext) {
		s := openMathWithReset(ctx, "avg", types.I64(), nil)
		resetNode := s.state.Node("reset_signal")

		inputSeries := telem.NewSeriesV[int64](10, 20, 30)
		inputSeries.Alignment = 100
		inputSeries.TimeRange = telem.TimeRange{Start: 50 * telem.SecondTS, End: 150 * telem.SecondTS}
		*s.inputNode.Output(0) = inputSeries
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(50, 100, 150)

		resetSeries := telem.NewSeriesV[uint8](0)
		resetSeries.Alignment = 75
		resetSeries.TimeRange = telem.TimeRange{Start: 25 * telem.SecondTS, End: 175 * telem.SecondTS}
		*resetNode.Output(0) = resetSeries
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(25)
		nextChanged(ctx, s.n)

		result := *s.state.Node("math").Output(0)
		Expect(result.Alignment).To(Equal(telem.Alignment(175)))
		Expect(result.TimeRange.Start).To(Equal(25 * telem.SecondTS))
		Expect(result.TimeRange.End).To(Equal(175 * telem.SecondTS))

		resultTime := *s.state.Node("math").OutputTime(0)
		Expect(resultTime.Alignment).To(Equal(telem.Alignment(175)))
	})
})

var _ = Describe("Derivative", func() {
	makeDerivGraph := func(dt types.Type) graph.Graph {
		return graph.Graph{
			Nodes: []graph.Node{
				{Key: "input", Type: "input"},
				{Key: "deriv", Type: "derivative"},
			},
			Edges: []graph.Edge{{
				Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "deriv", Param: ir.DefaultInputParam},
			}},
			Functions: []graph.Function{{
				Key:     "input",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: dt}},
			}},
		}
	}

	openDeriv := func(ctx SpecContext, dt types.Type) mathSetup {
		g := makeDerivGraph(dt)
		analyzed, diagnostics := graph.Analyze(ctx, g, stlmath.SymbolResolver)
		Expect(diagnostics.Ok()).To(BeTrue())
		s := node.New(analyzed)
		inputNode := s.Node("input")
		m := MustSucceed(stlmath.NewModule(ctx, nil))
		n := MustSucceed(m.Create(ctx, node.Config{
			Node:  ir.Node{Type: "derivative"},
			State: s.Node("deriv"),
		}))
		return mathSetup{state: s, inputNode: inputNode, n: n}
	}

	expectDerivOutput := func(s *node.ProgramState, values ...float64) {
		result := *s.Node("deriv").Output(0)
		Expect(result.Len()).To(Equal(int64(len(values))))
		vals := telem.UnmarshalSeries[float64](result)
		for i, v := range values {
			Expect(vals[i]).To(BeNumerically("~", v, 0.01))
		}
	}

	It("Should compute pointwise derivative for float64 input", func(ctx SpecContext) {
		s := openDeriv(ctx, types.F64())
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0, 40.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 4)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeTrue())
		expectDerivOutput(s.state, 0.0, 10.0, 10.0)
	})

	It("Should maintain state across batches", func(ctx SpecContext) {
		s := openDeriv(ctx, types.F64())
		*s.inputNode.Output(0) = telem.NewSeriesV(0.0, 10.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
		nextChanged(ctx, s.n)

		*s.inputNode.Output(0) = telem.NewSeriesV(30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4)
		nextChanged(ctx, s.n)
		expectDerivOutput(s.state, 10.0)
	})

	It("Should output zero for the first sample", func(ctx SpecContext) {
		s := openDeriv(ctx, types.F64())
		*s.inputNode.Output(0) = telem.NewSeriesV(5.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeTrue())
		expectDerivOutput(s.state, 0.0)
	})

	It("Should reset state and output zero after reset", func(ctx SpecContext) {
		s := openDeriv(ctx, types.F64())
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
		nextChanged(ctx, s.n)

		s.n.Reset()

		*s.inputNode.Output(0) = telem.NewSeriesV(100.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
		nextChanged(ctx, s.n)
		expectDerivOutput(s.state, 0.0)
	})

	It("Should output zero when timestamps are identical", func(ctx SpecContext) {
		s := openDeriv(ctx, types.F64())
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 1)
		nextChanged(ctx, s.n)
		expectDerivOutput(s.state, 0.0, 0.0)
	})

	It("Should output float64 for int32 input type", func(ctx SpecContext) {
		s := openDeriv(ctx, types.I32())
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](0, 100, 300)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 4)
		nextChanged(ctx, s.n)
		expectDerivOutput(s.state, 0.0, 100.0, 100.0)
	})

	It("Should compute negative derivatives", func(ctx SpecContext) {
		s := openDeriv(ctx, types.F64())
		*s.inputNode.Output(0) = telem.NewSeriesV(100.0, 80.0, 50.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 4)
		nextChanged(ctx, s.n)
		expectDerivOutput(s.state, 0.0, -20.0, -15.0)
	})

	It("Should propagate alignment from input to output", func(ctx SpecContext) {
		s := openDeriv(ctx, types.F64())
		inputSeries := telem.NewSeriesV(10.0, 20.0)
		inputSeries.Alignment = 250
		inputSeries.TimeRange = telem.TimeRange{
			Start: 100 * telem.SecondTS,
			End:   200 * telem.SecondTS,
		}
		*s.inputNode.Output(0) = inputSeries
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(100, 200)
		nextChanged(ctx, s.n)

		result := *s.state.Node("deriv").Output(0)
		Expect(result.Alignment).To(Equal(telem.Alignment(250)))
		Expect(result.TimeRange.Start).To(Equal(100 * telem.SecondTS))
		Expect(result.TimeRange.End).To(Equal(200 * telem.SecondTS))
	})
})
