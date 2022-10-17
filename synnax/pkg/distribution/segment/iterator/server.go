package iterator

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type server struct {
	resolver aspen.HostResolver
	ts       cesium.DB
	logger   *zap.Logger
}

func NewServer(cfg Config) *server {
	sf := &server{ts: cfg.TS, resolver: cfg.Resolver, logger: cfg.Logger}
	cfg.TransportServer.BindHandler(sf.Handle)
	return sf
}

// Handle handles incoming req from the freighter.
func (sf *server) Handle(_ctx context.Context, server ServerStream) error {
	ctx, cancel := signal.WithCancel(_ctx)
	defer cancel()

	// Block until we receive the first request from the remoteIterator. This message should
	// have an AcquireSearcher command that provides context for opening the cesium iterator.
	req, err := server.Receive()
	if err != nil {
		return err
	}
	if req.Command != Open {
		return errors.New("[segment.iterator] - server expected AcquireSearcher command")
	}

	// receiver receives requests from the client and pipes them into the
	// requestPipeline.
	receiver := &freightfluence.Receiver[Request]{Receiver: server}

	// sender receives responses from the pipeline and sends them
	// over the network.
	sender := &freightfluence.Sender[Response]{
		Sender: freighter.SenderNopCloser[Response]{StreamSender: server},
	}

	iter, err := newLocalIterator(req.Keys, Config{
		TimeRange: req.Range,
		TS:        sf.ts,
		Resolver:  sf.resolver,
		Logger:    sf.logger,
	})
	if err != nil {
		return err
	}

	pipe := plumber.New()
	plumber.SetSegment[Request, Response](pipe, "iterator", iter)
	plumber.SetSource[Request](pipe, "receiver", receiver)
	plumber.SetSink[Response](pipe, "sender", sender)

	plumber.UnaryRouter[Request]{
		SourceTarget: "receiver",
		SinkTarget:   "iterator",
	}.MustRoute(pipe)

	plumber.UnaryRouter[Response]{
		SourceTarget: "iterator",
		SinkTarget:   "sender",
	}.MustRoute(pipe)

	pipe.Flow(ctx, confluence.CloseInletsOnExit())

	return ctx.Wait()
}
