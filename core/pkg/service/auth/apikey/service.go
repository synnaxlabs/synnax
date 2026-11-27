// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package apikey authenticates subjects by API key. A subject holds any number of
// [APIKey] records, and the package implements [auth.Authenticator] for
// [auth.MethodAPIKey].
package apikey

import (
	"context"
	"crypto/subtle"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey/versions"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

const prefix = "syk_"

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// DB stores API keys.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Clock tells the time, for key expiry.
	//
	// [OPTIONAL] - Defaults to the system clock.
	Clock xtime.Clock
	// Instrumentation is for logging, tracing, metrics, etc.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Clock = override.Nil(c.Clock, other.Clock)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("apikey")
	v.NotNil("db", c.DB)
	v.NotNil("clock", c.Clock)
	return v.Error()
}

// Service stores API keys and checks proofs of them.
type Service struct {
	cfg     ServiceConfig
	table   *gorp.Table[Key, APIKey]
	indexes indexes
}

var _ auth.Authenticator = (*Service)(nil)

// OpenService opens a [Service] with the given configurations.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{Clock: xtime.Real}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, indexes: newIndexes()}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, APIKey]{
		DB:              cfg.DB,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases its resources.
func (s *Service) Close() error { return s.table.Close() }

// NewRetrieve opens a query for stored API keys.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}

// NewWriter opens a [Writer] that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: tx}
}

func (s *Service) now() telem.TimeStamp { return telem.NewTimeStamp(s.cfg.Clock.Now()) }

func (s *Service) expired(k APIKey) bool {
	return s.now() >= k.Expires
}

// Authenticate implements [auth.Authenticator]. A malformed key, an unknown key, a
// wrong secret, and an expired key all return [auth.ErrInvalidCredentials].
func (s *Service) Authenticate(
	ctx context.Context,
	proof auth.Proof,
) (auth.Identity, error) {
	creds, ok := proof.Variant.(auth.APIKeyProof)
	if !ok {
		return auth.Identity{}, errors.Wrapf(
			validate.ErrValidation,
			"API key authenticator cannot check a %T proof",
			proof.Variant,
		)
	}
	if err := creds.Validate(); err != nil {
		return auth.Identity{}, err
	}
	key, hash, err := auth.ParseToken(prefix, creds.Key)
	if err != nil {
		return auth.Identity{}, auth.ErrInvalidCredentials
	}
	var k APIKey
	if err = s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.NewRetrieve().Where(MatchKeys(key)).Entry(&k).Exec(ctx, snap)
	}); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return auth.Identity{}, auth.ErrInvalidCredentials
		}
		return auth.Identity{}, err
	}
	if subtle.ConstantTimeCompare(k.Hash, hash) != 1 {
		return auth.Identity{}, auth.ErrInvalidCredentials
	}
	if s.expired(k) {
		return auth.Identity{}, errors.Wrap(
			auth.ErrInvalidCredentials,
			"API key expired",
		)
	}
	return auth.Identity{
		Subject:    k.Subject,
		Credential: k.Key,
		Version:    k.Created,
	}, nil
}

// Version implements [auth.Authenticator]. An API key keeps one version for life. An
// expired key returns [auth.ErrInvalidCredentials], so its sessions end.
func (s *Service) Version(
	ctx context.Context,
	snap gorp.Snapshot,
	credential uuid.UUID,
) (telem.TimeStamp, error) {
	var k APIKey
	if err := s.NewRetrieve().
		Where(MatchKeys(credential)).
		Entry(&k).
		Exec(ctx, snap); err != nil {
		return 0, err
	}
	if s.expired(k) {
		return 0, errors.Wrapf(auth.ErrInvalidCredentials, "API key %s expired", k.Key)
	}
	return k.Created, nil
}
