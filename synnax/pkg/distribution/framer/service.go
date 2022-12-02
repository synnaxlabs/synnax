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
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type Service struct {
	writer   *writer.Service
	iterator *iterator.Service
}

type ServiceConfig struct {
	ChannelReader channel.Reader
	TS            storage.TS
	Transport     Transport
	HostResolver  core.HostResolver
	Logger        *zap.Logger
}

var (
	_             config.Config[ServiceConfig] = ServiceConfig{}
	DefaultConfig                              = ServiceConfig{Logger: zap.NewNop()}
)

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.framer")
	validate.NotNil(v, "ChannelReader", c.ChannelReader)
	validate.NotNil(v, "TS", c.TS)
	validate.NotNil(v, "Transport", c.Transport)
	validate.NotNil(v, "HostResolver", c.HostResolver)
	validate.NotNil(v, "Logger", c.Logger)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.ChannelReader = override.Nil(c.ChannelReader, other.ChannelReader)
	c.TS = override.Nil(c.TS, other.TS)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.Logger = override.Nil(c.Logger, other.Logger)
	return c
}

func Open(configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{}
	s.iterator, err = iterator.OpenService(iterator.ServiceConfig{
		TS:           cfg.TS,
		HostResolver: cfg.HostResolver,
		Transport:    cfg.Transport.Iterator(),
	})
	if err != nil {
		return nil, err
	}
	s.writer, err = writer.OpenService(writer.ServiceConfig{
		TS:           cfg.TS,
		HostResolver: cfg.HostResolver,
		Transport:    cfg.Transport.Writer(),
	})
	return s, err
}

func (s *Service) NewIterator(ctx context.Context, cfg IteratorConfig) (Iterator, error) {
	return s.iterator.New(ctx, cfg)
}

func (s *Service) NewStreamIterator(ctx context.Context, cfg IteratorConfig) (StreamIterator, error) {
	return s.iterator.NewStream(ctx, cfg)
}

func (s *Service) NewWriter(ctx context.Context, cfg WriterConfig) (Writer, error) {
	return s.writer.New(ctx, cfg)
}

func (s *Service) NewStreamWriter(ctx context.Context, cfg WriterConfig) (StreamWriter, error) {
	return s.writer.NewStream(ctx, cfg)
}
