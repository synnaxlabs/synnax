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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx            gorp.Tx
	allowInternal bool
	table         *gorp.Table[Key, Policy]
	roles         *role.Service
	dispatcher    actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates a policy, and its commit carries a create action that holds the stored
// policy. It returns a validation error when a policy with the key exists or the
// policy's role does not exist. A writer that does not allow internal writes also
// returns a validation error for an internal policy and a policy of a built-in role.
func (w Writer) Create(ctx context.Context, p *Policy) error {
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
		if err := w.checkRole(ctx, p.Role); err != nil {
			return err
		}
	}
	if err := w.table.NewCreate().Exclusive().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, p.Key, NewCreateAction(CreatePayload{Policy: *p}))
	return nil
}

// Apply applies the actions to the policy with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns a validation error when an action is a create, is not a set of a whole
// member, or leaves the policy invalid, or when the policy belongs to or moves to a
// built-in role and w does not allow internal writes, and an error wrapping
// query.ErrNotFound when no policy has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Policy](key)).
		ChangeErr(func(ctx gorp.Context, p Policy) (Policy, error) {
			if !w.allowInternal {
				if err := w.checkRole(ctx, p.Role); err != nil {
					return Policy{}, err
				}
			}
			next, err := Reduce(p, acts...)
			if err != nil {
				return Policy{}, err
			}
			if err = next.Validate(); err != nil {
				return Policy{}, err
			}
			if !w.allowInternal && next.Role != p.Role {
				if err = w.checkRole(ctx, next.Role); err != nil {
					return Policy{}, err
				}
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// CreateMany creates the given policies. It returns an error wrapping
// validate.ErrValidation when a policy with one of their keys exists.
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
