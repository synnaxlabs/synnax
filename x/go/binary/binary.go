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

	"github.com/synnaxlabs/x/errors"
)

// MakeCopy returns a copy of the given byte slice.
func MakeCopy(bytes []byte) []byte {
	copied := make([]byte, len(bytes))
	copy(copied, bytes)
	return copied
}

// UVarint decodes a varint from a byte slice. It wraps the encoding/binary.UVarint
// function and returns an error if the varint is not found.
func UVarint(buf []byte) (uint64, int, error) {
	value, n := binary.Uvarint(buf)
	if value != 0 && n > 0 {
		return value, n, nil
	}
	if n == 0 {
		return 0, 0, errors.New("buffer too small")
	}
	return 0, 0, errors.New("varint too large")
}

// VarintLength returns the number of bytes required to encode the given uint as a
// varint.
func VarintLength(x uint) int {
	switch {
	case x < 1<<(7*1):
		return 1
	case x < 1<<(7*2):
		return 2
	case x < 1<<(7*3):
		return 3
	case x < 1<<(7*4):
		return 4
	case x < 1<<(7*5):
		return 5
	case x < 1<<(7*6):
		return 6
	case x < 1<<(7*7):
		return 7
	case x < 1<<(7*8):
		return 8
	case x < 1<<(7*9):
		return 9
	default:
		return 10
	}
}
