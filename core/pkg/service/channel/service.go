// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/synnaxlabs/alamos"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	graph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type (
	Key          = distchannel.Key
	Keys         = distchannel.Keys
	Channel      = distchannel.Channel
	Operation    = distchannel.Operation
	CreateOption = distchannel.CreateOption
)

var (
	RetrieveIfNameExists                        = distchannel.RetrieveIfNameExists
	OverwriteIfNameExistsAndDifferentProperties = distchannel.OverwriteIfNameExistsAndDifferentProperties
	CreateWithoutGroupRelationship              = distchannel.CreateWithoutGroupRelationship
)

type StatusDetails = graph.StatusDetails

type ServiceConfig struct {
	DB           *gorp.DB
	Distribution *distchannel.Service
	Status       *status.Service
	Arc          *arc.Service
	alamos.Instrumentation
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

func (c ServiceConfig) Validate() error {
	v := validate.New("service.channel")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "distribution", c.Distribution)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "arc", c.Arc)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Distribution = override.Nil(c.Distribution, other.Distribution)
	c.Status = override.Nil(c.Status, other.Status)
	c.Arc = override.Nil(c.Arc, other.Arc)
	return c
}

type Service struct {
	cfg    ServiceConfig
	Writer Writer
	graph  *graph.Graph
}

func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.graph, err = graph.Open(ctx, graph.Config{
		Channel: cfg.Distribution,
		Status:  cfg.Status,
	}); err != nil {
		return nil, err
	}
	s.Writer = s.NewWriter(nil)
	return s, nil
}

func (s *Service) Close() error {
	if s.graph != nil {
		return s.graph.Close()
	}
	return nil
}

func (s *Service) Group() group.Group { return s.cfg.Distribution.Group() }

func (s *Service) NewRetrieve() distchannel.Retrieve { return s.cfg.Distribution.NewRetrieve() }

func (s *Service) NewObservable() observe.Observable[gorp.TxReader[Key, Channel]] {
	return s.cfg.Distribution.NewObservable()
}

func (s *Service) CountExternalNonVirtual() uint32 {
	return s.cfg.Distribution.CountExternalNonVirtual()
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		Writer:   s.cfg.Distribution.NewWriter(tx),
		tx:       gorp.OverrideTx(s.cfg.DB, tx),
		analyzer: analyzer.New(s.cfg.Arc.NewSymbolResolver(tx)),
	}
}

func (s *Service) Create(ctx context.Context, ch *Channel, opts ...CreateOption) error {
	return s.NewWriter(nil).Create(ctx, ch, opts...)
}

func (s *Service) CreateMany(ctx context.Context, channels *[]Channel, opts ...CreateOption) error {
	return s.NewWriter(nil).CreateMany(ctx, channels, opts...)
}

func (s *Service) Delete(ctx context.Context, key Key, allowInternal bool) error {
	return s.NewWriter(nil).Delete(ctx, key, allowInternal)
}

func (s *Service) DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error {
	return s.NewWriter(nil).DeleteMany(ctx, keys, allowInternal)
}

func (s *Service) DeleteByName(ctx context.Context, name string, allowInternal bool) error {
	return s.NewWriter(nil).DeleteByName(ctx, name, allowInternal)
}

func (s *Service) DeleteManyByNames(ctx context.Context, names []string, allowInternal bool) error {
	return s.NewWriter(nil).DeleteManyByNames(ctx, names, allowInternal)
}

func (s *Service) Rename(ctx context.Context, key Key, newName string, allowInternal bool) error {
	return s.NewWriter(nil).Rename(ctx, key, newName, allowInternal)
}

func (s *Service) RenameMany(ctx context.Context, keys []Key, names []string, allowInternal bool) error {
	return s.NewWriter(nil).RenameMany(ctx, keys, names, allowInternal)
}

func (s *Service) MapRename(ctx context.Context, names map[string]string, allowInternal bool) error {
	return s.NewWriter(nil).MapRename(ctx, names, allowInternal)
}
