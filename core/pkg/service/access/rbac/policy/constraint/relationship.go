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
	// what does this relationship try to match? this constraint matches when the
	// requested object matches the relationship to one of the given IDs in the
	// constraint. so if we have labeled_by, this constraint matches when the requested
	// object is labeled by one of c.Objects. This means we can start with the
	// constraint objects and then traverse the ontology to find all objects labeled by
	// that. However this is likely to be slow as that is likely to cause a lot of
	// matches. Instead, we can start with the requested objects and then traverse the
	// ontology backwards, and see if we match any or all of the c.Objects.
	var resources []ontology.Resource
	traverser := ontology.Traverser{
		Filter: func(res *ontology.Resource, rel *ontology.Relationship) bool {
			return rel.Type == c.RelationshipType && rel.From == res.ID
		},
		Direction: ontology.Backward,
	}
	if err := params.Ontology.NewRetrieve().WhereIDs(params.Request.Objects...).
		ExcludeFieldData(true).
		TraverseTo(traverser).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil {
		return false, err
	}
	relatedIDs := ontology.ResourceIDs(resources)

	// make a set of type IDs and normal IDs. This way the contains check for each
	// constraint object is O(1) instead of O(n).
	relatedIDsSet := set.FromSlice(relatedIDs)
	relatedTypesSet := set.FromMapSlice(relatedIDs, func(id ontology.ID) ontology.Type {
		return id.Type
	})

	switch c.Operator {
	case OpContainsAny:
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
