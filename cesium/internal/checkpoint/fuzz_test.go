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
}

// roundTrip fails t when b decodes but does not encode to the same bytes, apart from
// the bytes between the header sector and the body, which carry nothing.
func roundTrip(t *testing.T, b []byte) {
	t.Helper()
	h, s, err := checkpoint.Decode(b)
	if err != nil {
		return
	}
	again := checkpoint.Encode(h.DB, h.Generation, s)
	if len(again) != len(b) ||
		!bytes.Equal(again[:checkpoint.SectorSize], b[:checkpoint.SectorSize]) ||
		!bytes.Equal(again[checkpoint.BodyStart:], b[checkpoint.BodyStart:]) {
		t.Fatalf("head of %d bytes encodes to %d other bytes", len(b), len(again))
	}
}

// FuzzDecode checks that the head decoder never panics, and that a head it accepts
// encodes to the same bytes.
func FuzzDecode(f *testing.F) {
	addHeads(f)
	f.Fuzz(roundTrip)
}

// FuzzDecodeResealed sets the checksums of each input again before it decodes it, so
// the fuzzer reaches the checks after the checksums.
func FuzzDecodeResealed(f *testing.F) {
	addHeads(f)
	f.Fuzz(func(t *testing.T, b []byte) { roundTrip(t, reseal(b)) })
}
