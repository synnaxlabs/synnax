// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace

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

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	Signals  *signals.Provider
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Group    *group.Service
}

var (
	_             config.Config[ServiceConfig] = ServiceConfig{}
	DefaultConfig                              = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("workspace")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	return v.Error()
}

type Service struct {
	cfg             ServiceConfig
	group           group.Group
	shutdownSignals io.Closer
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, "Workspaces", ontology.RootID)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals == nil {
		return s, nil
	}
	if s.shutdownSignals, err = signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signals.GorpPublisherConfigUUID[Workspace](cfg.DB),
	); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Close() error { return s.shutdownSignals.Close() }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    gorp.OverrideTx(s.cfg.DB, tx),
		otg:   s.cfg.Ontology.NewWriter(tx),
		group: s.group,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:    s.cfg.Ontology,
		baseTX: s.cfg.DB,
		gorp:   gorp.NewRetrieve[uuid.UUID, Workspace](),
	}
}
