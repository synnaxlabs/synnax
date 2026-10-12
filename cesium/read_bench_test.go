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
	"github.com/synnaxlabs/x/io/fs"
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
