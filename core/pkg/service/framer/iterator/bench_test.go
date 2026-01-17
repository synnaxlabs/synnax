// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/telem"
)

type benchIterEnv struct {
	ctx         context.Context
	builder     *mock.Cluster
	dist        mock.Node
	iteratorSvc *iterator.Service
	arcSvc      *arc.Service
}

func newBenchIterEnv(b *testing.B) *benchIterEnv {
	gomega.RegisterTestingT(b)
	ctx := context.Background()
	builder := mock.NewCluster()
	dist := builder.Provision(ctx)

	arcSvc, err := arc.OpenService(ctx, arc.ServiceConfig{
		DB:       dist.DB,
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
	})
	if err != nil {
		b.Fatalf("failed to open arc service: %v", err)
	}

	iteratorSvc, err := iterator.NewService(iterator.ServiceConfig{
		DistFramer: dist.Framer,
		Channel:    dist.Channel,
		Arc:        arcSvc,
	})
	if err != nil {
		b.Fatalf("failed to open iterator service: %v", err)
	}

	return &benchIterEnv{
		ctx:         ctx,
		builder:     builder,
		dist:        dist,
		iteratorSvc: iteratorSvc,
		arcSvc:      arcSvc,
	}
}

func (e *benchIterEnv) close(b *testing.B) {
	if err := e.builder.Close(); err != nil {
		b.Errorf("failed to close cluster: %v", err)
	}
}

func (e *benchIterEnv) createChannels(
	b *testing.B,
	prefix string,
	numDataChannels int,
) (*channel.Channel, []*channel.Channel) {
	indexCh := &channel.Channel{
		Name:     prefix + "_time",
		DataType: telem.TimeStampT,
		IsIndex:  true,
	}
	if err := e.dist.Channel.Create(e.ctx, indexCh); err != nil {
		b.Fatalf("failed to create index channel: %v", err)
	}
	dataChannels := make([]*channel.Channel, numDataChannels)
	for i := 0; i < numDataChannels; i++ {
		dataChannels[i] = &channel.Channel{
			Name:       fmt.Sprintf("%s_sensor_%d", prefix, i),
			DataType:   telem.Float32T,
			LocalIndex: indexCh.LocalKey,
		}
		if err := e.dist.Channel.Create(e.ctx, dataChannels[i]); err != nil {
			b.Fatalf("failed to create data channel: %v", err)
		}
	}
	return indexCh, dataChannels
}

func (e *benchIterEnv) writeData(
	b *testing.B,
	indexCh *channel.Channel,
	dataChannels []*channel.Channel,
	samplesPerChannel int,
) {
	keys := make([]channel.Key, len(dataChannels)+1)
	keys[0] = indexCh.Key()
	for i, ch := range dataChannels {
		keys[i+1] = ch.Key()
	}
	w, err := e.dist.Framer.OpenWriter(e.ctx, framer.WriterConfig{
		Start:            telem.SecondTS,
		Keys:             keys,
		EnableAutoCommit: config.True(),
	})
	if err != nil {
		b.Fatalf("failed to open writer: %v", err)
	}
	timestamps := make([]telem.TimeStamp, samplesPerChannel)
	for i := 0; i < samplesPerChannel; i++ {
		timestamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
	}
	idxSeries := telem.NewSeriesV(timestamps...)
	series := make([]telem.Series, len(dataChannels)+1)
	series[0] = idxSeries
	for i := range dataChannels {
		data := make([]float32, samplesPerChannel)
		for j := 0; j < samplesPerChannel; j++ {
			data[j] = float32(i*100 + j)
		}
		series[i+1] = telem.NewSeriesV(data...)
	}
	fr := frame.NewMulti(keys, series)
	if _, err := w.Write(fr); err != nil {
		b.Fatalf("failed to write frame: %v", err)
	}
	if err := w.Close(); err != nil {
		b.Fatalf("failed to close writer: %v", err)
	}
}

func (e *benchIterEnv) createCalculation(
	b *testing.B,
	name, expression string,
) *channel.Channel {
	calc := &channel.Channel{
		Name:       name,
		DataType:   telem.Float32T,
		Expression: expression,
	}
	if err := e.dist.Channel.Create(e.ctx, calc); err != nil {
		b.Fatalf("failed to create calculation channel: %v", err)
	}
	return calc
}

func BenchmarkIteratorCalc_SingleResponse(b *testing.B) {
	env := newBenchIterEnv(b)
	defer env.close(b)

	indexCh, dataChannels := env.createChannels(b, "single", 1)
	env.writeData(b, indexCh, dataChannels, 100)
	calc := env.createCalculation(b, "single_calc", "return single_sensor_0 * 2")

	iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
		Keys:   []channel.Key{calc.Key(), calc.Index()},
		Bounds: telem.TimeRangeMax,
	})
	if err != nil {
		b.Fatalf("failed to open iterator: %v", err)
	}
	defer func() {
		if err := iter.Close(); err != nil {
			b.Errorf("failed to close iterator: %v", err)
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()
	iter.SeekFirst()
	for i := 0; i < b.N; i++ {
		for iter.Next(iterator.AutoSpan) {
			_ = iter.Value()
		}
		b.StopTimer()
		iter.SeekFirst()
		b.StartTimer()
	}
	b.StopTimer()
	b.ReportMetric(float64(100*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkIteratorCalc_ManyChannels(b *testing.B) {
	for _, count := range []int{1, 10, 20} {
		b.Run(fmt.Sprintf("channels=%d", count), func(b *testing.B) {
			env := newBenchIterEnv(b)
			defer env.close(b)

			prefix := fmt.Sprintf("many%d", count)
			indexCh, dataChannels := env.createChannels(b, prefix, count)
			env.writeData(b, indexCh, dataChannels, 100)
			calc := env.createCalculation(b, prefix+"_calc", fmt.Sprintf("return %s_sensor_0", prefix))

			iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
				Keys:   []channel.Key{calc.Key(), calc.Index()},
				Bounds: telem.TimeRangeMax,
			})
			if err != nil {
				b.Fatalf("failed to open iterator: %v", err)
			}
			defer func() {
				if err := iter.Close(); err != nil {
					b.Errorf("failed to close iterator: %v", err)
				}
			}()

			b.ReportAllocs()
			b.ResetTimer()
			iter.SeekFirst()
			for i := 0; i < b.N; i++ {
				for iter.Next(iterator.AutoSpan) {
					_ = iter.Value()
				}
				b.StopTimer()
				iter.SeekFirst()
				b.StartTimer()
			}
			b.StopTimer()
			b.ReportMetric(float64(100*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}

func BenchmarkIteratorCalc_LargeFrames(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("samples=%d", size), func(b *testing.B) {
			env := newBenchIterEnv(b)
			defer env.close(b)

			prefix := fmt.Sprintf("large%d", size)
			indexCh, dataChannels := env.createChannels(b, prefix, 1)
			env.writeData(b, indexCh, dataChannels, size)
			calc := env.createCalculation(b, prefix+"_calc", fmt.Sprintf("return %s_sensor_0 * 2", prefix))

			iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
				Keys:   []channel.Key{calc.Key(), calc.Index()},
				Bounds: telem.TimeRangeMax,
			})
			if err != nil {
				b.Fatalf("failed to open iterator: %v", err)
			}
			defer func() {
				if err := iter.Close(); err != nil {
					b.Errorf("failed to close iterator: %v", err)
				}
			}()

			b.ReportAllocs()
			b.ResetTimer()
			iter.SeekFirst()
			for i := 0; i < b.N; i++ {
				for iter.Next(iterator.AutoSpan) {
					_ = iter.Value()
				}
				b.StopTimer()
				iter.SeekFirst()
				b.StartTimer()
			}
			b.StopTimer()
			b.ReportMetric(float64(size*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}

func BenchmarkIteratorCalc_CalculatorChain(b *testing.B) {
	for _, length := range []int{1, 3} {
		b.Run(fmt.Sprintf("chain=%d", length), func(b *testing.B) {
			env := newBenchIterEnv(b)
			defer env.close(b)

			prefix := fmt.Sprintf("chain%d", length)
			indexCh, dataChannels := env.createChannels(b, prefix, 1)
			env.writeData(b, indexCh, dataChannels, 100)

			var finalCalc *channel.Channel
			prevName := prefix + "_sensor_0"
			for i := 0; i < length; i++ {
				name := fmt.Sprintf("%s_calc_%d", prefix, i)
				finalCalc = env.createCalculation(b, name, fmt.Sprintf("return %s + 1", prevName))
				prevName = name
			}

			iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
				Keys:   []channel.Key{finalCalc.Key(), finalCalc.Index()},
				Bounds: telem.TimeRangeMax,
			})
			if err != nil {
				b.Fatalf("failed to open iterator: %v", err)
			}
			defer func() {
				if err := iter.Close(); err != nil {
					b.Errorf("failed to close iterator: %v", err)
				}
			}()

			b.ReportAllocs()
			b.ResetTimer()
			iter.SeekFirst()
			for i := 0; i < b.N; i++ {
				for iter.Next(iterator.AutoSpan) {
					_ = iter.Value()
				}
				b.StopTimer()
				iter.SeekFirst()
				b.StartTimer()
			}
			b.StopTimer()
			b.ReportMetric(float64(100*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}

func BenchmarkIteratorCalc_MultipleDomains(b *testing.B) {
	for _, numDomains := range []int{1, 3} {
		b.Run(fmt.Sprintf("domains=%d", numDomains), func(b *testing.B) {
			env := newBenchIterEnv(b)
			defer env.close(b)

			prefix := fmt.Sprintf("domain%d", numDomains)
			indexCh := &channel.Channel{
				Name:     prefix + "_time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			if err := env.dist.Channel.Create(env.ctx, indexCh); err != nil {
				b.Fatalf("failed to create index channel: %v", err)
			}
			dataCh := &channel.Channel{
				Name:       prefix + "_sensor",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			if err := env.dist.Channel.Create(env.ctx, dataCh); err != nil {
				b.Fatalf("failed to create data channel: %v", err)
			}

			keys := []channel.Key{indexCh.Key(), dataCh.Key()}
			for d := 0; d < numDomains; d++ {
				startTS := telem.TimeStamp(d*1000+1) * telem.SecondTS
				w, err := env.dist.Framer.OpenWriter(env.ctx, framer.WriterConfig{
					Start:            startTS,
					Keys:             keys,
					EnableAutoCommit: config.True(),
				})
				if err != nil {
					b.Fatalf("failed to open writer: %v", err)
				}
				timestamps := make([]telem.TimeStamp, 50)
				data := make([]float32, 50)
				for i := 0; i < 50; i++ {
					timestamps[i] = telem.TimeStamp(d*1000+i+1) * telem.SecondTS
					data[i] = float32(d*100 + i)
				}
				fr := frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV(timestamps...),
					telem.NewSeriesV(data...),
				})
				if _, err := w.Write(fr); err != nil {
					b.Fatalf("failed to write: %v", err)
				}
				if err := w.Close(); err != nil {
					b.Fatalf("failed to close writer: %v", err)
				}
			}

			calc := env.createCalculation(b, prefix+"_calc", "return "+prefix+"_sensor * 2")
			iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
				Keys:   []channel.Key{calc.Key(), calc.Index()},
				Bounds: telem.TimeRangeMax,
			})
			if err != nil {
				b.Fatalf("failed to open iterator: %v", err)
			}
			defer func() {
				if err := iter.Close(); err != nil {
					b.Errorf("failed to close iterator: %v", err)
				}
			}()

			b.ReportAllocs()
			b.ResetTimer()
			iter.SeekFirst()
			for i := 0; i < b.N; i++ {
				for iter.Next(iterator.AutoSpan) {
					_ = iter.Value()
				}
				b.StopTimer()
				iter.SeekFirst()
				b.StartTimer()
			}
			b.StopTimer()
			b.ReportMetric(float64(50*numDomains*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}

func BenchmarkIteratorCalc_TwoInputAdd(b *testing.B) {
	env := newBenchIterEnv(b)
	defer env.close(b)

	indexCh, dataChannels := env.createChannels(b, "twoinput", 2)
	env.writeData(b, indexCh, dataChannels, 100)
	calc := env.createCalculation(b, "twoinput_calc", "return twoinput_sensor_0 + twoinput_sensor_1")

	iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
		Keys:   []channel.Key{calc.Key(), calc.Index()},
		Bounds: telem.TimeRangeMax,
	})
	if err != nil {
		b.Fatalf("failed to open iterator: %v", err)
	}
	defer func() {
		if err := iter.Close(); err != nil {
			b.Errorf("failed to close iterator: %v", err)
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()
	iter.SeekFirst()
	for i := 0; i < b.N; i++ {
		for iter.Next(iterator.AutoSpan) {
			_ = iter.Value()
		}
		b.StopTimer()
		iter.SeekFirst()
		b.StartTimer()
	}
	b.StopTimer()
	b.ReportMetric(float64(100*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkIteratorCalc_MixedConcreteAndCalc(b *testing.B) {
	env := newBenchIterEnv(b)
	defer env.close(b)

	indexCh, dataChannels := env.createChannels(b, "mixed", 2)
	env.writeData(b, indexCh, dataChannels, 100)
	calc := env.createCalculation(b, "mixed_calc", "return mixed_sensor_0 + mixed_sensor_1")

	requestKeys := []channel.Key{
		dataChannels[0].Key(),
		dataChannels[1].Key(),
		calc.Key(),
		calc.Index(),
	}
	iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
		Keys:   requestKeys,
		Bounds: telem.TimeRangeMax,
	})
	if err != nil {
		b.Fatalf("failed to open iterator: %v", err)
	}
	defer func() {
		if err := iter.Close(); err != nil {
			b.Errorf("failed to close iterator: %v", err)
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()
	iter.SeekFirst()
	for i := 0; i < b.N; i++ {
		for iter.Next(iterator.AutoSpan) {
			_ = iter.Value()
		}
		b.StopTimer()
		iter.SeekFirst()
		b.StartTimer()
	}
	b.StopTimer()
	b.ReportMetric(float64(100*b.N)/b.Elapsed().Seconds(), "samples/sec")
}
