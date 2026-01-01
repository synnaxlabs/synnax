// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

// Uint12 is a 12-bit precision unsigned integer that uses uint16 as it's underlying
// type. It is designed to represent values with a maximum precision of 2^12 (4096).
type Uint12 uint16

// Uint20 is a 20-bit precision unsigned integer that uses uint20 as it's underlying
// type. It is designed to represent values with a maximum precision of 2^20 (1048576)
type Uint20 uint32

// BoolToUint8 returns uint8(1) if b is true and uint8(0) if b is false.
func BoolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
