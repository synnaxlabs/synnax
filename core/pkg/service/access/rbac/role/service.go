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

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	alamos.Instrumentation
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Search   *search.Index
	Signals  *signals.Provider
	Group    *group.Service
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("policy")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

type Service struct {
	cfg     ServiceConfig
	signals io.Closer
	group   group.Group
	table   *gorp.Table[uuid.UUID, Role]
}

func (s *Service) Close() error {
	var err error
	if s.signals != nil {
		err = s.signals.Close()
	}
	return errors.Join(err, s.table.Close())
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	table, err := gorp.OpenTable(ctx, gorp.TableConfig[Role]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
		Migrations:      RoleMigrations(),
	})
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, table: table}
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	cfg.Search.RegisterService(s)
	if s.group, err = cfg.Group.CreateOrRetrieve(ctx, "Users", ontology.RootID); err != nil {
		return nil, err
	}
	if cfg.Signals != nil {
		if s.signals, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[Role](s.table.Observe()),
		); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) UsersGroup() group.Group { return s.group }

func (s *Service) NewWriter(tx gorp.Tx, allowInternal bool) Writer {
	return Writer{
		tx:            gorp.OverrideTx(s.cfg.DB, tx),
		otg:           s.cfg.Ontology.NewWriter(tx),
		group:         s.group,
		allowInternal: allowInternal,
		table:         s.table,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{baseTX: s.cfg.DB, gorp: s.table.NewRetrieve()}
}
