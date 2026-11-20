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
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
)

// headRefusal changes a head of sample(2) before its checksums are set again, so that
// the decoder refuses it with an error that wraps wraps and holds msg. The specs check
// each refusal, and the fuzz targets start from them.
type headRefusal struct {
	name   string
	mutate func(b []byte) []byte
	wraps  error
	msg    string
}

func headRefusalEntries() []TableEntry {
	entries := make([]TableEntry, len(headRefusals))
	for i, r := range headRefusals {
		entries[i] = Entry(r.name, r.mutate, r.wraps, r.msg)
	}
	return entries
}

func put32(b []byte, off uint64, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }
func put64(b []byte, off, v uint64)        { binary.LittleEndian.PutUint64(b[off:], v) }
func table(b []byte) uint64                { return binary.LittleEndian.Uint64(b[72:]) }

// at returns the file offset of field of record of section kind of head b.
func at(b []byte, kind int, record, field uint64) uint64 {
	off, _ := section(b, kind)
	size := []uint64{0, 48, 1, 24, 20, 32, 36, 96, 4, 4}[kind]
	return off + record*size + field
}

var headRefusals = []headRefusal{
	{"format version", func(b []byte) []byte {
		b[8] = 2
		return b
	}, log.ErrUnknown, "head format version 2"},
	{"file kind", func(b []byte) []byte {
		b[10] = 5
		return b
	}, place.ErrCorrupt, "kind 5"},
	{"section count", func(b []byte) []byte {
		put32(b, 80, 10)
		return b
	}, log.ErrUnknown, "head with 10 sections"},
	{"reserved byte", func(b []byte) []byte {
		b[200] = 1
		return b
	}, place.ErrCorrupt, "set reserved byte"},
	{"file length", func(b []byte) []byte {
		return append(b, 0)
	}, place.ErrCorrupt, "but its header says"},
	{"table offset", func(b []byte) []byte {
		put64(b, 72, table(b)+8)
		return b
	}, place.ErrCorrupt, "do not fit"},
	{"section kind", func(b []byte) []byte {
		binary.LittleEndian.PutUint16(b[table(b)+24:], 3)
		return b
	}, place.ErrCorrupt, "section 2 of the table has kind 3"},
	{"record version", func(b []byte) []byte {
		binary.LittleEndian.PutUint16(b[table(b)+24*4+2:], 2)
		return b
	}, log.ErrUnknown, "section 5 version 2"},
	{"record size", func(b []byte) []byte {
		put32(b, table(b)+24*5+4, 40)
		return b
	}, place.ErrCorrupt, "section 6 has records of 40 bytes"},
	{"section offset", func(b []byte) []byte {
		put64(b, table(b)+24*3+8, at(b, 4, 0, 8))
		return b
	}, place.ErrCorrupt, "section 4 at"},
	{"huge count", func(b []byte) []byte {
		put64(b, table(b)+24*5+16, 1<<60)
		return b
	}, place.ErrCorrupt, "out of place"},
	{"a count that wraps to the table", func(b []byte) []byte {
		put64(b, table(b)+24*8+16, 1<<62+2)
		return b
	}, place.ErrCorrupt, "section 9 at"},
	{"a part", func(b []byte) []byte {
		put64(b, table(b)+16, 1)
		put64(b, table(b)+24*1+8, table(b))
		return b
	}, place.ErrCorrupt, "out of place"},
	{"keys out of order", func(b []byte) []byte {
		put32(b, at(b, 4, 1, 0), 5)
		return b
	}, place.ErrCorrupt, "key 5 is not above key 5"},
	{"segment index", func(b []byte) []byte {
		put32(b, at(b, 4, 0, 4), 1)
		return b
	}, place.ErrCorrupt, "key 5 has segment records 1 to 2"},
	{"a tombstone index", func(b []byte) []byte {
		put32(b, at(b, 4, 0, 8), 1)
		return b
	}, place.ErrCorrupt, "names tombstones or parts"},
	{"a part index", func(b []byte) []byte {
		put32(b, at(b, 4, 2, 16), 0)
		return b
	}, place.ErrCorrupt, "names tombstones or parts"},
	{"segment flag", func(b []byte) []byte {
		b[at(b, 5, 2, 27)] = 2
		return b
	}, place.ErrCorrupt, "segment 3 has unknown flags"},
	{"segment reserved byte", func(b []byte) []byte {
		b[at(b, 5, 2, 30)] = 1
		return b
	}, place.ErrCorrupt, "segment 3 has unknown flags or a set reserved byte"},
	{"segments out of order", func(b []byte) []byte {
		put64(b, at(b, 5, 0, 0), 13)
		return b
	}, place.ErrCorrupt, "segment 12 of key 5 is not above segment 13"},
	{"a segment named twice in a key", func(b []byte) []byte {
		put64(b, at(b, 5, 1, 0), 2)
		return b
	}, place.ErrCorrupt, "segment 2 of key 5 is not above segment 2"},
	{"pointer index", func(b []byte) []byte {
		put32(b, at(b, 5, 0, 20), 1)
		return b
	}, place.ErrCorrupt, "segment 2 has pointer records 1 to 1"},
	{"a clock the section leaves out", func(b []byte) []byte {
		put32(b, at(b, 5, 2, 8), 8)
		return b
	}, place.ErrCorrupt, "2 clocks, but the segments name 3"},
	{"a clock the segments do not name", func(b []byte) []byte {
		put32(b, at(b, 8, 1, 0), 8)
		return b
	}, place.ErrCorrupt, "clock 1 is 8"},
	{"intents out of order", func(b []byte) []byte {
		put32(b, at(b, 9, 1, 0), 2)
		return b
	}, place.ErrCorrupt, "intent of key 2 is not above key 2"},
}
