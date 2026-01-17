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

// enforceLogical combines the results of child constraints using logical operators.
// The behavior depends on the operator:
//   - OpContainsAny (OR): Returns the union of all child results
//   - OpContainsAll (AND): Returns the intersection of all child results
//   - OpContainsNone (NAND): Returns objects that appear in NO child results
func (c Constraint) enforceLogical(
	ctx context.Context,
	params EnforceParams,
) ([]ontology.ID, error) {
	switch c.Operator {
	case OpContainsAny:
		return c.enforceLogicalOr(ctx, params)
	case OpContainsAll:
		return c.enforceLogicalAnd(ctx, params)
	case OpContainsNone:
		return c.enforceLogicalNone(ctx, params)
	default:
		return nil, ErrInvalidOperator
	}
}

// enforceLogicalOr returns the union of all child constraint results.
func (c Constraint) enforceLogicalOr(
	ctx context.Context,
	params EnforceParams,
) ([]ontology.ID, error) {
	if len(c.Constraints) == 0 {
		return params.Request.Objects, nil
	}
	unionSet := make(set.Set[ontology.ID])
	for _, child := range c.Constraints {
		covered, err := child.Enforce(ctx, params)
		if err != nil {
			return nil, err
		}
		for _, id := range covered {
			unionSet.Add(id)
		}
	}
	return unionSet.Keys(), nil
}

// enforceLogicalAnd returns the intersection of all child constraint results.
func (c Constraint) enforceLogicalAnd(
	ctx context.Context,
	params EnforceParams,
) ([]ontology.ID, error) {
	if len(c.Constraints) == 0 {
		return params.Request.Objects, nil
	}

	// Start with the result of the first constraint
	first, err := c.Constraints[0].Enforce(ctx, params)
	if err != nil {
		return nil, err
	}
	if len(first) == 0 {
		return nil, nil
	}
	intersectionSet := set.FromSlice(first)

	// Intersect with each subsequent constraint
	for _, child := range c.Constraints[1:] {
		covered, err := child.Enforce(ctx, params)
		if err != nil {
			return nil, err
		}
		coveredSet := set.FromSlice(covered)
		// Keep only IDs that are in both sets
		for id := range intersectionSet {
			if !coveredSet.Contains(id) {
				delete(intersectionSet, id)
			}
		}
		if len(intersectionSet) == 0 {
			return nil, nil
		}
	}
	return intersectionSet.Keys(), nil
}

// enforceLogicalNone returns objects that appear in NO child constraint results.
func (c Constraint) enforceLogicalNone(
	ctx context.Context,
	params EnforceParams,
) ([]ontology.ID, error) {
	if len(c.Constraints) == 0 {
		return params.Request.Objects, nil
	}

	// Collect all objects covered by any child
	excludeSet := make(set.Set[ontology.ID])
	for _, child := range c.Constraints {
		covered, err := child.Enforce(ctx, params)
		if err != nil {
			return nil, err
		}
		for _, id := range covered {
			excludeSet.Add(id)
		}
	}

	// Return requested objects that are NOT in the exclude set
	var result []ontology.ID
	for _, obj := range params.Request.Objects {
		if !excludeSet.Contains(obj) {
			result = append(result, obj)
		}
	}
	return result, nil
}
