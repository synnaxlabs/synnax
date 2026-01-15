// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import "context"

func (c Constraint) enforceLogical(
	ctx context.Context,
	params EnforceParams,
) (bool, error) {
	switch c.Operator {
	case OpContainsAny:
		if len(c.Constraints) == 0 {
			return true, nil
		}
		for _, constraint := range c.Constraints {
			if ok, err := constraint.Enforce(ctx, params); err != nil || ok {
				return ok, err
			}
		}
		return false, nil
	case OpContainsAll:
		for _, constraint := range c.Constraints {
			if ok, err := constraint.Enforce(ctx, params); err != nil || !ok {
				return ok, err
			}
		}
		return true, nil
	case OpContainsNone:
		for _, constraint := range c.Constraints {
			if ok, err := constraint.Enforce(ctx, params); err != nil || ok {
				return false, err
			}
		}
		return true, nil
	default:
		return false, ErrInvalidOperator
	}
}
