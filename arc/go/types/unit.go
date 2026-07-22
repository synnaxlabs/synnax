// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

// UnitsAssignable reports whether a value with unit a may be assigned to a
// target with unit b. A nil unit on either side is treated as a wildcard,
// matching the WASM and Cesium representation where i64 and i64 ns share the
// same wire type (timestamp <-> int64). Two non-nil units must match exactly,
// so f32 psi vs f32 bar still fails.
func UnitsAssignable(a, b *Unit) bool {
	if a == nil || b == nil {
		return true
	}
	return a.Equal(*b)
}
