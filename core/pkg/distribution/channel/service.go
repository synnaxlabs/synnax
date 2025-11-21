// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

type CalculationAnalyzer = func(ctx context.Context, expr string) (telem.DataType, error)

// Service is central entity for managing channels within delta's distribution layer. It
// provides facilities for creating and retrieving channels.
type Service struct {
	cfg Config
	db  *gorp.DB
	Writer
	proxy *leaseProxy
	otg   *ontology.Ontology
	group group.Group
}

func (s *Service) SetCalculationAnalyzer(calc CalculationAnalyzer) {
	s.proxy.analyzeCalculation = calc
}

type IntOverflowChecker = func(types.Uint20) error

func FixedOverflowChecker(limit int) IntOverflowChecker {
	return func(count types.Uint20) error {
		if count > types.Uint20(limit) {
			return errors.New("channel limit exceeded")
		}
		return nil
	}
}

type Config struct {
	HostResolver     cluster.HostResolver
	ClusterDB        *gorp.DB
	TSChannel        *ts.DB
	Transport        Transport
	Ontology         *ontology.Ontology
	Group            *group.Service
	IntOverflowCheck IntOverflowChecker
	// ValidateNames sets whether to validate channel names during creation and
	// renaming.
	ValidateNames *bool
	// ForceMigration will force all migrations to run, regardless of whether they have
	// already been run.
	ForceMigration *bool
}

var _ config.Config[Config] = Config{}

func (c Config) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "host_provider", c.HostResolver)
	validate.NotNil(v, "cluster_db", c.ClusterDB)
	validate.NotNil(v, "ts_channel", c.TSChannel)
	validate.NotNil(v, "transport", c.Transport)
	validate.NotNil(v, "int_overflow_check", c.IntOverflowCheck)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.ClusterDB = override.Nil(c.ClusterDB, other.ClusterDB)
	c.TSChannel = override.Nil(c.TSChannel, other.TSChannel)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.IntOverflowCheck = override.Nil(c.IntOverflowCheck, other.IntOverflowCheck)
	return c
}

var DefaultConfig = Config{ValidateNames: config.True(), ForceMigration: config.False()}

func New(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	var g group.Group
	if cfg.Group != nil {
		if g, err = cfg.Group.CreateOrRetrieve(ctx, "Channels", ontology.RootID); err != nil {
			return nil, err
		}
	}
	proxy, err := newLeaseProxy(ctx, cfg, g)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:   cfg,
		db:    cfg.ClusterDB,
		proxy: proxy,
		otg:   cfg.Ontology,
		group: g,
	}
	s.Writer = s.NewWriter(nil)
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: s.db.OverrideTx(tx)}
}

func (s *Service) Group() group.Group { return s.group }

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:                      gorp.NewRetrieve[Key, Channel](),
		tx:                        s.db,
		otg:                       s.otg,
		validateRetrievedChannels: s.validateChannels,
	}
}

func (s *Service) validateChannels(channels []Channel) ([]Channel, error) {
	res := make([]Channel, 0, len(channels))
	s.proxy.mu.RLock()
	defer s.proxy.mu.RUnlock()
	for i, key := range KeysFromChannels(channels) {
		if s.proxy.mu.externalNonVirtualSet.Contains(key) {
			channelNumber := s.proxy.mu.externalNonVirtualSet.NumLessThan(key) + 1
			if err := s.proxy.IntOverflowCheck(types.Uint20(channelNumber)); err != nil {
				return nil, err
			}
		}
		res = append(res, channels[i])
	}
	return res, nil
}

func TryToRetrieveStringer(ctx context.Context, svc *Service, key Key) string {
	if svc == nil {
		return key.String()
	}
	var ch Channel
	if err := svc.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
		return key.String()
	}
	return ch.String()
}
