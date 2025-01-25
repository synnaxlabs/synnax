package calculated

import (
	"context"
	"io"
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

func (c Config) Validate() error {
	v := validate.New("calculate")
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Channel", c.Channel)
	validate.NotNil(v, "ChannelObservable", c.ChannelObservable)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.ChannelObservable = override.Nil(c.ChannelObservable, other.ChannelObservable)
	return c
}

type entry struct {
	count    int
	inlet    confluence.Inlet[framer.StreamerRequest]
	shutdown io.Closer
}

type Service struct {
	cfg Config
	mu  struct {
		sync.Mutex
		entries map[channel.Key]*entry
	}
}

func Open(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	cfg.ChannelObservable.OnChange(s.handleChange)
	s.mu.entries = make(map[channel.Key]*entry)
	return s, nil
}

func (s *Service) handleChange(ctx context.Context, reader gorp.TxReader[channel.Key, channel.Channel]) {
	c, ok := reader.Next(ctx)
	if !ok {
		return
	}
	if c.Variant == change.Set && c.Value.IsCalculated() {
		s.Update(ctx, c.Key)
	}
}

func (s *Service) Update(ctx context.Context, ch channel.Key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, found := s.mu.entries[ch]
	if !found {
		return
	}
	e.inlet.Close()
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
		e.inlet.Close()
		delete(s.mu.entries, key)
		return
	})
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.mu.entries {
		e.inlet.Close()
	}
	c := errors.NewCatcher(errors.WithAggregation())
	for _, e := range s.mu.entries {
		c.Exec(e.shutdown.Close)
	}
	return c.Error()
}

const defaultPipelineBufferSize = 50

func (s *Service) Request(ctx context.Context, key channel.Key) (io.Closer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startCalculation(ctx, key, 1)
}

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

	calculation, err := computron.OpenExpression(ch.Expression)
	if err != nil {
		return nil, err
	}
	c := &calculator{
		ch:          ch,
		calculation: calculation,
		requires: lo.SliceToMap(requires, func(item channel.Channel) (channel.Key, channel.Channel) {
			return item.Key(), item
		}),
	}
	sc := &streamCalculator{internal: c, cfg: s.cfg}
	sc.Transform = sc.transform
	plumber.SetSegment[framer.StreamerResponse, framer.WriterRequest](p, "calculator", sc)

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
		count:    initialCount,
		inlet:    streamerRequests,
		shutdown: signal.NewShutdown(sCtx, cancel),
	}
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	s.cfg.L.Debug("started calculated channel", zap.Stringer("key", key))
	return s.releaseEntryCloser(key), nil
}

type streamCalculator struct {
	internal *calculator
	cfg      Config
	lastErr  error
	confluence.LinearTransform[framer.StreamerResponse, framer.WriterRequest]
}

func (s *streamCalculator) transform(_ context.Context, i framer.StreamerResponse) (framer.WriterRequest, bool, error) {
	frame, err := s.internal.calculate(i.Frame)
	if err != nil {
		s.cfg.L.Error("calculation error",
			zap.Error(err),
			zap.String("channel_name", s.internal.ch.Name),
			zap.String("expression", s.internal.ch.Expression))
		return framer.WriterRequest{}, false, nil
	}

	return framer.WriterRequest{Command: writer.Data, Frame: frame}, true, nil
}

type calculator struct {
	ch          channel.Channel
	calculation *computron.Expression
	requires    map[channel.Key]channel.Channel
}

func (c calculator) Close() { c.calculation.Close() }

func (c calculator) calculate(fr framer.Frame) (of framer.Frame, err error) {
	var alignment telem.AlignmentPair
	for _, k := range c.ch.Requires {
		s := fr.Get(k)
		if len(s) == 0 {
			continue
		}
		alignment = s[0].Alignment
		if ch, found := c.requires[k]; found {
			c.calculation.Set(ch.Name, computron.LValueFromSeries(s[0]))
		}
	}
	res, err := c.calculation.Run()
	if err != nil {
		return of, err
	}
	os := computron.SeriesToLValue(res, c.ch.DataType)
	os.Alignment = alignment
	of.Keys = []channel.Key{c.ch.Key()}
	of.Series = []telem.Series{os}
	return of, nil
}
