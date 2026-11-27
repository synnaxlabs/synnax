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
	"fmt"
	"testing"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/telem"
)

// history returns a durable state of key 5, its own clock, with sealed segments 1 to n
// of one pointer each, then the open segment n+1 with no pointer.
func history(n uint32) *state.Snapshot {
	ch := &state.Channel{Key: 5, Segments: make([]state.Segment, 0, n+1)}
	for id := uint32(1); id <= n; id++ {
		at := int64(id) * 10
		ch.Segments = append(ch.Segments, state.Segment{
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
	ch.Segments = append(
		ch.Segments,
		state.Segment{ID: open, Clock: 5, Version: 1, DataType: 4},
	)
	return &state.Snapshot{
		Limits:   state.Limits{Segments: open + 1, Alignments: open + 1},
		Channels: []*state.Channel{ch},
	}
}

// BenchmarkCommit measures the replay of a commit that extends the open segment of a
// key, after a history of sealed segments of that key.
func BenchmarkCommit(b *testing.B) {
	for _, n := range []uint32{1, 100, 10_000} {
		b.Run(fmt.Sprintf("segments=%d", n), func(b *testing.B) {
			s, err := state.Open(config(&appended{}), history(n))
			if err != nil {
				b.Fatal(err)
			}
			c := &log.Commit{Samples: 1, Slots: []log.Slot{{Segment: uint64(n) + 1}}}
			records := []log.Record{c}
			start := int64(n+1) * 10
			var i uint32
			b.ReportAllocs()
			for b.Loop() {
				c.Alignment = uint64(telem.NewAlignment(n+1, i))
				c.First = start + int64(i)
				c.Last = c.First
				c.Slots[0].Length = 8 * (i + 1)
				if err = s.Replay(uint64(i)+1, records); err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	}
}
