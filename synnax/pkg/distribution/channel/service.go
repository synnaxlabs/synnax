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
	HostResolver           core.HostResolver
	ClusterDB              *gorp.DB
	TSChannel              *ts.DB
	Transport              Transport
	Ontology               *ontology.Ontology
	Group                  *group.Service
	ValidateChannelCount   func(count int64) error
	ChannelsNeedValidation func() (int, error)
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "HostProvider", c.HostResolver)
	validate.NotNil(v, "ClusterDB", c.ClusterDB)
	validate.NotNil(v, "TSChannel", c.TSChannel)
	validate.NotNil(v, "Transport", c.Transport)
	validate.NotNil(v, "ValidateChannelCount", c.ValidateChannelCount)
	validate.NotNil(v, "ChannelsNeedValidation", c.ChannelsNeedValidation)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.ClusterDB = override.Nil(c.ClusterDB, other.ClusterDB)
	c.TSChannel = override.Nil(c.TSChannel, other.TSChannel)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.ValidateChannelCount = override.Nil(c.ValidateChannelCount, other.ValidateChannelCount)
	c.ChannelsNeedValidation = override.Nil(c.ChannelsNeedValidation, other.ChannelsNeedValidation)
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
	return newRetrieve(s.DB, s.otg, func(channels []Channel) ([]Channel, error) {
		maxAllowed, err := s.proxy.ChannelsNeedValidation()

		if err != nil {
			var fErr error
			keys := KeysFromChannels(channels)

			var returnedChannels []Channel
			var err error
			for i, key := range keys {
				deletedCount := s.proxy.deletedChannels.GetChannelsBeforeKey(key)
				internalCount := s.proxy.internalChannels.GetChannelsBeforeKey(key)
				keyNumber := key.LocalKey() - deletedCount - internalCount
				if keyNumber < uint16(maxAllowed) {
					returnedChannels = append(returnedChannels, channels[i])
				} else {
					fErr = err
				}
			}
			return returnedChannels, fErr
		}
		return channels, nil
	})
}
