// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package kv is a gorp-backed implementation of [auth.Authenticator]. Credentials are
// persisted as [secureCredentials] entries in a single Gorp table; the on-disk shape is
// an unexported implementation detail.
package kv

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/crypto/bcrypt"
)

type secureCredentials struct {
	Username string
	Password []byte
}

func (s secureCredentials) GorpKey() string { return s.Username }

func (secureCredentials) SetOptions() []any { return nil }

// AuthenticatorConfig is the configuration for opening a KV-backed [Authenticator].
type AuthenticatorConfig struct {
	// DB is the [gorp.DB] to use for the KV-backed authenticator.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Instrumentation is for logging, tracing, metrics, etc.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[AuthenticatorConfig] = AuthenticatorConfig{}

// Override implements config.Config.
func (c AuthenticatorConfig) Override(other AuthenticatorConfig) AuthenticatorConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

// Validate implements config.Config.
func (c AuthenticatorConfig) Validate() error {
	v := validate.New("auth.kv.authenticator")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// Authenticator is a simple key-value backed [auth.Authenticator]. It saves data to the
// provided [gorp.DB]. It's important to note that all [gorp.Tx] values provided to the
// Authenticator  must be spawned from the same [gorp.DB].
type Authenticator struct {
	cfg   AuthenticatorConfig
	table *gorp.Table[string, secureCredentials]
}

var _ auth.Authenticator = (*Authenticator)(nil)

// OpenAuthenticator opens a new KV-backed [Authenticator] with the given
// [AuthenticatorConfig]s.
func OpenAuthenticator(
	ctx context.Context,
	cfgs ...AuthenticatorConfig,
) (*Authenticator, error) {
	cfg, err := config.New(AuthenticatorConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	a := &Authenticator{cfg: cfg}
	if a.table, err = gorp.OpenTable(ctx, gorp.TableConfig[secureCredentials]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return a, nil
}

// Close closes the KV authenticator and releases any resources.
func (a *Authenticator) Close() error { return a.table.Close() }

// Authenticate implements [auth.Authenticator].
func (a *Authenticator) Authenticate(
	ctx context.Context,
	creds auth.Credentials,
) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	var stored secureCredentials
	if err := a.table.NewRetrieve().
		Where(gorp.MatchKeys[string, secureCredentials](creds.Username)).
		Entry(&stored).
		Exec(ctx, a.cfg.DB); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return auth.ErrInvalidCredentials
		}
		return errors.Combine(auth.ErrInvalidCredentials, err)
	}
	if err := bcrypt.
		CompareHashAndPassword(stored.Password, []byte(creds.Password)); err != nil {
		return errors.Combine(auth.ErrInvalidCredentials, err)
	}
	return nil
}

// NewWriter implements [auth.Authenticator].
func (a *Authenticator) NewWriter(tx gorp.Tx) auth.Writer {
	return &writer{service: a, tx: gorp.OverrideTx(a.cfg.DB, tx)}
}
