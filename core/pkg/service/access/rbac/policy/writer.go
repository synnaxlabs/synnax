// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx            gorp.Tx
	otg           ontology.Writer
	allowInternal bool
	table         *gorp.Table[Key, Policy]
}

// Create creates a new policy in the database.
func (w Writer) Create(ctx context.Context, p *Policy) error {
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	}
	if p.Internal && !w.allowInternal {
		return errors.Wrap(validate.ErrValidation, "cannot create internal policy")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DefineResources(ctx, p.OntologyID())
}

// CreateMany creates the given policies. If policies with the same key already exist,
// they will be overwritten.
func (w Writer) CreateMany(ctx context.Context, policies *[]Policy) error {
	for i := range *policies {
		if err := w.Create(ctx, &(*policies)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes policies with the given keys from the database.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, Policy](keys...)).
		Exec(ctx, w.tx)
}

func (w Writer) SetOnRole(
	ctx context.Context,
	roleKey role.Key,
	policies ...Key,
) error {
	return w.otg.DefineRelationships(
		ctx,
		role.OntologyID(roleKey),
		ontology.RelationshipTypeParentOf,
		OntologyIDs(policies)...,
	)
}
