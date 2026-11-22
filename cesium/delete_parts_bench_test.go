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
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkDeleteParts deletes the gap between two stretches of 100 samples of a data
// channel, among 100 stretches written before a switch. The gap holds no sample, so
// each delete repeats: it loads the records of the segments that it reaches, claims the
// range, and releases it. With parts, the switch wrote every sealed segment to a part,
// and the delete loads their records through the part cache. Without, it takes them
// from the state.
func BenchmarkDeleteParts(b *testing.B) {
	for _, parted := range []bool{false, true} {
		b.Run(fmt.Sprintf("parts=%t", parted), func(b *testing.B) {
			benchmarkDeleteParts(b, parted)
		})
	}
}

func benchmarkDeleteParts(b *testing.B, parted bool) {
	b.Helper()
	const stretches, samples = 100, 100
	ctx, fs := b.Context(), xfs.NewMem()
	size := 0
	if parted {
		size = 1
	}
	keys := []cesium.ChannelKey{1, 2}
	db, stop := runPartsDB(b, ctx, fs, size, 0)
	if err := db.CreateChannel(
		ctx,
		cesium.Channel{Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT},
		cesium.Channel{Key: 2, Name: "data", Index: 1, DataType: telem.Int64T},
	); err != nil {
		b.Fatal(err)
	}
	for s := range stretches {
		writePartsStretch(b, ctx, db, keys, s*samples, samples)
	}
	if err := switchLog(db, fs, 2); err != nil {
		b.Fatal(err)
	}
	waitPartsHead(b, fs)
	id := db.Identity()
	stop()
	db, stop = runPartsDB(b, ctx, fs, size, id)
	defer stop()
	// Stretch 50 ends at second 5,000, and stretch 51 starts at second 5,001.
	gap := telem.TimeRange{
		Start: 5_000*telem.SecondTS + 1,
		End:   5_001 * telem.SecondTS,
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := db.DeleteTimeRange(ctx, []cesium.ChannelKey{2}, gap); err != nil {
			b.Fatal(err)
		}
	}
}
