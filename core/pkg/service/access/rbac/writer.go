// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx gorp.Tx
}

// Create creates a new policy in the database.
func (w Writer) Create(
	ctx context.Context,
	p *Policy,
) error {
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	}
	return gorp.NewCreate[uuid.UUID, Policy]().Entry(p).Exec(ctx, w.tx)
}

// Delete removes policies with the given keys from the database.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	return gorp.NewDelete[uuid.UUID, Policy]().WhereKeys(keys...).Exec(ctx, w.tx)
}

// CreateRole creates a new role in the database.
func (w Writer) CreateRole(
	ctx context.Context,
	r *Role,
) error {
	if r.Key == uuid.Nil {
		r.Key = uuid.New()
	}
	return gorp.NewCreate[uuid.UUID, Role]().Entry(r).Exec(ctx, w.tx)
}

// DeleteRole removes a role from the database. It will fail if the role is builtin
// or if any users are assigned to the role.
func (w Writer) DeleteRole(
	ctx context.Context,
	key uuid.UUID,
) error {
	return gorp.NewDelete[uuid.UUID, Role]().WhereKeys(key).Guard(func(_ gorp.Context, r Role) error {
		if r.Builtin {
			return errors.New("cannot delete builtin role")
		}
		return nil
	}).Exec(ctx, w.tx)
}
