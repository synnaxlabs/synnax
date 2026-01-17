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

// enforceMatch returns the subset of requested objects that match this constraint's IDs.
// The behavior depends on the operator:
//   - OpContainsAny: Returns requested objects that match ANY of the constraint's IDs
//   - OpContainsAll: If ALL constraint IDs are present in the request, returns all
//     requested objects; otherwise returns empty
//   - OpContainsNone: Returns requested objects that match NONE of the constraint's IDs
func (c Constraint) enforceMatch(params EnforceParams) ([]ontology.ID, error) {
	constraintIDsSet := set.FromSlice(c.IDs)
	// Only collect types from type-only IDs (where Key is empty)
	constraintTypesSet := make(set.Set[ontology.Type])
	for _, id := range c.IDs {
		if id.IsType() {
			constraintTypesSet.Add(id.Type)
		}
	}

	switch c.Operator {
	case OpContainsAny:
		if len(c.IDs) == 0 {
			// No constraint IDs means all objects match
			return params.Request.Objects, nil
		}
		// Return objects that match any constraint ID
		var matched []ontology.ID
		for _, obj := range params.Request.Objects {
			if constraintIDsSet.Contains(obj) || constraintTypesSet.Contains(obj.Type) {
				matched = append(matched, obj)
			}
		}
		return matched, nil

	case OpContainsAll:
		// Check if ALL constraint IDs are present in the request
		requestedIDsSet := set.FromSlice(params.Request.Objects)
		requestedTypesSet := set.FromMapSlice(params.Request.Objects, func(id ontology.ID) ontology.Type {
			return id.Type
		})
		for _, id := range c.IDs {
			if id.IsType() {
				if !requestedTypesSet.Contains(id.Type) {
					return nil, nil
				}
			} else {
				if !requestedIDsSet.Contains(id) {
					return nil, nil
				}
			}
		}
		// All constraint IDs present, so all requested objects are covered
		return params.Request.Objects, nil

	case OpContainsNone:
		// Return objects that don't match any constraint ID
		var matched []ontology.ID
		for _, obj := range params.Request.Objects {
			if !constraintIDsSet.Contains(obj) && !constraintTypesSet.Contains(obj.Type) {
				matched = append(matched, obj)
			}
		}
		return matched, nil

	default:
		return nil, ErrInvalidOperator
	}
}
