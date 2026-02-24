// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package humanize converts Arc internal types to user-friendly strings.
package humanize

import (
	"github.com/synnaxlabs/arc/types"
)

// Type converts an internal type representation to a user-friendly string.
// Type variables with constraints are displayed by their constraint (e.g., "integer", "float")
// rather than internal names like "$T1".
func Type(t types.Type) string {
	switch t.Kind {
	case types.KindVariable:
		if t.Constraint != nil {
			return constraint(*t.Constraint)
		}
		return "unknown type"
	case types.KindChan:
		if t.Elem != nil {
			return "chan " + Type(*t.Elem)
		}
		return "chan"
	case types.KindSeries:
		if t.Elem != nil {
			return "series " + Type(*t.Elem)
		}
		return "series"
	default:
		return t.String()
	}
}

func constraint(c types.Type) string {
	switch c.Kind {
	case types.KindIntegerConstant:
		return "integer"
	case types.KindFloatConstant:
		return "float"
	case types.KindNumericConstant:
		return "numeric"
	case types.KindExactIntegerFloatConstant:
		return "numeric"
	default:
		return c.String()
	}
}
