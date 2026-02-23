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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	Signals  *signals.Provider
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Group    *group.Service
	Codec    gorp.Codec[Workspace]
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Codec = override.Nil(c.Codec, other.Codec)
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
	shutdownSignals io.Closer
	table           *gorp.Table[uuid.UUID, Workspace]
	group           group.Group
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	table, err := gorp.OpenTable[uuid.UUID, Workspace](ctx, gorp.TableConfig[Workspace]{
		DB:    cfg.DB,
		Codec: cfg.Codec,
		Migrations: []gorp.Migration{
			gorp.NewCodecTransition[uuid.UUID, Workspace]("msgpack_to_protobuf", cfg.Codec),
		},
	})
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, "Workspaces", ontology.RootID)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, group: g, table: table}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals == nil {
		return s, nil
	}
	signalsCfg := signals.GorpPublisherConfigUUID[Workspace](cfg.DB)
	signalsCfg.Observable = s.table.Observe()
	if s.shutdownSignals, err = signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signalsCfg,
	); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Close() error {
	err := s.shutdownSignals.Close()
	return errors.Join(err, s.table.Close())
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    gorp.OverrideTx(s.cfg.DB, tx),
		otg:   s.cfg.Ontology.NewWriter(tx),
		group: s.group,
		table: s.table,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:    s.cfg.Ontology,
		baseTX: s.cfg.DB,
		gorp:   s.table.NewRetrieve(),
	}
}
