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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a user.Service.
type ServiceConfig struct {
	// DB is the underlying database that the service will use to store Users.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology will be used to create relationships between users and other resources,
	// such as projects, within the Synnax cluster.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create the top level "Users" group that will be the default
	// parent of all users.
	//
	// [REQUIRED]
	Group *group.Service
	// Search is the search index for fuzzy searching users.
	//
	// [REQUIRED]
	Search *search.Index
	// Auth is the auth service used to register and rotate credentials for the root
	// user during startup-time reconciliation.
	//
	// [OPTIONAL] - Required when [ServiceConfig.RootCredentials] is set; ignored
	// otherwise.
	Auth *auth.Service
	// Signals is used to propagate user changes through the Synnax signals' channel
	// communication mechanism.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// RootCredentials is the username/password pair the cluster's root user must
	// satisfy after startup. When non-zero, [OpenService] runs root-user
	// reconciliation: it ensures exactly one user has RootUser=true with this username,
	// and the stored credentials match this password (rotating the password if it
	// diverges, registering it if no row exists).
	//
	// [OPTIONAL] - When the zero value is provided, root-user reconciliation limits
	// itself to demoting all but one stale root user (if multiple exist) and otherwise
	// leaves cluster state untouched.
	RootCredentials auth.Credentials
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Search = override.Nil(c.Search, other.Search)
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.Signals = override.Nil(c.Signals, other.Signals)
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
	if c.RootCredentials.Username != "" {
		validate.NotNil(v, "auth", c.Auth)
		v.Exec(c.RootCredentials.Validate)
	}
	return v.Error()
}

// A Service is how [User]s are managed in the Synnax cluster.
type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, User]
	indexes indexes
}

// OpenService opens a new [Service] with the given context and configurations.
func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (_ *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, indexes: newIndexes()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, User]{
		DB: cfg.DB,
		Migrations: []migrate.Migration{
			gorp.CodecMigration[Key, User]("msgpack_to_orc"),
		},
		Indexes:         s.indexes.all(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if cfg.Signals != nil {
		var sig io.Closer
		if sig, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID(s.table.Observe()),
		); !ok(err, sig) {
			return nil, err
		}
	}
	// reconcileRootUser makes sure that there is exactly one root user that matches the
	// credentials in cfg.RootCredentials, or — when no credentials are configured —
	// collapses multiple stale roots into one.
	if err = s.reconcileRootUser(ctx); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

// NewWriter opens a new [Writer] capable of creating, updating, and deleting [User]s.
// The [Writer] operates within the given transaction [gorp.Tx].
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    gorp.OverrideTx(s.cfg.DB, tx),
		otg:   s.cfg.Ontology.NewWriter(tx),
		svc:   s,
		table: s.table,
	}
}

// NewRetrieve opens a new [Retrieve] query capable of retrieving [User]s.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		baseTX:  s.cfg.DB,
		indexes: s.indexes,
	}
}

// Close closes the [Service] and stops any signal publishing.
func (s *Service) Close() error { return s.closer.Close() }
