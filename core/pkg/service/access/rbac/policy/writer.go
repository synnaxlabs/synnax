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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	svcrole "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
)

// Writer is a writer used for creating, updating, and deleting policies.
type Writer struct {
	tx  gorp.Tx
	otg ontology.Writer
}

// NewWriter opens a new Writer to create, update, and delete policies.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{tx: tx, otg: s.cfg.Ontology.NewWriter(tx)}
}

// Create creates a new policy. If the policy does not have a key, a new key will be
// generated.
func (w Writer) Create(ctx context.Context, policy *Policy) error {
	if policy.Key == uuid.Nil {
		policy.Key = uuid.New()
	}
	if err := gorp.
		NewCreate[uuid.UUID, Policy]().
		Entry(policy).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DefineResource(ctx, policy.OntologyID())
}

// Delete removes policies with the given keys from the database.
func (w Writer) Delete(ctx context.Context, keys ...uuid.UUID) error {
	return gorp.NewDelete[uuid.UUID, Policy]().WhereKeys(keys...).Exec(ctx, w.tx)
}

// SetOnRole associates the given policies with the role with the given key.
func (w Writer) SetOnRole(
	ctx context.Context,
	role uuid.UUID,
	policies ...uuid.UUID,
) error {
	for _, policy := range OntologyIDs(policies) {
		if err := w.otg.DefineRelationship(
			ctx,
			svcrole.OntologyID(role),
			ontology.RelationshipTypeParentOf,
			policy,
		); err != nil {
			return err
		}
	}
	return nil
}
