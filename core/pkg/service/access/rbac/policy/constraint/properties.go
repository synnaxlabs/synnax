// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import "slices"

func (c Constraint) enforceProperties(params EnforceParams) (bool, error) {
	switch c.Operator {
	case OpContainsAny:
		if slices.ContainsFunc(c.Properties, params.Request.Properties.Contains) {
			return true, nil
		}
		return false, nil
	case OpContainsAll:
		for _, property := range c.Properties {
			if !params.Request.Properties.Contains(property) {
				return false, nil
			}
		}
		return true, nil
	case OpContainsNone:
		if slices.ContainsFunc(c.Properties, params.Request.Properties.Contains) {
			return false, nil
		}
		return true, nil
	default:
		return false, ErrInvalidOperator
	}
}
