// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

// Numeric represents a generic numeric value.
type Numeric interface {
	~uint | ~int | ~float64 | ~float32 | ~int64 | ~int32 | ~int16 | ~int8 | ~uint64 | ~uint32 | ~uint16 | ~uint8
}
