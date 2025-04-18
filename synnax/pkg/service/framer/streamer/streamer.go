// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package streamer

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	Keys             channel.Keys `json:"keys" msgpack:"keys"`
	SendOpenAck      bool         `json:"send_open_ack" msgpack:"send_open_ack"`
	DownSampleFactor int          `json:"down_sample_factor" msgpack:"down_sample_factor"`
	ThrottleRate     telem.Rate   `json:"throttle_rate" msgpack:"throttle_rate"`
}

func (cfg Config) distribution() framer.StreamerConfig {
	return framer.StreamerConfig{
		Keys:        cfg.Keys,
		SendOpenAck: cfg.SendOpenAck,
	}
}

// ServiceConfig is the configuration for opening a new streamer service.
type ServiceConfig struct {
	alamos.Instrumentation
	Calculation *calculation.Service `json:"calculation" msgpack:"calculation"`
	Channel     channel.Readable     `json:"channel" msgpack:"channel"`
	Framer      *framer.Service      `json:"framer" msgpack:"framer"`
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.Calculation = override.Nil(cfg.Calculation, other.Calculation)
	cfg.Channel = override.Nil(cfg.Channel, other.Channel)
	cfg.Framer = override.Nil(cfg.Framer, other.Framer)
	return cfg
}

func (cfg ServiceConfig) Validate() error {
	v := validate.New("streamer")
	validate.NotNil(v, "calculation", cfg.Calculation)
	validate.NotNil(v, "channel", cfg.Channel)
	validate.NotNil(v, "framer", cfg.Framer)
	return v.Error()
}

type (
	Request  = Config
	Streamer = confluence.Segment[Request, framer.StreamerResponse]
)

type Service struct {
	cfg ServiceConfig
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	return &Service{cfg: cfg}, err
}

func (s *Service) New(ctx context.Context, cfg Config) (Streamer, error) {
	p := plumber.New()
	dist, err := s.cfg.Framer.NewStreamer(ctx, cfg.distribution())
	if err != nil {
		return nil, err
	}
	plumber.SetSegment(p, "distribution", dist)
	ut, err := s.newUpdaterTransform(ctx, cfg)
	if err != nil {
		return nil, err
	}
	plumber.SetSegment(p, "updater-transform", ut)
	plumber.MustConnect[framer.StreamerRequest](p, "updater-transform", "distribution", 25)
	var routeOutletFrom address.Address = "distribution"
	if cfg.ThrottleRate > 0 {
		plumber.SetSegment(p, "throttle", newThrottle(cfg))
		plumber.MustConnect[framer.StreamerResponse](p, routeOutletFrom, "throttle", 25)
		routeOutletFrom = "throttle"
	}
	if cfg.DownSampleFactor > 0 {
		plumber.SetSegment(p, "down-sample", newDownSampler(cfg))
		plumber.MustConnect[framer.StreamerResponse](p, routeOutletFrom, "down-sample", 25)
		routeOutletFrom = "down-sample"
	}
	return &plumber.Segment[Request, framer.StreamerResponse]{
		Pipeline:         p,
		RouteInletsTo:    []address.Address{"updater-transform"},
		RouteOutletsFrom: []address.Address{routeOutletFrom},
	}, nil
}

func (s *Service) newUpdaterTransform(
	ctx context.Context,
	cfg Config,
) (confluence.Segment[Request, framer.StreamerRequest], error) {
	ut := &calculationUpdaterTransform{
		Instrumentation: s.cfg.Instrumentation,
		c:               s.cfg.Calculation,
		readable:        s.cfg.Channel,
	}
	ut.Transform = ut.transform
	return ut, ut.update(ctx, cfg.Keys)
}
