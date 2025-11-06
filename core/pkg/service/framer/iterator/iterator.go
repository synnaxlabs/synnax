// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	svcarc "github.com/synnaxlabs/synnax/pkg/service/arc"
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

// ServiceConfig is the configuration for opening the service layer frame Service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	// [OPTIONAL] - defaults to noop instrumentation.
	alamos.Instrumentation
	// DistFramer is the distribution layer frame service to extend.
	// [REQUIRED]
	DistFramer *framer.Service
	// Channel is used to retrieve information about channels.
	// [REQUIRED]
	Channel channel.Readable
	Arc     *svcarc.Service
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

type (
	Config         = framer.IteratorConfig
	StreamIterator = framer.StreamIterator
	Request        = framer.IteratorRequest
	Response       = framer.IteratorResponse
)

const (
	AutoSpan    = iterator.AutoSpan
	SeekFirst   = iterator.SeekFirst
	SeekLast    = iterator.SeekLast
	SeekLE      = iterator.SeekLE
	SeekGE      = iterator.SeekGE
	Next        = iterator.Next
	Prev        = iterator.Prev
	SetBounds   = iterator.SetBounds
	AckResponse = iterator.AckResponse
	Error       = iterator.Error
	Valid       = iterator.Valid
)

type ResponseSegment = confluence.Segment[Response, Response]

func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamIterator, error) {
	p := plumber.New()
	calcTransform, err := s.newCalculationTransform(ctx, &cfg)
	if err != nil {
		return nil, err
	}
	legacyCalcTransform, err := s.newLegacyCalculationTransform(ctx, &cfg)
	if err != nil {
		return nil, err
	}
	dist, err := s.cfg.DistFramer.NewStreamIterator(ctx, cfg)
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

	if legacyCalcTransform != nil {
		plumber.SetSegment(
			p,
			"legacy_calculation",
			legacyCalcTransform,
		)
		plumber.MustConnect[Response](p, routeOutletFrom, "legacy_calculation", 25)
		routeOutletFrom = "legacy_calculation"
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
	alloc := graph.New(graph.Config{
		Channel:        s.cfg.Channel,
		SymbolResolver: s.cfg.Arc.SymbolResolver(),
	})

	// Add all calculated channels to the allocator
	for _, ch := range channels {
		if ch.IsCalculated() && !ch.IsLegacyCalculated() {
			if err := alloc.Add(ctx, ch); err != nil {
				return nil, err
			}
		}
	}

	// Get topologically sorted modules
	modules := alloc.CalculateFlat()

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

	calculatedKeys := alloc.CalculatedKeys()
	concreteBaseKeys := alloc.ConcreteBaseKeys()

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

type Iterator struct {
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	shutdown  context.CancelFunc
	wg        signal.WaitGroup
	value     []Response
}

// Next reads all channel data occupying the next span of time. Returns true
// if the current IteratorServer.View is pointing to any valid segments.
func (i *Iterator) Next(span telem.TimeSpan) bool {
	i.value = nil
	return i.exec(Request{Command: Next, Span: span})
}

// Prev reads all channel data occupying the previous span of time. Returns true
// if the current IteratorServer.View is pointing to any valid segments.
func (i *Iterator) Prev(span telem.TimeSpan) bool {
	i.value = nil
	return i.exec(Request{Command: Prev, Span: span})
}

// SeekFirst seeks the Iterator the start of the Iterator range.
// Returns true if the current IteratorServer.View is pointing to any valid segments.
func (i *Iterator) SeekFirst() bool {
	i.value = nil
	return i.exec(Request{Command: SeekFirst})
}

// SeekLast seeks the Iterator the end of the Iterator range.
// Returns true if the current IteratorServer.View is pointing to any valid segments.
func (i *Iterator) SeekLast() bool {
	i.value = nil
	return i.exec(Request{Command: SeekLast})
}

// SeekLE seeks the Iterator to the first whose timestamp is less than or equal
// to the given timestamp. Returns true if the current IteratorServer.View is pointing
// to any valid segments.
func (i *Iterator) SeekLE(stamp telem.TimeStamp) bool {
	i.value = nil
	return i.exec(Request{Command: SeekLE, Stamp: stamp})
}

// SeekGE seeks the Iterator to the first whose timestamp is greater than the
// given timestamp. Returns true if the current IteratorServer.View is pointing to
// any valid segments.
func (i *Iterator) SeekGE(stamp telem.TimeStamp) bool {
	i.value = nil
	return i.exec(Request{Command: SeekGE, Stamp: stamp})
}

// Valid returns true if the Iterator is pointing at valid data and is error free.
func (i *Iterator) Valid() bool {
	return i.exec(Request{Command: Valid})
}

// Error returns any errors accumulated during the iterators lifetime.
func (i *Iterator) Error() error {
	_, err := i.execErr(Request{Command: Error})
	return err
}

// Close closes the Iterator, ensuring that all in-progress reads complete
// before closing the Source outlet. All iterators must be Closed, or the
// distribution layer will panic.
func (i *Iterator) Close() error {
	defer i.shutdown()
	i.requests.Close()
	return i.wg.Wait()
}

// SetBounds sets the lower and upper bounds of the Iterator.
func (i *Iterator) SetBounds(bounds telem.TimeRange) bool {
	return i.exec(Request{Command: SetBounds, Bounds: bounds})
}

func (i *Iterator) Value() core.Frame {
	frames := make([]core.Frame, len(i.value))
	for i, v := range i.value {
		frames[i] = v.Frame
	}
	return core.MergeFrames(frames)
}

func (i *Iterator) exec(req Request) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *Iterator) execErr(req Request) (bool, error) {
	i.requests.Inlet() <- req
	for res := range i.responses.Outlet() {
		if res.Variant == AckResponse {
			return res.Ack, res.Error
		}
		i.value = append(i.value, res)
	}
	return false, nil
}
