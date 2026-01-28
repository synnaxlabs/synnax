// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package diagnostics

import (
	"github.com/synnaxlabs/arc/types"
)

// HumanizeType converts an internal type representation to a user-friendly string.
// Type variables with constraints are displayed by their constraint (e.g., "integer", "float")
// rather than internal names like "$T1".
func HumanizeType(t types.Type) string {
	switch t.Kind {
	case types.KindVariable:
		if t.Constraint != nil {
			return humanizeConstraint(*t.Constraint)
		}
		return "unknown type"
	case types.KindChan:
		if t.Elem != nil {
			return "chan " + HumanizeType(*t.Elem)
		}
		return "chan"
	case types.KindSeries:
		if t.Elem != nil {
			return "series " + HumanizeType(*t.Elem)
		}
		return "series"
	default:
		return t.String()
	}
}

func humanizeConstraint(c types.Type) string {
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
