// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"fmt"
	"go/types"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/legacy"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for opening the calculation service.
type ServiceConfig struct {
	alamos.Instrumentation
	// Framer is the underlying frame service to stream cache channel values and write
	// calculated samples.
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used to retrieve information about the channels being calculated.
	// [REQUIRED]
	Channel channel.Service
	// ChannelObservable is used to listen to real-time changes in calculated channels
	// so the calculation routines can be updated accordingly.
	// [REQUIRED]
	ChannelObservable observe.Observable[gorp.TxReader[channel.Key, channel.Channel]]
	// Arc is used for compiling arc programs used for executing calculations.
	// [REQUIRED]
	Arc *arc.Service
	// StateCodec is the encoder/decoder used to communicate calculation state
	// changes.
	// [OPTIONAL]
	StateCodec binary.Codec
	// EnableLegacyCalculations sets whether to enable the legacy, lua-based calculated
	// channel engine.
	// [OPTIONAL] - Default false
	EnableLegacyCalculations *bool
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening the calculation service.
	DefaultConfig = ServiceConfig{
		StateCodec:               &binary.JSONCodec{},
		EnableLegacyCalculations: config.False(),
	}
)

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("calculate")
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "channel_observable", c.ChannelObservable)
	validate.NotNil(v, "state_codec", c.StateCodec)
	validate.NotNil(v, "enable_legacy_calculations", c.EnableLegacyCalculations)
	validate.NotNil(v, "arc", c.Arc)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.ChannelObservable = override.Nil(c.ChannelObservable, other.ChannelObservable)
	c.StateCodec = override.Nil(c.StateCodec, other.StateCodec)
	c.Arc = override.Nil(c.Arc, other.Arc)
	c.EnableLegacyCalculations = override.Nil(c.EnableLegacyCalculations, other.EnableLegacyCalculations)
	return c
}

// entry is used to manage the lifecycle of a calculation.
type entry struct {
	// channel is the calculated channel.
	ch channel.Channel
	// count is the number of active requests for the calculation.
	count int
	// calculation is used to gracefully stop the calculation.
	calculation confluence.Closable
	// shutdown is used to force stop the calculation by cancelling the context.
	shutdown io.Closer
}

type Status = status.Status[types.Nil]

// Service creates and operates calculations on channels.
type Service struct {
	cfg ServiceConfig
	mu  struct {
		sync.Mutex
		entries map[channel.Key]*entry
	}
	disconnectFromChannelChanges observe.Disconnect
	stateKey                     channel.Key
	writer                       *framer.Writer
	legacy                       *legacy.Service
}

const StatusChannelName = "sy_calculation_status"

// OpenService opens the service with the provided configuration. The service must be closed
// when it is no longer needed.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	calculationStateCh := channel.Channel{
		Name:        StatusChannelName,
		DataType:    telem.JSONT,
		Virtual:     true,
		Leaseholder: cluster.Free,
		Internal:    true,
	}

	if err = cfg.Channel.MapRename(ctx, map[string]string{
		"sy_calculation_state": StatusChannelName,
	}, true); err != nil {
		return nil, err
	}

	if err = cfg.Channel.Create(
		ctx,
		&calculationStateCh,
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}

	w, err := cfg.Framer.OpenWriter(ctx, framer.WriterConfig{
		Keys:        []channel.Key{calculationStateCh.Key()},
		Start:       telem.Now(),
		Authorities: []control.Authority{255},
	})
	if err != nil {
		return nil, err
	}

	s := &Service{cfg: cfg, writer: w, stateKey: calculationStateCh.Key()}
	s.disconnectFromChannelChanges = cfg.ChannelObservable.OnChange(s.handleChange)
	s.mu.entries = make(map[channel.Key]*entry)

	if *cfg.EnableLegacyCalculations {
		s.legacy, err = legacy.OpenService(ctx, legacy.ServiceConfig{
			Channel:           cfg.Channel,
			Framer:            cfg.Framer,
			ChannelObservable: cfg.ChannelObservable,
			StateCodec:        cfg.StateCodec,
		})
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) setStatus(
	_ context.Context,
	status Status,
) {
	if _, err := s.writer.Write(core.UnaryFrame(
		s.stateKey,
		telem.NewSeriesStaticJSONV(status),
	)); err != nil {
		s.cfg.L.Error("failed to encode state", zap.Error(err))
	}
}

func (s *Service) handleChange(
	ctx context.Context,
	reader gorp.TxReader[channel.Key, channel.Channel],
) {
	c, ok := reader.Next(ctx)
	if !ok {
		return
	}
	// Don't stop calculating if the channel is deleted. The calculation will be
	// automatically shut down when it is no longer needed.
	if c.Variant != change.Set || !c.Value.IsCalculated() || c.Value.IsLegacyCalculated() {
		return
	}
	existing, found := s.mu.entries[c.Key]
	if !found {
		s.update(ctx, c.Value)
		return
	}
	if existing.ch.Equals(c.Value, "Name") {
		return
	}
	s.update(ctx, c.Value)
}

func (s *Service) update(ctx context.Context, ch channel.Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, found := s.mu.entries[ch.Key()]
	if !found {
		return
	}
	e.calculation.Close()
	if err := e.shutdown.Close(); err != nil {
		s.cfg.L.Error("failed to close calculated channel", zap.Error(err), zap.Stringer("key", ch))
	}
	delete(s.mu.entries, ch.Key())
	if _, err := s.startCalculation(ctx, ch.Key(), e.count); err != nil {
		s.cfg.L.Error("failed to restart calculated channel", zap.Error(err), zap.Stringer("key", ch))
		// Even if the operation is not successful, we still want to store the
		// latest requirements and expression in the entry.
		e.ch.Operations = ch.Operations
		e.ch.Expression = ch.Expression
		s.mu.entries[ch.Key()] = e
	}
}

func (s *Service) releaseEntryCloser(key channel.Key) io.Closer {
	return xio.CloserFunc(func() (err error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		e, found := s.mu.entries[key]
		if !found {
			return
		}
		e.count--
		if e.count != 0 {
			return
		}
		s.cfg.L.Debug("closing calculated channel", zap.Stringer("key", key))
		e.calculation.Close()
		delete(s.mu.entries, key)
		return
	})
}

// Close stops all calculations and closes the service. No other methods should be
// called after Close.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disconnectFromChannelChanges()
	for _, e := range s.mu.entries {
		e.calculation.Close()
	}
	c := errors.NewCatcher(errors.WithAggregation())
	for _, e := range s.mu.entries {
		c.Exec(e.shutdown.Close)
	}
	c.Exec(s.writer.Close)
	return c.Error()
}

// Request requests that the Service starts calculation the channel with the provided
// key. The calculation will be started if the channel is calculated and not already
// being calculated. If the channel is already being calculated, the number of active
// requests will be increased. The caller must close the returned io.Closer when the
// calculation is no longer needed, which will decrement the number of active requests.
func (s *Service) Request(ctx context.Context, key channel.Key) (io.Closer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startCalculation(ctx, key, 1)
}

const (
	defaultPipelineBufferSize                 = 500
	streamerAddr              address.Address = "streamer"
	calculatorAddr            address.Address = "calculator"
	writerAddr                address.Address = "writer"
	writerObserverAddr        address.Address = "writer_observer"
)

func (s *Service) startCalculation(
	ctx context.Context,
	key channel.Key,
	initialCount int,
) (io.Closer, error) {
	var ch channel.Channel
	// Wrap everything in a closure so we can properly propagate status changes.
	closer, err := func() (io.Closer, error) {
		ch.LocalKey = key.LocalKey()
		ch.Leaseholder = key.Leaseholder()
		if err := s.cfg.Channel.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
			return nil, err
		}
		if !ch.IsCalculated() {
			return nil, errors.Wrapf(validate.Error, "channel %v is not calculated", ch)
		}
		if _, exists := s.mu.entries[key]; exists {
			s.mu.entries[key].count++
			return s.releaseEntryCloser(key), nil
		}

		c, err := OpenCalculator(ctx, CalculatorConfig{
			ChannelSvc: s.cfg.Channel,
			Channel:    ch,
			Resolver:   s.cfg.Arc.SymbolResolver(),
		})
		if err != nil {
			return nil, err
		}
		writer_, err := s.cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{ch.Key(), ch.Index()},
			Start: telem.Now(),
		})
		if err != nil {
			return nil, err
		}
		streamer_, err := s.cfg.Framer.NewStreamer(ctx, framer.StreamerConfig{Keys: c.ReadFrom()})
		if err != nil {
			return nil, err
		}
		p := plumber.New()
		plumber.SetSegment(p, streamerAddr, streamer_)
		plumber.SetSegment(p, writerAddr, writer_)

		sc := newCalculationTransform([]*Calculator{c}, s.setStatus)
		plumber.SetSegment(
			p,
			calculatorAddr,
			sc,
			confluence.DeferErr(sc.close),
		)

		o := confluence.NewObservableSubscriber[framer.WriterResponse]()
		o.OnChange(func(ctx context.Context, i framer.WriterResponse) {
			s.cfg.L.DPanic(
				"write of calculated channel value failed",
				zap.Stringer("channel", ch),
			)
		})
		plumber.SetSink(p, writerObserverAddr, o)
		plumber.MustConnect[framer.StreamerResponse](p, streamerAddr, calculatorAddr, defaultPipelineBufferSize)
		plumber.MustConnect[framer.WriterRequest](p, calculatorAddr, writerAddr, defaultPipelineBufferSize)
		plumber.MustConnect[framer.WriterResponse](p, writerAddr, writerObserverAddr, defaultPipelineBufferSize)
		streamerRequests := confluence.NewStream[framer.StreamerRequest](1)
		streamer_.InFrom(streamerRequests)
		sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.cfg.Instrumentation))
		s.mu.entries[ch.Key()] = &entry{
			ch:          ch,
			count:       initialCount,
			calculation: streamerRequests,
			shutdown:    signal.NewHardShutdown(sCtx, cancel),
		}
		p.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.WithRetryOnPanic())
		s.cfg.L.Debug("started calculated channel", zap.Stringer("key", key))
		return s.releaseEntryCloser(key), nil
	}()
	if err != nil {
		s.setStatus(ctx, status.Status[types.Nil]{
			Key:         ch.Key().String(),
			Variant:     status.ErrorVariant,
			Message:     fmt.Sprintf("Failed to start calculation for %s", ch),
			Description: err.Error(),
		})
	}
	return closer, err
}

type onStatusChange func(ctx context.Context, status Status)

type streamCalculationTransform struct {
	confluence.LinearTransform[framer.StreamerResponse, framer.WriterRequest]
	calculators   []*Calculator
	onStateChange onStatusChange
}

func newCalculationTransform(
	calculators []*Calculator,
	onChange onStatusChange,
) *streamCalculationTransform {
	t := &streamCalculationTransform{calculators: calculators}
	t.Transform = t.transform
	t.onStateChange = onChange
	return t
}

func (t *streamCalculationTransform) transform(
	ctx context.Context,
	req framer.StreamerResponse,
) (res framer.WriterRequest, send bool, err error) {
	res.Command = writer.Write
	var changed bool
	for _, c := range t.calculators {
		res.Frame, changed, err = c.Next(ctx, req.Frame, res.Frame)
		if err != nil {
			t.onStateChange(ctx, Status{
				Key:         c.ch.Key().String(),
				Variant:     status.ErrorVariant,
				Message:     fmt.Sprintf("Failed to start calculation for %s", c.ch),
				Description: err.Error(),
			})
		} else if changed {
			send = true
		}
	}
	return res, send, nil
}

func (t *streamCalculationTransform) close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, calc := range t.calculators {
		c.Exec(calc.Close)
	}
	return c.Error()
}
