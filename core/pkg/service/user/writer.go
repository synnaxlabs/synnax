// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// A Writer creates, updates, and deletes users in one transaction. Of the credentials
// of a user, it changes only the username of the password, through a set username
// action.
type Writer struct {
	svc        *Service
	tx         gorp.Tx
	table      *gorp.Table[Key, User]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create persists a new user record from u. If u.Key is the zero UUID, a new key is
// assigned. The returned User has Key populated, and the commit carries a create action
// that holds it. Returns an error if u.RootUser is true, and a validation error if
// u.Name is empty, u.Role is zero or does not exist, or a user with u.Key already
// exists.
func (w Writer) Create(ctx context.Context, u User) (User, error) {
	if u.RootUser {
		return User{}, errors.New(
			"cannot create a root user; root users are provisioned at startup",
		)
	}
	return w.create(ctx, u)
}

func (w Writer) create(ctx context.Context, u User) (User, error) {
	if err := u.Validate(); err != nil {
		return User{}, err
	}
	if u.Key == uuid.Nil() {
		u.Key = uuid.New()
	}
	if err := w.table.NewCreate().Exclusive().Entry(&u).Exec(ctx, w.tx); err != nil {
		return User{}, err
	}
	w.dispatcher.Attach(w.tx, u.Key, NewCreateAction(CreatePayload{User: u}))
	return u, nil
}

// Apply applies the actions to the user with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. A set
// username action renames the user's password. Apply returns a validation error when an
// action is a create, changes the role of a root user, or leaves the user invalid, the
// errors of password.Writer.Apply for a set username action, and an error wrapping
// query.ErrNotFound when no user has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, User](key)).
		ChangeErr(func(_ gorp.Context, u User) (User, error) {
			next, err := Reduce(u, acts...)
			if err != nil {
				return User{}, err
			}
			if u.RootUser && next.Role != u.Role {
				return User{}, errors.Wrapf(
					validate.ErrValidation,
					"cannot change the role of root user %s",
					u.Name,
				)
			}
			if err = next.Validate(); err != nil {
				return User{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	passwords := w.svc.cfg.Password.NewWriter(w.tx)
	for _, a := range acts {
		if a.Type != ActionTypeSetUsername {
			continue
		}
		if err := passwords.Apply(
			ctx,
			ResourceID(key),
			password.NewSetUsernameAction(a.SetUsername.Username),
		); err != nil {
			return err
		}
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// SetRole gives the role r to each user with the given keys, root users included, and
// attaches a set of the role for each one. SetRole returns a validation error when r
// does not exist, and query.ErrNotFound when a user does not exist.
func (w Writer) SetRole(ctx context.Context, r role.Key, keys ...Key) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, User](keys...)).
		Change(func(_ gorp.Context, u User) User { u.Role = r; return u }).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range keys {
		w.dispatcher.Attach(w.tx, k, newSetAction("role", r))
	}
	return nil
}

func (w Writer) promoteRoot(ctx context.Context, key Key, r role.Key) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, User](key)).
		Change(func(_ gorp.Context, u User) User {
			u.RootUser = true
			u.Role = r
			return u
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		key,
		newSetAction("root_user", true),
		newSetAction("role", r),
	)
	return nil
}

func (w Writer) demoteRoot(ctx context.Context, key Key) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, User](key)).
		Change(func(_ gorp.Context, u User) User { u.RootUser = false; return u }).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, newSetAction("root_user", false))
	return nil
}

// Delete removes the users with the given keys from the key-value store.
// Deleting a root user is rejected.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, User](keys...)).
		Guard(func(_ gorp.Context, u User) error {
			if u.RootUser {
				return errors.New("cannot delete root user")
			}
			return nil
		}).Exec(ctx, w.tx)
}
