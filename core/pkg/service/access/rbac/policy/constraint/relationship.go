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
	"github.com/synnaxlabs/x/gorp"
)

// TypeRelationship is the type discriminator for Relationship constraints.
const TypeRelationship Type = "relationship"

func init() { Register(TypeRelationship, func() Constraint { return &Relationship{} }) }

// Relationship checks ontology graph relationships.
// Examples:
//   - resource created_by subject (creator-only access)
//   - resource labeled_by any of [label1, label2]
type Relationship struct {
	// Relationship is the ontology edge type to traverse.
	Relationship ontology.RelationshipType `json:"relationship" msgpack:"relationship"`
	// Operator specifies how to match relationships (contains_any, contains_all,
	// contains_none).
	Operator Operator `json:"operator" msgpack:"operator"`
	// Value is the list of IDs to match against the related entities. Ignored if
	// MatchSubject is true.
	Value []ontology.ID `json:"value,omitempty" msgpack:"value,omitempty"`
	// MatchSubject, when true, matches the relationship against the requesting subject.
	// This is used for "creator only" or "owner only" access patterns. When true, Value
	// is ignored and the subject ID is used instead.
	MatchSubject bool `json:"match_subject,omitempty" msgpack:"match_subject,omitempty"`
}

var _ Constraint = Relationship{}

// Type implements Constraint.
func (c Relationship) Type() Type { return TypeRelationship }

// Enforce checks if the constraint is satisfied.
func (c Relationship) Enforce(ctx context.Context, params EnforceParams) bool {
	relatedIDs := resolveRelationship(ctx, params, c.Relationship)

	var targetIDs []ontology.ID
	if c.MatchSubject {
		targetIDs = []ontology.ID{params.Request.Subject}
	} else {
		targetIDs = c.Value
	}

	switch c.Operator {
	case OpContainsAny:
		for _, target := range targetIDs {
			if containsID(relatedIDs, target) {
				return true
			}
		}
		return false
	case OpContainsAll:
		for _, target := range targetIDs {
			if !containsID(relatedIDs, target) {
				return false
			}
		}
		return true
	case OpContainsNone:
		for _, target := range targetIDs {
			if containsID(relatedIDs, target) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func resolveRelationship(ctx context.Context, params EnforceParams, relType ontology.RelationshipType) []ontology.ID {
	if len(params.Request.Objects) == 0 {
		return nil
	}
	obj := params.Request.Objects[0]
	var relationships []ontology.Relationship
	if err := gorp.NewRetrieve[[]byte, ontology.Relationship]().
		Where(func(_ gorp.Context, rel *ontology.Relationship) (bool, error) {
			return rel.From == obj && rel.Type == relType, nil
		}).
		Entries(&relationships).
		Exec(ctx, params.Tx); err != nil {
		return nil
	}
	relatedIDs := make([]ontology.ID, 0, len(relationships))
	for _, rel := range relationships {
		relatedIDs = append(relatedIDs, rel.To)
	}
	return relatedIDs
}

func containsID(ids []ontology.ID, target ontology.ID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
