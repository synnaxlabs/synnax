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
	"github.com/synnaxlabs/x/uuid"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Signals  *signals.Provider
	Group    *group.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements [config.Config].
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements [config.Config].
func (c Config) Validate() error {
	v := validate.New("policy")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "ontology", c.Ontology)
	return v.Error()
}

type Service struct {
	cfg     Config
	signals io.Closer
	group   group.Group
}

func (s *Service) Close() error {
	if s.signals != nil {
		return s.signals.Close()
	}
	return nil
}

func OpenService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(ctx, "Users", ontology.RootID); err != nil {
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

func (s *Service) UsersGroup() group.Group { return s.group }

func (s *Service) NewWriter(tx gorp.Tx, allowInternal bool) Writer {
	return Writer{
		tx:            gorp.OverrideTx(s.cfg.DB, tx),
		otg:           s.cfg.Ontology.NewWriter(tx),
		group:         s.group,
		allowInternal: allowInternal,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{baseTx: s.cfg.DB, gorp: gorp.NewRetrieve[uuid.UUID, Role]()}
}
