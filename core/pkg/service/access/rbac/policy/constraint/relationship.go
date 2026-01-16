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
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/set"
)

func (c Constraint) enforceRelationship(
	ctx context.Context,
	params EnforceParams,
) (bool, error) {
	var resources []ontology.Resource
	traverser := ontology.Traverser{
		Filter: func(res *ontology.Resource, rel *ontology.Relationship) bool {
			return rel.Type == c.RelationshipType && rel.From == res.ID
		},
		Direction: ontology.Forward,
	}
	if err := params.Ontology.NewRetrieve().WhereIDs(params.Request.Objects...).
		ExcludeFieldData(true).
		TraverseTo(traverser).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil {
		return false, err
	}
	relatedIDs := ontology.ResourceIDs(resources)
	relatedIDsSet := set.FromSlice(relatedIDs)
	relatedTypesSet := set.FromMapSlice(relatedIDs, func(id ontology.ID) ontology.Type {
		return id.Type
	})

	switch c.Operator {
	case OpContainsAny:
		if len(c.IDs) == 0 {
			return true, nil
		}
		for _, id := range c.IDs {
			if id.IsType() {
				if relatedTypesSet.Contains(id.Type) {
					return true, nil
				}
			} else {
				if relatedIDsSet.Contains(id) {
					return true, nil
				}
			}
		}
		return false, nil
	case OpContainsAll:
		for _, id := range c.IDs {
			if id.IsType() {
				if !relatedTypesSet.Contains(id.Type) {
					return false, nil
				}
			} else {
				if !relatedIDsSet.Contains(id) {
					return false, nil
				}
			}
		}
		return true, nil
	case OpContainsNone:
		for _, id := range c.IDs {
			if id.IsType() {
				if relatedTypesSet.Contains(id.Type) {
					return false, nil
				}
			} else {
				if relatedIDsSet.Contains(id) {
					return false, nil
				}
			}
		}
		return true, nil
	default:
		return false, ErrInvalidOperator
	}
}
