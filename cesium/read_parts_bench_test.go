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
	"time"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkReadParts reads an index and a data channel of 100 stretches of 100 samples
// after a switch. With parts, the switch wrote every sealed segment to a part, and a
// read takes their pointers through the part cache. Without, the same switch writes no
// part, and a read takes them from the state.
func BenchmarkReadParts(b *testing.B) {
	for _, parted := range []bool{false, true} {
		b.Run(fmt.Sprintf("parts=%t", parted), func(b *testing.B) {
			benchmarkReadParts(b, parted)
		})
	}
}

func benchmarkReadParts(b *testing.B, parted bool) {
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
		cesium.Channel{
			Key:      3,
			Name:     "filler",
			IsIndex:  true,
			DataType: telem.TimestampT,
		},
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
	// A writer's open waits for the state owner, which applies the tags of the parts
	// that the open loaded first.
	writePartsStretch(b, ctx, db, keys, stretches*samples, 1)
	b.ReportAllocs()
	for b.Loop() {
		if read := readAll(b, db, keys); read != 2*(stretches*samples+1) {
			b.Fatalf("read %d samples, want %d", read, 2*(stretches*samples+1))
		}
	}
}

// waitPartsHead waits for head 2 on fs to read whole. The checkpointer writes the parts
// before the head, and the head exists once it is created.
func waitPartsHead(b *testing.B, fs xfs.FS) {
	b.Helper()
	for {
		if _, _, err := checkpoint.Read(fs, 2); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// runPartsDB opens and runs a database of the pointer log layout on fs with partSize,
// stating identity id, 0 for a new root, and returns the function that stops and
// closes it.
func runPartsDB(
	b *testing.B,
	ctx context.Context,
	fs xfs.FS,
	partSize int,
	id uint64,
) (*cesium.DB, func()) {
	b.Helper()
	db, err := cesium.Open(
		ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithPointerLog(),
		cesium.WithPartSize(partSize),
		cesium.WithIdentity(id),
	)
	if err != nil {
		b.Fatal(err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	ran := make(chan error, 1)
	go func() { ran <- db.Run(runCtx) }()
	return db, func() {
		cancel()
		if err := <-ran; err != nil {
			b.Error(err)
		}
		if err := db.Close(); err != nil {
			b.Error(err)
		}
	}
}

// writePartsStretch writes samples samples from sample first to keys, an index and a
// data channel, with one writer.
func writePartsStretch(
	b *testing.B,
	ctx context.Context,
	db *cesium.DB,
	keys []cesium.ChannelKey,
	first, samples int,
) {
	b.Helper()
	stamps, values := make([]telem.TimeStamp, samples), make([]int64, samples)
	for i := range samples {
		stamps[i] = telem.TimeStamp(first+i+1) * telem.SecondTS
		values[i] = int64(first + i)
	}
	w, err := db.OpenWriter(ctx, cesium.WriterConfig{
		Channels: keys,
		Clock:    keys[0],
		Start:    stamps[0],
	})
	if err != nil {
		b.Fatal(err)
	}
	fr := telem.MultiFrame(keys, []telem.Series{
		telem.NewSeries(stamps),
		telem.NewSeries(values),
	})
	if _, _, err = w.Write(ctx, fr, nil); err == nil {
		_, _, err = w.Commit(ctx, nil)
	}
	if err = errors.Join(err, w.Close()); err != nil {
		b.Fatal(err)
	}
}
