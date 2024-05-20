// Copyright 2023 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Service interface {
	Readable
	Writeable
	ontology.Service
}

type Writeable interface {
	Writer
	NewWriter(tx gorp.Tx) Writer
}

type Readable interface {
	NewRetrieve() Retrieve
}

type ReadWriteable interface {
	Writeable
	Readable
}

// service is central entity for managing channels within delta's distribution layer. It provides facilities for creating
// and retrieving channels.
type service struct {
	*gorp.DB
	Writer
	proxy *leaseProxy
	otg   *ontology.Ontology
	group group.Group
}

var _ Service = (*service)(nil)

type ServiceConfig struct {
	HostResolver     core.HostResolver
	ClusterDB        *gorp.DB
	TSChannel        *ts.DB
	Transport        Transport
	Ontology         *ontology.Ontology
	Group            *group.Service
	IntOverflowCheck func(count int64) error
	GetChannelCount  func() (int, error)
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "HostProvider", c.HostResolver)
	validate.NotNil(v, "ClusterDB", c.ClusterDB)
	validate.NotNil(v, "TSChannel", c.TSChannel)
	validate.NotNil(v, "Transport", c.Transport)
	validate.NotNil(v, "IntOverflowCheck", c.IntOverflowCheck)
	validate.NotNil(v, "GetChannelCount", c.GetChannelCount)
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
	c.GetChannelCount = override.Nil(c.GetChannelCount, other.GetChannelCount)
	return c
}

var DefaultConfig = ServiceConfig{}

const groupName = "Channels"

func New(ctx context.Context, configs ...ServiceConfig) (Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	var g group.Group
	if cfg.Group != nil {
		g, err = cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
		if err != nil {
			return nil, err
		}
	}
	proxy, err := newLeaseProxy(cfg, g)
	if err != nil {
		return nil, err
	}

	s := &service{
		DB:    cfg.ClusterDB,
		proxy: proxy,
		otg:   cfg.Ontology,
	}
	s.Writer = s.NewWriter(nil)
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	return s, nil
}

func (s *service) NewWriter(tx gorp.Tx) Writer {
	return writer{proxy: s.proxy, tx: s.DB.OverrideTx(tx)}
}

func (s *service) NewRetrieve() Retrieve {
	return newRetrieve(s.DB, s.otg, s.validateChannels)
}

func (s *service) validateChannels(channels []Channel) ([]Channel, error) {
	maxAllowed, err := s.proxy.GetChannelCount()
	if err == nil {
		return channels, nil
	}
	var vErr error = nil
	keys := KeysFromChannels(channels)
	returnedChannels := make([]Channel, 0, len(channels))
	for i, key := range keys {
		deletedCount := s.proxy.deleted.NumLessThan(key.LocalKey())
		internalCount := s.proxy.internal.NumLessThan(key.LocalKey())
		keyNumber := key.LocalKey() - deletedCount - internalCount
		if keyNumber < uint16(maxAllowed) {
			returnedChannels = append(returnedChannels, channels[i])
		} else {
			vErr = err
		}
	}
	return returnedChannels, vErr
}
