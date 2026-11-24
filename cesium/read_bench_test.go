// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkReadDomains reads an index and a data channel stored as one-sample domains,
// the shape that many short writer sessions leave behind. With an open writer, a writer
// that has committed one more domain stays open during the reads.
func BenchmarkReadDomains(b *testing.B) {
	for _, count := range []int{1, 100, 1_000} {
		for _, open := range []bool{false, true} {
			name := fmt.Sprintf("domains=%d", count)
			if open {
				name += "/writer=open"
			}
			b.Run(name, func(b *testing.B) { benchmarkReadDomains(b, count, open) })
		}
	}
}

func benchmarkReadDomains(b *testing.B, count int, open bool) {
	b.Helper()
	ctx := b.Context()
	db := openBenchDB(b, true)
	keys := []cesium.ChannelKey{1, 2}
	createBenchChannels(b, db, keys)
	frame := func(i int) (telem.TimeStamp, cesium.Frame) {
		start := telem.TimeStamp(i+1) * telem.SecondTS
		return start, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesV(start),
			telem.NewSeriesV(float32(i)),
		})
	}
	for i := range count {
		start, fr := frame(i)
		if err := write(ctx, db, start, fr); err != nil {
			b.Fatal(err)
		}
	}
	if open {
		start, fr := frame(count)
		count++
		w, err := db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    keys[0],
			Start:    start,
		})
		if err != nil {
			b.Fatal(err)
		}
		defer func() {
			if err := w.Close(); err != nil {
				b.Error(err)
			}
		}()
		if _, _, err := w.Write(ctx, fr, nil); err != nil {
			b.Fatal(err)
		}
		if _, _, err := w.Commit(ctx, nil); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		if read := readAll(b, db, keys); read != 2*count {
			b.Fatalf("read %d samples, want %d", read, 2*count)
		}
	}
}

// readAll reads every sample of keys with AutoSpan and returns the number read.
func readAll(b *testing.B, db *cesium.DB, keys []cesium.ChannelKey) int {
	b.Helper()
	iter, err := db.OpenIterator(cesium.IteratorConfig{
		Bounds:   telem.TimeRangeMax,
		Channels: keys,
	})
	if err != nil {
		b.Fatal(err)
	}
	read := 0
	for iter.SeekFirst(); iter.Next(cesium.AutoSpan); {
		for _, series := range iter.Value().SeriesSlice() {
			read += int(series.Len())
		}
	}
	if err := iter.Close(); err != nil {
		b.Fatal(err)
	}
	return read
}

// layout returns the options of the old layout, or of the pointer log. On the pointer
// log, a rate of 1 byte a second holds each background rewrite at its first job, so no
// compaction or encoding runs during the reads.
func layout(pointerLog bool) []cesium.Option {
	if pointerLog {
		return []cesium.Option{
			cesium.WithPointerLog(),
			cesium.WithGCConfig(cesium.GCConfig{Rate: 1}),
		}
	}
	return nil
}

// writeBenchData writes n timestamps one millisecond apart to the index keys[0], and
// n-late zeros to each data key of keys from the stamp at late, and returns the stamps.
// On the pointer log, where a frame holds every key of its writer, a writer of the
// index alone writes the stamps before late.
func writeBenchData(
	b *testing.B,
	db *cesium.DB,
	keys []cesium.ChannelKey,
	n, late int,
	pointerLog bool,
) []telem.TimeStamp {
	b.Helper()
	if !pointerLog {
		stamps := writeBenchStamps(b, db, keys[0], n)
		data := telem.NewSeriesV(make([]float32, n-late)...)
		for _, key := range keys[1:] {
			if err := writeSeries(b.Context(), db, key, stamps[late], data); err != nil {
				b.Fatal(err)
			}
		}
		return stamps
	}
	stamps := make([]telem.TimeStamp, n)
	for i := range stamps {
		stamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
	}
	if late > 0 {
		writeBenchStamps(b, db, keys[0], late)
	}
	w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
		Channels: keys,
		Clock:    keys[0],
		Start:    stamps[late],
	})
	if err != nil {
		b.Fatal(err)
	}
	series := []telem.Series{telem.NewSeriesV(stamps[late:]...)}
	data := telem.NewSeriesV(make([]float32, n-late)...)
	for range keys[1:] {
		series = append(series, data)
	}
	if _, _, err = w.Write(
		b.Context(),
		telem.MultiFrame(keys, series),
		nil,
	); err != nil {
		b.Fatal(err)
	}
	if _, _, err = w.Commit(b.Context(), nil); err != nil {
		b.Fatal(err)
	}
	if err = w.Close(); err != nil {
		b.Fatal(err)
	}
	return stamps
}

// writeBenchStamps writes n timestamps one millisecond apart to the index channel key
// and returns them.
func writeBenchStamps(
	b *testing.B,
	db *cesium.DB,
	key cesium.ChannelKey,
	n int,
) []telem.TimeStamp {
	b.Helper()
	stamps := make([]telem.TimeStamp, n)
	for i := range stamps {
		stamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
	}
	if err := writeSeries(
		b.Context(), db,
		key,
		stamps[0],
		telem.NewSeriesV(stamps...),
	); err != nil {
		b.Fatal(err)
	}
	return stamps
}

// BenchmarkReadSamples reads an index and a data channel that hold a million samples in
// one domain each, with the data starting at the first index sample or partway through
// the index.
func BenchmarkReadSamples(b *testing.B) { readSamples(b, false) }

// BenchmarkReadSamplesPointerLog is BenchmarkReadSamples on the pointer log, where each
// read checks its blocks.
func BenchmarkReadSamplesPointerLog(b *testing.B) { readSamples(b, true) }

func readSamples(b *testing.B, pointerLog bool) {
	b.Helper()
	const n = 1_000_000
	for _, late := range []int{0, n / 2} {
		b.Run(fmt.Sprintf("late=%d", late), func(b *testing.B) {
			db := openBenchDB(b, true, layout(pointerLog)...)
			keys := []cesium.ChannelKey{1, 2}
			createBenchChannels(b, db, keys)
			writeBenchData(b, db, keys, n, late, pointerLog)
			b.ReportAllocs()
			for b.Loop() {
				if read := readAll(b, db, keys); read != 2*n-late {
					b.Fatalf("read %d samples, want %d", read, 2*n-late)
				}
			}
		})
	}
}

// BenchmarkReadOneClock reads the middle half of the data channels of one clock, which
// share the clock's index lookups, on each layout.
func BenchmarkReadOneClock(b *testing.B) {
	const n = 100_000
	layouts := []struct {
		name string
		opts []cesium.Option
	}{{name: "old"}, {name: "log", opts: []cesium.Option{cesium.WithPointerLog()}}}
	for _, layout := range layouts {
		for _, count := range []int{4, 16} {
			name := fmt.Sprintf("layout=%s/keys=%d", layout.name, count)
			b.Run(name, func(b *testing.B) {
				db := openBenchDB(b, true, layout.opts...)
				keys := createIndexedChannels(b, db, 1, count)
				stamps := make([]telem.TimeStamp, n)
				for i := range stamps {
					stamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
				}
				series := []telem.Series{telem.NewSeriesV(stamps...)}
				for range count {
					series = append(series, telem.NewSeriesV(make([]float32, n)...))
				}
				if err := write(
					b.Context(),
					db,
					stamps[0],
					telem.MultiFrame(keys, series),
				); err != nil {
					b.Fatal(err)
				}
				tr := stamps[n/4].Range(stamps[3*n/4])
				b.ReportAllocs()
				for b.Loop() {
					fr, err := db.Read(b.Context(), tr, keys[1:]...)
					if err != nil {
						b.Fatal(err)
					}
					if got := fr.Count(); got < count {
						b.Fatalf("read %d series, want at least %d", got, count)
					}
				}
			})
		}
	}
}

// BenchmarkStreamFixedSpan reads an index and its data channels with one fixed-span
// Next over every sample, taking each chunk as the stream iterator sends it.
func BenchmarkStreamFixedSpan(b *testing.B) { streamFixedSpan(b, false) }

// BenchmarkStreamFixedSpanPointerLog is BenchmarkStreamFixedSpan on the pointer log,
// where each read checks its blocks.
func BenchmarkStreamFixedSpanPointerLog(b *testing.B) { streamFixedSpan(b, true) }

func streamFixedSpan(b *testing.B, pointerLog bool) {
	b.Helper()
	for _, c := range []struct{ samples, channels int }{
		{samples: 1_000_000, channels: 1},
		{samples: 1_000_000, channels: 8},
		{samples: 4_000_000, channels: 1},
		{samples: 4_000_000, channels: 8},
	} {
		name := fmt.Sprintf("samples=%d/channels=%d", c.samples, c.channels)
		b.Run(name, func(b *testing.B) {
			db := openBenchDB(b, true, layout(pointerLog)...)
			keys := createIndexedChannels(b, db, 1, c.channels)
			stamps := writeBenchData(b, db, keys, c.samples, 0, pointerLog)
			span := stamps[0].Span(stamps[c.samples-1] + 1)
			want := c.samples * len(keys)
			b.ReportAllocs()
			for b.Loop() {
				if read := streamSpan(b, db, keys, span); read != want {
					b.Fatalf("read %d samples, want %d", read, want)
				}
			}
		})
	}
}

// createIndexedChannels creates the given number of index channels, each with perIndex
// float32 data channels that it indexes. It returns their keys, each index followed by
// its data channels.
func createIndexedChannels(
	b *testing.B,
	db *cesium.DB,
	indexes, perIndex int,
) []cesium.ChannelKey {
	b.Helper()
	channels := make([]cesium.Channel, 0, indexes*(1+perIndex))
	for i := range indexes {
		index := cesium.ChannelKey(len(channels) + 1)
		channels = append(channels, cesium.Channel{
			Key:      index,
			Name:     fmt.Sprintf("time_%d", i),
			IsIndex:  true,
			DataType: telem.TimestampT,
		})
		for j := range perIndex {
			channels = append(channels, cesium.Channel{
				Key:      cesium.ChannelKey(len(channels) + 1),
				Name:     fmt.Sprintf("data_%d_%d", i, j),
				Index:    index,
				DataType: telem.Float32T,
			})
		}
	}
	if err := db.CreateChannel(b.Context(), channels...); err != nil {
		b.Fatal(err)
	}
	keys := make([]cesium.ChannelKey, len(channels))
	for i, ch := range channels {
		keys[i] = ch.Key
	}
	return keys
}

// streamSpan reads keys with a stream iterator in one Next of span from the first
// sample, and returns the number of samples read.
func streamSpan(
	b *testing.B,
	db *cesium.DB,
	keys []cesium.ChannelKey,
	span telem.TimeSpan,
) int {
	b.Helper()
	iter, err := db.NewStreamIterator(cesium.IteratorConfig{
		Bounds:   telem.TimeRangeMax,
		Channels: keys,
	})
	if err != nil {
		b.Fatal(err)
	}
	sCtx, cancel := signal.Isolated()
	requests, responses := confluence.Attach(iter, 1)
	iter.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	requests.Inlet() <- cesium.IteratorRequest{
		Command: cesium.IteratorCommandSeekFirst,
	}
	<-responses.Outlet()
	requests.Inlet() <- cesium.IteratorRequest{
		Command: cesium.IteratorCommandNext,
		Span:    span,
	}
	read := 0
	for res := range responses.Outlet() {
		if res.Variant == cesium.IteratorResponseVariantAck {
			break
		}
		for _, series := range res.Frame.SeriesSlice() {
			read += int(series.Len())
		}
	}
	requests.Close()
	err = sCtx.Wait()
	cancel()
	if err != nil {
		b.Fatal(err)
	}
	return read
}

// BenchmarkPointReadsPointerLog reads one sample of a data channel at a random stamp,
// which looks the stamp up in its clock. The block cache holds a clock of 1M stamps,
// and a clock of 10M stamps (80 MB) passes its budget. With goroutines=16, the reads
// run in parallel.
func BenchmarkPointReadsPointerLog(b *testing.B) {
	for _, n := range []int{1_000_000, 10_000_000} {
		for _, goroutines := range []int{1, 16} {
			name := fmt.Sprintf("stamps=%d/goroutines=%d", n, goroutines)
			b.Run(name, func(b *testing.B) { pointReads(b, n, goroutines) })
		}
	}
}

func pointReads(b *testing.B, n, goroutines int) {
	b.Helper()
	db := openBenchDB(b, true, layout(true)...)
	keys := []cesium.ChannelKey{1, 2}
	createBenchChannels(b, db, keys)
	stamps := writeBenchData(b, db, keys, n, 0, true)
	read := func(r *rand.Rand) {
		s := stamps[r.IntN(n)]
		fr, err := db.Read(b.Context(), telem.TimeRange{Start: s, End: s + 1}, keys[1])
		if err != nil {
			b.Error(err)
			return
		}
		if read := fr.Get(keys[1]).Len(); read != 1 {
			b.Errorf("read %d samples at %d, want 1", read, s)
		}
	}
	b.ReportAllocs()
	if goroutines == 1 {
		r := rand.New(rand.NewPCG(1, 2))
		for b.Loop() {
			read(r)
		}
		return
	}
	procs := runtime.GOMAXPROCS(0)
	b.SetParallelism((goroutines + procs - 1) / procs)
	var seed atomic.Uint64
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewPCG(seed.Add(1), 2))
		for pb.Next() {
			read(r)
		}
	})
}
