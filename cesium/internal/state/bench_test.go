// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/telem"
)

// history returns a durable state of key 5, its own clock, with sealed segments 1 to n
// of one pointer each, then the open segment n+1 with no pointer.
func history(n uint32) *state.Snapshot {
	segs := make([]state.Segment, 0, n+1)
	for id := uint32(1); id <= n; id++ {
		at := int64(id) * 10
		segs = append(segs, state.Segment{
			ID:       uint64(id),
			Clock:    5,
			Version:  1,
			DataType: 4,
			Sealed:   true,
			Length:   8,
			Pointers: []state.Pointer{pointer(id, 0, at, at, 0, 8, 1)},
		})
	}
	open := uint64(n) + 1
	segs = append(segs, state.Segment{ID: open, Clock: 5, Version: 1, DataType: 4})
	return &state.Snapshot{
		Limits:   state.Limits{Segments: open + 1, Alignments: open + 1},
		Channels: []*state.Channel{state.NewChannel(5, segs...)},
	}
}

// extendOpen sets c to the commit of sample i on the open segment of history(n).
func extendOpen(c *log.Commit, n, i uint32) {
	c.Samples = 1
	c.Alignment = uint64(telem.NewAlignment(n+1, i))
	c.First = int64(n+1)*10 + int64(i)
	c.Last = c.First
	c.Slots[0] = log.Slot{Segment: uint64(n) + 1, Length: 8 * (i + 1)}
}

// BenchmarkCommit measures a commit that extends the open segment of a key, after a
// history of sealed segments of that key.
func BenchmarkCommit(b *testing.B) {
	for _, n := range []uint32{1, 100, 10_000} {
		b.Run(fmt.Sprintf("replay/segments=%d", n), func(b *testing.B) {
			commitReplay(b, n)
		})
		b.Run(fmt.Sprintf("writer/segments=%d", n), func(b *testing.B) {
			commitWriter(b, n)
		})
	}
}

func commitReplay(b *testing.B, n uint32) {
	b.Helper()
	s, err := state.Open(config(&appended{}), history(n))
	if err != nil {
		b.Fatal(err)
	}
	c := &log.Commit{Slots: make([]log.Slot, 1)}
	records := []log.Record{c}
	var i uint32
	b.ReportAllocs()
	for b.Loop() {
		extendOpen(c, n, i)
		if err = s.Replay(uint64(i)+1, records); err != nil {
			b.Fatal(err)
		}
		i++
	}
}

func commitWriter(b *testing.B, n uint32) {
	b.Helper()
	cfg := config(&appended{})
	cfg.Executor = func(f func()) { f() }
	s, err := state.Open(cfg, history(n))
	if err != nil {
		b.Fatal(err)
	}
	ran := make(chan error, 1)
	go func() { ran <- s.Run(context.Background()) }()
	b.Cleanup(func() {
		if err := s.Close(); err != nil {
			b.Error(err)
		}
		if err := <-ran; err != nil {
			b.Error(err)
		}
	})
	w, err := s.OpenWriter(context.Background(), clockWriter(int64(n+1)*10, false, 5))
	if err != nil {
		b.Fatal(err)
	}
	c := &log.Commit{Slots: make([]log.Slot, 1)}
	var i uint32
	b.ReportAllocs()
	for b.Loop() {
		extendOpen(c, n, i)
		if err = w.Commit(c); err != nil {
			b.Fatal(err)
		}
		i++
	}
}
