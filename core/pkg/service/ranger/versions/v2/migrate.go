// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

// Migrations move each range to its v2 shape, then move its parent from a parent
// relationship between two ranges into the parent field, and its labels from
// labeled_by relationships into the labels field. They delete each relationship they
// move, and drop parents and labels that are not stored.
var Migrations = []gorp.Migration{
	migrate.NewEntry("v61_range_shape", autoMigrateRange),
	ontology.NewFieldMigration(ontology.FieldConfig[Key, Range]{
		Name:     "v61_range_parent_labels",
		Converts: converts,
		Holder: func(rel ontology.Relationship) string {
			if rel.Type == ontology.RelationshipTypeLabeledBy {
				return rel.From.Key
			}
			return rel.To.Key
		},
		Set: moveRelationship,
	}),
}

func converts(rel ontology.Relationship) bool {
	if rel.From.Type != resource.TypeRange {
		return false
	}
	return rel.Type == ontology.RelationshipTypeLabeledBy ||
		rel.Type == ontology.RelationshipTypeParentOf &&
			rel.To.Type == resource.TypeRange
}

func moveRelationship(
	ctx context.Context,
	tx gorp.Tx,
	r *Range,
	rel ontology.Relationship,
) (bool, error) {
	if rel.Type == ontology.RelationshipTypeLabeledBy {
		label, err := uuid.Parse(rel.To.Key)
		if err != nil {
			return false, errors.Wrapf(err, "label of range %s", rel.From.Key)
		}
		found, err := ontology.ResourceExists(ctx, tx, rel.To)
		if !found || err != nil {
			return false, err
		}
		r.Labels = append(r.Labels, label)
		return true, nil
	}
	if r.Parent != nil {
		return false, nil
	}
	parent, err := uuid.Parse(rel.From.Key)
	if err != nil {
		return false, errors.Wrapf(err, "parent of range %s", rel.To.Key)
	}
	found, err := gorp.NewRetrieve[Key, Range]().
		Where(gorp.MatchKeys[Key, Range](parent)).
		Exists(ctx, tx)
	if !found || err != nil {
		return false, err
	}
	r.Parent = &parent
	return true, nil
}
