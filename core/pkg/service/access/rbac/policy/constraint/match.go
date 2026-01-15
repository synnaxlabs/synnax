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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/set"
)

func (c Constraint) enforceMatch(params EnforceParams) (bool, error) {
	requestedIDsSet := set.FromSlice(params.Request.Objects)
	requestedTypesSet := set.FromMapSlice(params.Request.Objects, func(id ontology.ID) ontology.Type {
		return id.Type
	})
	switch c.Operator {
	case OpContainsAny:
		for _, id := range c.IDs {
			if id.IsType() {
				if requestedTypesSet.Contains(id.Type) {
					return true, nil
				}
			} else {
				if requestedIDsSet.Contains(id) {
					return true, nil
				}
			}
		}
		return false, nil
	case OpContainsAll:
		for _, id := range c.IDs {
			if id.IsType() {
				if !requestedTypesSet.Contains(id.Type) {
					return false, nil
				}
			} else {
				if !requestedIDsSet.Contains(id) {
					return false, nil
				}
			}
		}
		return true, nil
	case OpContainsNone:
		for _, id := range c.IDs {
			if id.IsType() {
				if requestedTypesSet.Contains(id.Type) {
					return false, nil
				}
			} else {
				if requestedIDsSet.Contains(id) {
					return false, nil
				}
			}
		}
		return true, nil
	default:
		return false, ErrInvalidOperator
	}
}
