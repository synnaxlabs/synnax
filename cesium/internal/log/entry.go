// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"encoding/binary"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
)

// Entry builds one entry, which applies all or nothing: its bytes and its records. The
// zero value is an empty entry.
type Entry struct {
	b       []byte
	records []Record
	commit  bool
}

// Add appends r to the entry. The entry keeps r, and Log.Append hands it to the
// OnDurable hook of the log, so r must not change after Add.
func (e *Entry) Add(r Record) {
	if len(e.b) == 0 {
		e.b = append(e.b, 0, 0, 0, 0, 0, 0, 0, 0)
	}
	e.b = r.append(e.b)
	e.records = append(e.records, r)
	if _, ok := r.(*Commit); ok {
		e.commit = true
	}
}

// Len returns the number of records in the entry.
func (e *Entry) Len() int { return len(e.records) }

// Bytes returns the encoded entry. The bytes stay valid until the next Add or Reset.
func (e *Entry) Bytes() []byte {
	if len(e.b) == 0 {
		e.b = append(e.b, 0, 0, 0, 0, 0, 0, 0, 0)
	}
	binary.LittleEndian.PutUint32(e.b, uint32(len(e.b)-headerSize))
	binary.LittleEndian.PutUint32(e.b[4:], uint32(len(e.records)))
	return e.b
}

// Reset empties the entry and keeps its byte buffer. It does not reuse the slice of
// records that an Append took.
func (e *Entry) Reset() {
	e.b = e.b[:0]
	e.records = nil
	e.commit = false
}

// Decoder reads the records of entries. It owns one record of each type, so it
// allocates nothing per record. The zero value is ready for Reset.
type Decoder struct {
	b         []byte
	left      uint32
	reserve   Reserve
	create    Create
	commit    Commit
	seal      Seal
	tombstone Tombstone
	replace   Replace
	drop      Drop
	restart   Restart
	check     Check
	intent    Intent
}

// Reset starts reading the entry at the start of b, and returns the length of the
// entry in b.
func (d *Decoder) Reset(b []byte) (int, error) {
	if len(b) < headerSize {
		return 0, errors.Wrapf(
			place.ErrCorrupt,
			"entry header needs 8 bytes, but %d remain",
			len(b),
		)
	}
	size := binary.LittleEndian.Uint32(b)
	count := binary.LittleEndian.Uint32(b[4:])
	if uint64(size) > uint64(len(b)-headerSize) {
		return 0, errors.Wrapf(
			place.ErrCorrupt,
			"entry holds %d bytes of records, but %d remain",
			size,
			len(b)-headerSize,
		)
	}
	d.b = b[headerSize : headerSize+int(size)]
	d.left = count
	return headerSize + int(size), nil
}

// Next returns the next record of the entry, or false at its end. The record stays
// valid until the next call. Next skips a record of an unknown type that an older
// binary may skip, and returns ErrUnknown for one that it may not.
func (d *Decoder) Next() (Record, bool, error) {
	for d.left > 0 {
		if len(d.b) < headerSize {
			return nil, false, errors.Wrapf(
				place.ErrCorrupt,
				"entry ends with %d records left",
				d.left,
			)
		}
		t := binary.LittleEndian.Uint16(d.b)
		zero := binary.LittleEndian.Uint16(d.b[2:])
		size := binary.LittleEndian.Uint32(d.b[4:])
		if zero != 0 {
			return nil, false, errors.Wrapf(
				place.ErrCorrupt,
				"record of type %d has a set reserved field",
				t,
			)
		}
		if uint64(size) > uint64(len(d.b)-headerSize) {
			return nil, false, errors.Wrapf(
				place.ErrCorrupt,
				"record of type %d holds %d bytes, but %d remain in its entry",
				t,
				size,
				len(d.b)-headerSize,
			)
		}
		if size%8 != 0 {
			return nil, false, errors.Wrapf(
				place.ErrCorrupt,
				"record of type %d holds %d bytes, not a multiple of 8",
				t,
				size,
			)
		}
		body := d.b[headerSize : headerSize+int(size)]
		d.b = d.b[headerSize+int(size):]
		d.left--
		r := d.record(t)
		if r == nil {
			if t&typeSkippable != 0 {
				continue
			}
			return nil, false, errors.Wrapf(ErrUnknown, "record type %d", t)
		}
		if err := r.decode(body); err != nil {
			return nil, false, err
		}
		return r, true, nil
	}
	if len(d.b) != 0 {
		return nil, false, errors.Wrapf(
			place.ErrCorrupt,
			"entry has %d bytes after its last record",
			len(d.b),
		)
	}
	return nil, false, nil
}

func (d *Decoder) record(t uint16) Record {
	switch t {
	case typeReserve:
		return &d.reserve
	case typeCreate:
		return &d.create
	case typeCommit:
		return &d.commit
	case typeSeal:
		return &d.seal
	case typeTombstone:
		return &d.tombstone
	case typeReplace:
		return &d.replace
	case typeDrop:
		return &d.drop
	case typeRestart:
		return &d.restart
	case typeCheck:
		return &d.check
	case typeIntent:
		return &d.intent
	default:
		return nil
	}
}
