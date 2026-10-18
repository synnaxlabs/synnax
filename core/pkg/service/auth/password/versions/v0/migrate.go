// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	authv1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// SubjectLookup resolves the subject that owns the username of a stored credential.
// It returns an error wrapping query.ErrNotFound when no subject has the username.
type SubjectLookup func(
	ctx context.Context,
	tx gorp.Tx,
	username string,
) (ontology.ID, error)

// MoveConfig configures [NewMove].
type MoveConfig struct {
	// SubjectByUsername finds the subject of each stored credential. It is called once
	// per credential, so it may be nil when none are stored.
	SubjectByUsername SubjectLookup
}

// NewMove returns the migration that moves every row of the credentials table into
// the password table. The hash moves as it is, still tagged bcrypt. A credential whose
// username no subject holds is dropped with a warning.
func NewMove(cfg MoveConfig) migrate.Migration {
	return gorp.NewMigration(
		"v0.move_credentials_to_passwords",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			return move(ctx, tx, ins, cfg)
		},
	)
}

func move(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	cfg MoveConfig,
) (err error) {
	reader := gorp.WrapReader[string, authv1.SecureCredentials](tx)
	iter, err := reader.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	var (
		passwords []Password
		usernames []string
	)
	for iter.First(); iter.Valid(); iter.Next() {
		old := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return err
		}
		usernames = append(usernames, old.Username)
		if cfg.SubjectByUsername == nil {
			return errors.Wrap(
				validate.ErrValidation,
				"stored credentials need a subject lookup to migrate",
			)
		}
		subject, err := cfg.SubjectByUsername(ctx, tx, old.Username)
		if errors.Is(err, query.ErrNotFound) {
			ins.L.Warn(
				"dropping stored credential with no user",
				zap.String("username", old.Username),
			)
			continue
		}
		if err != nil {
			return errors.Wrapf(err, "credential %q", old.Username)
		}
		passwords = append(passwords, Password{
			Key:      uuid.New(),
			Subject:  subject,
			Username: old.Username,
			Hash:     old.Password,
			Created:  telem.Now(),
		})
	}
	if len(usernames) == 0 {
		return nil
	}
	if err = gorp.WrapWriter[Key, Password](tx).Set(ctx, passwords...); err != nil {
		return err
	}
	return gorp.WrapWriter[string, authv1.SecureCredentials](tx).
		Delete(ctx, usernames...)
}
