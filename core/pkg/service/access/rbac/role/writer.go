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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx            gorp.Tx
	allowInternal bool
	table         *gorp.Table[Key, Role]
	dispatcher    actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates a new role in the database. It returns an error wrapping
// validate.ErrValidation when a role with the key exists. The commit carries a create
// action that holds the stored role.
func (w Writer) Create(ctx context.Context, r *Role) error {
	if r.Key == uuid.Nil() {
		r.Key = uuid.New()
	}
	if r.Internal && !w.allowInternal {
		return errors.Wrap(validate.ErrValidation, "cannot create internal role")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, r.Key, NewCreateAction(CreatePayload{Role: *r}))
	return nil
}

// Apply applies the actions to the role with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns a validation error when an action is a create or leaves the role invalid, or
// when the role is built in and w does not allow internal writes, and an error wrapping
// query.ErrNotFound when no role has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Role](key)).
		ChangeErr(func(_ gorp.Context, r Role) (Role, error) {
			if r.Internal && !w.allowInternal {
				return Role{}, errors.Wrapf(
					validate.ErrValidation,
					"cannot change built-in role %s",
					r.Name,
				)
			}
			next, err := Reduce(r, acts...)
			if err != nil {
				return Role{}, err
			}
			if err = next.Validate(); err != nil {
				return Role{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// CreateMany creates the given roles. It returns an error wrapping
// validate.ErrValidation when a role with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, roles *[]Role) error {
	for i := range *roles {
		if err := w.Create(ctx, &(*roles)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a role and its policies from the database. It returns a validation
// error for a built-in role, or for a role that a user or rack holds.
func (w Writer) Delete(ctx context.Context, key Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Role](key)).
		Guard(func(_ gorp.Context, r Role) error {
			if r.Internal && !w.allowInternal {
				return errors.Wrap(validate.ErrValidation, "cannot delete builtin role")
			}
			return nil
		}).
		Exec(ctx, w.tx)
}
