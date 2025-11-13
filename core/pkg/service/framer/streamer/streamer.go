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
	"github.com/synnaxlabs/x/validate"
)

type (
	Request         = Config
	Response        = framer.StreamerResponse
	Streamer        = confluence.Segment[Request, Response]
	responseSegment = confluence.Segment[Response, Response]
)

type Config struct {
	Keys             channel.Keys `json:"keys" msgpack:"keys"`
	SendOpenAck      bool         `json:"send_open_ack" msgpack:"send_open_ack"`
	DownsampleFactor int          `json:"downsample_factor" msgpack:"downsample_factor"`
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (cfg Config) Validate() error {
	v := validate.New("streamer.config")
	validate.GreaterThanEq(v, "downsample_factor", cfg.DownsampleFactor, 0)
	return v.Error()
}

// Override implements config.Config.
func (cfg Config) Override(other Config) Config {
	cfg.Keys = override.Slice(cfg.Keys, other.Keys)
	cfg.SendOpenAck = other.SendOpenAck
	cfg.DownsampleFactor = override.Numeric(cfg.DownsampleFactor, other.DownsampleFactor)
	return cfg
}

func (cfg Config) distribution() framer.StreamerConfig {
	return framer.StreamerConfig{Keys: cfg.Keys, SendOpenAck: &cfg.SendOpenAck}
}

// ServiceConfig is the configuration for opening a new streamer service.
type ServiceConfig struct {
	alamos.Instrumentation
	Calculation *calculation.Service
	Channel     channel.Readable
	DistFramer  *framer.Service
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.Calculation = override.Nil(cfg.Calculation, other.Calculation)
	cfg.Channel = override.Nil(cfg.Channel, other.Channel)
	cfg.DistFramer = override.Nil(cfg.DistFramer, other.DistFramer)
	return cfg
}

func (cfg ServiceConfig) Validate() error {
	v := validate.New("streamer")
	validate.NotNil(v, "calculation", cfg.Calculation)
	validate.NotNil(v, "channel", cfg.Channel)
	validate.NotNil(v, "dist_framer", cfg.DistFramer)
	return v.Error()
}

type Service struct {
	cfg ServiceConfig
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

var (
	distAddr       address.Address = "distribution"
	utAddr         address.Address = "updater_transform"
	downsampleAddr address.Address = "downsample"
)

const (
	responseBufferSize = 200
	requestBufferSize  = 10
)

func (s *Service) New(ctx context.Context, cfgs ...Config) (Streamer, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	p := plumber.New()
	dist, err := s.cfg.DistFramer.NewStreamer(ctx, cfg.distribution())
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
	var routeOutletFrom = distAddr
	if cfg.DownsampleFactor > 1 {
		plumber.SetSegment(p, downsampleAddr, newDownsampler(cfg))
		plumber.MustConnect[Response](p, routeOutletFrom, downsampleAddr, responseBufferSize)
		routeOutletFrom = downsampleAddr
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
		readable:        s.cfg.Channel,
	}
	ut.Transform = ut.transform
	return ut, ut.calcManager.Set(ctx, cfg.Keys)
}
