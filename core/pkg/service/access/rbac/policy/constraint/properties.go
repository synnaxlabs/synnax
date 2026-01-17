// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
)

// enforceProperties checks if the request's properties match this constraint.
// If the properties match, all requested objects are covered (returned).
// If the properties don't match, no objects are covered (empty slice returned).
func (c Constraint) enforceProperties(params EnforceParams) ([]ontology.ID, error) {
	switch c.Operator {
	case OpContainsAny:
		if len(c.Properties) == 0 {
			return params.Request.Objects, nil
		}
		if slices.ContainsFunc(c.Properties, params.Request.Properties.Contains) {
			return params.Request.Objects, nil
		}
		return nil, nil
	case OpContainsAll:
		for _, property := range c.Properties {
			if !params.Request.Properties.Contains(property) {
				return nil, nil
			}
		}
		return params.Request.Objects, nil
	case OpContainsNone:
		if slices.ContainsFunc(c.Properties, params.Request.Properties.Contains) {
			return nil, nil
		}
		return params.Request.Objects, nil
	default:
		return nil, ErrInvalidOperator
	}
}
