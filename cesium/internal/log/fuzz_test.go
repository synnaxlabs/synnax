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

// FuzzDecodeFrame checks that the frame decoder never panics, and that a frame it
// accepts encodes to the same bytes.
func FuzzDecodeFrame(f *testing.F) {
	frame := log.Frame{Count: 1, Sequence: 9, PrevCRC: 7, Entries: entryOf(&commit)}
	f.Add(frame.Append(nil, db, generation, 8_192))
	mark := log.Frame{Sequence: 10, Clean: true}
	f.Add(mark.Append(nil, db, generation, 8_192))
	f.Fuzz(func(t *testing.T, b []byte) {
		decoded, err := log.DecodeFrame(b, db, generation, 8_192)
		if err != nil {
			return
		}
		again := decoded.Append(nil, db, generation, 8_192)
		if !bytes.Equal(again, b[:decoded.Size()]) {
			t.Fatalf("frame %x encodes to %x", b[:decoded.Size()], again)
		}
	})
}

// FuzzDecodeHeader checks that the header decoder never panics, and that a header it
// accepts encodes to the same bytes.
func FuzzDecodeHeader(f *testing.F) {
	f.Add(header.Encode(0))
	f.Fuzz(func(t *testing.T, b []byte) {
		decoded, err := log.DecodeHeader(b, 0)
		if err != nil {
			return
		}
		if again := decoded.Encode(0); !bytes.Equal(again, b[:log.HeaderSize]) {
			t.Fatalf("header %x encodes to %x", b[:log.HeaderSize], again)
		}
	})
}
