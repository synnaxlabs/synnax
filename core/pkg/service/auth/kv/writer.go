// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"golang.org/x/crypto/bcrypt"
)

type writer struct {
	service *Authenticator
	tx      gorp.Tx
}

var _ auth.Writer = (*writer)(nil)

func (w *writer) Register(ctx context.Context, creds auth.Credentials) error {
	if err := w.assertUsernameAvailable(ctx, creds.Username); err != nil {
		return err
	}
	hashed, err := hashPassword(creds.Password)
	if err != nil {
		return err
	}
	return w.service.table.NewCreate().Entry(&secureCredentials{
		Username: creds.Username,
		Password: hashed,
	}).Exec(ctx, w.tx)
}

func (w *writer) UpdateUsername(
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
		Where(gorp.MatchKeys[string, secureCredentials](oldUsername)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	stored.Username = newUsername
	return w.service.table.NewCreate().Entry(&stored).Exec(ctx, w.tx)
}

func (w *writer) ChangePassword(ctx context.Context, creds auth.Credentials) error {
	hashed, err := hashPassword(creds.Password)
	if err != nil {
		return err
	}
	err = w.service.table.NewUpdate().
		Where(gorp.MatchKeys[string, secureCredentials](creds.Username)).
		Change(func(_ gorp.Context, c secureCredentials) secureCredentials {
			c.Password = hashed
			return c
		}).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		return auth.ErrInvalidCredentials
	}
	if err != nil {
		return errors.Combine(auth.ErrInvalidCredentials, err)
	}
	return nil
}

func (w *writer) Deactivate(ctx context.Context, usernames ...string) error {
	return w.service.table.NewDelete().
		Where(gorp.MatchKeys[string, secureCredentials](usernames...)).
		Exec(ctx, w.tx)
}

func (w *writer) retrieve(
	ctx context.Context,
	username string,
) (secureCredentials, error) {
	var stored secureCredentials
	if err := w.service.table.NewRetrieve().
		Where(gorp.MatchKeys[string, secureCredentials](username)).
		Entry(&stored).
		Exec(ctx, w.tx); err != nil {
		return secureCredentials{}, err
	}
	return stored, nil
}

func (w *writer) assertUsernameAvailable(ctx context.Context, username string) error {
	exists, err := w.service.table.NewRetrieve().
		Where(gorp.MatchKeys[string, secureCredentials](username)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if exists {
		return errors.Wrap(
			auth.ErrRepeatedUsername,
			fmt.Sprintf("username %s already exists", username),
		)
	}
	return nil
}

func hashPassword(plaintext string) ([]byte, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Combine(auth.ErrInvalidCredentials, err)
	}
	return h, nil
}
