// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/crypto/bcrypt"
)

// Writer registers and mutates credentials within a [Service]. Every Writer method is a
// primitive: it performs no identity check itself and writes directly within the
// writer's transaction. Verification flows (e.g. "user proves they know the old
// password before rotating it") are composed by higher-level packages by combining
// [Service.Authenticate] with these primitives.
type Writer struct {
	service *Service
	tx      gorp.Tx
}

// Register sets the credentials of the user with the given key, replacing any stored
// credentials. Returns [ErrRepeatedUsername] if another user holds creds.Username,
// or a validation error if creds has an empty username or password.
func (w Writer) Register(ctx context.Context, key Key, creds Credentials) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	if err := w.assertUsernameAvailable(ctx, key, creds.Username); err != nil {
		return err
	}
	hashed, err := hashPassword(creds.Password)
	if err != nil {
		return err
	}
	return w.service.table.NewCreate().
		Entry(&SecureCredentials{Key: key, Username: creds.Username, Password: hashed}).
		Exec(ctx, w.tx)
}

// ChangeUsername replaces the stored username of the user with the given key. No
// identity check; caller is responsible for authorization. Returns [query.ErrNotFound]
// if the user has no stored credentials, [ErrRepeatedUsername] if another user holds
// username, or a validation error if username is empty.
func (w Writer) ChangeUsername(ctx context.Context, key Key, username string) error {
	v := validate.New("auth.credentials")
	v.NotEmptyString("username", username)
	if err := v.Error(); err != nil {
		return err
	}
	if err := w.assertUsernameAvailable(ctx, key, username); err != nil {
		return err
	}
	return w.service.table.NewUpdate().
		Where(gorp.MatchKeys[Key, SecureCredentials](key)).
		Change(func(_ gorp.Context, c SecureCredentials) SecureCredentials {
			c.Username = username
			return c
		}).
		Exec(ctx, w.tx)
}

// ChangePassword replaces the stored password of the user with the given key. No
// identity check; caller is responsible for authorization. Returns
// [query.ErrNotFound] if the user has no stored password, or a validation error if
// password is empty.
func (w Writer) ChangePassword(ctx context.Context, key Key, password string) error {
	hashed, err := hashPassword(password)
	if err != nil {
		return err
	}
	return w.service.table.NewUpdate().
		Where(gorp.MatchKeys[Key, SecureCredentials](key)).
		Change(func(_ gorp.Context, c SecureCredentials) SecureCredentials {
			c.Password = hashed
			return c
		}).
		Exec(ctx, w.tx)
}

// Deactivate deletes the stored passwords of the users with the given keys.
func (w Writer) Deactivate(ctx context.Context, keys ...Key) error {
	return w.service.table.NewDelete().
		Where(gorp.MatchKeys[Key, SecureCredentials](keys...)).
		Exec(ctx, w.tx)
}

func (w Writer) assertUsernameAvailable(
	ctx context.Context,
	key Key,
	username string,
) error {
	exists, err := w.service.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, c *SecureCredentials) (bool, error) {
			return c.Username == username && c.Key != key, nil
		})).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if exists {
		return errors.Wrapf(ErrRepeatedUsername, "username %s already exists", username)
	}
	return nil
}

func hashPassword(plaintext string) ([]byte, error) {
	if err := validatePassword(plaintext); err != nil {
		return nil, err
	}
	return bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
}

func validatePassword(password string) error {
	v := validate.New("auth.credentials")
	v.NotEmptyString("password", password)
	return v.Error()
}
