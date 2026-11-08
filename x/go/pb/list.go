// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

// ListValues returns the values of a list wrapper that is present. Protobuf decodes an
// empty list as nil, so it returns an empty slice for nil, keeping an empty list apart
// from an absent one.
func ListValues[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
