// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import "github.com/synnaxlabs/synnax/pkg/distribution/ontology"

// enforceAction checks if the request's action matches this constraint.
// If the action matches, all requested objects are covered (returned).
// If the action doesn't match, no objects are covered (empty slice returned).
func (c Constraint) enforceAction(params EnforceParams) ([]ontology.ID, error) {
	switch c.Operator {
	case OpIsIn:
		if c.Actions.Contains(params.Request.Action) {
			return params.Request.Objects, nil
		}
		return nil, nil
	case OpIsNotIn:
		if !c.Actions.Contains(params.Request.Action) {
			return params.Request.Objects, nil
		}
		return nil, nil
	default:
		return nil, ErrInvalidOperator
	}
}
