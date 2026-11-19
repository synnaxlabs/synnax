// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculator_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type benchEnv struct {
	ctx        context.Context
	node       testutil.Node
	closer     io.MultiCloser
	channelSvc *channel.Service
	creator    calctestutil.Creator
}

func newBenchEnv(b *testing.B) *benchEnv {
	b.Helper()
	gomega.RegisterTestingT(b)
	node := testutil.OpenNode(b.Context())
	groupSvc, err := group.OpenService(b.Context(), group.ServiceConfig{
		DB: node.DB,
	})
	if err != nil {
		b.Fatalf("failed to open group service: %v", err)
	}

	labelSvc := MustSucceed(label.OpenService(b.Context(), label.ServiceConfig{
		DB: node.DB,
	}))
	statusSvc := MustSucceed(status.OpenService(b.Context(), status.ServiceConfig{
		DB: node.DB,
	}))
	channelSvc := MustSucceed(channel.OpenService(b.Context(), channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	return &benchEnv{
		ctx:        b.Context(),
		node:       node,
		channelSvc: channelSvc,
		creator:    calctestutil.Creator{DB: node.DB, Channel: channelSvc},
		closer: io.MultiCloser{
			node, groupSvc, channelSvc, statusSvc, labelSvc,
		},
	}
}

func (e *benchEnv) close(b *testing.B) {
	b.Helper()
	if err := e.closer.Close(); err != nil {
		b.Errorf("failed to close env: %v", err)
	}
}

func (e *benchEnv) openCalculator(
	b *testing.B,
	bases []channel.Channel,
	calc calctestutil.Calculation,
) *calculator.Calculator {
	b.Helper()
	if len(bases) > 0 {
		if err := e.node.DB.WithTx(e.ctx, func(tx gorp.Tx) error {
			return e.channelSvc.NewWriter(tx).CreateMany(e.ctx, &bases)
		}); err != nil {
			b.Fatalf("failed to create base channels: %v", err)
		}
	}
	out, err := e.creator.Create(e.ctx, calc)
	if err != nil {
		b.Fatalf("failed to create calc channel: %v", err)
	}
	mod, err := compiler.Compile(e.ctx, e.node.DB, compiler.Config{
		ChannelService: e.channelSvc,
		Channel:        out,
	})
	if err != nil {
		b.Fatalf("failed to compile calculator: %v", err)
	}
	c, err := calculator.Open(e.ctx, calculator.Config{Module: mod})
	if err != nil {
		b.Fatalf("failed to open calculator: %v", err)
	}
	return c
}

// advanceAlignment moves every series in f past its own upper bound so the next call to
// the calculator sees fresh data instead of skipping it as already consumed. It runs
// inside the timed loop: a few series updates cost far less than a timer restart.
func advanceAlignment(f frame.Frame) {
	for i, ser := range f.RawSeries() {
		ser.Alignment = ser.AlignmentBounds().Upper
		f.SetRawSeriesAt(i, ser)
	}
}

func BenchmarkCalculator_SingleInput(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close(b)

	base := []channel.Channel{{Name: "input", DataType: telem.Int64T, Virtual: true}}
	calc := calctestutil.Calculation{
		Name:       "output",
		DataType:   telem.Int64T,
		Expression: "return input",
	}

	c := env.openCalculator(b, base, calc)
	defer func() {
		if err := c.Close(); err != nil {
			b.Errorf("failed to close calculator: %v", err)
		}
	}()

	inputFrame := frame.NewUnary(base[0].Key(), telem.NewSeriesV[int64](10, 20, 30))
	outputFrame := frame.Frame{}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		advanceAlignment(inputFrame)
		if _, _, err := c.Next(env.ctx, inputFrame, outputFrame); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkCalculator_TwoInputs_Add(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close(b)

	bases := []channel.Channel{
		{Name: "a", DataType: telem.Float32T, Virtual: true},
		{Name: "b", DataType: telem.Float32T, Virtual: true},
	}
	calc := calctestutil.Calculation{
		Name:       "sum",
		DataType:   telem.Float32T,
		Expression: "return a + b",
	}

	c := env.openCalculator(b, bases, calc)
	defer func() {
		if err := c.Close(); err != nil {
			b.Errorf("failed to close calculator: %v", err)
		}
	}()

	inputFrame := frame.NewMulti(
		[]channel.Key{bases[0].Key(), bases[1].Key()},
		[]telem.Series{
			telem.NewSeriesV[float32](1.0, 2.0, 3.0),
			telem.NewSeriesV[float32](4.0, 5.0, 6.0),
		},
	)
	outputFrame := frame.Frame{}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		advanceAlignment(inputFrame)
		if _, _, err := c.Next(env.ctx, inputFrame, outputFrame); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkCalculator_MultipleInputs(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close(b)

	bases := []channel.Channel{
		{Name: "w", DataType: telem.Float64T, Virtual: true},
		{Name: "x", DataType: telem.Float64T, Virtual: true},
		{Name: "y", DataType: telem.Float64T, Virtual: true},
		{Name: "z", DataType: telem.Float64T, Virtual: true},
	}
	calc := calctestutil.Calculation{
		Name:       "result",
		DataType:   telem.Float64T,
		Expression: "return w + x + y + z",
	}

	c := env.openCalculator(b, bases, calc)
	defer func() {
		if err := c.Close(); err != nil {
			b.Errorf("failed to close calculator: %v", err)
		}
	}()

	inputFrame := frame.NewMulti(
		[]channel.Key{bases[0].Key(), bases[1].Key(), bases[2].Key(), bases[3].Key()},
		[]telem.Series{
			telem.NewSeriesV(1.0, 2.0, 3.0),
			telem.NewSeriesV(4.0, 5.0, 6.0),
			telem.NewSeriesV(7.0, 8.0, 9.0),
			telem.NewSeriesV(10.0, 11.0, 12.0),
		},
	)
	outputFrame := frame.Frame{}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		advanceAlignment(inputFrame)
		if _, _, err := c.Next(env.ctx, inputFrame, outputFrame); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkCalculator_NestedTwoLevel(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close(b)

	base := []channel.Channel{{Name: "base", DataType: telem.Int64T, Virtual: true}}
	calc1 := calctestutil.Calculation{
		Name:       "calc1",
		DataType:   telem.Int64T,
		Expression: "return base + 1",
	}
	c1 := env.openCalculator(b, base, calc1)

	calc2 := calctestutil.Calculation{
		Name:       "calc2",
		DataType:   telem.Int64T,
		Expression: "return calc1 * 2",
	}
	c2 := env.openCalculator(b, nil, calc2)

	group := calculator.Group{c1, c2}
	defer func() {
		if err := group.Close(); err != nil {
			b.Errorf("failed to close calculator group: %v", err)
		}
	}()

	inputFrame := frame.NewUnary(base[0].Key(), telem.NewSeriesV[int64](10, 20, 30))

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		advanceAlignment(inputFrame)
		_, _ = group.Next(env.ctx, inputFrame, saveReport)
	}
	b.StopTimer()
	b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkCalculator_SampleCount(b *testing.B) {
	for _, count := range []int{100, 1000, 10000, 100000} {
		b.Run(fmt.Sprintf("samples=%d", count), func(b *testing.B) {
			env := newBenchEnv(b)
			defer env.close(b)

			bases := []channel.Channel{
				{Name: "a", DataType: telem.Float64T, Virtual: true},
				{Name: "b", DataType: telem.Float64T, Virtual: true},
			}
			calc := calctestutil.Calculation{
				Name:       "result",
				DataType:   telem.Float64T,
				Expression: "return a * b",
			}

			c := env.openCalculator(b, bases, calc)
			defer func() {
				if err := c.Close(); err != nil {
					b.Errorf("failed to close calculator: %v", err)
				}
			}()

			aData := make([]float64, count)
			bData := make([]float64, count)
			for i := range count {
				aData[i] = float64(i)
				bData[i] = float64(i + 1)
			}

			inputFrame := frame.NewMulti(
				[]channel.Key{bases[0].Key(), bases[1].Key()},
				[]telem.Series{telem.NewSeriesV(aData...), telem.NewSeriesV(bData...)},
			)
			outputFrame := frame.Frame{}

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				advanceAlignment(inputFrame)
				if _, _, err := c.Next(env.ctx, inputFrame, outputFrame); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(count*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}

func BenchmarkCalculator_ComplexExpression(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close(b)

	bases := []channel.Channel{
		{Name: "a", DataType: telem.Float64T, Virtual: true},
		{Name: "b", DataType: telem.Float64T, Virtual: true},
	}
	calc := calctestutil.Calculation{
		Name:       "result",
		DataType:   telem.Float64T,
		Expression: "if a > b { return a * 2 } else { return b * 3 }",
	}

	c := env.openCalculator(b, bases, calc)
	defer func() {
		if err := c.Close(); err != nil {
			b.Errorf("failed to close calculator: %v", err)
		}
	}()

	inputFrame := frame.NewMulti(
		[]channel.Key{bases[0].Key(), bases[1].Key()},
		[]telem.Series{
			telem.NewSeriesV(10.0, 5.0, 15.0),
			telem.NewSeriesV(5.0, 10.0, 10.0),
		},
	)
	outputFrame := frame.Frame{}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		advanceAlignment(inputFrame)
		if _, _, err := c.Next(env.ctx, inputFrame, outputFrame); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkCalculator_GroupScaling(b *testing.B) {
	for _, size := range []int{1, 3, 5} {
		b.Run(fmt.Sprintf("calculators=%d", size), func(b *testing.B) {
			env := newBenchEnv(b)
			defer env.close(b)

			base := []channel.Channel{
				{Name: "base", DataType: telem.Int64T, Virtual: true},
			}

			group := make(calculator.Group, 0, size)
			prevName := "base"
			for i := range size {
				calcCh := calctestutil.Calculation{
					Name:       fmt.Sprintf("calc_%d", i),
					DataType:   telem.Int64T,
					Expression: fmt.Sprintf("return %s + 1", prevName),
				}
				var bases []channel.Channel
				if i == 0 {
					bases = base
				}
				c := env.openCalculator(b, bases, calcCh)
				group = append(group, c)
				prevName = calcCh.Name
			}
			defer func() {
				if err := group.Close(); err != nil {
					b.Errorf("failed to close calculator group: %v", err)
				}
			}()

			inputFrame := frame.NewUnary(
				base[0].Key(),
				telem.NewSeriesV[int64](10, 20, 30),
			)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				advanceAlignment(inputFrame)
				_, _ = group.Next(env.ctx, inputFrame, saveReport)
			}
			b.StopTimer()
			b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}

// clockFrame returns a frame of n float32 samples for each data channel and the stamps
// of their index, one period apart.
func clockFrame(
	index channel.Key,
	data []channel.Key,
	n int,
	period telem.TimeSpan,
) frame.Frame {
	stamps := make([]telem.TimeStamp, n)
	for i := range stamps {
		stamps[i] = telem.SecondTS + telem.TimeStamp(i)*telem.TimeStamp(period)
	}
	keys := append([]channel.Key{index}, data...)
	series := make([]telem.Series, len(keys))
	series[0] = telem.NewSeries(stamps)
	for i := range data {
		series[i+1] = telem.NewSeries(make([]float32, n))
	}
	return frame.NewMulti(keys, series)
}

// advanceClock moves the stamps of f forward by span and its series past their own
// alignment bounds, so f reads as the next frame of its clock.
func advanceClock(f frame.Frame, span telem.TimeSpan) {
	stamps := f.RawSeriesAt(0)
	for i := range int(stamps.Len()) {
		stamps.SetValueAt(i, stamps.ValueAt[telem.TimeStamp](i)+telem.TimeStamp(span))
	}
	advanceAlignment(f)
}

// BenchmarkCalculator_Clocks steps the sum of two persisted inputs, a and b, with one
// frame of a per iteration at 100 frames per second. With one clock, b shares the index
// of a. With two clocks, b has its own index at 1 Hz, so one sample of b arrives before
// every 100th frame of a.
func BenchmarkCalculator_Clocks(b *testing.B) {
	for _, c := range []struct {
		name string
		// samples is the number of samples of a in each frame.
		samples int
		clocks  int
	}{
		{name: "clocks=1/rate=1kHz", samples: 10, clocks: 1},
		{name: "clocks=2/rates=1kHz+1Hz", samples: 10, clocks: 2},
		{name: "clocks=1/rate=10kHz", samples: 100, clocks: 1},
	} {
		b.Run(c.name, func(b *testing.B) { benchmarkClocks(b, c.samples, c.clocks) })
	}
}

func benchmarkClocks(b *testing.B, samples, clocks int) {
	b.Helper()
	const framesPerSecond = 100
	env := newBenchEnv(b)
	defer env.close(b)
	indexes, bases, calc := env.openClockCalculator(b, clocks)
	defer func() {
		if err := calc.Close(); err != nil {
			b.Errorf("failed to close calculator: %v", err)
		}
	}()
	onFast := []channel.Key{bases[0].Key(), bases[1].Key()}
	var slow frame.Frame
	if clocks == 2 {
		onFast = onFast[:1]
		slow = clockFrame(
			indexes[1].Key(),
			[]channel.Key{bases[1].Key()},
			1,
			telem.Second,
		)
	}
	period := telem.Second / telem.TimeSpan(framesPerSecond*samples)
	fast := clockFrame(indexes[0].Key(), onFast, samples, period)
	output := calc.Channel().Key()
	outputs := 0
	next := func(f frame.Frame, span telem.TimeSpan) {
		advanceClock(f, span)
		out, changed, err := calc.Next(env.ctx, f, frame.Frame{})
		if err != nil {
			b.Fatal(err)
		}
		if !changed {
			return
		}
		for key, series := range out.Entries() {
			if key == output {
				outputs += int(series.Len())
			}
		}
	}
	b.ReportAllocs()
	frames := 0
	for b.Loop() {
		if clocks == 2 && frames%framesPerSecond == 0 {
			next(slow, telem.Second)
		}
		next(fast, telem.Second/framesPerSecond)
		frames++
	}
	if outputs == 0 {
		b.Fatal("the calculator made no output")
	}
	b.ReportMetric(float64(samples*frames)/b.Elapsed().Seconds(), "samples/sec")
	b.ReportMetric(float64(outputs)/float64(frames), "outputs/op")
}

// openClockCalculator creates the given number of index channels, the inputs a and b,
// and a calculator for their sum. Input a is on the first index and b on the last.
func (e *benchEnv) openClockCalculator(
	b *testing.B,
	clocks int,
) (indexes, bases []channel.Channel, calc *calculator.Calculator) {
	b.Helper()
	indexes = make([]channel.Channel, clocks)
	for i := range indexes {
		indexes[i] = channel.Channel{
			Name:     fmt.Sprintf("time_%d", i),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
	}
	if err := e.node.DB.WithTx(e.ctx, func(tx gorp.Tx) error {
		return e.channelSvc.NewWriter(tx).CreateMany(e.ctx, &indexes)
	}); err != nil {
		b.Fatalf("failed to create index channels: %v", err)
	}
	bases = []channel.Channel{
		{Name: "a", DataType: telem.Float32T, LocalIndex: indexes[0].LocalKey},
		{Name: "b", DataType: telem.Float32T, LocalIndex: indexes[clocks-1].LocalKey},
	}
	sum := calctestutil.Calculation{
		Name:       "sum",
		DataType:   telem.Float32T,
		Expression: "return a + b",
	}
	return indexes, bases, e.openCalculator(b, bases, sum)
}
