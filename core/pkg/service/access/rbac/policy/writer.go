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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx            gorp.Tx
	otg           ontology.Writer
	allowInternal bool
}

// Create creates a new policy in the database.
func (w Writer) Create(
	ctx context.Context,
	p *Policy,
) error {
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	}
	if p.Internal && !w.allowInternal {
		return errors.Wrap(validate.Error, "cannot create internal policy")
	}
	if err := gorp.NewCreate[uuid.UUID, Policy]().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DefineResource(ctx, OntologyID(p.Key))
}

// Delete removes policies with the given keys from the database.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	return gorp.NewDelete[uuid.UUID, Policy]().WhereKeys(keys...).Exec(ctx, w.tx)
}

func (w Writer) SetOnRole(
	ctx context.Context,
	roleKey uuid.UUID,
	policyKeys ...uuid.UUID,
) error {
	policyIDs := OntologyIDs(policyKeys)
	for _, p := range policyIDs {
		if err := w.otg.DefineRelationship(ctx, role.OntologyID(roleKey), ontology.ParentOf, p); err != nil {
			return err
		}
	}
	return nil
}
