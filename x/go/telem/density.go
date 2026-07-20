// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import v0 "github.com/synnaxlabs/x/telem/types/v0"

// Density represents a density in bytes per value.
type Density = v0.Density

const (
	// UnknownDensity is for type structure occupying an unknown number of bytes.
	UnknownDensity = v0.UnknownDensity
	// Bit128 is for a type occupying 16 bytes.
	Bit128 = v0.Bit128
	// Bit64 is for a type occupying 8 bytes.
	Bit64 = v0.Bit64
	// Bit32 is for a type occupying 4 bytes.
	Bit32 = v0.Bit32
	// Bit16 is for a data type occupying 2 bytes.
	Bit16 = v0.Bit16
	// Bit8 is for a data type occupying 1 byte.
	Bit8 = v0.Bit8
)
