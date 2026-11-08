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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	svc           *Service
	tx            gorp.Tx
	allowInternal bool
	table         *gorp.Table[Key, Policy]
	roles         *role.Service
}

// inTx calls f with w, or with a Writer on a transaction of its own that keeps the
// internal write permission of w when w has no transaction.
func (w Writer) inTx(ctx context.Context, f func(Writer) error) error {
	db, ok := w.tx.(*gorp.DB)
	if !ok {
		return f(w)
	}
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		return f(w.svc.NewWriter(tx, w.allowInternal))
	})
}

// Create creates or overwrites a policy. It returns a validation error when the
// policy's role does not exist. A writer that does not allow internal writes also
// returns a validation error for an internal policy, a policy of a built-in role, and a
// policy that would overwrite one of a built-in role.
func (w Writer) Create(ctx context.Context, p *Policy) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, p) })
}

func (w Writer) create(ctx context.Context, p *Policy) error {
	if p.Key == uuid.Nil() {
		p.Key = uuid.New()
	}
	if p.Internal && !w.allowInternal {
		return errors.Wrap(validate.ErrValidation, "cannot create internal policy")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if !w.allowInternal {
		if err := w.checkStored(ctx, p.Key); err != nil {
			return err
		}
		if err := w.checkRole(ctx, p.Role); err != nil {
			return err
		}
	}
	return w.table.NewCreate().Entry(p).Exec(ctx, w.tx)
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

// Delete removes policies with the given keys from the database. A writer that does
// not allow internal writes returns a validation error for a policy of a built-in role.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	q := w.table.NewDelete().Where(gorp.MatchKeys[Key, Policy](keys...))
	if !w.allowInternal {
		q = q.Guard(func(ctx gorp.Context, p Policy) error {
			return w.checkRole(ctx, p.Role)
		})
	}
	return q.Exec(ctx, w.tx)
}

func (w Writer) checkStored(ctx context.Context, key Key) error {
	var stored Policy
	err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Policy](key)).
		Entry(&stored).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return w.checkRole(ctx, stored.Role)
}

// checkRole returns a validation error when the role with key r is built in. It leaves
// a missing role to the reference check of the write.
func (w Writer) checkRole(ctx context.Context, r role.Key) error {
	var rol role.Role
	err := w.roles.NewRetrieve().
		Where(role.MatchKeys(r)).
		Entry(&rol).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if rol.Internal {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot change the policies of built-in role %s",
			rol.Name,
		)
	}
	return nil
}
