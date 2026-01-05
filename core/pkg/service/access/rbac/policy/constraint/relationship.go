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
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// RelationshipOperator defines operators specific to relationship constraints.
type RelationshipOperator string

const (
	// RelOpContainsSome checks if the related entities include ANY of the specified IDs.
	RelOpContainsSome RelationshipOperator = "contains_some"
	// RelOpContainsAll checks if the related entities include ALL of the specified IDs.
	RelOpContainsAll RelationshipOperator = "contains_all"
	// RelOpContainsNone checks if the related entities include NONE of the specified IDs.
	RelOpContainsNone RelationshipOperator = "contains_none"
)

// Relationship checks ontology graph relationships.
// Examples:
//   - resource created_by subject (creator-only access)
//   - resource labeled_by any of [label1, label2]
type Relationship struct {
	// Relationship is the ontology edge type to traverse.
	Relationship ontology.RelationshipType `json:"relationship" msgpack:"relationship"`
	// Operator specifies how to match relationships (contains_some, contains_all, contains_none).
	Operator RelationshipOperator `json:"operator" msgpack:"operator"`
	// Value is the list of IDs to match against the related entities.
	// Ignored if MatchSubject is true.
	Value []ontology.ID `json:"value,omitempty" msgpack:"value,omitempty"`
	// MatchSubject, when true, matches the relationship against the requesting subject.
	// This is used for "creator only" or "owner only" access patterns.
	// When true, Value is ignored and the subject ID is used instead.
	MatchSubject bool `json:"match_subject,omitempty" msgpack:"match_subject,omitempty"`
}

// Type implements Constraint.
func (c Relationship) Type() Type { return TypeRelationship }

// MarshalJSON implements json.Marshaler.
func (c Relationship) MarshalJSON() ([]byte, error) {
	type rc Relationship
	return json.Marshal(struct {
		Type Type `json:"type"`
		rc
	}{Type: TypeRelationship, rc: rc(c)})
}

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
	case RelOpContainsSome:
		for _, target := range targetIDs {
			if containsID(relatedIDs, target) {
				return true
			}
		}
		return false
	case RelOpContainsAll:
		for _, target := range targetIDs {
			if !containsID(relatedIDs, target) {
				return false
			}
		}
		return true
	case RelOpContainsNone:
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
	var relationships []ontology.Relationship
	if err := gorp.NewRetrieve[[]byte, ontology.Relationship]().
		Where(func(_ gorp.Context, rel *ontology.Relationship) (bool, error) {
			return rel.From == params.Object && rel.Type == relType, nil
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
