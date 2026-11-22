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
	"testing"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkCommitPointerLog is BenchmarkCommit on the pointer log layout, on an
// in-memory file system, so the state of each commit is what it measures. A writer of
// the pointer log writes one clock, so each case has one index.
func BenchmarkCommitPointerLog(b *testing.B) {
	for _, c := range []struct{ indexes, channels, samples int }{
		{indexes: 1, channels: 32, samples: 125},
		{indexes: 1, channels: 990, samples: 100},
	} {
		name := fmt.Sprintf(
			"indexes=%d/channels=%d/samples=%d",
			c.indexes,
			c.channels,
			c.samples,
		)
		b.Run(name, func(b *testing.B) {
			db := openBenchDB(b, true, cesium.WithPointerLog())
			benchmarkCommit(b, db, c.indexes, c.channels, c.samples)
		})
	}
}

// BenchmarkSeekPointerLog seeks iterators of a data channel on the pointer log to
// random stamps, from one reader per goroutine of RunParallel. Each seek loads the data
// channel and its clock. Run it with -cpu 8 for 8 readers.
func BenchmarkSeekPointerLog(b *testing.B) {
	const n = 100_000
	db := openBenchDB(b, true, cesium.WithPointerLog())
	keys := []cesium.ChannelKey{1, 2}
	createBenchChannels(b, db, keys)
	stamps := make([]telem.TimeStamp, n)
	for i := range stamps {
		stamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
	}
	if err := db.Write(b.Context(), stamps[0], telem.MultiFrame(
		keys,
		[]telem.Series{
			telem.NewSeriesV(stamps...),
			telem.NewSeriesV(make([]float32, n)...),
		},
	)); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		iter, err := db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: keys[1:],
		})
		if err != nil {
			b.Error(err)
			return
		}
		r := rand.New(rand.NewPCG(1, 2))
		for pb.Next() {
			if !iter.SeekGE(stamps[r.IntN(n)]) {
				b.Error("seek found no sample")
				break
			}
		}
		if err := iter.Close(); err != nil {
			b.Error(err)
		}
	})
}
