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
	"encoding/binary"
	"testing"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/set"
)

// knownTypes are the record types that this binary decodes.
var knownTypes = set.New[uint16](1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

// walkEntry reads the structure of the entry at the start of b with no decoder: an
// entry header whose length and count fit, then records whose headers have a zero
// reserved field and whose bodies are multiples of 8 inside the entry. It returns the
// records and false when the structure breaks.
func walkEntry(b []byte) ([]raw, bool) {
	if len(b) < 8 {
		return nil, false
	}
	size := int(binary.LittleEndian.Uint32(b))
	count := int(binary.LittleEndian.Uint32(b[4:]))
	if size > len(b)-8 {
		return nil, false
	}
	body := b[8 : 8+size]
	var records []raw
	for range count {
		if len(body) < 8 || binary.LittleEndian.Uint16(body[2:]) != 0 {
			return nil, false
		}
		n := int(binary.LittleEndian.Uint32(body[4:]))
		if n%8 != 0 || n > len(body)-8 {
			return nil, false
		}
		records = append(records, raw{binary.LittleEndian.Uint16(body), body[8 : 8+n]})
		body = body[8+n:]
	}
	return records, len(body) == 0
}

// checkEntry fails t when the decoder accepts entry b but b does not have the structure
// that walkEntry reads, or a known record of b does not encode to its own bytes, or an
// unknown one is not skippable.
func checkEntry(t *testing.T, b []byte) {
	t.Helper()
	var d log.Decoder
	if _, err := d.Reset(b); err != nil {
		return
	}
	var got [][]byte
	for {
		r, ok, err := d.Next()
		if err != nil {
			return
		}
		if !ok {
			break
		}
		var e log.Entry
		e.Add(r)
		got = append(got, e.Bytes()[8:])
	}
	records, ok := walkEntry(b)
	if !ok {
		t.Fatalf("decoder accepts entry %x, which breaks the structure", b)
	}
	var known int
	for _, r := range records {
		if !knownTypes.Contains(r.t) {
			if r.t&(1<<15) == 0 {
				t.Fatalf(
					"decoder skips record type %d, which is not skippable",
					r.t,
				)
			}
			continue
		}
		want := rawEntry(r)[8:]
		if known >= len(got) || !bytes.Equal(got[known], want) {
			t.Fatalf(
				"record %d of entry %x does not encode to its own bytes",
				known,
				b,
			)
		}
		known++
	}
	if known != len(got) {
		t.Fatalf("decoder returns %d records of %d known ones", len(got), known)
	}
}

// FuzzDecodeEntry checks that the entry decoder never panics, and that an entry it
// accepts has the structure that walkEntry reads, with each known record encoding to
// its own bytes and each unknown one skippable.
func FuzzDecodeEntry(f *testing.F) {
	f.Add(entryOf(
		&log.Reserve{Segments: 17, Alignments: 1 << 20},
		&log.Create{Segment: 2, Key: 4, Clock: 4, Version: 1, DataType: 6},
		&commit,
		&log.Seal{Segment: 2, Length: 24, TailCRC: 7},
		&tombstone,
		&replace,
		&log.Drop{Key: 8},
		&log.Restart{},
		&log.Check{Segment: 3},
		&log.Intent{Key: 8, Cleared: true},
	))
	f.Add(rawEntry(raw{1<<15 | 40, []byte{1, 2, 3, 4, 5, 6, 7, 8}}))
	f.Add(entryOf())
	f.Add(rawEntry(raw{11, make([]byte, 8)}))
	for _, r := range entryRefusals {
		f.Add(r.b)
	}
	f.Fuzz(checkEntry)
}

// FuzzDecodeFrame checks that the frame decoder never panics, and that a frame it
// accepts encodes to the same bytes and has room for the headers of its entries.
func FuzzDecodeFrame(f *testing.F) {
	frame := log.Frame{Count: 1, Sequence: 9, PrevCRC: 7, Entries: entryOf(&commit)}
	f.Add(frame.Append(nil, db, generation, 8_192))
	mark := log.Frame{Sequence: 10, Clean: true}
	f.Add(mark.Append(nil, db, generation, 8_192))
	for _, r := range frameRefusals {
		f.Add(r.b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		decoded, err := log.DecodeFrame(b, db, generation, 8_192)
		if err != nil {
			return
		}
		if 8*uint64(decoded.Count) > uint64(len(decoded.Entries)) {
			t.Fatalf("frame of %d entry bytes holds %d entries", len(decoded.Entries),
				decoded.Count)
		}
		again := decoded.Append(nil, db, generation, 8_192)
		if !bytes.Equal(again, b[:decoded.Size()]) {
			t.Fatalf("frame %x encodes to %x", b[:decoded.Size()], again)
		}
	})
}

// FuzzDecodeHeader checks that the header decoder never panics, and that a header it
// accepts is of the log it reads and encodes to the same bytes.
func FuzzDecodeHeader(f *testing.F) {
	f.Add(header.Encode(0))
	other := header
	other.DB++
	f.Add(other.Encode(0))
	for _, at := range []int{0, 8, 12, 55} {
		b := header.Encode(0)
		b[at] ^= 2
		f.Add(sealedHeader(b))
	}
	f.Add(header.Encode(0)[:63])
	f.Fuzz(func(t *testing.T, b []byte) {
		decoded, err := log.DecodeHeader(b, db, generation, 0)
		if err != nil {
			return
		}
		if decoded.DB != db || decoded.Generation != generation {
			t.Fatalf("header of database %d and generation %d passes", decoded.DB,
				decoded.Generation)
		}
		if again := decoded.Encode(0); !bytes.Equal(again, b[:log.HeaderSize]) {
			t.Fatalf("header %x encodes to %x", b[:log.HeaderSize], again)
		}
	})
}
