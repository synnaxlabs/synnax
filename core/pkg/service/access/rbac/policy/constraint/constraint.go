// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package constraint defines the constraint types used in access control policies.
package constraint

import (
	"context"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/gorp"
)

// Kind discriminates between constraint types.
type Kind string

const (
	// KindField checks a stored value on a resource, subject, request, or system.
	KindField Kind = "field"
	// KindRelationship checks ontology graph relationships.
	KindRelationship Kind = "relationship"
	// KindAnd requires all child constraints to be satisfied.
	KindAnd Kind = "and"
	// KindOr requires at least one child constraint to be satisfied.
	KindOr Kind = "or"
)

// Operator defines how a constraint compares values.
type Operator string

const (
	OpEqual              Operator = "eq"
	OpNotEqual           Operator = "not_eq"
	OpEqualSubject       Operator = "eq_subject"
	OpIn                 Operator = "in"
	OpNotIn              Operator = "not_in"
	OpContains           Operator = "contains"
	OpContainsAny        Operator = "contains_any"
	OpContainsAll        Operator = "contains_all"
	OpContainsNone       Operator = "contains_none"
	OpWithin             Operator = "within"
	OpSubsetOf           Operator = "subset_of"
	OpLessThan           Operator = "lt"
	OpLessThanOrEqual    Operator = "lte"
	OpGreaterThan        Operator = "gt"
	OpGreaterThanOrEqual Operator = "gte"
)

type Target string

const (
	TargetResource Target = "resource"
	TargetRequest  Target = "request"
)

// Constraint is a condition that must be satisfied for a policy to apply. The Kind
// field determines which other fields are used.
type Constraint struct {
	// Kind specifies the type of constraint.
	Kind Kind `json:"kind" msgpack:"kind"`

	// Objects specifies which ontology IDs this constraint applies to. Used to
	// scope the constraint to specific resources.
	Objects []ontology.ID `json:"objects,omitempty" msgpack:"objects,omitempty"`
	// Actions specifies which actions this constraint applies to.
	Actions []access.Action `json:"actions,omitempty" msgpack:"actions,omitempty"`

	// --- Field constraint fields ---
	// Target is the namespace: "resource", "subject", "request", "system"
	Target string `json:"target,omitempty" msgpack:"target,omitempty"`
	// Field is the path to the field within the target.
	Field []string `json:"field,omitempty" msgpack:"field,omitempty"`

	// --- Shared fields (Field, Relationship, Computed) ---
	// Operator for comparison.
	Operator Operator `json:"operator,omitempty" msgpack:"operator,omitempty"`
	// Value to compare against.
	Value any `json:"value,omitempty" msgpack:"value,omitempty"`

	// --- Relationship constraint fields ---
	// Relationship is the ontology edge type to traverse.
	Relationship ontology.RelationshipType `json:"relationship,omitempty" msgpack:"relationship,omitempty"`
	// RelationshipIDs is the list of IDs to match against related entities.
	RelationshipIDs []ontology.ID `json:"relationship_ids,omitempty" msgpack:"relationship_ids,omitempty"`
	// MatchSubject, when true, matches the relationship against the requesting subject.
	MatchSubject bool `json:"match_subject,omitempty" msgpack:"match_subject,omitempty"`

	// --- Computed constraint fields ---
	// Property is the computed value to evaluate (e.g., "duration", "age", "count").
	Property string `json:"property,omitempty" msgpack:"property,omitempty"`
	// Source is the path to compute from (e.g., ["request", "time_range"]).
	Source []string `json:"source,omitempty" msgpack:"source,omitempty"`

	// Constraints is the list of child constraints for KindAnd and KindOr.
	Constraints []Constraint `json:"constraints,omitempty" msgpack:"constraints,omitempty"`
}

// EnforceParams provides the context needed for constraint evaluation.
type EnforceParams struct {
	// Request is the access request being evaluated.
	Request access.Request
	// Ontology provides access to the ontology graph for relationship lookups.
	Ontology *ontology.Ontology
	// Tx is the database transaction for queries.
	Tx gorp.Tx
}

// Enforce checks if the constraint is satisfied.
func (c Constraint) Enforce(ctx context.Context, params EnforceParams) bool {
	switch c.Kind {
	case KindField:
		return c.enforceField(ctx, params)
	case KindRelationship:
		return c.enforceRelationship(ctx, params)
	case KindAnd:
		return c.enforceAnd(ctx, params)
	case KindOr:
		return c.enforceOr(ctx, params)
	default:
		return false
	}
}

func (c Constraint) enforceField(ctx context.Context, params EnforceParams) bool {
	actual, ok := resolveFieldValue(ctx, params, c.Target, c.Field)
	if !ok {
		return false
	}
	return applyOperator(c.Operator, actual, c.Value, params.Request.Subject)
}

func (c Constraint) enforceRelationship(ctx context.Context, params EnforceParams) bool {
	// Determine which entity's relationships to check
	var fromID ontology.ID
	switch c.Target {
	case "subject":
		fromID = params.Request.Subject
	default: // "resource" or empty
		if len(params.Request.Objects) == 0 {
			return false
		}
		fromID = params.Request.Objects[0]
	}

	relatedIDs := resolveRelationship(ctx, params, fromID, c.Relationship)

	var targetIDs []ontology.ID
	if c.MatchSubject {
		targetIDs = []ontology.ID{params.Request.Subject}
	} else {
		targetIDs = c.RelationshipIDs
	}

	switch c.Operator {
	case OpContainsAny:
		for _, target := range targetIDs {
			if slices.Contains(relatedIDs, target) {
				return true
			}
		}
		return false
	case OpContainsAll:
		for _, target := range targetIDs {
			if !slices.Contains(relatedIDs, target) {
				return false
			}
		}
		return true
	case OpContainsNone:
		for _, target := range targetIDs {
			if slices.Contains(relatedIDs, target) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (c Constraint) enforceAnd(ctx context.Context, params EnforceParams) bool {
	for _, child := range c.Constraints {
		if !child.Enforce(ctx, params) {
			return false
		}
	}
	return true
}

func (c Constraint) enforceOr(ctx context.Context, params EnforceParams) bool {
	for _, child := range c.Constraints {
		if child.Enforce(ctx, params) {
			return true
		}
	}
	return false
}
