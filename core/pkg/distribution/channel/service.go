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
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

type Service interface {
	Readable
	Writeable
	ontology.Service
	Group() group.Group
	SetCalculationAnalyzer(calc CalculationAnalyzer)
}

type Writeable interface {
	Writer
	NewWriter(tx gorp.Tx) Writer
}

type Readable interface {
	NewRetrieve() Retrieve
	NewObservable() observe.Observable[gorp.TxReader[Key, Channel]]
}

type ReadWriteable interface {
	Writeable
	Readable
}

type CalculationAnalyzer = func(ctx context.Context, expr string) (telem.DataType, error)

// service is central entity for managing channels within delta's distribution layer. It
// provides facilities for creating and retrieving channels.
type service struct {
	*gorp.DB
	Writer
	proxy *leaseProxy
	otg   *ontology.Ontology
	group group.Group
}

func (s *service) SetCalculationAnalyzer(calc CalculationAnalyzer) {
	s.proxy.analyzeCalculation = calc
}

var _ Service = (*service)(nil)

type IntOverflowChecker = func(types.Uint20) error

func FixedOverflowChecker(limit int) IntOverflowChecker {
	return func(count types.Uint20) error {
		if count > types.Uint20(limit) {
			return errors.New("channel limit exceeded")
		}
		return nil
	}
}

type ServiceConfig struct {
	HostResolver     cluster.HostResolver
	ClusterDB        *gorp.DB
	TSChannel        *ts.DB
	Transport        Transport
	Ontology         *ontology.Ontology
	Group            *group.Service
	IntOverflowCheck IntOverflowChecker
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "host_provider", c.HostResolver)
	validate.NotNil(v, "cluster_db", c.ClusterDB)
	validate.NotNil(v, "ts_channel", c.TSChannel)
	validate.NotNil(v, "transport", c.Transport)
	validate.NotNil(v, "int_overflow_check", c.IntOverflowCheck)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.ClusterDB = override.Nil(c.ClusterDB, other.ClusterDB)
	c.TSChannel = override.Nil(c.TSChannel, other.TSChannel)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.IntOverflowCheck = override.Nil(c.IntOverflowCheck, other.IntOverflowCheck)
	return c
}

var DefaultConfig = ServiceConfig{}

func New(ctx context.Context, cfgs ...ServiceConfig) (Service, error) {
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
	s := &service{
		DB:    cfg.ClusterDB,
		proxy: proxy,
		otg:   cfg.Ontology,
		group: g,
	}
	s.Writer = s.NewWriter(nil)
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	return s, nil
}

func (s *service) NewWriter(tx gorp.Tx) Writer {
	return writer{svc: s, tx: s.OverrideTx(tx)}
}

func (s *service) Group() group.Group { return s.group }

func (s *service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:                      gorp.NewRetrieve[Key, Channel](),
		tx:                        s.DB,
		otg:                       s.otg,
		validateRetrievedChannels: s.validateChannels,
	}
}

func (s *service) validateChannels(channels []Channel) ([]Channel, error) {
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

func TryToRetrieveStringer(ctx context.Context, readable Readable, key Key) string {
	var ch Channel
	if readable == nil {
		return key.String()
	}
	if err := readable.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
		return key.String()
	}
	return ch.String()
}
