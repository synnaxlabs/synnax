package calculated

import (
	"context"
	"fmt"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/computron"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"io"
	"sync"
)

// Config is the configuration for opening the calculated channel service.
type Config struct {
	alamos.Instrumentation
	// Framer is the underlying frame service to stream required channel values and write
	// calculated samples.
	// [REQUIRED]
	Framer *framer.Service
	// Computron is the computation service used to perform the Python based calculations.
	// [REQUIRED]
	Computron *computron.Interpreter
	// Channel is used to retrieve information about the channels being calculated.
	// [REQUIRED]
	Channel channel.Readable
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening calculated channel service.
	DefaultConfig = Config{}
)

// Validate implements config.Config
func (c Config) Validate() error {
	v := validate.New("calculate")
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Computron", c.Computron)
	validate.NotNil(v, "Channel", c.Channel)
	return v.Error()
}

// Override implements config.Config
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Computron = override.Nil(c.Computron, other.Computron)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

type entry struct {
	// count is the number of requests for the channel.
	count int
	// Input stream for calculation pipeline
	inlet confluence.Inlet[framer.StreamerRequest]
}

// Service manages collections of go-routines used to perform calculations on channels.
type Service struct {
	cfg      Config
	sCtx     signal.Context
	shutdown io.Closer
	mu       sync.Mutex
	entries  map[channel.Key]*entry
}

// Open opens a new calculated channel service using the provided configuration. See
// Config for more information on the configuration options. If the service is opened
// successfully, it must be closed using the Close method after it is no longer needed.
func Open(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	s.sCtx = sCtx
	s.shutdown = signal.NewShutdown(sCtx, cancel)
	s.entries = make(map[channel.Key]*entry)
	return s, nil
}

func (s *Service) releaseEntryCloser(key channel.Key) io.Closer {
	return xio.CloserFunc(func() (err error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		e, found := s.entries[key]
		if !found {
			return
		}
		e.count--
		if e.count != 0 {
			return
		}
		s.cfg.L.Debug("closing calculated channel", zap.Stringer("key", key))
		e.inlet.Close()
		delete(s.entries, key)
		return
	})
}

// Close stops the calculated channel service, halting all calculation go-routines and
// releasing any resources used by the service.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		e.inlet.Close()
	}
	return s.shutdown.Close()
}

const defaultPipelineBufferSize = 50

// Request starts calculating values for the channel identified by key and returns a closer
// that must be called when the calculation is no longer needed. If the same channel is
// requested multiple times, the calculations are shared between requests.
func (s *Service) Request(ctx context.Context, key channel.Key) (io.Closer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ch channel.Channel
	if err := s.cfg.Channel.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
		return nil, err
	}
	if !ch.IsCalculated() {
		return nil, errors.Newf("channel %v is not calculated", ch)
	}
	// Check if the channel is already being calculated.
	if _, exists := s.entries[key]; exists {
		s.entries[key].count++
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
	calculation, err := s.cfg.Computron.NewCalculation(ch.Expression)
	if err != nil {
		return nil, err
	}
	c := &calculator{
		ch:          ch,
		calculation: calculation,
		requires: lo.SliceToMap(requires, func(item channel.Channel) (channel.Key, channel.Channel) {
			return item.Key(), item
		}),
		locals: make(map[string]interface{}, len(requires)),
	}
	sc := &streamCalculator{
		internal: c,
		cfg:      s.cfg,
	}
	sc.Transform = sc.transform
	plumber.SetSegment[framer.StreamerResponse, framer.WriterRequest](p, "calculator", sc)

	// Wire up an observer that logs any errors that occurred with writing the calculated
	// samples. Since we're writing to a free virtual channel, these errors should never
	// occur.
	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(ctx context.Context, i framer.WriterResponse) {
		s.cfg.L.DPanic(
			"write of calculated channel value failed",
			zap.Error(i.Error),
			zap.Stringer("channel", ch),
		)
	})
	plumber.SetSink[framer.WriterResponse](p, "obs", o)
	plumber.SetSegment[framer.StreamerResponse, framer.WriterRequest](p, "calculator", sc)
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "calculator", defaultPipelineBufferSize)
	plumber.MustConnect[framer.WriterRequest](p, "calculator", "writer", defaultPipelineBufferSize)
	plumber.MustConnect[framer.WriterResponse](p, "writer", "obs", defaultPipelineBufferSize)
	streamerRequests := confluence.NewStream[framer.StreamerRequest](1)
	streamer_.InFrom(streamerRequests)
	s.entries[ch.Key()] = &entry{count: 1, inlet: streamerRequests}
	p.Flow(s.sCtx, confluence.CloseOutputInletsOnExit())
	s.cfg.L.Debug("started calculated channel", zap.Stringer("key", key))
	return s.releaseEntryCloser(key), nil
}

type streamCalculator struct {
	internal *calculator
	cfg      Config
	confluence.LinearTransform[framer.StreamerResponse, framer.WriterRequest]
}

func (s *streamCalculator) transform(_ context.Context, i framer.StreamerResponse) (framer.WriterRequest, bool, error) {
	frame, warning, err := s.internal.calculate(i.Frame)
	if err != nil {
		fmt.Printf("Calculation error occurred: %v\n", err)
		// Log the full error including Python traceback
		s.cfg.L.Error("Python calculation error",
			zap.Error(err),
			zap.String("channel_name", s.internal.ch.Name),
			zap.String("expression", s.internal.ch.Expression))
		return framer.WriterRequest{}, false, err
	}

	if warning != "" {
		s.cfg.L.Warn("Python runtime warning",
			zap.String("warning", warning),
			zap.String("channel_name", s.internal.ch.Name),
			zap.String("expression", s.internal.ch.Expression))
	}

	return framer.WriterRequest{Command: writer.Data, Frame: frame}, true, nil
}

type calculator struct {
	ch          channel.Channel
	calculation *computron.Calculation
	requires    map[channel.Key]channel.Channel
	locals      map[string]interface{}
}

func (c calculator) calculate(fr framer.Frame) (of framer.Frame, warning string, err error) {
	var alignment telem.AlignmentPair
	for _, k := range c.ch.Requires {
		s := fr.Get(k)
		if len(s) == 0 {
			continue
		}
		alignment = s[0].Alignment
		obj, err := computron.NewSeries(s[0])
		if err != nil {
			return of, "", err
		}
		ch, found := c.requires[k]
		if !found {
			continue
		}
		c.locals[ch.Name] = obj
	}
	os, warning, err := c.calculation.RunWarning(c.locals)
	if err != nil {
		return of, "", err
	}

	os.Alignment = alignment
	of.Keys = []channel.Key{c.ch.Key()}
	of.Series = []telem.Series{os}
	return of, warning, nil
}
