// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package place computes every checksum of Cesium's files. A checksum is a CRC32C over
// a 32-byte place and then the bytes it covers, so bytes of another database, file,
// offset, kind, version, data type, or clock fail the check. A place is never stored.
package place

import (
	"encoding/binary"
	"hash/crc32"
)

// Size is the number of bytes of an encoded place.
const Size = 32

const (
	kindSegment uint8 = 1
	kindCheck   uint8 = 2
	kindLog     uint8 = 3
	kindHead    uint8 = 4
	kindPart    uint8 = 5
)

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// Place is where covered bytes live. The zero value is not a valid place.
type Place struct {
	db       uint64
	file     uint64
	offset   uint64
	kind     uint8
	dataType uint8
	version  uint16
	clock    uint32
}

// Segment returns the place of bytes at offset in segment file of database db, with
// the segment's format version, data type code, and clock key.
func Segment(
	db, file, offset uint64,
	version uint16,
	dataType uint8,
	clock uint32,
) Place {
	return Place{
		db:       db,
		file:     file,
		offset:   offset,
		kind:     kindSegment,
		dataType: dataType,
		version:  version,
		clock:    clock,
	}
}

// Check returns the place of bytes at offset in the check file of segment file.
func Check(db, file, offset uint64, version uint16) Place {
	return Place{db: db, file: file, offset: offset, kind: kindCheck, version: version}
}

// Log returns the place of bytes at offset in the log of generation.
func Log(db, generation, offset uint64) Place {
	return Place{db: db, file: generation, offset: offset, kind: kindLog, version: 1}
}

// Head returns the place of bytes at offset in the head of generation.
func Head(db, generation, offset uint64) Place {
	return Place{db: db, file: generation, offset: offset, kind: kindHead, version: 1}
}

// Part returns the place of bytes at offset in part file.
func Part(db, file, offset uint64) Place {
	return Place{db: db, file: file, offset: offset, kind: kindPart, version: 1}
}

// At returns p at another offset of the same file.
func (p Place) At(offset uint64) Place {
	p.offset = offset
	return p
}

// Start returns the CRC state after p, from which Update continues over the covered
// bytes.
func (p Place) Start() uint32 {
	var b [Size]byte
	binary.LittleEndian.PutUint64(b[0:], p.db)
	binary.LittleEndian.PutUint64(b[8:], p.file)
	binary.LittleEndian.PutUint64(b[16:], p.offset)
	b[24] = p.kind
	b[25] = p.dataType
	binary.LittleEndian.PutUint16(b[26:], p.version)
	binary.LittleEndian.PutUint32(b[28:], p.clock)
	return crc32.Update(0, castagnoli, b[:])
}

// Sum returns the checksum of b at p.
func (p Place) Sum(b []byte) uint32 { return Update(p.Start(), b) }

// Update returns the CRC state crc after b. A state from Start, updated over the
// covered bytes in any number of calls, is their checksum.
func Update(crc uint32, b []byte) uint32 { return crc32.Update(crc, castagnoli, b) }
