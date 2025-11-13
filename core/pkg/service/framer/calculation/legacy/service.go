// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy

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
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
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
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "state_codec", c.StateCodec)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
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

type Status = status.Status[types.Nil]

// Service creates and operates calculations on channels.
type Service struct {
	cfg ServiceConfig
	mu  struct {
		sync.Mutex
		entries map[channel.Key]*entry
	}
	stateKey channel.Key
	w        *framer.Writer
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

	s := &Service{cfg: cfg, w: w, stateKey: calculationStateCh.Key()}
	s.mu.entries = make(map[channel.Key]*entry)

	return s, nil
}

func (s *Service) setStatus(
	_ context.Context,
	status Status,
) {
	if _, err := s.w.Write(core.UnaryFrame(
		s.stateKey,
		telem.NewSeriesStaticJSONV(status),
	)); err != nil {
		s.cfg.L.Error("failed to encode state", zap.Error(err))
	}
}

func (s *Service) Update(ctx context.Context, ch channel.Channel) {
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

// Add requests that the Service starts calculating the channel with the provided
// key. The calculation will be started if the channel is calculated and not already
// being calculated. If the channel is already being calculated, the number of active
// requests will be increased. The caller must call Remove when the calculation is no
// longer needed, which will decrement the number of active requests.
func (s *Service) Add(ctx context.Context, key channel.Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.startCalculation(ctx, key, 1)
	return err
}

// Remove decrements the reference count for the calculation of the channel with the
// provided key. If the reference count reaches zero, the calculation will be stopped
// and the channel will be removed from the service.
func (s *Service) Remove(ctx context.Context, key channel.Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, found := s.mu.entries[key]
	if !found {
		return nil
	}
	e.count--
	if e.count > 0 {
		s.cfg.L.Debug("decremented calculation reference count",
			zap.Stringer("key", key),
			zap.Int("count", e.count),
		)
		return nil
	}
	s.cfg.L.Debug("closing calculated channel", zap.Stringer("key", key))
	e.calculation.Close()
	delete(s.mu.entries, key)
	return nil
}

const defaultPipelineBufferSize = 50

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
		if !ch.IsLegacyCalculated() {
			return nil, nil
		}
		if _, exists := s.mu.entries[key]; exists {
			s.mu.entries[key].count++
			return s.releaseEntryCloser(key), nil
		}

		var requires []channel.Channel
		if err := s.cfg.Channel.NewRetrieve().
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
		sc := newCalculationTransform([]*Calculator{c}, s.setStatus)
		plumber.SetSegment(
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
		plumber.SetSink(p, "obs", o)
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
	for _, c := range t.calculators {
		s, err := c.Next(req.Frame)
		if err != nil {
			t.onStateChange(ctx, Status{
				Key:         c.ch.Key().String(),
				Variant:     status.ErrorVariant,
				Message:     fmt.Sprintf("Failed to start calculation for %s", c.ch),
				Description: err.Error(),
			})
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
