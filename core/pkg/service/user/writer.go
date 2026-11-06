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
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// A Writer is used to create, update, and delete user records. It does not touch
// authentication credentials; callers that need to keep credentials in sync (the API
// layer, the rbac root-user reconciler) coordinate user and auth writes themselves
// within a single transaction.
type Writer struct {
	svc   *Service
	tx    gorp.Tx
	otg   ontology.Writer
	table *gorp.Table[Key, User]
}

// Create persists a new user record from u. If u.Key is the zero UUID, a new key is
// assigned. The returned User has Key populated. Returns an error if u.RootUser is
// true, and a validation error if u.Name is empty, a user with u.Key already exists, or
// a role in u.Roles does not exist.
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
	} else if exists, err := w.svc.NewRetrieve().
		Where(MatchKeys(u.Key)).
		Exists(ctx, w.tx); err != nil {
		return User{}, err
	} else if exists {
		return User{}, errors.Wrapf(
			validate.ErrValidation,
			"user %s already exists",
			u.Key,
		)
	}
	if err := w.table.NewCreate().Entry(&u).Exec(ctx, w.tx); err != nil {
		return User{}, err
	}
	if err := w.otg.DefineResources(ctx, u.ResourceID()); err != nil {
		return User{}, err
	}
	return u, nil
}

// ChangeName sets the name of the user with the given key. It returns a validation
// error when name is empty.
func (w Writer) ChangeName(ctx context.Context, key Key, name string) error {
	if err := (User{Name: name}).Validate(); err != nil {
		return err
	}
	return w.table.NewUpdate().Where(gorp.MatchKeys[Key, User](key)).
		Change(func(_ gorp.Context, u User) User {
			u.Name = name
			return u
		}).Exec(ctx, w.tx)
}

// SetRoles removes the roles in remove from each user with the given keys, then puts
// each role in add that the user does not hold ahead of the roles it keeps. A role in
// both lists is held after the call. SetRoles returns a validation error when a role in
// add does not exist or a user is a root user, and query.ErrNotFound when a user does
// not exist.
func (w Writer) SetRoles(
	ctx context.Context,
	add, remove []role.Key,
	keys ...Key,
) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, User](keys...)).
		ChangeErr(func(_ gorp.Context, u User) (User, error) {
			if u.RootUser {
				return u, errors.Wrapf(
					validate.ErrValidation,
					"cannot change the roles of root user %s",
					u.Name,
				)
			}
			u.Roles = mergeRoles(u.Roles, add, remove)
			return u, nil
		}).
		Exec(ctx, w.tx)
}

// AssignRole puts the role r ahead of the roles of each user with the given keys that
// does not hold it, root users included. It returns a validation error when r does not
// exist.
func (w Writer) AssignRole(ctx context.Context, r role.Key, keys ...Key) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, User](keys...)).
		Change(func(_ gorp.Context, u User) User {
			u.Roles = mergeRoles(u.Roles, []role.Key{r}, nil)
			return u
		}).
		Exec(ctx, w.tx)
}

func mergeRoles(held, add, remove []role.Key) []role.Key {
	kept := lo.Without(held, remove...)
	added := lo.Reject(lo.Uniq(add), func(r role.Key, _ int) bool {
		return slices.Contains(kept, r)
	})
	return append(added, kept...)
}

func (w Writer) setRootUser(ctx context.Context, key Key, root bool) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, User](key)).
		Change(func(_ gorp.Context, u User) User { u.RootUser = root; return u }).
		Exec(ctx, w.tx)
}

// Delete removes the users with the given keys from the key-value store.
// Deleting a root user is rejected.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().Where(gorp.MatchKeys[Key, User](keys...)).
		Guard(func(_ gorp.Context, u User) error {
			if u.RootUser {
				return errors.New("cannot delete root user")
			}
			return nil
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, ResourceIDsFromKeys(keys)...)
}
