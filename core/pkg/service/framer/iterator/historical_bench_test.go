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
	"testing"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/telem"
)

const histWriteChunk = 250_000

type histSpan struct {
	name string
	rate float64
	span telem.TimeSpan
}

var histSpans = []histSpan{
	{"rate=0.1hz/span=minute", 0.1, 60 * telem.Second},
	{"rate=1hz/span=minute", 1, 60 * telem.Second},
	{"rate=0.1hz/span=hour", 0.1, 3600 * telem.Second},
	{"rate=1hz/span=hour", 1, 3600 * telem.Second},
	{"rate=100hz/span=minute", 100, 60 * telem.Second},
	{"rate=0.1hz/span=day", 0.1, 86400 * telem.Second},
	{"rate=1khz/span=minute", 1000, 60 * telem.Second},
	{"rate=1hz/span=day", 1, 86400 * telem.Second},
	{"rate=0.1hz/span=month", 0.1, 2592000 * telem.Second},
	{"rate=100hz/span=hour", 100, 3600 * telem.Second},
	{"rate=10khz/span=minute", 10000, 60 * telem.Second},
	{"rate=1hz/span=month", 1, 2592000 * telem.Second},
	{"rate=1khz/span=hour", 1000, 3600 * telem.Second},
}

func histSampleCount(rate float64, span telem.TimeSpan) int {
	return int(float64(span) / float64(telem.Second) * rate)
}

// writeHist writes count samples to the given channels, spaced by the given interval,
// in chunks so a month of high-rate data never lands in one frame.
func (e *benchIterEnv) writeHist(
	b *testing.B,
	indexCh *channel.Channel,
	dataChannels []*channel.Channel,
	count int,
	spacing telem.TimeSpan,
) {
	keys := make([]channel.Key, len(dataChannels)+1)
	keys[0] = indexCh.Key()
	for i, ch := range dataChannels {
		keys[i+1] = ch.Key()
	}
	w, err := e.node.Framer.OpenWriter(e.ctx, framer.WriterConfig{
		Start:            telem.SecondTS,
		Keys:             keys,
		EnableAutoCommit: new(true),
	})
	if err != nil {
		b.Fatalf("failed to open writer: %v", err)
	}
	for written := 0; written < count; {
		size := min(histWriteChunk, count-written)
		timestamps := make([]telem.TimeStamp, size)
		for i := range size {
			timestamps[i] = telem.SecondTS + telem.TimeStamp(written+i)*
				telem.TimeStamp(spacing)
		}
		series := make([]telem.Series, len(dataChannels)+1)
		series[0] = telem.NewSeriesV(timestamps...)
		for i := range dataChannels {
			data := make([]float32, size)
			for j := range size {
				data[j] = float32(i*100 + written + j)
			}
			series[i+1] = telem.NewSeriesV(data...)
		}
		if _, err := w.Write(frame.NewMulti(keys, series)); err != nil {
			b.Fatalf("failed to write frame: %v", err)
		}
		written += size
	}
	if err := w.Close(); err != nil {
		b.Fatalf("failed to close writer: %v", err)
	}
}

// BenchmarkIteratorHistorical compares a raw read against calculated evaluation across
// sample rate and time span. Each iteration opens through iterator.Service.Open, so it
// pays the channel retrieve, the graph build, and the frame plumbing a real read pays.
func BenchmarkIteratorHistorical(b *testing.B) {
	for _, hs := range histSpans {
		b.Run(hs.name, func(b *testing.B) {
			env := newBenchIterEnv(b)
			defer env.close(b)
			indexCh, data := env.createChannels(b, "hist", 4)
			env.writeHist(
				b,
				indexCh,
				data,
				histSampleCount(hs.rate, hs.span),
				telem.TimeSpan(float64(telem.Second)/hs.rate),
			)
			simple := env.createCalculation(
				b, "hist_simple", "return hist_sensor_0 * 2",
			)
			complx := env.createCalculation(
				b, "hist_complex", "return hist_sensor_0 * hist_sensor_1 + 1",
			)
			env.createCalculation(b, "hist_nested_1", "return hist_sensor_0 * 2")
			env.createCalculation(b, "hist_nested_2", "return hist_nested_1 + 1")
			nested3 := env.createCalculation(
				b, "hist_nested_3", "return hist_nested_2 + 1",
			)
			rawMulti := make([]channel.Key, len(data)+1)
			rawMulti[0] = indexCh.Key()
			for i, ch := range data {
				rawMulti[i+1] = ch.Key()
			}
			cases := []struct {
				name string
				keys []channel.Key
			}{
				{"raw/single", []channel.Key{indexCh.Key(), data[0].Key()}},
				{"raw/multi=4", rawMulti},
				{"calc/simple", []channel.Key{simple.Key(), simple.Index()}},
				{"calc/complex", []channel.Key{complx.Key(), complx.Index()}},
				{"calc/nested=3", []channel.Key{nested3.Key(), nested3.Index()}},
				{"calc/index-only", []channel.Key{nested3.Index()}},
			}
			for _, c := range cases {
				b.Run(c.name, func(b *testing.B) { runHistCase(b, env, c.keys) })
			}
		})
	}
}

func runHistCase(b *testing.B, env *benchIterEnv, keys []channel.Key) {
	b.ReportAllocs()
	b.ResetTimer()
	read := 0
	for range b.N {
		iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
			Keys:   keys,
			Bounds: telem.TimeRangeMax,
		})
		if err != nil {
			b.Fatalf("failed to open iterator: %v", err)
		}
		iter.SeekFirst()
		for iter.Next(iterator.AutoSpan) {
			for _, s := range iter.Value().RawSeries() {
				read += int(s.Len())
			}
		}
		if err := iter.Close(); err != nil {
			b.Fatalf("failed to close iterator: %v", err)
		}
	}
	b.StopTimer()
	if read == 0 {
		b.Fatal("read no samples, the case timed an empty range")
	}
	b.ReportMetric(float64(read)/b.Elapsed().Seconds(), "samples/sec")
}
