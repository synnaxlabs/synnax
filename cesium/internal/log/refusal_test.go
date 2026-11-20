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
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/cesium/internal/log"
)

// refusal is bytes that a decoder refuses, with a part of the message of its error.
// The specs check each refusal, and the fuzz targets start from them.
type refusal struct {
	name string
	b    []byte
	msg  string
}

func refusalEntries(rs []refusal) []TableEntry {
	entries := make([]TableEntry, len(rs))
	for i, r := range rs {
		entries[i] = Entry(r.name, r.b, r.msg)
	}
	return entries
}

// changedFrame returns a frame with one byte changed after its checksum was set.
func changedFrame() []byte {
	f := log.Frame{Count: 1, Sequence: 9, Entries: entryOf(&log.Drop{Key: 4})}
	b := f.Append(nil, db, generation, 8_192)
	b[len(b)-1] ^= 1
	return b
}

// long returns an entry of one record of type t whose body is 8 bytes longer than body.
func long(t uint16, body []byte) []byte {
	return rawEntry(raw{t, concat(body, make([]byte, 8))})
}

var entryRefusals = []refusal{
	{"a long reserve", long(1, le64(1, 1)), "reserve record has 24 bytes, not 16"},
	{
		"a long create",
		long(2, concat(le64(1), le32(1, 1), le64(1))),
		"create record has 32 bytes, not 24",
	},
	{
		"a long commit",
		long(3, concat(le32(1, 1), le64(1, 1, 1, 1), le32(1, 1))),
		"commit record of 56 bytes does not hold 32 bytes and its 1 slots",
	},
	{"a long seal", long(4, concat(le64(1), le32(1, 1))), "seal record has 24 bytes"},
	{"a long tombstone", long(5, make([]byte, 72)), "tombstone record has 80 bytes"},
	{
		"a long replace",
		long(6, concat(le64(1, 2), le32(1, 1, 0, 0))),
		"replace record of 40 bytes does not hold 32 bytes and its 0 pointers",
	},
	{"a long drop", long(7, le32(1, 0)), "drop record has 16 bytes, not 8"},
	{"a long intent", long(10, le32(1, 0)), "intent record has 16 bytes, not 8"},
	{"a short entry header", []byte{1, 2, 3}, "entry header needs 8 bytes"},
	{
		"records past the end",
		concat(le32(100, 1), make([]byte, 8)),
		"entry holds 100 bytes of records",
	},
	{
		"a count that its bytes cannot hold",
		concat(le32(8, 2), []byte{8, 0, 0, 0}, le32(0)),
		"entry ends with 1 records left",
	},
	{
		"bytes after the last record",
		concat(le32(16, 1), []byte{8, 0, 0, 0}, le32(0), make([]byte, 8)),
		"8 bytes after its last record",
	},
	{
		"fewer records than its count",
		concat(le32(16, 1), []byte{8, 0, 0, 0}, le32(8), make([]byte, 8)),
		"restart record has 8 bytes, not 0",
	},
	{
		"a body that is not a multiple of 8 bytes",
		rawEntry(raw{7, le32(1)}),
		"holds 4 bytes, not a multiple of 8",
	},
	{
		"a skippable body that is not a multiple of 8 bytes",
		rawEntry(raw{1<<15 | 40, []byte{1, 2, 3}}),
		"holds 3 bytes, not a multiple of 8",
	},
	{
		"a set reserved field in a record header",
		concat(le32(8, 1), []byte{8, 0, 1, 0}, le32(0)),
		"set reserved field",
	},
	{
		"a record past its entry",
		concat(le32(8, 1), []byte{7, 0, 0, 0}, le32(8)),
		"holds 8 bytes, but 0 remain",
	},
	{
		"a set reserved byte in a create",
		rawEntry(raw{
			2,
			concat(le64(1), le32(1, 1), []byte{1, 0, 3, 0, 0, 9, 0, 0}),
		}),
		"create record has a set reserved byte",
	},
	{
		"a commit whose slot count differs from its size",
		rawEntry(raw{3, concat(le32(2, 1), le64(1, 1, 1, 1), le32(1, 1))}),
		"commit record of 48 bytes does not hold 32 bytes and its 2 slots",
	},
	{
		"a short commit",
		rawEntry(raw{3, le32(0, 1, 0, 0, 0, 0)}),
		"commit record of 24 bytes",
	},
	{
		"a set reserved field in a drop",
		rawEntry(raw{7, le32(1, 1)}),
		"drop record has a set reserved byte",
	},
	{
		"unknown intent flags",
		rawEntry(raw{10, le32(1, 2)}),
		"unknown flags 0x2",
	},
	{
		"a short seal",
		rawEntry(raw{4, le64(1)}),
		"seal record has 8 bytes, not 16",
	},
	{
		"a short tombstone",
		rawEntry(raw{5, make([]byte, 64)}),
		"tombstone record has 64 bytes, not 72",
	},
	{
		"unknown tombstone flags",
		rawEntry(raw{5, concat(le32(4, 4), make([]byte, 64))}),
		"tombstone record of key 4 has unknown flags 0x4",
	},
	{
		"a set byte in an absent cut",
		rawEntry(
			raw{5, concat(le32(4, 2), make([]byte, 16), le64(1), make([]byte, 40))},
		),
		"tombstone record has a set reserved byte",
	},
	{
		"a set reserved field in a cut",
		rawEntry(raw{5, concat(le32(4, 1), make([]byte, 16), le64(1), le32(0, 1),
			make([]byte, 32))}),
		"tombstone record has a set reserved byte",
	},
	{
		"a replace whose pointer count differs from its size",
		rawEntry(raw{6, concat(le64(1, 2), le32(1, 1, 2, 0), make([]byte, 40))}),
		"replace record of 72 bytes does not hold 32 bytes and its 2 pointers",
	},
	{
		"a set reserved field in a replace",
		rawEntry(raw{6, concat(le64(1, 2), le32(1, 1, 0, 1))}),
		"replace record has a set reserved byte",
	},
	{
		"a set reserved field in a replace pointer",
		rawEntry(raw{6, concat(le64(1, 2), le32(1, 1, 1, 0), le32(0, 1, 1, 1),
			le64(1, 1, 1))}),
		"replace record has a set reserved byte",
	},
	{
		"a long check",
		rawEntry(raw{9, le64(1, 2)}),
		"check record has 16 bytes, not 8",
	},
	{
		"a short reserve",
		rawEntry(raw{1, le64(1)}),
		"reserve record has 8 bytes, not 16",
	},
	{
		"a short create",
		rawEntry(raw{2, le64(1, 1)}),
		"create record has 16 bytes, not 24",
	},
}

var frameRefusals = []refusal{
	{"a short header", make([]byte, 31), "needs 32 bytes, but 31 remain"},
	{"no magic", make([]byte, 32), "no frame magic at 8192"},
	{
		"no magic in a sealed frame",
		sealedFrame(concat(le32(0x464C5044, 0, 0, 0), le64(1), le32(0, 0)), 8_192),
		"no frame magic at 8192",
	},
	{"a changed byte", changedFrame(), "frame at 8192 fails its check"},
	{
		"entries past the bytes",
		concat(le32(0x464C5043, 0, 8, 0), make([]byte, 16)),
		"holds 8 bytes of entries, past its bounds",
	},
	{
		"entries past the largest frame",
		concat(le32(0x464C5043, 0, 16<<20, 0), make([]byte, 16)),
		"past its bounds",
	},
	{
		"unknown flags",
		sealedFrame(concat(le32(0x464C5043, 0, 0, 0), le64(1), le32(0, 2)), 8_192),
		"unknown flags 0x2",
	},
	{
		"more entries than its bytes hold",
		sealedFrame(
			concat(le32(0x464C5043, 0, 8, 2), le64(1), le32(0, 0), le32(0, 0)),
			8_192,
		),
		"cannot hold 2 entries",
	},
}
