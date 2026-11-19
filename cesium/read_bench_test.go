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
		if err := db.Write(ctx, start, fr); err != nil {
			b.Fatal(err)
		}
	}
	if open {
		start, fr := frame(count)
		count++
		w, err := db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
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
		if _, err := w.Write(ctx, fr); err != nil {
			b.Fatal(err)
		}
		if _, err := w.Commit(ctx); err != nil {
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
	if err := db.WriteSeries(
		b.Context(),
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
func BenchmarkReadSamples(b *testing.B) {
	const n = 1_000_000
	for _, late := range []int{0, n / 2} {
		b.Run(fmt.Sprintf("late=%d", late), func(b *testing.B) {
			db := openBenchDB(b, true)
			keys := []cesium.ChannelKey{1, 2}
			createBenchChannels(b, db, keys)
			stamps := writeBenchStamps(b, db, keys[0], n)
			if err := db.WriteSeries(
				b.Context(),
				keys[1],
				stamps[late],
				telem.NewSeriesV(make([]float32, n-late)...),
			); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if read := readAll(b, db, keys); read != 2*n-late {
					b.Fatalf("read %d samples, want %d", read, 2*n-late)
				}
			}
		})
	}
}

// BenchmarkStreamFixedSpan reads an index and its data channels with one fixed-span
// Next over every sample, taking each chunk as the stream iterator sends it.
func BenchmarkStreamFixedSpan(b *testing.B) {
	for _, c := range []struct{ samples, channels int }{
		{samples: 1_000_000, channels: 1},
		{samples: 1_000_000, channels: 8},
		{samples: 4_000_000, channels: 1},
		{samples: 4_000_000, channels: 8},
	} {
		name := fmt.Sprintf("samples=%d/channels=%d", c.samples, c.channels)
		b.Run(name, func(b *testing.B) {
			db := openBenchDB(b, true)
			keys := createIndexedChannels(b, db, 1, c.channels)
			stamps := writeBenchStamps(b, db, keys[0], c.samples)
			data := telem.NewSeriesV(make([]float32, c.samples)...)
			for _, key := range keys[1:] {
				if err := db.WriteSeries(
					b.Context(),
					key,
					stamps[0],
					data,
				); err != nil {
					b.Fatal(err)
				}
			}
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
