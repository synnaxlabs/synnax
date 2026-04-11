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
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a user.Service.
type ServiceConfig struct {
	// DB is the underlying database that the service will use to store Users.
	DB *gorp.DB
	// Ontology will be used to create relationships between users and other resources,
	// such as workspaces, within the Synnax cluster.
	Ontology *ontology.Ontology
	// Group is used to create the top level "Users" group that will be the default
	// parent of all users.
	Group *group.Service
	// Search is the search index for fuzzy searching users.
	// [REQUIRED]
	Search *search.Index
	// Signals is used to propagate user changes through the Synnax signals' channel
	// communication mechanism.
	// [OPTIONAL]
	Signals *signals.Provider
	// Auth is used to register credentials for the root user during provisioning.
	// [OPTIONAL] - If nil, root user provisioning is skipped.
	Auth auth.Authenticator
	// RootCredentials are the credentials for the root user.
	// [OPTIONAL] - Only used when Auth is provided.
	RootCredentials auth.InsecureCredentials
	// Instrumentation for logging, tracing, and metrics.
	alamos.Instrumentation
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	defaultServiceConfig                              = ServiceConfig{}
)

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.RootCredentials = override.Zero(c.RootCredentials, other.RootCredentials)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("user")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

// A Service is how users are managed in the Synnax cluster.
type Service struct {
	cfg    ServiceConfig
	closer xio.MultiCloser
	table  *gorp.Table[uuid.UUID, User]
}

// OpenService opens a new Service with the given context ctx and configurations configs.
func OpenService(ctx context.Context, configs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(defaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable[uuid.UUID, User](ctx, gorp.TableConfig[User]{
		DB:              cfg.DB,
		Migrations:      []migrate.Migration{gorp.CodecMigration[uuid.UUID, User]("msgpack_to_orc")},
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if cfg.Signals != nil {
		var sig io.Closer
		if sig, err = signals.PublishFromGorp[uuid.UUID, User](
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[User](s.table.Observe()),
		); !ok(err, sig) {
			return nil, err
		}
	}
	if cfg.Auth != nil {
		if err = s.provisionRoot(ctx, cfg); !ok(err, nil) {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) provisionRoot(ctx context.Context, cfg ServiceConfig) error {
	return cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var rootUser User
		if err := s.NewRetrieve().
			Where(WhereUsernames(cfg.RootCredentials.Username)).
			Entry(&rootUser).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if rootUser.Key != uuid.Nil {
			return nil
		}
		rootUser.Username = cfg.RootCredentials.Username
		rootUser.RootUser = true
		if err := cfg.Auth.NewWriter(tx).Register(ctx, cfg.RootCredentials); err != nil {
			return err
		}
		return s.NewWriter(tx).Create(ctx, &rootUser)
	})
}

// NewWriter opens a new writer capable of creating, updating, and deleting Users. The
// writer operates within the given transaction tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    gorp.OverrideTx(s.cfg.DB, tx),
		otg:   s.cfg.Ontology.NewWriter(tx),
		svc:   s,
		table: s.table,
	}
}

// NewRetrieve opens a new retrieve query capable of retrieving Users.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   s.table.NewRetrieve(),
		baseTX: s.cfg.DB,
	}
}

// UsernameExists reports whether a User with the given username exists.
func (s *Service) UsernameExists(ctx context.Context, username string) (bool, error) {
	return s.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, u *User) (bool, error) {
			return u.Username == username, nil
		})).
		Exists(ctx, s.cfg.DB)
}

// Close closes the service and stops any signal publishing.
func (s *Service) Close() error { return s.closer.Close() }
