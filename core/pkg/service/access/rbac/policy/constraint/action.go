// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

func (c Constraint) enforceAction(params EnforceParams) (bool, error) {
	switch c.Operator {
	case OpIsIn:
		return c.Actions.Contains(params.Request.Action), nil
	case OpIsNotIn:
		return !c.Actions.Contains(params.Request.Action), nil
	default:
		return false, ErrInvalidOperator
	}
}
