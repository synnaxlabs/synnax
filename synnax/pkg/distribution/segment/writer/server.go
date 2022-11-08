package writer

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type server struct {
	Config
	ts storage.TS
}

func NewServer(cfg Config) *server {
	sf := &server{ts: cfg.TS, Config: cfg}
	cfg.TransportServer.BindHandler(sf.Handle)
	return sf
}

func (sf *server) Handle(_ctx context.Context, server ServerStream) error {
	ctx, cancel := signal.WithCancel(_ctx)
	defer cancel()

	// Block until we receive the first request from the remote w. This message
	// should have an Keys command that provides context for opening the cesium
	// w.
	req, err := server.Receive()
	if err != nil {
		return err
	}
	if len(req.Keys) == 0 {
		return errors.AssertionFailedf("[segment.w] - server expected Keys to be defined")
	}

	receiver := &freightfluence.Receiver[Request]{Receiver: server}
	sender := &freightfluence.Sender[Response]{Sender: freighter.SenderNopCloser[Response]{StreamSender: server}}

	w, err := newLocalWriter(ctx, req.Keys, sf.Config)
	if err != nil {
		return errors.Wrap(err, "[segment.w] - failed to open cesium w")
	}
	pipe := plumber.New()
	plumber.SetSegment[Request, Response](pipe, "writerClient", w)
	plumber.SetSource[Request](pipe, "receiver", receiver)
	plumber.SetSink[Response](pipe, "sender", sender)
	plumber.MustConnect[Request](pipe, "receiver", "writerClient", 1)
	plumber.MustConnect[Response](pipe, "writerClient", "sender", 1)
	pipe.Flow(ctx, confluence.CloseInletsOnExit())
	return ctx.Wait()
}
