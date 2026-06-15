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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
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
// assigned. The returned User has Key populated. Returns an error if u.RootUser is true
// and [auth.ErrRepeatedUsername] if a user with u.Username already exists.
func (w Writer) Create(ctx context.Context, u User) (User, error) {
	if u.RootUser {
		return User{}, errors.New("cannot create a root user; root users are provisioned at startup")
	}
	return w.create(ctx, u)
}

func (w Writer) create(ctx context.Context, u User) (User, error) {
	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}
	exists, err := w.svc.
		NewRetrieve().Where(MatchUsernames(u.Username)).Exists(ctx, w.tx)
	if err != nil {
		return User{}, err
	}
	if exists {
		return User{}, auth.ErrRepeatedUsername
	}
	if err := w.table.NewCreate().Entry(&u).Exec(ctx, w.tx); err != nil {
		return User{}, err
	}
	if err := w.otg.DefineResource(ctx, OntologyID(u.Key)); err != nil {
		return User{}, err
	}
	return u, nil
}

// ChangeUsername renames the user record identified by key to newUsername. No identity
// check; callers must already have authorized the operation. Returns
// [auth.ErrRepeatedUsername] if newUsername already belongs to a different user.
func (w Writer) ChangeUsername(ctx context.Context, key Key, newUsername string) error {
	exists, err := w.svc.NewRetrieve().
		Where(MatchUsernames(newUsername)).Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if exists {
		return auth.ErrRepeatedUsername
	}
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, User](key)).
		Change(func(_ gorp.Context, u User) User {
			u.Username = newUsername
			return u
		}).
		Exec(ctx, w.tx)
}

// ChangeName updates the first and last name of the user with the given key. If either
// first or last is empty, the corresponding field is left unchanged.
func (w Writer) ChangeName(ctx context.Context, key Key, first, last string) error {
	return w.table.NewUpdate().Where(gorp.MatchKeys[Key, User](key)).
		Change(func(_ gorp.Context, u User) User {
			if first != "" {
				u.FirstName = first
			}
			if last != "" {
				u.LastName = last
			}
			return u
		}).Exec(ctx, w.tx)
}

// ChangeAvatar sets the avatar of the user with the given key. avatar is a base64 data
// URI; passing an empty string clears it, falling the user back to a generated avatar.
func (w Writer) ChangeAvatar(ctx context.Context, key Key, avatar string) error {
	return w.table.NewUpdate().Where(gorp.MatchKeys[Key, User](key)).
		Change(func(_ gorp.Context, u User) User { u.Avatar = avatar; return u }).
		Exec(ctx, w.tx)
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
	return w.otg.DeleteResource(ctx, OntologyIDsFromKeys(keys)...)
}
