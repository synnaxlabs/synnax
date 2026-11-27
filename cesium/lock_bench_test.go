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
	"testing"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkOpenWriter opens and closes a writer on an index and a data channel.
func BenchmarkOpenWriter(b *testing.B) {
	db := openBenchDB(b, true)
	keys := []cesium.ChannelKey{1, 2}
	createBenchChannels(b, db, keys)
	b.ReportAllocs()
	for b.Loop() {
		w := openBenchWriter(b, db, keys, control.AuthorityAbsolute)
		if err := w.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCreateChannel creates a data channel on an existing index.
func BenchmarkCreateChannel(b *testing.B) {
	db := openBenchDB(b, true)
	createBenchChannels(b, db, []cesium.ChannelKey{1, 2})
	b.ReportAllocs()
	key := cesium.ChannelKey(3)
	for b.Loop() {
		if err := db.CreateChannel(b.Context(), cesium.Channel{
			Key:      key,
			Name:     "data",
			Index:    1,
			DataType: telem.Float32T,
		}); err != nil {
			b.Fatal(err)
		}
		key++
	}
}

// BenchmarkDeleteTimeRange deletes a frame of samples from an index and a data
// channel. Writing the frame is not timed.
func BenchmarkDeleteTimeRange(b *testing.B) {
	db := openBenchDB(b, true)
	keys := []cesium.ChannelKey{1, 2}
	createBenchChannels(b, db, keys)
	b.ReportAllocs()
	start := telem.SecondTS
	for range b.N {
		b.StopTimer()
		if err := db.Write(b.Context(), start, benchFrame(keys, start)); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := db.DeleteTimeRange(
			b.Context(),
			keys,
			start.Range(start+benchFrameSamples),
		); err != nil {
			b.Fatal(err)
		}
		start += benchFrameSamples
	}
}
