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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// KVConfig is the configuration for opening the KV backed authenticator.
type KVConfig struct {
	DB *gorp.DB
	alamos.Instrumentation
}

var _ config.Config[KVConfig] = KVConfig{}

// Override implements config.Config.
func (c KVConfig) Override(other KVConfig) KVConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

// Validate implements config.Config.
func (c KVConfig) Validate() error {
	v := validate.New("auth.kv")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// KV is a simple key-value backed Authenticator. It saves data to the provided Gorp DB.
// It's important to note that all gorp.tx(s) provided to the Authenticator interface
// must be spawned from the same gorp DB.
type KV struct {
	cfg   KVConfig
	table *gorp.Table[string, SecureCredentials]
}

// OpenKV opens a new KV authenticator with the given database.
func OpenKV(ctx context.Context, cfgs ...KVConfig) (*KV, error) {
	cfg, err := config.New(KVConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	table, err := gorp.OpenTable(ctx, gorp.TableConfig[string, SecureCredentials]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return nil, err
	}
	return &KV{cfg: cfg, table: table}, nil
}

// Close closes the KV authenticator and releases any resources.
func (kv *KV) Close() error { return kv.table.Close() }

var _ Authenticator = (*KV)(nil)

// Authenticate implements Authenticator.
func (kv *KV) Authenticate(ctx context.Context, creds InsecureCredentials) error {
	_, err := kv.authenticate(ctx, creds, kv.cfg.DB)
	return err
}

func (kv *KV) authenticate(
	ctx context.Context,
	creds InsecureCredentials,
	tx gorp.Tx,
) (SecureCredentials, error) {
	if err := creds.Validate(); err != nil {
		return SecureCredentials{}, err
	}
	var secureCreds SecureCredentials
	err := kv.retrieve(ctx, tx, creds.Username, &secureCreds)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			err = InvalidCredentials
		}
		return SecureCredentials{}, err
	}
	if err = secureCreds.Password.Validate(creds.Password); err != nil {
		return SecureCredentials{}, err
	}
	return secureCreds, nil
}

// NewWriter implements Authenticator.
func (kv *KV) NewWriter(tx gorp.Tx) Writer {
	return &kvWriter{service: kv, tx: kv.cfg.DB.OverrideTx(tx), table: kv.table}
}

type kvWriter struct {
	service *KV
	tx      gorp.Tx
	table   *gorp.Table[string, SecureCredentials]
}

// Register implements Authenticator.
func (w *kvWriter) Register(ctx context.Context, creds InsecureCredentials) error {
	err := w.checkUsernameExists(ctx, creds.Username)
	if err != nil {
		return err
	}
	sec := SecureCredentials{Username: creds.Username}
	sec.Password, err = creds.Password.Hash()
	if err != nil {
		return err
	}
	return w.set(ctx, sec)
}

// UpdateUsername implements Authenticator.
func (w *kvWriter) UpdateUsername(ctx context.Context, creds InsecureCredentials, newUser string) error {
	secureCreds, err := w.service.authenticate(ctx, creds, w.tx)
	if err != nil {
		return err
	}
	return w.changeUsername(ctx, secureCreds.Username, newUser)
}

// InsecureUpdateUsername implements Authenticator.
func (w *kvWriter) InsecureUpdateUsername(ctx context.Context, oldUsername, newUsername string) error {
	return w.changeUsername(ctx, oldUsername, newUsername)
}

// UpdatePassword implements Authenticator.
func (w *kvWriter) UpdatePassword(ctx context.Context, creds InsecureCredentials, newPass password.Raw) error {
	secureCreds, err := w.service.authenticate(ctx, creds, w.tx)
	if err != nil {
		return err
	}
	secureCreds.Password, err = newPass.Hash()
	if err != nil {
		return err
	}
	return w.set(ctx, secureCreds)
}

// InsecureDeactivate implements Authenticator.
func (w *kvWriter) InsecureDeactivate(ctx context.Context, usernames ...string) error {
	return w.delete(ctx, usernames...)
}

func (w *kvWriter) changeUsername(ctx context.Context, oldUsername, newUsername string) error {
	if oldUsername == newUsername {
		return nil
	}
	if err := w.checkUsernameExists(ctx, newUsername); err != nil {
		return err
	}
	var secureCreds SecureCredentials
	if err := w.service.retrieve(ctx, w.tx, oldUsername, &secureCreds); err != nil {
		return err
	}
	if err := w.delete(ctx, oldUsername); err != nil {
		return err
	}
	secureCreds.Username = newUsername
	return w.set(ctx, secureCreds)
}

func (w *kvWriter) set(ctx context.Context, creds SecureCredentials) error {
	return w.table.NewCreate().Entry(&creds).Exec(ctx, w.tx)
}

func (w *kvWriter) delete(ctx context.Context, usernames ...string) error {
	return w.table.NewDelete().Where(gorp.MatchKeys[string, SecureCredentials](usernames...)).Exec(ctx, w.tx)
}

func (w *kvWriter) checkUsernameExists(ctx context.Context, user string) error {
	exists, err := w.service.table.NewRetrieve().Where(gorp.MatchKeys[string, SecureCredentials](user)).Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if exists {
		return errors.Wrap(RepeatedUsername, fmt.Sprintf("A user with the username %s already exists", user))
	}
	return err
}

func (kv *KV) retrieve(ctx context.Context, tx gorp.Tx, user string, creds *SecureCredentials) error {
	return kv.table.NewRetrieve().
		Where(gorp.MatchKeys[string, SecureCredentials](user)).
		Entry(creds).
		Exec(ctx, gorp.OverrideTx(kv.cfg.DB, tx))
}
