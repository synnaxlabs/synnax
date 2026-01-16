// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"
	"io"

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a role service.
type ServiceConfig struct {
	// DB is the underlying database that the service will use to store roles.
	DB *gorp.DB
	// Ontology is the ontology that the service will use to create relationships
	// between roles and other resources, such as policies, within the Synnax cluster.
	Ontology *ontology.Ontology
	// Signals is the signals provider that the service will use to publish changes to
	// roles.
	Signals *signals.Provider
	// Group is the group that the service will use to create the "Users" group that
	// will be the default parent of all roles.
	Group *group.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("role")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	return v.Error()
}

// Service is the main entrypoint for managing roles within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting roles. It also provides
// mechanisms for listening to changes in roles.
type Service struct {
	cfg     ServiceConfig
	signals io.Closer
	group   group.Group
}

// OpenService opens a new role service using the provided configuration(s). If error is
// nil, the service is ready for use and must be closed by calling Close in order to
// prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(
		ctx,
		"Users",
		ontology.RootID,
	); err != nil {
		return nil, err
	}
	if cfg.Signals != nil {
		if s.signals, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[Role](cfg.DB),
		); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the role service and releases any resources that it may have acquired.
func (s *Service) Close() error {
	if s.signals != nil {
		return s.signals.Close()
	}
	return nil
}

// UsersGroup returns the group for users.
func (s *Service) UsersGroup() group.Group { return s.group }
