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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/compiler"
	"github.com/synnaxlabs/x/telem"
)

type benchEnv struct {
	ctx    context.Context
	dist   mock.Node
	arcSvc *arc.Service
}

func newBenchEnv(b *testing.B) *benchEnv {
	gomega.RegisterTestingT(b)
	ctx := context.Background()
	distB := mock.NewCluster()
	dist := distB.Provision(ctx)

	arcSvc, err := arc.OpenService(ctx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Signals:  dist.Signals,
	})
	if err != nil {
		b.Fatalf("failed to open arc service: %v", err)
	}

	return &benchEnv{ctx: ctx, dist: dist, arcSvc: arcSvc}
}

func (e *benchEnv) close(b *testing.B) {
	if err := e.dist.Close(); err != nil {
		b.Errorf("failed to close distribution: %v", err)
	}
}

func (e *benchEnv) openCalculator(
	b *testing.B,
	indexes, bases []channel.Channel,
	calc *channel.Channel,
) *calculator.Calculator {
	if len(indexes) > 0 {
		if err := e.dist.Channel.CreateMany(e.ctx, &indexes); err != nil {
			b.Fatalf("failed to create index channels: %v", err)
		}
	}
	if len(bases) > 0 {
		for i, ch := range bases {
			if ch.Virtual {
				continue
			}
			toGet := i
			if len(indexes) == 1 {
				toGet = 0
			}
			ch.LocalIndex = indexes[toGet].LocalKey
			bases[i] = ch
		}
		if err := e.dist.Channel.CreateMany(e.ctx, &bases); err != nil {
			b.Fatalf("failed to create base channels: %v", err)
		}
	}
	if err := e.dist.Channel.Create(e.ctx, calc); err != nil {
		b.Fatalf("failed to create calc channel: %v", err)
	}
	mod, err := compiler.Compile(e.ctx, compiler.Config{
		ChannelService: e.dist.Channel,
		Channel:        *calc,
		SymbolResolver: e.arcSvc.SymbolResolver(),
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

func BenchmarkCalculator_SingleInput(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close(b)

	base := []channel.Channel{{Name: "input", DataType: telem.Int64T, Virtual: true}}
	calc := channel.Channel{
		Name:       "output",
		DataType:   telem.Int64T,
		Virtual:    true,
		Expression: "return input",
	}

	c := env.openCalculator(b, nil, base, &calc)
	defer func() {
		if err := c.Close(); err != nil {
			b.Errorf("failed to close calculator: %v", err)
		}
	}()

	inputFrame := frame.NewUnary(base[0].Key(), telem.NewSeriesV[int64](10, 20, 30))
	outputFrame := frame.Frame{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = c.Next(env.ctx, inputFrame, outputFrame)
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
	calc := channel.Channel{
		Name:       "sum",
		DataType:   telem.Float32T,
		Virtual:    true,
		Expression: "return a + b",
	}

	c := env.openCalculator(b, nil, bases, &calc)
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
	for i := 0; i < b.N; i++ {
		_, _, _ = c.Next(env.ctx, inputFrame, outputFrame)
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
	calc := channel.Channel{
		Name:       "result",
		DataType:   telem.Float64T,
		Virtual:    true,
		Expression: "return w + x + y + z",
	}

	c := env.openCalculator(b, nil, bases, &calc)
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
	for i := 0; i < b.N; i++ {
		_, _, _ = c.Next(env.ctx, inputFrame, outputFrame)
	}
	b.StopTimer()
	b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkCalculator_NestedTwoLevel(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close(b)

	base := []channel.Channel{{Name: "base", DataType: telem.Int64T, Virtual: true}}
	calc1 := channel.Channel{
		Name:       "calc1",
		DataType:   telem.Int64T,
		Virtual:    true,
		Expression: "return base + 1",
	}
	c1 := env.openCalculator(b, nil, base, &calc1)

	calc2 := channel.Channel{
		Name:       "calc2",
		DataType:   telem.Int64T,
		Virtual:    true,
		Expression: "return calc1 * 2",
	}
	c2 := env.openCalculator(b, nil, nil, &calc2)

	group := calculator.Group{c1, c2}
	defer func() {
		if err := group.Close(); err != nil {
			b.Errorf("failed to close calculator group: %v", err)
		}
	}()

	inputFrame := frame.NewUnary(base[0].Key(), telem.NewSeriesV[int64](10, 20, 30))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = group.Next(env.ctx, inputFrame)
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
			calc := channel.Channel{
				Name:       "result",
				DataType:   telem.Float64T,
				Virtual:    true,
				Expression: "return a * b",
			}

			c := env.openCalculator(b, nil, bases, &calc)
			defer func() {
				if err := c.Close(); err != nil {
					b.Errorf("failed to close calculator: %v", err)
				}
			}()

			aData := make([]float64, count)
			bData := make([]float64, count)
			for i := 0; i < count; i++ {
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
			for i := 0; i < b.N; i++ {
				_, _, _ = c.Next(env.ctx, inputFrame, outputFrame)
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
	calc := channel.Channel{
		Name:       "result",
		DataType:   telem.Float64T,
		Virtual:    true,
		Expression: "if a > b { return a * 2 } else { return b * 3 }",
	}

	c := env.openCalculator(b, nil, bases, &calc)
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
	for i := 0; i < b.N; i++ {
		_, _, _ = c.Next(env.ctx, inputFrame, outputFrame)
	}
	b.StopTimer()
	b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkCalculator_GroupScaling(b *testing.B) {
	for _, size := range []int{1, 3, 5} {
		b.Run(fmt.Sprintf("calculators=%d", size), func(b *testing.B) {
			env := newBenchEnv(b)
			defer env.close(b)

			base := []channel.Channel{{Name: "base", DataType: telem.Int64T, Virtual: true}}

			var group calculator.Group
			prevName := "base"
			for i := 0; i < size; i++ {
				calcCh := channel.Channel{
					Name:       fmt.Sprintf("calc_%d", i),
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: fmt.Sprintf("return %s + 1", prevName),
				}
				var bases []channel.Channel
				if i == 0 {
					bases = base
				}
				c := env.openCalculator(b, nil, bases, &calcCh)
				group = append(group, c)
				prevName = calcCh.Name
			}
			defer func() {
				if err := group.Close(); err != nil {
					b.Errorf("failed to close calculator group: %v", err)
				}
			}()

			inputFrame := frame.NewUnary(base[0].Key(), telem.NewSeriesV[int64](10, 20, 30))

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = group.Next(env.ctx, inputFrame)
			}
			b.StopTimer()
			b.ReportMetric(float64(3*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}
