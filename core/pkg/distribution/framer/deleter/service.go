// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deleter

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Service struct {
	channel *channel.Service
	Deleter
	proxy *leaseProxy
}

type ServiceConfig struct {
	HostResolver cluster.HostResolver
	Channel      *channel.Service
	TSChannel    *ts.DB
	Transport    Transport
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.deleter")
	validate.NotNil(v, "host_resolver", c.HostResolver)
	validate.NotNil(v, "ts_channel", c.TSChannel)
	validate.NotNil(v, "aspen_transport", c.Transport)
	validate.NotNil(v, "channel", c.Channel)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.TSChannel = override.Nil(c.TSChannel, other.TSChannel)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

var DefaultConfig = ServiceConfig{}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	proxy, err := newLeaseProxy(cfg)
	if err != nil {
		return nil, err
	}
	s := &Service{
		proxy:   proxy,
		channel: cfg.Channel,
	}
	s.Deleter = s.New()
	return s, nil
}

func (s *Service) New() Deleter {
	return Deleter{proxy: s.proxy, channel: s.channel}
}
