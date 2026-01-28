// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bit

import "github.com/synnaxlabs/x/types"

// FlagPos tracks the location of a bit flag within it's parent byte. The best way to
// use this type is to assign it to a global variable with the bit position and then
// call the Get and Set methods to work with the flag value.
type FlagPos int

// Get returns true if the bit at the flag position is 1, and false if it is 0.
func (f FlagPos) Get(b byte) bool {
	return ((b >> f) & 1) == 1
}

// Set sets the bit at the flag position to 1 if the value is true, and false if it is
// 0.
func (f FlagPos) Set(b byte, value bool) byte {
	mask := ^(byte(1) << f)
	b &= mask
	v := types.BoolToUint8(value) << f
	return b | v
}
