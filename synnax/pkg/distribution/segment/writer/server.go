package writer

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
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
	cfg.Transport.BindHandler(sf.Handle)
	return sf
}

func (sf *server) Handle(_ctx context.Context, server Server) error {
	ctx, cancel := signal.WithCancel(_ctx)
	defer cancel()

	// Block until we receive the first request from the remote w. This message
	// should have an OpenKeys command that provides context for opening the cesium
	// w.
	req, err := server.Receive()
	if err != nil {
		return err
	}
	if len(req.OpenKeys) == 0 {
		return errors.AssertionFailedf("[segment.w] - server expected OpenKeys to be defined")
	}

	receiver := &freightfluence.Receiver[Request]{Receiver: server}
	sender := &freightfluence.Sender[Response]{Sender: freighter.SenderEmptyCloser[Response]{StreamSender: server}}

	transientErrors := confluence.NewStream[error](0)

	w, err := newLocalWriter(ctx, req.OpenKeys, transientErrors, sf.Config)
	if err != nil {
		return errors.Wrap(err, "[segment.w] - failed to open cesium w")
	}

	pipe := plumber.New()
	plumber.SetSegment[Request, Response](pipe, "writer", w)
	plumber.SetSource[Request](pipe, "receiver", receiver)
	plumber.SetSink[Response](pipe, "sender", sender)
	plumber.SetSource[Response](pipe, "transient", &TransientSource{transient: transientErrors})

	plumber.UnaryRouter[Request]{SourceTarget: "receiver", SinkTarget: "writer"}.MustRoute(pipe)

	plumber.MultiRouter[Response]{
		SourceTargets: []address.Address{"writer", "transient"},
		SinkTargets:   []address.Address{"sender"},
	}.MustRoute(pipe)

	pipe.Flow(ctx, confluence.CloseInletsOnExit())

	return ctx.Wait()
}
