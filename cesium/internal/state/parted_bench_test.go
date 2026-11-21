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

	"github.com/synnaxlabs/cesium/internal/state"
)

// BenchmarkParted tags every sealed segment of one key in one Parted. It times the
// Parted step alone.
func BenchmarkParted(b *testing.B) {
	for _, n := range []int{2_000, 20_000} {
		b.Run(fmt.Sprintf("segments=%d", n), func(b *testing.B) {
			for range b.N {
				b.StopTimer()
				s, tags, done := runSealed(b, n)
				b.StartTimer()
				if err := s.Parted(2, tags, []state.Range{
					{Key: 5, Oldest: 1, Newest: 1},
				}, func() {}); err != nil {
					b.Fatal(err)
				}
				// TakeDead waits for the owner to run the Parted step, at a small fixed
				// cost.
				if _, err := s.TakeDead(); err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				if err := s.Close(); err != nil {
					b.Fatal(err)
				}
				if err := <-done; err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// runSealed runs a state of key 5 with n sealed segments, and returns it, the tags of
// its segments in part 1, and the result of its Run.
func runSealed(b *testing.B, n int) (*state.State, []state.Tag, chan error) {
	b.Helper()
	segs := make([]state.Segment, n)
	tags := make([]state.Tag, n)
	for i := range segs {
		segs[i] = state.Segment{
			ID: uint64(i + 1), Clock: 5, Version: 1, DataType: 7,
			Sealed: true, Length: 8, TailCRC: 1,
			Pointers: []state.Pointer{pointer(uint32(i+1), 0,
				int64(i*10), int64(i*10), 0, 8, 1)},
		}
		tagged := segs[i]
		tagged.Part = 1
		tags[i] = state.Tag{Part: 1, Segment: tagged}
	}
	head := &state.Snapshot{
		Sequence: 1,
		Limits:   state.Limits{Segments: uint64(n + 1), Alignments: uint64(n + 1)},
		Channels: []*state.Channel{{Key: 5, Segments: segs}},
	}
	s, err := state.Open(config(&appended{}), head)
	if err != nil {
		b.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- s.Run(b.Context()) }()
	// Run ends replay mode before its first step, so a step here keeps that work out
	// of the timer.
	if _, err := s.TakeDead(); err != nil {
		b.Fatal(err)
	}
	return s, tags, done
}
