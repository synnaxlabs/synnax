// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening a user.Service.
type Config struct {
	// DB is the underlying database that the service will use to store Users.
	DB *gorp.DB
	// Ontology will be used to create relationships between users and other resources,
	// such as workspaces, within the Synnax cluster.
	Ontology *ontology.Ontology
	// Group is used to create the top level "Users" group that will be the default
	// parent of all users.
	Group *group.Service
}

var (
	_             config.Config[Config] = Config{}
	defaultConfig                       = Config{}
)

// Override implements [config.Config].
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements [config.Config].
func (c Config) Validate() error {
	v := validate.New("user")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	validate.NotNil(v, "Group", c.Group)
	return v.Error()
}

// A Service is how users are managed in the Synnax cluster.
type Service struct {
	// Config is the configuration for the service.
	Config
	group group.Group
}

const groupName = "Users"

// NewService opens a new Service with the given context ctx and configurations configs.
func NewService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(defaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

// NewWriter opens a new writer capable of creating, updating, and deleting Users. The
// writer operates within the given transaction tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:  gorp.OverrideTx(s.DB, tx),
		otg: s.Ontology.NewWriter(tx),
		svc: s,
	}
}

// NewRetrieve opens a new retrieve query capable of retrieving Users.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, User](),
		baseTX: s.DB,
	}
}

// UsernameExists reports whether a User with the given username exists.
func (s *Service) UsernameExists(ctx context.Context, username string) (bool, error) {
	return gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(_ gorp.Context, u *User) (bool, error) {
			return u.Username == username, nil
		}).
		Exists(ctx, s.DB)
}
