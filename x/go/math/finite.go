// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math

import (
	"math"

	"github.com/synnaxlabs/x/types"
)

// Finite returns true when v is neither NaN nor infinite.
func Finite[T types.Numeric](v T) bool {
	f := float64(v)
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
