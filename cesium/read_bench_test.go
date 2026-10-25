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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkReadDomains reads an index and a data channel stored as one-sample domains,
// the shape that many short writer sessions leave behind.
func BenchmarkReadDomains(b *testing.B) {
	for _, count := range []int{1, 100, 1_000} {
		b.Run(fmt.Sprintf("domains=%d", count), func(b *testing.B) {
			ctx := b.Context()
			db, err := cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs.NewMem()),
				cesium.WithInstrumentation(alamos.New("bench")),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					b.Error(err)
				}
			}()
			keys := []cesium.ChannelKey{1, 2}
			if err := db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      keys[0],
					Name:     "time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      keys[1],
					Name:     "data",
					Index:    keys[0],
					DataType: telem.Float32T,
				},
			); err != nil {
				b.Fatal(err)
			}
			for i := range count {
				start := telem.TimeStamp(i+1) * telem.SecondTS
				if err := db.Write(ctx, start, telem.MultiFrame(
					keys,
					[]telem.Series{
						telem.NewSeriesV(start),
						telem.NewSeriesV(float32(i)),
					},
				)); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			for b.Loop() {
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
				if read != 2*count {
					b.Fatalf("read %d samples, want %d", read, 2*count)
				}
			}
		})
	}
}

// BenchmarkReadSamples reads an index and a data channel that hold a million samples in
// one domain each, with the data starting at the first index sample or partway through
// the index.
func BenchmarkReadSamples(b *testing.B) {
	const n = 1_000_000
	for _, late := range []int{0, n / 2} {
		b.Run(fmt.Sprintf("late=%d", late), func(b *testing.B) {
			ctx := b.Context()
			db, err := cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs.NewMem()),
				cesium.WithInstrumentation(alamos.New("bench")),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					b.Error(err)
				}
			}()
			keys := []cesium.ChannelKey{1, 2}
			if err := db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      keys[0],
					Name:     "time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      keys[1],
					Name:     "data",
					Index:    keys[0],
					DataType: telem.Float32T,
				},
			); err != nil {
				b.Fatal(err)
			}
			stamps := make([]telem.TimeStamp, n)
			for i := range stamps {
				stamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
			}
			if err := db.WriteSeries(
				ctx,
				keys[0],
				stamps[0],
				telem.NewSeriesV(stamps...),
			); err != nil {
				b.Fatal(err)
			}
			if err := db.WriteSeries(
				ctx,
				keys[1],
				stamps[late],
				telem.NewSeriesV(make([]float32, n-late)...),
			); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
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
				if read != 2*n-late {
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
			ctx := b.Context()
			db, err := cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs.NewMem()),
				cesium.WithInstrumentation(alamos.New("bench")),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					b.Error(err)
				}
			}()
			keys := []cesium.ChannelKey{1}
			channels := []cesium.Channel{{
				Key:      1,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			}}
			for i := range c.channels {
				key := cesium.ChannelKey(i + 2)
				keys = append(keys, key)
				channels = append(channels, cesium.Channel{
					Key:      key,
					Name:     fmt.Sprintf("data_%d", i),
					Index:    1,
					DataType: telem.Float32T,
				})
			}
			if err := db.CreateChannel(ctx, channels...); err != nil {
				b.Fatal(err)
			}
			stamps := make([]telem.TimeStamp, c.samples)
			for i := range stamps {
				stamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
			}
			if err := db.WriteSeries(
				ctx,
				keys[0],
				stamps[0],
				telem.NewSeriesV(stamps...),
			); err != nil {
				b.Fatal(err)
			}
			data := telem.NewSeriesV(make([]float32, c.samples)...)
			for _, key := range keys[1:] {
				if err := db.WriteSeries(ctx, key, stamps[0], data); err != nil {
					b.Fatal(err)
				}
			}
			span := stamps[0].Span(stamps[c.samples-1] + 1)
			want := c.samples * len(keys)
			b.ReportAllocs()
			for b.Loop() {
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
				if read != want {
					b.Fatalf("read %d samples, want %d", read, want)
				}
			}
		})
	}
}
