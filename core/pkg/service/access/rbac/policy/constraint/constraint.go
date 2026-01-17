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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/set"
)

// Kind discriminates between constraint types.
type Kind string

const (
	// KindProperties checks for modifying certain properties of a resource.
	KindProperties Kind = "properties"
	// KindRelationship checks ontology graph relationships.
	KindRelationship Kind = "relationship"
	// KindLogical checks for logical relationships between constraints.
	KindLogical Kind = "logical"
	// KindMatch checks if the requested object matches the IDs in the constraint.
	KindMatch Kind = "match"
	// KindAction checks if the requested action matches the actions in the constraint.
	KindAction Kind = "action"
)

// Operator defines how a constraint compares values.
type Operator string

const (
	OpContainsAny  Operator = "contains_any"
	OpContainsAll  Operator = "contains_all"
	OpContainsNone Operator = "contains_none"
	OpIsIn         Operator = "is_in"
	OpIsNotIn      Operator = "is_not_in"
)

var ErrInvalidOperator = errors.New("invalid constraint operator")
var ErrInvalidKind = errors.New("invalid constraint kind")

// Constraint is a condition that must be satisfied for a policy to apply. The Kind
// field determines which other fields are used.
type Constraint struct {
	// Kind is the type of constraint.
	Kind Kind `json:"kind" msgpack:"kind"`

	// IDs is the list of ontology IDs that this constraint applies to. Used to scope
	// the constraint to specific resources.
	IDs []ontology.ID `json:"ids,omitempty" msgpack:"ids,omitempty"`
	// Operator is the operator used to compare the values in the constraint.
	Operator Operator `json:"operator,omitempty" msgpack:"operator,omitempty"`

	// Actions is the set of actions that this constraint applies to for KindAction.
	Actions set.Set[access.Action] `json:"actions,omitempty" msgpack:"actions,omitempty"`
	// Properties is the list of properties to check for KindProperties.
	Properties []string `json:"properties,omitempty" msgpack:"properties,omitempty"`
	// RelationshipType is the ontology edge type to traverse for KindRelationship.
	RelationshipType ontology.RelationshipType `json:"relationship_type,omitempty" msgpack:"relationship_type,omitempty"`
	// Constraints is the list of child constraints for KindLogical.
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

// Enforce checks which objects from the request are covered by this constraint.
// It returns the subset of params.Request.Objects that satisfy the constraint.
// An empty slice means no objects are covered. An error is only returned for
// invalid configurations, not for access denials.
func (c Constraint) Enforce(ctx context.Context, params EnforceParams) ([]ontology.ID, error) {
	switch c.Kind {
	case KindProperties:
		return c.enforceProperties(params)
	case KindRelationship:
		return c.enforceRelationship(ctx, params)
	case KindLogical:
		return c.enforceLogical(ctx, params)
	case KindMatch:
		return c.enforceMatch(params)
	case KindAction:
		return c.enforceAction(params)
	default:
		return nil, ErrInvalidKind
	}
}
