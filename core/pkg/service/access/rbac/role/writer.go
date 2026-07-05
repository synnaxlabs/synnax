// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx            gorp.Tx
	otg           ontology.Writer
	group         group.Group
	allowInternal bool
	table         *gorp.Table[Key, Role]
}

// Create creates a new role in the database.
func (w Writer) Create(ctx context.Context, r *Role) error {
	if r.Key == uuid.Nil {
		r.Key = uuid.New()
	}
	if r.Internal && !w.allowInternal {
		return errors.Wrap(validate.ErrValidation, "cannot create internal role")
	}
	if err := w.table.NewCreate().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.otg.DefineResources(ctx, OntologyID(r.Key)); err != nil {
		return err
	}
	return w.otg.DefineRelationships(
		ctx, w.group.OntologyID(), ontology.RelationshipTypeParentOf, r.OntologyID(),
	)
}

// CreateMany creates the given roles. If roles with the same key already exist, they
// will be overwritten.
func (w Writer) CreateMany(ctx context.Context, roles *[]Role) error {
	for i := range *roles {
		if err := w.Create(ctx, &(*roles)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a role from the database. It will fail if the role is builtin or if
// any users are assigned to the role.
func (w Writer) Delete(ctx context.Context, key Key) error {
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, Role](key)).Guard(func(_ gorp.Context, r Role) error {
		if r.Internal && !w.allowInternal {
			return errors.Wrap(validate.ErrValidation, "cannot delete builtin role")
		}
		return nil
	}).Exec(ctx, w.tx)
}

// AssignRole assigns a role to a subject (typically a user) by creating an ontology
// relationship. The relationship is idempotent - calling this multiple times with the
// same subject and role has no effect.
func (w Writer) AssignRole(ctx context.Context, subject ontology.ID, role Key) error {
	return w.otg.DefineRelationships(
		ctx, OntologyID(role), ontology.RelationshipTypeParentOf, subject,
	)
}

// UnassignRole removes a role from a subject by deleting the ontology relationship. If
// the relationship does not exist, this is a no-op.
func (w Writer) UnassignRole(ctx context.Context, subject ontology.ID, role Key) error {
	return w.otg.DeleteRelationships(ctx, ontology.Relationship{
		From: OntologyID(role),
		Type: ontology.RelationshipTypeParentOf,
		To:   subject,
	})
}
