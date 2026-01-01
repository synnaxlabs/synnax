// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bounds

import (
	"fmt"

	"github.com/synnaxlabs/x/types"
)

// Bounds are a set of numeric lower and upper values that define a bound of values.
// Bounds are start inclusive and end exclusive.
type Bounds[V types.Numeric] struct {
	// Lower is the lower value for the bounds.
	Lower V
	// Upper is the upper value for the bounds.
	Upper V
}

// Contains returns true if the bounds contain the given value i.e., the v is greater
// than or equal to b.Lower and strictly less than b.Upper.
func (b Bounds[V]) Contains(v V) bool {
	return b.Lower <= v && v < b.Upper
}

// String implements fmt.Stringer to return a nicely formatted string representation
// of the bounds.
func (b Bounds[V]) String() string { return fmt.Sprintf("Bounds[%v, %v)", b.Lower, b.Upper) }

func (b Bounds[V]) Span() V {
	return b.Upper - b.Lower
}
