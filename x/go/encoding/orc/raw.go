// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package orc

import (
	"encoding/binary"
	"math"
)

// Raw provides zero-allocation navigation of orc-encoded binary data. Callers
// can skip or read individual fields by type without fully deserializing the
// record. A nil Raw signals that previous navigation went out of bounds.
type Raw []byte

// NewRaw creates a Raw from orc-encoded data, stripping the magic header.
func NewRaw(data []byte) (Raw, error) {
	if err := validateMagic(data); err != nil {
		return nil, err
	}
	return data[len(magic):], nil
}

// Skip advances past n bytes of fixed-size data. Returns nil if there isn't
// enough data.
func (r Raw) Skip(n int) Raw {
	if len(r) < n {
		return nil
	}
	return r[n:]
}

// SkipLenPrefixed advances past one length-prefixed field (string, bytes, or
// nested record).
func (r Raw) SkipLenPrefixed() Raw {
	if len(r) < 4 {
		return nil
	}
	fieldLen := int(binary.BigEndian.Uint32(r[:4]))
	r = r[4:]
	if len(r) < fieldLen {
		return nil
	}
	return r[fieldLen:]
}

// SkipString advances past one length-prefixed string field.
func (r Raw) SkipString() Raw { return r.SkipLenPrefixed() }

// SkipStrings advances past n consecutive length-prefixed string fields.
func (r Raw) SkipStrings(n int) Raw {
	for range n {
		r = r.SkipString()
		if r == nil {
			return nil
		}
	}
	return r
}

// SkipBool advances past a 1-byte boolean field.
func (r Raw) SkipBool() Raw { return r.Skip(1) }

// SkipUint8 advances past a 1-byte uint8 field.
func (r Raw) SkipUint8() Raw { return r.Skip(1) }

// SkipUint16 advances past a 2-byte uint16 field.
func (r Raw) SkipUint16() Raw { return r.Skip(2) }

// SkipUint32 advances past a 4-byte uint32 field.
func (r Raw) SkipUint32() Raw { return r.Skip(4) }

// SkipUint64 advances past an 8-byte uint64 field.
func (r Raw) SkipUint64() Raw { return r.Skip(8) }

// SkipInt8 advances past a 1-byte int8 field.
func (r Raw) SkipInt8() Raw { return r.Skip(1) }

// SkipInt16 advances past a 2-byte int16 field.
func (r Raw) SkipInt16() Raw { return r.Skip(2) }

// SkipInt32 advances past a 4-byte int32 field.
func (r Raw) SkipInt32() Raw { return r.Skip(4) }

// SkipInt64 advances past an 8-byte int64 field.
func (r Raw) SkipInt64() Raw { return r.Skip(8) }

// SkipFloat32 advances past a 4-byte float32 field.
func (r Raw) SkipFloat32() Raw { return r.Skip(4) }

// SkipFloat64 advances past an 8-byte float64 field.
func (r Raw) SkipFloat64() Raw { return r.Skip(8) }

// ReadLenPrefixed reads one length-prefixed field, returning the field value
// and the remaining Raw. Returns (nil, nil) if the data is too short.
func (r Raw) ReadLenPrefixed() (value []byte, rest Raw) {
	if len(r) < 4 {
		return nil, nil
	}
	fieldLen := int(binary.BigEndian.Uint32(r[:4]))
	r = r[4:]
	if len(r) < fieldLen {
		return nil, nil
	}
	return r[:fieldLen], r[fieldLen:]
}

// ReadString reads one length-prefixed string field, returning the string
// bytes and the remaining Raw.
func (r Raw) ReadString() (value []byte, rest Raw) { return r.ReadLenPrefixed() }

// ReadBool reads a 1-byte boolean field.
func (r Raw) ReadBool() (bool, Raw) {
	if len(r) < 1 {
		return false, nil
	}
	return r[0] != 0, r[1:]
}

// ReadUint8 reads a 1-byte uint8 field.
func (r Raw) ReadUint8() (uint8, Raw) {
	if len(r) < 1 {
		return 0, nil
	}
	return r[0], r[1:]
}

// ReadUint16 reads a 2-byte big-endian uint16 field.
func (r Raw) ReadUint16() (uint16, Raw) {
	if len(r) < 2 {
		return 0, nil
	}
	return binary.BigEndian.Uint16(r[:2]), r[2:]
}

// ReadUint32 reads a 4-byte big-endian uint32 field.
func (r Raw) ReadUint32() (uint32, Raw) {
	if len(r) < 4 {
		return 0, nil
	}
	return binary.BigEndian.Uint32(r[:4]), r[4:]
}

// ReadUint64 reads an 8-byte big-endian uint64 field.
func (r Raw) ReadUint64() (uint64, Raw) {
	if len(r) < 8 {
		return 0, nil
	}
	return binary.BigEndian.Uint64(r[:8]), r[8:]
}

// ReadInt8 reads a 1-byte int8 field.
func (r Raw) ReadInt8() (int8, Raw) {
	v, rest := r.ReadUint8()
	return int8(v), rest
}

// ReadInt16 reads a 2-byte big-endian int16 field.
func (r Raw) ReadInt16() (int16, Raw) {
	v, rest := r.ReadUint16()
	return int16(v), rest
}

// ReadInt32 reads a 4-byte big-endian int32 field.
func (r Raw) ReadInt32() (int32, Raw) {
	v, rest := r.ReadUint32()
	return int32(v), rest
}

// ReadInt64 reads a 8-byte big-endian int64 field.
func (r Raw) ReadInt64() (int64, Raw) {
	v, rest := r.ReadUint64()
	return int64(v), rest
}

// ReadFloat32 reads a 4-byte big-endian float32 field.
func (r Raw) ReadFloat32() (float32, Raw) {
	v, rest := r.ReadUint32()
	return math.Float32frombits(v), rest
}

// ReadFloat64 reads an 8-byte big-endian float64 field.
func (r Raw) ReadFloat64() (float64, Raw) {
	v, rest := r.ReadUint64()
	return math.Float64frombits(v), rest
}
