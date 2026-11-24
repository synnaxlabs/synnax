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
	stdmath "math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/runtime/node"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// breakerEpoch opens nodeType from an epoch-windowed host with a window of duration.
func breakerEpoch(
	ctx SpecContext,
	nodeType string,
	dt types.Type,
	duration telem.TimeSpan,
	count int64,
) mathSetup {
	GinkgoHelper()
	params := types.Params{
		{Name: "duration", Type: types.TimeSpan(), Value: duration},
	}
	if count > 0 {
		params = append(
			params,
			types.Param{Name: "count", Type: types.I64(), Value: count},
		)
	}
	return openMathWindowing(ctx, stlmath.WindowFromEpoch, nodeType, dt, params)
}

// breakerEpochWithReset opens nodeType with a reset input and a window of duration from
// an epoch-windowed host.
func breakerEpochWithReset(
	ctx SpecContext,
	windowing stlmath.Windowing,
	nodeType string,
	dt types.Type,
	duration telem.TimeSpan,
) mathSetup {
	GinkgoHelper()
	g := makeMathGraphWithReset(nodeType, dt, types.Bool())
	inputs := types.Params{{Name: "duration", Type: types.TimeSpan(), Value: duration}}
	g.Inputs["math"] = nodeConfig(nodeType, inputs)
	analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	s := node.New(analyzed)
	m := MustSucceed(stlmath.NewHost(ctx, nil, windowing))
	n := MustSucceed(m.Create(node.Config{
		Node:    ir.Node{Key: "math", Type: nodeType, Inputs: inputs},
		State:   s.Node("math"),
		Program: program.Program{IR: analyzed},
	}))
	return mathSetup{state: s, inputNode: s.Node("input"), n: n}
}

func breakerPush[T telem.Sample](
	s mathSetup,
	alignment telem.Alignment,
	stamps []telem.TimeStamp,
	values ...T,
) {
	d := telem.NewSeriesV(values...)
	d.Alignment = alignment
	t := telem.NewSeries(stamps)
	t.Alignment = alignment
	*s.inputNode.Output(0) = d
	*s.inputNode.OutputTime(0) = t
	s.inputNode.MarkFresh(0)
}

func ns(v ...int64) []telem.TimeStamp {
	out := make([]telem.TimeStamp, len(v))
	for i, x := range v {
		out[i] = telem.TimeStamp(x)
	}
	return out
}

func secs(v ...float64) []telem.TimeStamp {
	out := make([]telem.TimeStamp, len(v))
	for i, x := range v {
		out[i] = telem.TimeStamp(x * float64(telem.Second))
	}
	return out
}

var _ = Describe("Breaker epoch windows", func() {
	It("Should put a stamp on a boundary in the next window", func(ctx SpecContext) {
		s := breakerEpoch(ctx, "avg", types.F64(), 5*telem.Second, 0)
		breakerPush(s, 0, ns(
			int64(5*telem.Second)-1,
			int64(5*telem.Second),
			int64(10*telem.Second)-1,
			int64(10*telem.Second),
		), 1.0, 10.0, 20.0, 100.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 1.0, 15.0, 100.0)
		expectOutputTime(
			s.state,
			5*telem.SecondTS-1,
			10*telem.SecondTS-1,
			10*telem.SecondTS,
		)
	})

	It("Should emit one sample per window for a frame with many windows", func(
		ctx SpecContext,
	) {
		s := breakerEpoch(ctx, "avg", types.F64(), telem.Second, 0)
		vals := make([]float64, 100)
		stamps := make([]float64, 100)
		for i := range vals {
			vals[i] = float64(i)
			stamps[i] = float64(i)
		}
		breakerPush(s, telem.NewAlignment(3, 7), secs(stamps...), vals...)
		nextChanged(ctx, s.n)
		expectOutput(s.state, vals...)
		out := *s.state.Node("math").Output(0)
		Expect(out.Alignment).To(Equal(telem.NewAlignment(3, 7)))
	})

	It("Should carry one window over several frames", func(ctx SpecContext) {
		s := breakerEpoch(ctx, "avg", types.F64(), 6*telem.Second, 0)
		breakerPush(s, 0, secs(1), 10.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 10.0)
		breakerPush(s, 1, secs(2), 20.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 15.0)
		breakerPush(s, 2, secs(3, 7), 30.0, 70.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 20.0, 70.0)
		expectOutputTime(s.state, 3*telem.SecondTS, 7*telem.SecondTS)
	})

	It("Should skip an empty window between samples", func(ctx SpecContext) {
		s := breakerEpoch(ctx, "avg", types.F64(), 6*telem.Second, 0)
		breakerPush(s, 0, secs(1), 10.0)
		nextChanged(ctx, s.n)
		breakerPush(s, 1, secs(13), 30.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 30.0)
		breakerPush(s, 2, secs(14, 25, 31), 50.0, 7.0, 9.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 40.0, 7.0, 9.0)
	})

	It("Should window a duration that does not divide a second", func(ctx SpecContext) {
		s := breakerEpoch(ctx, "avg", types.F64(), 333*telem.Millisecond, 0)
		ms := func(v ...int64) []telem.TimeStamp {
			out := make([]telem.TimeStamp, len(v))
			for i, x := range v {
				out[i] = telem.TimeStamp(x) * telem.MillisecondTS
			}
			return out
		}
		// Windows 0, 1, 1, 2, 2, 3; then 1000 ms is in window 3 and 1332 ms starts 4.
		breakerPush(
			s,
			0,
			ms(332, 333, 665, 666, 998, 999),
			1.0,
			2.0,
			4.0,
			8.0,
			16.0,
			32.0,
		)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 1.0, 3.0, 12.0, 32.0)
		breakerPush(s, 6, ms(1000, 1332, 1333), 64.0, 128.0, 256.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 48.0, 192.0)
	})

	It("Should window a 7 ns duration", func(ctx SpecContext) {
		s := breakerEpoch(ctx, "avg", types.I64(), 7*telem.Nanosecond, 0)
		breakerPush[int64](s, 0, ns(6, 7, 13, 14, 20, 21), 1, 3, 5, 7, 9, 11)
		nextChanged(ctx, s.n)
		expectOutput[int64](s.state, 1, 4, 8, 11)
	})

	It("Should keep every stamp in one window for the largest duration", func(
		ctx SpecContext,
	) {
		s := breakerEpoch(
			ctx,
			"avg",
			types.F64(),
			telem.TimeSpan(stdmath.MaxInt64),
			0,
		)
		breakerPush(s, 0, ns(0, 1, int64(stdmath.MaxInt64)-1), 1.0, 2.0, 3.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 2.0)
		breakerPush(s, 3, ns(int64(stdmath.MaxInt64)), 9.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 9.0)
	})

	DescribeTable("min and max windows", func(
		ctx SpecContext,
		nodeType string,
		first, second []float64,
	) {
		s := breakerEpoch(ctx, nodeType, types.F64(), 5*telem.Second, 0)
		breakerPush(s, 0, secs(3, 4, 6, 7, 11), 10.0, 20.0, 100.0, 5.0, 7.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, first...)
		expectOutputTime(s.state, 4*telem.SecondTS, 7*telem.SecondTS, 11*telem.SecondTS)
		breakerPush(s, 5, secs(12, 16), 1.0, 2.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, second...)
	},
		Entry("max", "max", []float64{20, 100, 7}, []float64{7, 2}),
		Entry("min", "min", []float64{10, 5, 7}, []float64{1, 2}),
	)

	DescribeTable(
		"integer types",
		func(ctx SpecContext, nodeType string, want []int32) {
			s := breakerEpoch(ctx, nodeType, types.I32(), 2*telem.Second, 0)
			breakerPush[int32](s, 0, secs(0, 1, 2, 3, 4), -1, -4, 9, 3, 6)
			nextChanged(ctx, s.n)
			expectOutput(s.state, want...)
		},
		Entry("avg", "avg", []int32{-2, 6, 6}),
		Entry("min", "min", []int32{-4, 3, 6}),
		Entry("max", "max", []int32{-1, 9, 6}),
	)

	It("Should keep a count reset inside epoch windows", func(ctx SpecContext) {
		s := breakerEpoch(ctx, "avg", types.F64(), 10*telem.Second, 2)
		breakerPush(s, 0, secs(1, 2), 10.0, 20.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 15.0)
		breakerPush(s, 2, secs(3), 30.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 30.0)
		breakerPush(s, 3, secs(4, 11), 40.0, 110.0)
		nextChanged(ctx, s.n)
		expectOutput(s.state, 35.0, 110.0)
	})

	It("Should run a count alone the same in both windowings", func(ctx SpecContext) {
		params := types.Params{{Name: "count", Type: types.I64(), Value: int64(3)}}
		epoch := openMathWindowing(
			ctx,
			stlmath.WindowFromEpoch,
			"avg",
			types.F64(),
			params,
		)
		first := openMathWindowing(
			ctx,
			stlmath.WindowFromFirst,
			"avg",
			types.F64(),
			params,
		)
		frames := [][]float64{{1, 2}, {3, 4}, {5}, {6, 7, 8}, {9}}
		a := telem.Alignment(0)
		for i, f := range frames {
			st := make([]float64, len(f))
			for j := range f {
				st[j] = float64(i*10 + j)
			}
			breakerPush(epoch, a, secs(st...), f...)
			breakerPush(first, a, secs(st...), f...)
			nextChanged(ctx, epoch.n)
			nextChanged(ctx, first.n)
			Expect(epoch.state.Node("math").Output(0).Unmarshal[float64]()).To(
				Equal(first.state.Node("math").Output(0).Unmarshal[float64]()),
			)
			a += telem.Alignment(len(f))
		}
	})

	It("Should run an untimed input as one window", func(ctx SpecContext) {
		s := breakerEpoch(ctx, "avg", types.F64(), telem.Second, 0)
		*s.inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0)
		*s.inputNode.OutputTime(0) = telem.Series{DataType: telem.TimestampT}
		s.inputNode.MarkFresh(0)
		changed := nextChanged(ctx, s.n)
		Expect(changed.Contains(0)).To(BeTrue())
		expectOutput(s.state, 15.0)
		Expect(s.state.Node("math").OutputTime(0).Len()).To(Equal(int64(1)))
	})

	It("Should give window outputs alignments that never overlap across frames", func(
		ctx SpecContext,
	) {
		s := breakerEpoch(ctx, "max", types.F64(), 2*telem.Second, 0)
		const frames = 6
		bounds := make([]telem.AlignmentBounds, 0, frames)
		a := telem.NewAlignment(4, 0)
		for f := range frames {
			st := []float64{float64(f*3) + 0, float64(f*3) + 1, float64(f*3) + 2}
			breakerPush(s, a, secs(st...), 1.0, 2.0, 3.0)
			nextChanged(ctx, s.n)
			out := *s.state.Node("math").Output(0)
			outTime := *s.state.Node("math").OutputTime(0)
			Expect(outTime.Alignment).To(Equal(out.Alignment))
			Expect(out.Len()).To(Equal(outTime.Len()))
			bounds = append(bounds, out.AlignmentBounds())
			a = a.AddSamples(3)
		}
		for i := 1; i < len(bounds); i++ {
			Expect(bounds[i].Lower).To(BeNumerically(">=", bounds[i-1].Upper))
		}
	})

	DescribeTable("Should not give two different window values one alignment", func(
		ctx SpecContext,
		windowing stlmath.Windowing,
	) {
		s := breakerEpochWithReset(ctx, windowing, "avg", types.F64(), 10*telem.Second)
		resetNode := s.state.Node("reset_signal")
		type sample struct {
			alignment telem.Alignment
			stamp     telem.TimeStamp
			value     float64
		}
		seen := map[telem.Alignment]sample{}
		record := func() {
			out := *s.state.Node("math").Output(0)
			outTime := *s.state.Node("math").OutputTime(0)
			for i := range out.Len() {
				smp := sample{
					alignment: out.Alignment.AddSamples(uint32(i)),
					stamp:     outTime.ValueAt[telem.TimeStamp](int(i)),
					value:     out.ValueAt[float64](int(i)),
				}
				if prev, ok := seen[smp.alignment]; ok {
					Expect(prev).To(Equal(smp), "two window values at one alignment")
				}
				seen[smp.alignment] = smp
			}
		}
		breakerPush(s, telem.NewAlignment(1, 0), secs(1), 10.0)
		nextChanged(ctx, s.n)
		record()
		breakerPush(s, telem.NewAlignment(1, 1), secs(2), 30.0)
		nextChanged(ctx, s.n)
		record()
		reset := telem.NewSeriesV(true)
		reset.Alignment = telem.NewAlignment(2, 0)
		*resetNode.Output(0) = reset
		*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(3)
		resetNode.MarkFresh(0)
		if nextChanged(ctx, s.n).Contains(0) {
			record()
		}
	},
		Entry("epoch", stlmath.WindowFromEpoch),
		Entry("first", stlmath.WindowFromFirst),
	)
})
