// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

// Uint12 is a 12-bit precision unsigned integer.

type Uint12 uint16

// Uint20 is a 20-bit precision unsigned integer.
type Uint20 uint32

func BoolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
