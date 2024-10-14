package calculated

import (
	"context"
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

type Config struct {
	alamos.Instrumentation
	Framer    *framer.Service
	Computron *computron.Interpreter
	Channel   channel.Readable
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
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
	count int
	inlet confluence.Inlet[framer.StreamerRequest]
}

type Service struct {
	cfg      Config
	sCtx     signal.Context
	shutdown io.Closer
	mu       sync.Mutex
	entries  map[channel.Key]*entry
}

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

func (s *Service) closerFunc(key channel.Key) io.Closer {
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
		e.inlet.Close()
		delete(s.entries, key)
		return
	})
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		e.inlet.Close()
	}
	return s.shutdown.Close()
}

func (s *Service) Request(ctx context.Context, key channel.Key) (io.Closer, error) {
	var ch channel.Channel
	if err := s.cfg.Channel.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
		return nil, err
	}
	if !ch.IsCalculated() {
		return nil, errors.Newf("Channel %v is not calculated", ch)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entries[key]; exists {
		s.entries[key].count++
		return s.closerFunc(key), nil
	}
	var requires []channel.Channel
	if err := s.cfg.Channel.NewRetrieve().WhereKeys(ch.Requires...).Entries(&requires).Exec(ctx, nil); err != nil {
		return nil, err
	}
	wrt, err := s.cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:  channel.Keys{ch.Key()},
		Start: telem.Now(),
	})
	if err != nil {
		return nil, err
	}
	str, err := s.cfg.Framer.NewStreamer(ctx, framer.StreamerConfig{Keys: ch.Requires})
	if err != nil {
		return nil, err
	}
	p := plumber.New()
	plumber.SetSegment(p, "streamer", str)
	plumber.SetSegment(p, "writer", wrt)
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
		globals: make(map[string]interface{}, len(requires)),
	}
	sc := &streamCalculator{internal: c}
	sc.Transform = sc.transform
	plumber.SetSegment[framer.StreamerResponse, framer.WriterRequest](p, "calculator", sc)
	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(ctx context.Context, i framer.WriterResponse) {
		s.cfg.L.Error("Calculated", zap.Error(i.Error))
	})
	plumber.SetSink[framer.WriterResponse](p, "obs", o)
	plumber.SetSegment[framer.StreamerResponse, framer.WriterRequest](p, "calculator", sc)
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "calculator", 10)
	plumber.MustConnect[framer.WriterRequest](p, "calculator", "writer", 10)
	plumber.MustConnect[framer.WriterResponse](p, "writer", "obs", 10)
	streamerRequests := confluence.NewStream[framer.StreamerRequest](1)
	str.InFrom(streamerRequests)
	s.entries[ch.Key()] = &entry{count: 1, inlet: streamerRequests}
	p.Flow(s.sCtx, confluence.CloseOutputInletsOnExit())
	return s.closerFunc(key), nil
}

type streamCalculator struct {
	internal *calculator
	confluence.LinearTransform[framer.StreamerResponse, framer.WriterRequest]
}

func (s *streamCalculator) transform(ctx context.Context, i framer.StreamerResponse) (framer.WriterRequest, bool, error) {
	i.Frame = s.internal.calculate(i.Frame)
	return framer.WriterRequest{Command: writer.Data, Frame: i.Frame}, true, nil
}

type calculator struct {
	ch          channel.Channel
	calculation *computron.Calculation
	requires    map[channel.Key]channel.Channel
	globals     map[string]interface{}
}

func (c calculator) calculate(fr framer.Frame) (of framer.Frame) {
	var alignment telem.AlignmentPair
	for _, k := range c.ch.Requires {
		s := fr.Get(k)
		if len(s) == 0 {
			continue
		}
		alignment = s[0].Alignment
		obj, err := computron.NewSeries(s[0])
		if err != nil {
			continue
		}
		ch, found := c.requires[k]
		if !found {
			continue
		}
		c.globals[ch.Name] = obj
	}
	os, err := c.calculation.Run(c.globals)
	if err != nil {
		return
	}
	os.Alignment = alignment
	of.Keys = []channel.Key{c.ch.Key()}
	of.Series = []telem.Series{os}
	return
}
