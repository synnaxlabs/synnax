// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary

import (
	"encoding/binary"
	"io"
)

// WriteLEB128Unsigned writes an unsigned LEB128 encoded integer to the given writer.
// LEB128 (Little Endian Base 128) is a variable-length encoding used in WASM and other formats.
func WriteLEB128Unsigned(w io.ByteWriter, val uint64) {
	for {
		b := byte(val & 0x7f)
		val >>= 7
		if val != 0 {
			b |= 0x80
		}
		_ = w.WriteByte(b)
		if val == 0 {
			break
		}
	}
}

// WriteLEB128Signed writes a signed LEB128 encoded integer to the given writer.
// Note: This is NOT the same as Go's binary.PutVarint, which uses zigzag encoding.
// LEB128 signed encoding uses two's complement representation.
func WriteLEB128Signed(w io.ByteWriter, val int64) {
	for {
		b := byte(val & 0x7f)
		signBit := b & 0x40
		val >>= 7
		if (val == 0 && signBit == 0) || (val == -1 && signBit != 0) {
			_ = w.WriteByte(b)
			break
		}
		_ = w.WriteByte(b | 0x80)
	}
}

// AppendLEB128Unsigned appends an unsigned LEB128 encoded integer to a byte slice.
func AppendLEB128Unsigned(dst []byte, val uint64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, val)
	return append(dst, buf[:n]...)
}

// AppendLEB128Signed appends a signed LEB128 encoded integer to a byte slice.
func AppendLEB128Signed(dst []byte, val int64) []byte {
	for {
		b := byte(val & 0x7f)
		signBit := b & 0x40
		val >>= 7
		if (val == 0 && signBit == 0) || (val == -1 && signBit != 0) {
			return append(dst, b)
		}
		dst = append(dst, b|0x80)
	}
}
