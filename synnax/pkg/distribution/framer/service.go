package framer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type Service struct{ Config }

type Config struct {
	ChannelService *channel.Service
	TS             storage.TS
	Transport      Transport
	HostResolver   core.HostResolver
	Logger         *zap.Logger
}

func (c Config) Validate() error {
	v := validate.New("distribution.framer")
	validate.NotNil(v, "channelService", c.ChannelService)
	validate.NotNil(v, "ts", c.TS)
	validate.NotNil(v, "transport", c.Transport)
	validate.NotNil(v, "hostResolver", c.HostResolver)
	validate.NotNil(v, "logger", c.Logger)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.ChannelService = override.Nil(c.ChannelService, other.ChannelService)
	c.TS = override.Nil(c.TS, other.TS)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.Logger = override.Nil(c.Logger, other.Logger)
	return c
}

var _ config.Config[Config] = Config{}

var DefaultConfig = Config{Logger: zap.NewNop()}

func Open(cfg ...Config) (*Service, error) {
	_cfg, err := config.OverrideAndValidate(DefaultConfig, cfg...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: _cfg}
	iterator.StartServer(iterator.Config{
		TS:              s.TS,
		HostResolver:    s.HostResolver,
		TransportServer: s.Transport.IteratorServer(),
	})
	writer.NewServer(writer.Config{
		TS:              s.TS,
		HostResolver:    s.HostResolver,
		TransportServer: s.Transport.WriterServer(),
	})
	return s, nil
}

func (s *Service) NewIterator(ctx context.Context, tr telem.TimeRange, keys ...channel.Key) (Iterator, error) {
	return iterator.New(ctx, s.newIteratorConfig(tr, keys))
}

func (s *Service) NewStreamIterator(ctx context.Context, tr telem.TimeRange, keys ...channel.Key) (StreamIterator, error) {
	return iterator.NewStream(ctx, s.newIteratorConfig(tr, keys))
}

func (s *Service) NewWriter(ctx context.Context, start telem.TimeStamp, keys ...channel.Key) (Writer, error) {
	return writer.New(ctx, s.newWriterConfig(start, keys))
}

func (s *Service) NewStreamWriter(ctx context.Context, start telem.TimeStamp, keys ...channel.Key) (StreamWriter, error) {
	return writer.NewStream(ctx, s.newWriterConfig(start, keys))
}

func (s *Service) newIteratorConfig(tr telem.TimeRange, keys []channel.Key) iterator.Config {
	return iterator.Config{
		TS:              s.TS,
		HostResolver:    s.HostResolver,
		TransportServer: s.Transport.IteratorServer(),
		TransportClient: s.Transport.IteratorClient(),
		ChannelKeys:     keys,
		TimeRange:       tr,
		Logger:          s.Logger,
		ChannelService:  s.ChannelService,
	}
}

func (s *Service) newWriterConfig(start telem.TimeStamp, keys []channel.Key) writer.Config {
	return writer.Config{
		Start:           start,
		Keys:            keys,
		TS:              s.TS,
		HostResolver:    s.HostResolver,
		TransportServer: s.Transport.WriterServer(),
		TransportClient: s.Transport.WriterClient(),
		Logger:          s.Logger,
		ChannelService:  s.ChannelService,
	}
}
