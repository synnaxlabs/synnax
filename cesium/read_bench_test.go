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
		// A synchronous writer has committed the write when Write returns.
		w, err := db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    start,
			Sync:     new(true),
		})
		if err != nil {
			b.Fatal(err)
		}
		defer func() {
			if err := w.Close(); err != nil {
				b.Error(err)
			}
		}()
		if _, err := w.Write(fr); err != nil {
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
