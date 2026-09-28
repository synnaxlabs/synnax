// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"encoding/json/v2"

	"github.com/synnaxlabs/alamos"
	v1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"go.uber.org/zap"
)

// MigrationConfig is the configuration for the migration returned by [NewMigration].
type MigrationConfig struct {
	// UserKeys returns the key of every user, indexed by username.
	UserKeys func(context.Context, gorp.Tx) (map[string]Key, error)
}

// QuarantineKVPrefix prefixes the KV keys of credentials that name no user.
const QuarantineKVPrefix = "sy_auth_quarantine/"

// QuarantineKVKey returns the KV key that holds the quarantined credentials for
// username.
func QuarantineKVKey(username string) []byte {
	return []byte(QuarantineKVPrefix + username)
}

// NewMigration returns a migration that re-keys every stored credential from its
// username to the key of the user with that username. Credentials whose username
// matches no user move to [QuarantineKVKey]. The user table must be readable when the
// migration runs.
func NewMigration(cfg MigrationConfig) migrate.Migration {
	return gorp.NewMigration(
		"v59_credentials_by_user_key",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			return rekey(ctx, tx, ins, cfg)
		},
	)
}

func rekey(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	cfg MigrationConfig,
) error {
	old, err := collect(ctx, gorp.WrapReader[string, v1.SecureCredentials](tx))
	if err != nil || len(old) == 0 {
		return err
	}
	userKeys, err := cfg.UserKeys(ctx, tx)
	if err != nil {
		return err
	}
	var (
		oldW = gorp.WrapWriter[string, v1.SecureCredentials](tx)
		newW = gorp.WrapWriter[Key, SecureCredentials](tx)
	)
	for _, o := range old {
		if err := oldW.Delete(ctx, o.Username); err != nil {
			return err
		}
		key, ok := userKeys[o.Username]
		if !ok {
			if err := quarantine(ctx, tx, ins, o); err != nil {
				return err
			}
			continue
		}
		creds, err := autoMigrateSecureCredentials(ctx, o)
		if err != nil {
			return err
		}
		creds.Key = key
		if err := newW.Set(ctx, creds); err != nil {
			return err
		}
	}
	return nil
}

func quarantine(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	o v1.SecureCredentials,
) error {
	b, err := json.Marshal(o, json.Deterministic(true))
	if err != nil {
		return err
	}
	if err := tx.Set(ctx, QuarantineKVKey(o.Username), b); err != nil {
		return err
	}
	ins.L.Warn(
		"quarantined credentials that belong to no user",
		zap.String("username", o.Username),
	)
	return nil
}

func collect(
	ctx context.Context,
	r gorp.Reader[string, v1.SecureCredentials],
) (out []v1.SecureCredentials, err error) {
	iter, err := r.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}
