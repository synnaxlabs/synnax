// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"bytes"
	"testing"

	"github.com/synnaxlabs/cesium/internal/log"
)

// reencode decodes the entry at the start of b and encodes its records again. It
// returns false when b does not decode.
func reencode(b []byte) ([]byte, bool) {
	var (
		d log.Decoder
		e log.Entry
	)
	if _, err := d.Reset(b); err != nil {
		return nil, false
	}
	for {
		r, ok, err := d.Next()
		if err != nil {
			return nil, false
		}
		if !ok {
			return e.Bytes(), true
		}
		e.Add(r)
	}
}

// FuzzDecodeEntry checks that the entry decoder never panics, and that an entry it
// accepts encodes to bytes that decode to the same records.
func FuzzDecodeEntry(f *testing.F) {
	f.Add(entryOf(
		&log.Reserve{Segments: 17, Alignments: 1 << 20},
		&log.Create{Segment: 2, Key: 4, Clock: 4, Version: 1, DataType: 6},
		&commit,
		&log.Seal{Segment: 2, Length: 24, TailCRC: 7},
		&log.Drop{Key: 8},
		&log.Restart{},
		&log.Intent{Key: 8, Clear: true},
	))
	f.Add(rawEntry(raw{1<<15 | 40, []byte{1, 2, 3}}))
	f.Add(entryOf())
	f.Fuzz(func(t *testing.T, b []byte) {
		first, ok := reencode(b)
		if !ok {
			return
		}
		second, ok := reencode(first)
		if !ok {
			t.Fatalf("encoded entry %x does not decode", first)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("entry %x encodes to %x, then to %x", b, first, second)
		}
	})
}
