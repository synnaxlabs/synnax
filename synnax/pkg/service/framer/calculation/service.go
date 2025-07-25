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
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
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
	// Framer is the underlying frame service to stream required channel values and write
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
	// StateCodec is the encoder/decoder used to communicate calculation state
	// changes.
	// [OPTIONAL]
	StateCodec binary.Codec
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening the calculation service.
	DefaultConfig = ServiceConfig{StateCodec: &binary.JSONCodec{}}
)

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("calculate")
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Channel", c.Channel)
	validate.NotNil(v, "ChannelObservable", c.ChannelObservable)
	validate.NotNil(v, "StateCodec", c.StateCodec)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.ChannelObservable = override.Nil(c.ChannelObservable, other.ChannelObservable)
	c.StateCodec = override.Nil(c.StateCodec, other.StateCodec)
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

type State struct {
	Key     channel.Key    `json:"key"`
	Variant status.Variant `json:"variant"`
	Message string         `json:"message"`
}

// Service creates and operates calculations on channels.
type Service struct {
	cfg ServiceConfig
	mu  struct {
		sync.Mutex
		entries map[channel.Key]*entry
	}
	disconnectFromChannelChanges observe.Disconnect
	stateKey                     channel.Key
	w                            *framer.Writer
}

// OpenService opens the service with the provided configuration. The service must be closed
// when it is no longer needed.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	calculationStateCh := channel.Channel{
		Name:        "sy_calculation_state",
		DataType:    telem.JSONT,
		Virtual:     true,
		Leaseholder: dcore.Free,
		Internal:    true,
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

	s := &Service{cfg: cfg, w: w, stateKey: calculationStateCh.Key()}
	s.disconnectFromChannelChanges = cfg.ChannelObservable.OnChange(s.handleChange)
	s.mu.entries = make(map[channel.Key]*entry)

	return s, nil
}

func (s *Service) setState(
	_ context.Context,
	ch channel.Channel,
	variant status.Variant,
	message string,
) {
	if _, err := s.w.Write(core.UnaryFrame(
		s.stateKey,
		telem.NewSeriesStaticJSONV(State{
			Key:     ch.Key(),
			Variant: variant,
			Message: message,
		}),
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
	if c.Variant != change.Set || !c.Value.IsCalculated() {
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
		e.ch.Requires = ch.Requires
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
	c.Exec(s.w.Close)
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

const defaultPipelineBufferSize = 50

func (s *Service) startCalculation(
	ctx context.Context,
	key channel.Key,
	initialCount int,
) (closer io.Closer, err error) {
	var ch channel.Channel
	ch.LocalKey = key.LocalKey()
	ch.Leaseholder = key.Leaseholder()
	defer func() {
		if err != nil {
			s.setState(ctx, ch, status.ErrorVariant, err.Error())
		}
	}()
	if err = s.cfg.Channel.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
		return nil, err
	}
	if !ch.IsCalculated() {
		return nil, errors.Newf("channel %v is not calculated", ch)
	}

	if _, exists := s.mu.entries[key]; exists {
		s.mu.entries[key].count++
		return s.releaseEntryCloser(key), nil
	}

	var requires []channel.Channel
	if err = s.cfg.Channel.NewRetrieve().
		WhereKeys(ch.Requires...).
		Entries(&requires).
		Exec(ctx, nil); err != nil {
		return nil, err
	}

	writer_, err := s.cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:  channel.Keys{ch.Key()},
		Start: telem.Now(),
	})
	if err != nil {
		return nil, err
	}
	streamer_, err := s.cfg.Framer.NewStreamer(ctx, framer.StreamerConfig{Keys: ch.Requires})
	if err != nil {
		return nil, err
	}
	p := plumber.New()
	plumber.SetSegment(p, "streamer", streamer_)
	plumber.SetSegment(p, "writer", writer_)

	c, err := OpenCalculator(ch, requires)
	if err != nil {
		return nil, err
	}
	sc := newCalculationTransform([]*Calculator{c}, s.setState)
	plumber.SetSegment[framer.StreamerResponse, framer.WriterRequest](
		p,
		"Calculator",
		sc,
		confluence.Defer(sc.close),
	)

	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(ctx context.Context, i framer.WriterResponse) {
		s.cfg.L.DPanic(
			"write of calculated channel value failed",
			zap.Stringer("channel", ch),
		)
	})
	plumber.SetSink[framer.WriterResponse](p, "obs", o)
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "Calculator", defaultPipelineBufferSize)
	plumber.MustConnect[framer.WriterRequest](p, "Calculator", "writer", defaultPipelineBufferSize)
	plumber.MustConnect[framer.WriterResponse](p, "writer", "obs", defaultPipelineBufferSize)
	streamerRequests := confluence.NewStream[framer.StreamerRequest](1)
	streamer_.InFrom(streamerRequests)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.cfg.Instrumentation))
	s.mu.entries[ch.Key()] = &entry{
		ch:          ch,
		count:       initialCount,
		calculation: streamerRequests,
		shutdown:    signal.NewHardShutdown(sCtx, cancel),
	}
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	s.cfg.L.Debug("started calculated channel", zap.Stringer("key", key))
	return s.releaseEntryCloser(key), nil
}

type onStateChange func(
	ctx context.Context,
	channel channel.Channel,
	variant status.Variant,
	message string,
)

type streamCalculationTransform struct {
	confluence.LinearTransform[framer.StreamerResponse, framer.WriterRequest]
	calculators   []*Calculator
	onStateChange onStateChange
}

func newCalculationTransform(
	calculators []*Calculator,
	onChange onStateChange,
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
	for _, c := range t.calculators {
		s, err := c.Next(req.Frame)
		if err != nil {
			t.onStateChange(ctx, c.ch, status.ErrorVariant, err.Error())
		} else if s.Len() > 0 {
			res.Frame = res.Frame.Append(c.ch.Key(), s)
			send = true
		}
	}
	return res, send, nil
}

func (t *streamCalculationTransform) close() {
	for _, c := range t.calculators {
		c.Close()
	}
}
