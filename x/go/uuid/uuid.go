// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package uuid extends the standard uuid package.
package uuid

import (
	"crypto/sha1"
	"uuid"
)

// NewV5 returns the name-based UUID of name in namespace, as RFC 9562 defines version
// 5. The same namespace and name always give the same UUID.
func NewV5(namespace uuid.UUID, name string) uuid.UUID {
	h := sha1.New()
	h.Write(namespace[:])
	h.Write([]byte(name))
	var u uuid.UUID
	copy(u[:], h.Sum(nil))
	u[6] = (u[6] & 0x0f) | 0x50
	u[8] = (u[8] & 0x3f) | 0x80
	return u
}
