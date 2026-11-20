// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"bytes"
	"testing"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/state"
)

func addHeads(f *testing.F) {
	f.Helper()
	f.Add(checkpoint.Encode(db, generation, sample(2)))
	f.Add(checkpoint.Encode(db, generation, &state.Snapshot{Sequence: 1}))
	for _, r := range headRefusals {
		f.Add(reseal(r.mutate(checkpoint.Encode(db, generation, sample(2)))))
	}
	for _, off := range []int{30, 4100, 4200} {
		b := checkpoint.Encode(db, generation, sample(2))
		b[off] ^= 1
		f.Add(b)
	}
	b := checkpoint.Encode(db, generation, sample(2))
	clear(b[:checkpoint.SectorSize])
	f.Add(b)
}

// checkOrder fails t when s breaks the record order of a head: keys rising, then
// identities, and intents rising.
func checkOrder(t *testing.T, s *state.Snapshot) {
	t.Helper()
	for i, c := range s.Channels {
		if i > 0 && c.Key <= s.Channels[i-1].Key {
			t.Fatalf("key %d is not above key %d", c.Key, s.Channels[i-1].Key)
		}
		for j, seg := range c.Segments {
			if j > 0 && seg.ID <= c.Segments[j-1].ID {
				t.Fatalf("segment %d of key %d is out of order", seg.ID, c.Key)
			}
		}
	}
	for i := 1; i < len(s.Intents); i++ {
		if s.Intents[i] <= s.Intents[i-1] {
			t.Fatalf("intent of key %d is out of order", s.Intents[i])
		}
	}
}

// roundTrip fails t when b decodes but does not encode to the same bytes, apart from
// the bytes between the header sector and the body, which carry nothing.
func roundTrip(t *testing.T, b []byte) {
	t.Helper()
	h, s, err := decode(b)
	if err != nil {
		return
	}
	checkOrder(t, s)
	again := checkpoint.Encode(h.DB, h.Generation, s)
	if len(again) != len(b) ||
		!bytes.Equal(again[:checkpoint.SectorSize], b[:checkpoint.SectorSize]) ||
		!bytes.Equal(again[checkpoint.BodyStart:], b[checkpoint.BodyStart:]) {
		t.Fatalf("head of %d bytes encodes to %d other bytes", len(b), len(again))
	}
}

// FuzzRead checks that the head reader never panics, and that a head it accepts
// encodes to the same bytes.
func FuzzRead(f *testing.F) {
	addHeads(f)
	f.Fuzz(roundTrip)
}

// FuzzReadResealed sets the checksums of each input again before it decodes it, so
// the fuzzer reaches the checks after the checksums.
func FuzzReadResealed(f *testing.F) {
	addHeads(f)
	f.Fuzz(func(t *testing.T, b []byte) { roundTrip(t, reseal(b)) })
}
