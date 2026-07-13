// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package writer is the service-layer entry point for writing telemetry frames to a
// Synnax cluster. It validates writes against channel metadata retrieved from the
// service-layer channel service before forwarding them to the distribution-layer
// writer.
package writer

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening a Writer or StreamWriter. It is an alias for
// the distribution-layer writer configuration.
type Config = writer.Config

// ServiceConfig is the configuration for opening the service-layer writer Service.
type ServiceConfig struct {
	// Framer is the distribution-layer framer service writes are forwarded to.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used to retrieve channel metadata for validating writes.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.Framer = override.Nil(cfg.Framer, other.Framer)
	cfg.Channel = override.Nil(cfg.Channel, other.Channel)
	return cfg
}

// Validate implements config.Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("service.framer.writer")
	validate.NotNil(v, "framer", cfg.Framer)
	validate.NotNil(v, "channel", cfg.Channel)
	return v.Error()
}

// Service is the service-layer entry point for opening Writers and StreamWriters that
// write telemetry frames to a Synnax cluster.
type Service struct{ cfg ServiceConfig }

// NewService creates a new service using the provided configuration(s). Each subsequent
// configuration overrides the one in the previous configuration. If the configuration
// is invalid, NewService returns a nil service and a non-nil error.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

const (
	validatorAddr    address.Address = "validator"
	distributionAddr address.Address = "distribution"
)

// NewStream opens a StreamWriter for the channels in cfg, driven through confluence
// inlet requests and outlet responses. It returns an error if cfg is invalid or any
// channel in cfg.Keys cannot be resolved.
func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamWriter, error) {
	keys := cfg.Keys.Unique()
	var channels []channel.Channel
	if err := s.cfg.Channel.NewRetrieve().
		Where(channel.MatchKeys(keys...)).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	channelMap := make(map[channel.Key]channel.Channel, len(channels))
	cfg.FreeIndexes = make(map[channel.Key]channel.Key)
	for _, ch := range channels {
		channelMap[ch.Key()] = ch
		if ch.Free() && ch.Index() != 0 {
			cfg.FreeIndexes[ch.Key()] = ch.Index()
		}
	}
	dist, err := s.cfg.Framer.NewStreamWriter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	plumber.SetSegment[Request, Request](pipe, validatorAddr, newValidator(channelMap))
	plumber.SetSegment[Request, Response](pipe, distributionAddr, dist)
	plumber.MustConnect[Request](pipe, validatorAddr, distributionAddr, 30)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo(validatorAddr))
	lo.Must0(seg.RouteOutletFrom(distributionAddr))
	return seg, nil
}

// Open opens a buffered Writer for the channels in cfg starting at cfg.Start. The
// caller must Close the returned Writer to release its control over the written region.
// It returns an error if cfg is invalid or any channel in cfg.Keys cannot be resolved.
func (s *Service) Open(ctx context.Context, cfg Config) (*Writer, error) {
	resolved, err := config.New(writer.DefaultConfig(), cfg)
	if err != nil {
		return nil, err
	}
	seg, err := s.NewStream(ctx, resolved)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithInstrumentation(s.cfg.Instrumentation),
	)
	req := confluence.NewStream[Request]()
	res := confluence.NewStream[Response]()
	seg.InFrom(req)
	seg.OutTo(res)
	seg.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.CancelOnFail(),
		confluence.RecoverWithErrOnPanic(),
	)
	return &Writer{
		cfg:       resolved,
		requests:  req,
		responses: res,
		shutdown:  signal.NewHardShutdown(sCtx, cancel),
	}, nil
}
