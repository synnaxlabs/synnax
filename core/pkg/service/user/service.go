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
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
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
	// Signals is used to propagate user changes through the Synnax signals' channel
	// communication mechanism.
	// [OPTIONAL]
	Signals *signals.Provider
}

var (
	_             config.Config[ServiceConfig] = ServiceConfig{}
	defaultConfig                              = ServiceConfig{}
)

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("user")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	return v.Error()
}

// A Service is how users are managed in the Synnax cluster.
type Service struct {
	cfg             ServiceConfig
	shutdownSignals io.Closer
}

// OpenService opens a new Service with the given context ctx and configurations configs.
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(defaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	s := &Service{cfg: cfg}
	cfg.Ontology.RegisterService(s)

	if cfg.Signals != nil {
		cdcS, err := signals.PublishFromGorp[uuid.UUID, User](
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[User](cfg.DB),
		)
		s.shutdownSignals = cdcS
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// NewWriter opens a new writer capable of creating, updating, and deleting Users. The
// writer operates within the given transaction tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:  gorp.OverrideTx(s.cfg.DB, tx),
		otg: s.cfg.Ontology.NewWriter(tx),
		svc: s,
	}
}

// NewRetrieve opens a new retrieve query capable of retrieving Users.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, User](),
		baseTX: s.cfg.DB,
	}
}

// UsernameExists reports whether a User with the given username exists.
func (s *Service) UsernameExists(ctx context.Context, username string) (bool, error) {
	return gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(_ gorp.Context, u *User) (bool, error) {
			return u.Username == username, nil
		}).
		Exists(ctx, s.cfg.DB)
}

// Close closes the service and stops any signal publishing.
func (s *Service) Close() error {
	if s.shutdownSignals == nil {
		return nil
	}
	return s.shutdownSignals.Close()
}
