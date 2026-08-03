// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bit

// Mask128 is a bitmask with 128 values.
type Mask128 [16]byte

func (m Mask128) Swap(i, j int) Mask128 {
	if i < 0 || i >= 128 || j < 0 || j >= 128 {
		panic("mask: position out of bounds")
	}
	if i == j {
		return m
	}
	iVal := m.Get(i)
	jVal := m.Get(j)
	return m.Set(i, jVal).Set(j, iVal)
}

// Cap returns the capacity of the mask in bits.
func (m Mask128) Cap() int { return 128 }

// TrueCount returns the number of 1s in the mask.
func (m Mask128) TrueCount() int {
	count := 0
	for _, b := range m {
		for b != 0 {
			count += int(b & 1)
			b >>= 1
		}
	}
	return count
}

// Set sets the value at the given position to true or false depending on val.
func (m Mask128) Set(pos int, val bool) Mask128 {
	if pos < 0 || pos >= 128 {
		panic("mask: position out of bounds")
	}
	if val {
		m[pos/8] = m[pos/8] | (1 << (pos % 8))
	} else {
		m[pos/8] = m[pos/8] & ^(1 << (pos % 8))
	}
	return m
}

// Get returns the value at the given position.
func (m Mask128) Get(pos int) bool {
	if pos < 0 || pos >= 128 {
		panic("mask: position out of bounds")
	}
	return m[pos/8]&(1<<(pos%8)) != 0
}
