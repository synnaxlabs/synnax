// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/graph"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening an Iterator.
type Config struct {
	// Keys are the keys of the channels to iterate over. At least one key must be
	// specified. An iterator cannot iterate over virtual channels or free channels, and
	// calls to Open or NewStream will return an error when attempting to iterate over
	// channels of these types.
	//
	// [REQUIRED]
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Bounds are the bounds of the time range to iterate over. This time range must be
	// valid i.e., the start value must be before or equal to the end value.
	//
	// [REQUIRED]
	Bounds telem.TimeRange `json:"bounds" msgpack:"bounds"`
	// ChunkSize sets the default number of samples to iterate over per-channel when
	// calling Next or Prev with AutoSpan.
	//
	// [OPTIONAL]
	ChunkSize int64 `json:"chunk_size" msgpack:"chunk_size"`
	// DownsampleFactor is the factor to downsample the data by If DownsampleFactor is
	// less than or equal to 1, no downsampling will be performed.
	//
	// [OPTIONAL]
	DownsampleFactor int `json:"downsample_factor" msgpack:"downsample_factor"`
}

func (c Config) distribution() framer.IteratorConfig {
	return iterator.Config{Keys: c.Keys, Bounds: c.Bounds, ChunkSize: c.ChunkSize}
}

// ServiceConfig is the configuration for opening the service layer frame Service.
type ServiceConfig struct {
	// DistFramer is the distribution layer frame service to extend.
	// [REQUIRED]
	DistFramer *framer.Service
	// Channel is used to retrieve information about channels.
	//
	// [REQUIRED]
	Channel *channel.Service
	Arc     *arc.Service
	// Instrumentation is for logging, tracing, and metrics.
	// [OPTIONAL] - defaults to noop instrumentation.
	alamos.Instrumentation
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a Service. This
	// configuration is not valid on its own and must be overridden with the required
	// fields specified in ServiceConfig.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.DistFramer = override.Nil(cfg.DistFramer, other.DistFramer)
	cfg.Channel = override.Nil(cfg.Channel, other.Channel)
	cfg.Arc = override.Nil(cfg.Arc, other.Arc)
	return cfg
}

// Validate implements config.Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("iterator")
	validate.NotNil(v, "framer", cfg.DistFramer)
	validate.NotNil(v, "channel", cfg.Channel)
	validate.NotNil(v, "arc", cfg.Arc)
	return v.Error()
}

// Service is the service layer entry point for using iterators to read historical
// telemetry from a multi-node Synnax cluster.
type Service struct{ cfg ServiceConfig }

// NewService creates a new service using the provided configuration(s). Each subsequent
// configuration overrides the one in the previous configuration. If the configuration
// is invalid, NewService returns a nil service and a non-nil error.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamIterator, error) {
	p := plumber.New()
	calcTransform, err := s.newCalculationTransform(ctx, &cfg)
	if err != nil {
		return nil, err
	}
	dist, err := s.cfg.DistFramer.NewStreamIterator(ctx, cfg.distribution())
	if err != nil {
		return nil, err
	}
	plumber.SetSegment(p, "distribution", dist)
	var routeOutletFrom address.Address = "distribution"
	if calcTransform != nil {
		plumber.SetSegment(
			p,
			"calculation",
			calcTransform,
			confluence.DeferErr(calcTransform.close),
		)
		plumber.MustConnect[Response](p, routeOutletFrom, "calculation", 25)
		routeOutletFrom = "calculation"
	}
	if cfg.DownsampleFactor > 1 {
		plumber.SetSegment(
			p,
			"downsampler",
			newDownsampler(cfg),
		)
		plumber.MustConnect[Response](p, routeOutletFrom, "downsampler", 25)
		routeOutletFrom = "downsampler"
	}
	return &plumber.Segment[Request, Response]{
		Pipeline:         p,
		RouteInletsTo:    []address.Address{"distribution"},
		RouteOutletsFrom: []address.Address{routeOutletFrom},
	}, nil
}

func (s *Service) Open(ctx context.Context, cfg Config) (*Iterator, error) {
	stream, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.cfg.Instrumentation))
	req := confluence.NewStream[Request]()
	res := confluence.NewStream[Response]()
	stream.InFrom(req)
	stream.OutTo(res)
	stream.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.CancelOnFail(),
	)
	return &Iterator{requests: req, responses: res, shutdown: cancel, wg: sCtx}, nil
}

func (s *Service) newCalculationTransform(ctx context.Context, cfg *Config) (*calculationTransform, error) {
	originalKeys := slices.Clone(cfg.Keys)

	// Fetch the requested channels
	var channels []channel.Channel
	if err := s.cfg.Channel.NewRetrieve().
		WhereKeys(cfg.Keys...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}

	// Use allocator to resolve dependencies and get topological order
	calcGraph, err := graph.New(graph.Config{
		Channel:        s.cfg.Channel,
		SymbolResolver: s.cfg.Arc.SymbolResolver(),
	})
	if err != nil {
		return nil, err
	}

	// Add all calculated channels to the allocator
	for _, ch := range channels {
		if ch.IsCalculated() {
			if err := calcGraph.Add(ctx, ch); err != nil {
				return nil, err
			}
		}
	}

	// Get topologically sorted modules
	modules := calcGraph.CalculateFlat()

	// If no calculated channels, no transform needed
	if len(modules) == 0 {
		return nil, nil
	}

	// Open calculators from modules
	calculators := make([]*calculator.Calculator, 0, len(modules))
	for _, mod := range modules {
		calc, err := calculator.Open(ctx, calculator.Config{Module: mod})
		if err != nil {
			return nil, err
		}
		calculators = append(calculators, calc)
	}

	calculatedKeys := calcGraph.CalculatedKeys()
	concreteBaseKeys := calcGraph.ConcreteBaseKeys()

	// Fetch concrete base channel metadata to get their indices
	var concreteBaseChannels []channel.Channel
	if len(concreteBaseKeys) > 0 {
		if err := s.cfg.Channel.NewRetrieve().
			Entries(&concreteBaseChannels).
			WhereKeys(concreteBaseKeys.Keys()...).
			Exec(ctx, nil); err != nil {
			return nil, err
		}
	}

	// Update cfg.Keys to include concrete base keys and their indices
	cfg.Keys = lo.Uniq(append(cfg.Keys, concreteBaseKeys.Keys()...))
	cfg.Keys = lo.Uniq(append(cfg.Keys, lo.FilterMap(
		concreteBaseChannels,
		func(item channel.Channel, index int) (channel.Key, bool) {
			return item.Index(), !item.Virtual
		})...,
	))

	// Remove ALL calculated keys (including nested ones) from cfg.Keys
	cfg.Keys = lo.Filter(cfg.Keys, func(item channel.Key, index int) bool {
		return !calculatedKeys.Contains(item) && !item.Free()
	})

	return newCalculationTransform(originalKeys, calculators), nil
}
