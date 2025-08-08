// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// package uuid provides utilities for using uuids.
package uuid

import (
	"encoding/hex"

	"github.com/google/uuid"
)

// EncodeStringBytes encodes the bytes of the string form of the given uuid into the
// provided byte slice. The provided byte slice must be at least 36 bytes long.
func EncodeStringBytes(dst []byte, uuid uuid.UUID) {
	hex.Encode(dst, uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:], uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:], uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:], uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], uuid[10:])
}
