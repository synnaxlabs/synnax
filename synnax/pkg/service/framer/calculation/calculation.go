package calculation

import (
	"context"
	"io"
	"slices"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/computron"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for opening the calculation service.
type Config struct {
	alamos.Instrumentation
	// Framer is the underlying frame service to stream required channel values and write
	// calculated samples.
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used to retrieve information about the channels being calculated.
	// [REQUIRED]
	Channel channel.Readable
	// ChannelObservable is used to listen to real-time changes in calculated channels
	// so the calculation routines can be updated accordingly.
	ChannelObservable observe.Observable[gorp.TxReader[channel.Key, channel.Channel]]
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening the calculation service.
	DefaultConfig = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("calculate")
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Channel", c.Channel)
	validate.NotNil(v, "ChannelObservable", c.ChannelObservable)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.ChannelObservable = override.Nil(c.ChannelObservable, other.ChannelObservable)
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

// Service creates and operates calculations on channels.
type Service struct {
	cfg Config
	mu  struct {
		sync.Mutex
		entries map[channel.Key]*entry
	}
	disconnectFromChannelChanges observe.Disconnect
}

// Open opens the service with the provided configuration. The service must be closed
// when it is no longer needed.
func Open(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	s.disconnectFromChannelChanges = cfg.ChannelObservable.OnChange(s.handleChange)
	s.mu.entries = make(map[channel.Key]*entry)
	return s, nil
}

func (s *Service) handleChange(
	ctx context.Context,
	reader gorp.TxReader[channel.Key, channel.Channel],
) {
	c, ok := reader.Next(ctx)
	if !ok {
		return
	}
	// Don't stop calculating if the channel is deleted.
	if c.Variant != change.Set || !c.Value.IsCalculated() {
		return
	}
	existing, found := s.mu.entries[c.Key]
	if !found {
		s.update(ctx, c.Key)
		return

	}
	expressionsEqual := existing.ch.Expression == c.Value.Expression
	if !expressionsEqual {
		s.update(ctx, c.Key)
		return
	}
	slices.Sort(existing.ch.Requires)
	slices.Sort(c.Value.Requires)
	requiresEqual := slices.Equal(existing.ch.Requires, c.Value.Requires)
	if requiresEqual {
		return
	}
	s.update(ctx, c.Key)
}

func (s *Service) update(ctx context.Context, ch channel.Key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, found := s.mu.entries[ch]
	if !found {
		return
	}
	e.calculation.Close()
	if err := e.shutdown.Close(); err != nil {
		s.cfg.L.Error("failed to close calculated channel", zap.Error(err), zap.Stringer("key", ch))
	}
	delete(s.mu.entries, ch)
	if _, err := s.startCalculation(ctx, ch, e.count); err != nil {
		s.cfg.L.Error("failed to restart calculated channel", zap.Error(err), zap.Stringer("key", ch))
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
	return c.Error()
}

// Request requests that the Service starts calculation the channel with the provided
// key. The calculation will be started if the channel is calculated and not already
// being calculated. If the channel is already being calculated, the number of active
// requests will be incremented. The caller must close the returned io.Closer when the
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
) (io.Closer, error) {
	var ch channel.Channel
	if err := s.cfg.Channel.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
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

	calculation, err := computron.Open(ch.Expression)
	if err != nil {
		return nil, err
	}
	c := &Calculator{
		ch:          ch,
		calculation: calculation,
		requires: lo.SliceToMap(requires, func(item channel.Channel) (channel.Key, channel.Channel) {
			return item.Key(), item
		}),
	}
	sc := &streamCalculator{internal: c, cfg: s.cfg}
	sc.Transform = sc.transform
	plumber.SetSegment[framer.StreamerResponse, framer.WriterRequest](
		p,
		"calculator",
		sc,
		confluence.Defer(sc.internal.close),
	)

	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(ctx context.Context, i framer.WriterResponse) {
		s.cfg.L.DPanic(
			"write of calculated channel value failed",
			zap.Error(i.Error),
			zap.Stringer("channel", ch),
		)
	})
	plumber.SetSink[framer.WriterResponse](p, "obs", o)
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "calculator", defaultPipelineBufferSize)
	plumber.MustConnect[framer.WriterRequest](p, "calculator", "writer", defaultPipelineBufferSize)
	plumber.MustConnect[framer.WriterResponse](p, "writer", "obs", defaultPipelineBufferSize)
	streamerRequests := confluence.NewStream[framer.StreamerRequest](1)
	streamer_.InFrom(streamerRequests)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.cfg.Instrumentation))
	s.mu.entries[ch.Key()] = &entry{
		ch:          ch,
		count:       initialCount,
		calculation: streamerRequests,
		shutdown:    signal.NewShutdown(sCtx, cancel),
	}
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	s.cfg.L.Debug("started calculated channel", zap.Stringer("key", key))
	return s.releaseEntryCloser(key), nil
}

type streamCalculator struct {
	internal *Calculator
	cfg      Config
	lastErr  error
	confluence.LinearTransform[framer.StreamerResponse, framer.WriterRequest]
}

func (s *streamCalculator) transform(_ context.Context, i framer.StreamerResponse) (framer.WriterRequest, bool, error) {
	frame, err := s.internal.Calculate(i.Frame)
	if err != nil {
		s.cfg.L.Error("calculation error",
			zap.Error(err),
			zap.String("channel_name", s.internal.ch.Name),
			zap.String("expression", s.internal.ch.Expression))
		return framer.WriterRequest{}, false, nil
	}
	return framer.WriterRequest{Command: writer.Data, Frame: frame}, true, nil
}

type Calculator struct {
	ch          channel.Channel
	calculation *computron.Calculator
	requires    map[channel.Key]channel.Channel
}

func (c Calculator) close() { c.calculation.Close() }

func (c Calculator) Calculate(fr framer.Frame) (of framer.Frame, err error) {
	if len(fr.Series) == 0 {
		return
	}
	os := telem.AllocSeries(c.ch.DataType, fr.Series[0].Len())
	os.Alignment = fr.Series[0].Alignment
	of.Keys = []channel.Key{c.ch.Key()}
	of.Series = []telem.Series{os}
	for i := range os.Len() {
		for _, k := range c.ch.Requires {
			sArray := fr.Get(k)
			if len(sArray) == 0 {
				continue
			}
			s := sArray[0]
			idx := i
			if idx >= s.Len() {
				idx = s.Len() - 1
			}
			if ch, found := c.requires[k]; found {
				c.calculation.Set(ch.Name, computron.LValueFromSeries(sArray[0], idx))
			}
		}
		res, err := c.calculation.Run()
		if err != nil {
			return of, err
		}
		computron.SetLValueOnSeries(res, c.ch.DataType, os, i)
	}
	return of, nil
}