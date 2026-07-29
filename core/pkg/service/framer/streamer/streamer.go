// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type (
	Request         = Config
	Response        = framer.StreamerResponse
	Streamer        = confluence.Segment[Request, Response]
	responseSegment = confluence.Segment[Response, Response]
)

type Config struct {
	Keys             channel.Keys `json:"keys"              msgpack:"keys"`
	SendOpenAck      bool         `json:"send_open_ack"     msgpack:"send_open_ack"`
	DownsampleFactor int          `json:"downsample_factor" msgpack:"downsample_factor"`
	ThrottleRate     telem.Rate   `json:"throttle_rate"     msgpack:"throttle_rate"`
	ExcludeGroups    []uint32     `json:"exclude_groups"    msgpack:"exclude_groups"`
}

var _ config.Config[Config] = Config{}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("streamer.config")
	validate.GreaterThanEq(v, "downsample_factor", c.DownsampleFactor, 0)
	validate.GreaterThanEq(v, "throttle_rate", c.ThrottleRate, 0)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Keys = override.Slice(c.Keys, other.Keys)
	c.SendOpenAck = other.SendOpenAck
	c.DownsampleFactor = override.Numeric(c.DownsampleFactor, other.DownsampleFactor)
	c.ThrottleRate = override.Numeric(c.ThrottleRate, other.ThrottleRate)
	c.ExcludeGroups = override.Slice(c.ExcludeGroups, other.ExcludeGroups)
	return c
}

func (c Config) distribution() framer.StreamerConfig {
	return framer.StreamerConfig{
		Keys:          c.Keys,
		SendOpenAck:   &c.SendOpenAck,
		ExcludeGroups: c.ExcludeGroups,
	}
}

// ServiceConfig is the configuration for opening a new streamer service.
type ServiceConfig struct {
	// Calculation is used to update the calculation request manager with the provided
	// keys.
	//
	// [REQUIRED]
	Calculation *calculation.Service
	// Channel is used to retrieve channel information.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer is the underlying framer service to stream telemetry frames.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

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

// Service is the service layer entry point for using streamers to stream telemetry
// frames.
type Service struct{ cfg ServiceConfig }

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

var (
	distAddr       address.Address = "distribution"
	utAddr         address.Address = "updater_transform"
	downsampleAddr address.Address = "downsample"
	throttleAddr   address.Address = "throttle"
)

const (
	responseBufferSize = 200
	requestBufferSize  = 10
)

func (s *Service) New(ctx context.Context, cfgs ...Config) (Streamer, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return nil, err
	}
	p := plumber.New()
	dist, err := s.cfg.Framer.NewStreamer(cfg.distribution())
	if err != nil {
		return nil, err
	}
	plumber.SetSegment(p, distAddr, dist)
	ut, err := s.newCalculationUpdaterTransform(ctx, cfg)
	if err != nil {
		return nil, err
	}
	plumber.SetSegment(p, utAddr, ut)
	plumber.MustConnect[framer.StreamerRequest](p, utAddr, distAddr, requestBufferSize)
	routeOutletFrom := distAddr
	if cfg.DownsampleFactor > 1 {
		plumber.SetSegment(p, downsampleAddr, newDownsampler(cfg))
		plumber.MustConnect[Response](
			p,
			routeOutletFrom,
			downsampleAddr,
			responseBufferSize,
		)
		routeOutletFrom = downsampleAddr
	}
	if cfg.ThrottleRate > 0 {
		plumber.SetSegment(p, throttleAddr, newThrottle(cfg))
		plumber.MustConnect[Response](
			p,
			routeOutletFrom,
			throttleAddr,
			responseBufferSize,
		)
		routeOutletFrom = throttleAddr
	}
	return &plumber.Segment[Request, Response]{
		Pipeline:         p,
		RouteInletsTo:    []address.Address{utAddr},
		RouteOutletsFrom: []address.Address{routeOutletFrom},
	}, nil
}

func (s *Service) newCalculationUpdaterTransform(
	ctx context.Context,
	cfg Config,
) (confluence.Segment[Request, framer.StreamerRequest], error) {
	ut := &calculationUpdaterTransform{
		Instrumentation: s.cfg.Instrumentation,
		calcManager:     s.cfg.Calculation.OpenRequestManager(),
		channelSvc:      s.cfg.Channel,
	}
	ut.Transform = ut.transform
	return ut, ut.calcManager.Set(ctx, cfg.Keys)
}
