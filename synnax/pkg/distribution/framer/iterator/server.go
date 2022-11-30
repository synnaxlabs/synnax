package iterator

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type server struct{ ServiceConfig }

func startServer(cfg ServiceConfig) *server {
	s := &server{ServiceConfig: cfg}
	cfg.Transport.Server().BindHandler(s.handle)
	return s
}

// Handle implements freighter.StreamServer.
func (sf *server) handle(ctx context.Context, server ServerStream) error {
	sCtx, cancel := signal.WithCancel(ctx)
	defer cancel()

	// Block until we receive the first request from the remoteIterator. This message should
	// have an open command that provides context for opening the cesium iterator.
	req, err := server.Receive()
	if err != nil {
		return err
	}

	// receiver receives requests from the client and pipes them into the
	// requestPipeline.
	receiver := &freightfluence.Receiver[Request]{Receiver: server}

	// sender receives responses from the pipeline and sends them
	// over the network.
	sender := &freightfluence.Sender[Response]{
		Sender: freighter.SenderNopCloser[Response]{StreamSender: server},
	}

	iter, err := newStorageIterator(
		sf.ServiceConfig,
		Config{
			Keys:   req.Keys,
			Bounds: req.Bounds,
		},
	)
	if err != nil {
		return err
	}

	pipe := plumber.New()
	plumber.SetSegment[Request, Response](pipe, "iterator", iter)
	plumber.SetSource[Request](pipe, "receiver", receiver)
	plumber.SetSink[Response](pipe, "sender", sender)
	plumber.MustConnect[Request](pipe, "receiver", "iterator", 1)
	plumber.MustConnect[Response](pipe, "iterator", "sender", 1)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())
	return sCtx.Wait()
}
