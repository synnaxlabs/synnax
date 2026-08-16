// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

// Equal checks if two units are identical.
func (u Unit) Equal(other Unit) bool {
	return u.Dimensions.Equal(other.Dimensions) &&
		u.Scale == other.Scale &&
		u.Name == other.Name
}
