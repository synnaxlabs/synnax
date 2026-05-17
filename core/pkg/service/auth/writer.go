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
	"fmt"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
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

// Register stores new credentials. Returns [ErrRepeatedUsername] if the username is
// already taken, or a validation error if creds has an empty username or password.
func (w Writer) Register(ctx context.Context, creds Credentials) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	if err := w.assertUsernameAvailable(ctx, creds.Username); err != nil {
		return err
	}
	hashed, err := hashPassword(creds.Password)
	if err != nil {
		return err
	}
	return w.service.table.NewCreate().Entry(&SecureCredentials{
		Username: creds.Username,
		Password: hashed,
	}).Exec(ctx, w.tx)
}

// UpdateUsername renames the credential entry from oldUsername to newUsername. No
// identity check; caller is responsible for authorization.
func (w Writer) UpdateUsername(
	ctx context.Context,
	oldUsername,
	newUsername string,
) error {
	if oldUsername == newUsername {
		return nil
	}
	if err := w.assertUsernameAvailable(ctx, newUsername); err != nil {
		return err
	}
	stored, err := w.retrieve(ctx, oldUsername)
	if err != nil {
		return err
	}
	if err := w.service.table.NewDelete().
		Where(gorp.MatchKeys[string, SecureCredentials](oldUsername)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	stored.Username = newUsername
	return w.service.table.NewCreate().Entry(&stored).Exec(ctx, w.tx)
}

// ChangePassword replaces the stored password for creds.Username with creds.Password.
// No identity check; caller is responsible for authorization. Returns a validation
// error if creds has an empty username or password.
func (w Writer) ChangePassword(ctx context.Context, creds Credentials) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	hashed, err := hashPassword(creds.Password)
	if err != nil {
		return err
	}
	err = w.service.table.NewUpdate().
		Where(gorp.MatchKeys[string, SecureCredentials](creds.Username)).
		Change(func(_ gorp.Context, c SecureCredentials) SecureCredentials {
			c.Password = hashed
			return c
		}).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		return ErrInvalidCredentials
	}
	return err
}

// Deactivate removes credentials for the given usernames. No identity check; caller
// is responsible for authorization.
func (w Writer) Deactivate(ctx context.Context, usernames ...string) error {
	return w.service.table.NewDelete().
		Where(gorp.MatchKeys[string, SecureCredentials](usernames...)).
		Exec(ctx, w.tx)
}

func (w Writer) retrieve(
	ctx context.Context,
	username string,
) (SecureCredentials, error) {
	var stored SecureCredentials
	if err := w.service.table.NewRetrieve().
		Where(gorp.MatchKeys[string, SecureCredentials](username)).
		Entry(&stored).
		Exec(ctx, w.tx); err != nil {
		return SecureCredentials{}, err
	}
	return stored, nil
}

func (w Writer) assertUsernameAvailable(ctx context.Context, username string) error {
	exists, err := w.service.table.NewRetrieve().
		Where(gorp.MatchKeys[string, SecureCredentials](username)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if exists {
		return errors.Wrap(
			ErrRepeatedUsername,
			fmt.Sprintf("username %s already exists", username),
		)
	}
	return nil
}

// hashPassword returns the bcrypt hash of plaintext, propagating any bcrypt error
// (e.g. password too long, out of memory) verbatim so callers can distinguish a real
// system failure from a credential mismatch.
func hashPassword(plaintext string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
}
