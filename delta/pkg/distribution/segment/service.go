package segment

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/x/telem"
	"go.uber.org/zap"
)

type Service struct {
	channel   *channel.Service
	db        cesium.DB
	transport Transport
	resolver  core.HostResolver
	logger    *zap.Logger
}

func New(
	channel *channel.Service,
	db cesium.DB,
	transport Transport,
	resolver core.HostResolver,
	logger *zap.Logger,
) *Service {
	s := &Service{
		channel:   channel,
		db:        db,
		transport: transport,
		resolver:  resolver,
		logger:    logger,
	}
	iterator.NewServer(iterator.Config{
		TS:        db,
		Resolver:  resolver,
		Transport: transport.Iterator(),
	})
	writer.NewServer(writer.Config{
		TS:        db,
		Resolver:  resolver,
		Transport: transport.Writer(),
	})
	return s
}

func (s *Service) NewIterator(ctx context.Context, tr telem.TimeRange, keys ...channel.Key) (Iterator, error) {
	return iterator.New(ctx, s.newIteratorConfig(tr, keys))
}

func (s *Service) NewStreamIterator(ctx context.Context, tr telem.TimeRange, keys ...channel.Key) (StreamIterator, error) {
	return iterator.NewStream(ctx, s.newIteratorConfig(tr, keys))
}

func (s *Service) NewWriter(ctx context.Context, keys ...channel.Key) (Writer, error) {
	return writer.New(ctx, s.newWriterConfig(keys))
}

func (s *Service) NewStreamWriter(ctx context.Context, keys ...channel.Key) (StreamWriter, error) {
	return writer.NewStream(ctx, s.newWriterConfig(keys))
}

func (s *Service) newIteratorConfig(tr telem.TimeRange, keys []channel.Key) iterator.Config {
	return iterator.Config{
		TS:             s.db,
		Resolver:       s.resolver,
		Transport:      s.transport.Iterator(),
		ChannelKeys:    keys,
		TimeRange:      tr,
		Logger:         s.logger,
		ChannelService: s.channel,
	}
}

func (s *Service) newWriterConfig(keys []channel.Key) writer.Config {
	return writer.Config{
		TS:             s.db,
		Resolver:       s.resolver,
		Transport:      s.transport.Writer(),
		ChannelKeys:    keys,
		Logger:         s.logger,
		ChannelService: s.channel,
	}
}
