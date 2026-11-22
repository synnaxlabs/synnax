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
	"context"
	"fmt"
	"testing"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkOpen opens a database, runs it through the orphan pass at the start of Run,
// and closes it. Each group of an index and a data channel holds one segment for each
// of its stretches, and a switch wrote every sealed segment to a part, so the orphan
// pass matches each segment file against the part records of its key.
func BenchmarkOpen(b *testing.B) {
	for _, c := range []struct{ groups, stretches int }{{500, 10}, {1, 5000}} {
		b.Run(fmt.Sprintf("groups=%d/stretches=%d", c.groups, c.stretches), func(
			b *testing.B,
		) {
			benchmarkOpen(b, c.groups, c.stretches)
		})
	}
}

func benchmarkOpen(b *testing.B, groups, stretches int) {
	b.Helper()
	const samples = 10
	ctx, fs := b.Context(), xfs.NewMem()
	db, stop := runPartsDB(b, ctx, fs, 1)
	channels := make([]cesium.Channel, 0, 2*groups)
	for g := range groups {
		index, data := cesium.ChannelKey(2*g+1), cesium.ChannelKey(2*g+2)
		channels = append(
			channels,
			cesium.Channel{
				Key:      index,
				Name:     fmt.Sprintf("time-%d", g),
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      data,
				Name:     fmt.Sprintf("data-%d", g),
				Index:    index,
				DataType: telem.Int64T,
			},
		)
	}
	if err := db.CreateChannel(ctx, channels...); err != nil {
		b.Fatal(err)
	}
	for s := range stretches {
		for g := range groups {
			keys := []cesium.ChannelKey{channels[2*g].Key, channels[2*g+1].Key}
			writePartsStretch(b, ctx, db, keys, s*samples, samples)
		}
	}
	if err := switchLog(db, fs, 2); err != nil {
		b.Fatal(err)
	}
	waitPartsHead(b, fs)
	stop()
	// With ctx done before it starts, Run returns once its orphan pass is done.
	done, cancel := context.WithCancel(ctx)
	cancel()
	b.ReportAllocs()
	for b.Loop() {
		db, err := cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithPartSize(1),
		)
		if err != nil {
			b.Fatal(err)
		}
		if err = errors.Join(db.Run(done), db.Close()); err != nil {
			b.Fatal(err)
		}
	}
}
