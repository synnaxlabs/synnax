// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stat_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/stat"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func makeStatGraph(nodeType string, dt types.Type) graph.Graph {
	return graph.Graph{
		Nodes: []graph.Node{
			{Key: "input", Type: "input"},
			{Key: "stat", Type: nodeType},
		},
		Edges: []graph.Edge{{
			Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "stat", Param: ir.DefaultInputParam},
		}},
		Functions: []graph.Function{{
			Key:     "input",
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: dt}},
		}},
	}
}

func makeStatGraphWithReset(nodeType string, dt types.Type) graph.Graph {
	return graph.Graph{
		Nodes: []graph.Node{
			{Key: "input", Type: "input"},
			{Key: "reset_signal", Type: "reset_signal"},
			{Key: "stat", Type: nodeType},
		},
		Edges: []graph.Edge{
			{
				Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "stat", Param: ir.DefaultInputParam},
			},
			{
				Source: ir.Handle{Node: "reset_signal", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "stat", Param: "reset"},
			},
		},
		Functions: []graph.Function{
			{Key: "input", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: dt}}},
			{Key: "reset_signal", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}},
		},
	}
}

type statSetup struct {
	state     *node.ProgramState
	inputNode *node.State
	n         node.Node
}

func openStat(
	ctx SpecContext,
	nodeType string,
	dt types.Type,
	config types.Params,
) statSetup {
	g := makeStatGraph(nodeType, dt)
	analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	s := node.New(analyzed)
	inputNode := s.Node("input")
	m := &stat.Module{}
	n := MustSucceed(m.Create(ctx, node.Config{
		Node:    ir.Node{Key: "stat", Type: nodeType, Config: config},
		State:   s.Node("stat"),
		Program: program.Program{IR: analyzed},
	}))
	return statSetup{state: s, inputNode: inputNode, n: n}
}

func openStatWithReset(
	ctx SpecContext,
	nodeType string,
	dt types.Type,
	config types.Params,
) statSetup {
	g := makeStatGraphWithReset(nodeType, dt)
	analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	s := node.New(analyzed)
	inputNode := s.Node("input")
	m := &stat.Module{}
	n := MustSucceed(m.Create(ctx, node.Config{
		Node:    ir.Node{Key: "stat", Type: nodeType, Config: config},
		State:   s.Node("stat"),
		Program: program.Program{IR: analyzed},
	}))
	return statSetup{state: s, inputNode: inputNode, n: n}
}

func nextChanged(ctx SpecContext, n node.Node) set.Set[string] {
	changed := make(set.Set[string])
	n.Next(node.Context{Context: ctx, MarkChanged: func(i int) { changed.Add(n.Outputs()[i]) }})
	return changed
}

func expectOutput[T telem.NumericSample](s *node.ProgramState, values ...T) {
	result := *s.Node("stat").Output(0)
	Expect(result.Len()).To(Equal(int64(len(values))))
	vals := telem.UnmarshalSeries[T](result)
	for i, v := range values {
		Expect(vals[i]).To(BeNumerically("~", v, 0.01))
	}
}

func expectOutputTime(s *node.ProgramState, timestamps ...telem.TimeStamp) {
	result := *s.Node("stat").OutputTime(0)
	Expect(result.Len()).To(Equal(int64(len(timestamps))))
	vals := telem.UnmarshalSeries[telem.TimeStamp](result)
	for i, ts := range timestamps {
		Expect(vals[i]).To(Equal(ts))
	}
}

var _ = Describe("Avg", func() {
	It("Should compute the average of a single batch", func(ctx SpecContext) {
		s := openStat(ctx, "avg", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
		expectOutput[float64](s.state, 20.0)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should accumulate a weighted average across batches", func(ctx SpecContext) {
		s := openStat(ctx, "avg", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
		nextChanged(ctx, s.n)
		// avg so far: 15.0, count: 2
		*s.inputNode.Output(0) = telem.NewSeriesV(40.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(3)
		nextChanged(ctx, s.n)
		// weighted: (15*2 + 40) / 3 = 23.33
		expectOutput[float64](s.state, 23.333)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should reset after count threshold", func(ctx SpecContext) {
		s := openStat(ctx, "avg", types.F64(), types.Params{
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
		s := openStat(ctx, "avg", types.F64(), types.Params{
			{Name: "duration", Type: types.TimeSpan(), Value: 5 * telem.Second},
		})
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 20.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		// 6s - 1s = 5s >= 5s, triggers reset
		*s.inputNode.Output(0) = telem.NewSeriesV(100.0, 200.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(6, 7)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 150.0)
		expectOutputTime(s.state, 7*telem.SecondTS)
	})

	It("Should reset on signal", func(ctx SpecContext) {
		s := openStatWithReset(ctx, "avg", types.F64(), nil)
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
		s := openStat(ctx, "avg", types.F64(), nil)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeFalse())
	})

	It("Should work with int32 type", func(ctx SpecContext) {
		s := openStat(ctx, "avg", types.I32(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](10, 20, 30)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
		expectOutput[int32](s.state, 20)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})
})

var _ = Describe("Min", func() {
	It("Should compute the minimum of a single batch", func(ctx SpecContext) {
		s := openStat(ctx, "min", types.I32(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](50, 10, 70)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
		expectOutput[int32](s.state, 10)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should maintain minimum across batches", func(ctx SpecContext) {
		s := openStat(ctx, "min", types.I32(), nil)
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
		s := openStat(ctx, "min", types.F64(), nil)
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
		s := openStat(ctx, "min", types.I32(), types.Params{
			{Name: "duration", Type: types.TimeSpan(), Value: 5 * telem.Second},
		})
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](50, 10, 70)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 10)
		expectOutputTime(s.state, 3*telem.SecondTS)

		// 6s - 1s = 5s >= 5s, triggers reset
		*s.inputNode.Output(0) = telem.NewSeriesV[int32](80, 40, 60)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(6, 7, 8)
		nextChanged(ctx, s.n)
		expectOutput[int32](s.state, 40)
		expectOutputTime(s.state, 8*telem.SecondTS)
	})

	It("Should reset after count threshold", func(ctx SpecContext) {
		s := openStat(ctx, "min", types.F64(), types.Params{
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
		s := openStatWithReset(ctx, "min", types.I32(), nil)
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
		s := openStat(ctx, "max", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 50.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
		expectOutput[float64](s.state, 50.0)
		expectOutputTime(s.state, 3*telem.SecondTS)
	})

	It("Should maintain maximum across batches", func(ctx SpecContext) {
		s := openStat(ctx, "max", types.F64(), nil)
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
		s := openStat(ctx, "max", types.F64(), nil)
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
		s := openStat(ctx, "max", types.F64(), types.Params{
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
		s := openStat(ctx, "max", types.I32(), types.Params{
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
		s := openStatWithReset(ctx, "max", types.F64(), nil)
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
		s := openStat(ctx, "max", types.F64(), nil)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 50.0, 30.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
		expectOutput[float64](s.state, 50.0)
		expectOutputTime(s.state, 3*telem.SecondTS)

		*s.inputNode.Output(0) = telem.NewSeriesV(25.0, 80.0, 40.0)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
		nextChanged(ctx, s.n)
		expectOutput[float64](s.state, 80.0)
		expectOutputTime(s.state, 6*telem.SecondTS)
	})

	It("Should catch fast reset pulses (1->0 transition)", func(ctx SpecContext) {
		s := openStatWithReset(ctx, "avg", types.I64(), nil)
		resetNode := s.state.Node("reset_signal")

		*s.inputNode.Output(0) = telem.NewSeriesV[int64](10, 20, 30)
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
		*resetNode.Output(0) = telem.NewSeriesV[uint8](0)
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		nextChanged(ctx, s.n)
		expectOutput[int64](s.state, 20)
		expectOutputTime(s.state, 3*telem.SecondTS)

		// Reset pulse: 1 at time 4, 0 at time 5
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
		s := openStat(ctx, "avg", types.F64(), nil)
		inputSeries := telem.NewSeriesV(10.0, 20.0, 30.0)
		inputSeries.Alignment = 250
		inputSeries.TimeRange = telem.TimeRange{Start: 100 * telem.SecondTS, End: 300 * telem.SecondTS}
		*s.inputNode.Output(0) = inputSeries
		*s.inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(100, 200, 300)
		nextChanged(ctx, s.n)

		result := *s.state.Node("stat").Output(0)
		Expect(result.Alignment).To(Equal(telem.Alignment(250)))
		Expect(result.TimeRange.Start).To(Equal(100 * telem.SecondTS))
		Expect(result.TimeRange.End).To(Equal(300 * telem.SecondTS))

		resultTime := *s.state.Node("stat").OutputTime(0)
		Expect(resultTime.Alignment).To(Equal(telem.Alignment(250)))
	})

	It("Should sum alignments when reset signal is connected", func(ctx SpecContext) {
		s := openStatWithReset(ctx, "avg", types.I64(), nil)
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

		result := *s.state.Node("stat").Output(0)
		Expect(result.Alignment).To(Equal(telem.Alignment(175)))
		Expect(result.TimeRange.Start).To(Equal(25 * telem.SecondTS))
		Expect(result.TimeRange.End).To(Equal(175 * telem.SecondTS))

		resultTime := *s.state.Node("stat").OutputTime(0)
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

	openDeriv := func(ctx SpecContext, dt types.Type) statSetup {
		g := makeDerivGraph(dt)
		analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
		Expect(diagnostics.Ok()).To(BeTrue())
		s := node.New(analyzed)
		inputNode := s.Node("input")
		m := &stat.Module{}
		n := MustSucceed(m.Create(ctx, node.Config{
			Node:  ir.Node{Type: "derivative"},
			State: s.Node("deriv"),
		}))
		return statSetup{state: s, inputNode: inputNode, n: n}
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
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
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
		Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
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
