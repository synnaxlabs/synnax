// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

// Density represents a density in bytes per value.
type Density uint32

// SampleCount returns the number of samples within the number of bytes provided in
// Size.
// For example,
//
//	Bit64.SampleCount(16) == 2
func (d Density) SampleCount(size Size) int64 {
	if d == 0 {
		panic("attempted to call SampleCount() on undefined density")
	}
	return int64(size) / int64(d)
}

// Size returns the number of bytes occupied by the number of samples provided
// in sampleCount.
// For example,
//
//	Bit64.Size(2) == 16
func (d Density) Size(sampleCount int64) Size {
	if d == 0 {
		panic("attempted to call Size() on undefined density")
	}
	return Size(sampleCount) * Size(d)
}

const (
	// UnknownDensity is for type structure occupying an unknown number of bytes.
	UnknownDensity Density = 0
	// Bit128 is for a type occupying 16 bytes.
	Bit128 Density = 16
	// Bit64 is for a type occupying 8 bytes.
	Bit64 Density = 8
	// Bit32 is for a type occupying 4 bytes.
	Bit32 Density = 4
	// Bit16 is for a data type occupying 2 bytes.
	Bit16 Density = 2
	// Bit8 is for a data type occupying 1 byte.
	Bit8 Density = 1
)
