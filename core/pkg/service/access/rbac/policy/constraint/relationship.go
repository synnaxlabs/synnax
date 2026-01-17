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

// enforceRelationship checks which requested objects have the specified relationships.
// It traverses the ontology graph backward from constraint IDs to efficiently find
// all objects that have the required relationships.
func (c Constraint) enforceRelationship(
	ctx context.Context,
	params EnforceParams,
) ([]ontology.ID, error) {
	if len(params.Request.Objects) == 0 {
		return nil, nil
	}

	// Validate operator first before any early returns
	switch c.Operator {
	case OpContainsAny, OpContainsAll, OpContainsNone:
		// Valid operators
	default:
		return nil, ErrInvalidOperator
	}

	if len(c.IDs) == 0 {
		// No constraint IDs means all objects pass
		return params.Request.Objects, nil
	}

	// Separate specific IDs from type-only IDs
	var specificIDs []ontology.ID
	typeOnlyTypes := make(set.Set[ontology.Type])
	for _, id := range c.IDs {
		if id.IsType() {
			typeOnlyTypes.Add(id.Type)
		} else {
			specificIDs = append(specificIDs, id)
		}
	}

	// If we have type-only constraints, we need forward traversal for those
	hasTypeOnly := len(typeOnlyTypes) > 0

	switch c.Operator {
	case OpContainsAny:
		return c.enforceRelationshipContainsAny(ctx, params, specificIDs, typeOnlyTypes, hasTypeOnly)
	case OpContainsAll:
		return c.enforceRelationshipContainsAll(ctx, params, specificIDs, typeOnlyTypes, hasTypeOnly)
	case OpContainsNone:
		return c.enforceRelationshipContainsNone(ctx, params, specificIDs, typeOnlyTypes, hasTypeOnly)
	default:
		return nil, ErrInvalidOperator
	}
}

// enforceRelationshipContainsAny returns requested objects that have a relationship
// to ANY of the constraint IDs.
func (c Constraint) enforceRelationshipContainsAny(
	ctx context.Context,
	params EnforceParams,
	specificIDs []ontology.ID,
	typeOnlyTypes set.Set[ontology.Type],
	hasTypeOnly bool,
) ([]ontology.ID, error) {
	requestedSet := set.FromSlice(params.Request.Objects)
	coveredSet := make(set.Set[ontology.ID])

	// For specific IDs, traverse backward to find objects related to them
	if len(specificIDs) > 0 {
		relatedObjects, err := c.traverseBackward(ctx, params, specificIDs)
		if err != nil {
			return nil, err
		}
		for _, obj := range relatedObjects {
			if requestedSet.Contains(obj) {
				coveredSet.Add(obj)
			}
		}
	}

	// For type-only IDs, use forward traversal from requested objects
	if hasTypeOnly {
		for _, obj := range params.Request.Objects {
			if coveredSet.Contains(obj) {
				continue // Already covered by specific ID match
			}
			relatedTypes, err := c.getRelatedTypes(ctx, params, obj)
			if err != nil {
				return nil, err
			}
			for t := range typeOnlyTypes {
				if relatedTypes.Contains(t) {
					coveredSet.Add(obj)
					break
				}
			}
		}
	}

	return coveredSet.Keys(), nil
}

// enforceRelationshipContainsAll returns requested objects that have relationships
// to ALL of the constraint IDs.
func (c Constraint) enforceRelationshipContainsAll(
	ctx context.Context,
	params EnforceParams,
	specificIDs []ontology.ID,
	typeOnlyTypes set.Set[ontology.Type],
	hasTypeOnly bool,
) ([]ontology.ID, error) {
	// Start with all requested objects as candidates
	candidateSet := set.FromSlice(params.Request.Objects)

	// For each specific constraint ID, narrow down candidates
	for _, constraintID := range specificIDs {
		relatedObjects, err := c.traverseBackward(ctx, params, []ontology.ID{constraintID})
		if err != nil {
			return nil, err
		}
		relatedSet := set.FromSlice(relatedObjects)
		// Keep only candidates that are related to this constraint ID
		for candidate := range candidateSet {
			if !relatedSet.Contains(candidate) {
				delete(candidateSet, candidate)
			}
		}
		if len(candidateSet) == 0 {
			return nil, nil // No candidates left
		}
	}

	// For type-only constraints, check each candidate via forward traversal
	if hasTypeOnly {
		for candidate := range candidateSet {
			relatedTypes, err := c.getRelatedTypes(ctx, params, candidate)
			if err != nil {
				return nil, err
			}
			// Check if candidate has relationships to all required types
			for t := range typeOnlyTypes {
				if !relatedTypes.Contains(t) {
					delete(candidateSet, candidate)
					break
				}
			}
		}
	}

	return candidateSet.Keys(), nil
}

// enforceRelationshipContainsNone returns requested objects that have NO relationships
// to any of the constraint IDs.
func (c Constraint) enforceRelationshipContainsNone(
	ctx context.Context,
	params EnforceParams,
	specificIDs []ontology.ID,
	typeOnlyTypes set.Set[ontology.Type],
	hasTypeOnly bool,
) ([]ontology.ID, error) {
	excludeSet := make(set.Set[ontology.ID])

	// For specific IDs, traverse backward to find objects to exclude
	if len(specificIDs) > 0 {
		relatedObjects, err := c.traverseBackward(ctx, params, specificIDs)
		if err != nil {
			return nil, err
		}
		for _, obj := range relatedObjects {
			excludeSet.Add(obj)
		}
	}

	// For type-only IDs, check each requested object via forward traversal
	if hasTypeOnly {
		for _, obj := range params.Request.Objects {
			if excludeSet.Contains(obj) {
				continue // Already excluded
			}
			relatedTypes, err := c.getRelatedTypes(ctx, params, obj)
			if err != nil {
				return nil, err
			}
			for t := range typeOnlyTypes {
				if relatedTypes.Contains(t) {
					excludeSet.Add(obj)
					break
				}
			}
		}
	}

	// Return requested objects that are NOT in the exclude set
	var covered []ontology.ID
	for _, obj := range params.Request.Objects {
		if !excludeSet.Contains(obj) {
			covered = append(covered, obj)
		}
	}
	return covered, nil
}

// traverseBackward traverses backward from the given IDs to find all objects
// that have the specified relationship type to those IDs.
func (c Constraint) traverseBackward(
	ctx context.Context,
	params EnforceParams,
	fromIDs []ontology.ID,
) ([]ontology.ID, error) {
	if len(fromIDs) == 0 {
		return nil, nil
	}
	var resources []ontology.Resource
	traverser := ontology.Traverser{
		Filter: func(res *ontology.Resource, rel *ontology.Relationship) bool {
			return rel.Type == c.RelationshipType && rel.To == res.ID
		},
		Direction: ontology.Backward,
	}
	if err := params.Ontology.NewRetrieve().WhereIDs(fromIDs...).
		ExcludeFieldData(true).
		TraverseTo(traverser).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil {
		return nil, err
	}
	return ontology.ResourceIDs(resources), nil
}

// getRelatedTypes returns the types of all resources that the given object
// has the specified relationship to (forward traversal).
func (c Constraint) getRelatedTypes(
	ctx context.Context,
	params EnforceParams,
	obj ontology.ID,
) (set.Set[ontology.Type], error) {
	var resources []ontology.Resource
	traverser := ontology.Traverser{
		Filter: func(res *ontology.Resource, rel *ontology.Relationship) bool {
			return rel.Type == c.RelationshipType && rel.From == res.ID
		},
		Direction: ontology.Forward,
	}
	if err := params.Ontology.NewRetrieve().WhereIDs(obj).
		ExcludeFieldData(true).
		TraverseTo(traverser).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil {
		return nil, err
	}
	types := make(set.Set[ontology.Type])
	for _, r := range resources {
		types.Add(r.ID.Type)
	}
	return types, nil
}
