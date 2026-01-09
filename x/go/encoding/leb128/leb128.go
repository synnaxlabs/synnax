// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package leb128

import (
	"io"
)

// WriteUnsigned writes an unsigned LEB128 encoded integer to the given writer.
// LEB128 (Little Endian Base 128) is a variable-length encoding used in WASM and other formats.
func WriteUnsigned(w io.ByteWriter, val uint64) error {
	for {
		b := byte(val & 0x7f)
		val >>= 7
		if val != 0 {
			b |= 0x80
		}
		if err := w.WriteByte(b); err != nil {
			return err
		}
		if val == 0 {
			break
		}
	}
	return nil
}

// WriteSigned writes a signed LEB128 encoded integer to the given writer.
// Note: This is NOT the same as Go's binary.PutVarint, which uses zigzag encoding.
// LEB128 signed encoding uses two's complement representation.
func WriteSigned(w io.ByteWriter, val int64) error {
	for {
		b := byte(val & 0x7f)
		signBit := b & 0x40
		val >>= 7
		if (val == 0 && signBit == 0) || (val == -1 && signBit != 0) {
			if err := w.WriteByte(b); err != nil {
				return err
			}
			break
		}
		if err := w.WriteByte(b | 0x80); err != nil {
			return err
		}
	}
	return nil
}
