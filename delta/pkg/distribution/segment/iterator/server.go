package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

type server struct {
	host   core.NodeID
	db     cesium.DB
	logger *zap.SugaredLogger
}

func NewServer(db cesium.DB, host core.NodeID, transport Transport) *server {
	sf := &server{db: db, host: host}
	transport.BindHandler(sf.Handle)
	return sf
}

// Handle handles incoming req from the freighter.
func (sf *server) Handle(_ctx context.Context, server Server) error {
	ctx, cancel := signal.WithCancel(_ctx)
	defer cancel()

	// Block until we receive the first request from the remoteIterator. This message should
	// have an Open command that provides context for opening the cesium iterator.
	req, err := server.Receive()
	if err != nil {
		return err
	}
	if req.Command != Open {
		return errors.New("[segment.iterator] - server expected Open command")
	}

	// receiver receives requests from the client and pipes them into the
	// requestPipeline.
	receiver := &freightfluence.Receiver[Request]{Receiver: server}

	// sender receives responses from the pipeline and sends them
	// over the network.
	sender := &freightfluence.Sender[Response]{
		Sender: freighter.SenderEmptyCloser[Response]{StreamSender: server},
	}

	iter, err := newLocalIterator(sf.db, sf.host, req.Range, req.Keys)
	if err != nil {
		return errors.Wrap(err, "[segment.iterator] - cesium iterator failed to open")
	}

	pipe := plumber.New()
	plumber.SetSegment[Request, Response](pipe, "iterator", iter)
	plumber.SetSource[Request](pipe, "receiver", receiver)
	plumber.SetSink[Response](pipe, "sender", sender)

	c := errutil.NewCatch()

	c.Exec(plumber.UnaryRouter[Request]{
		SourceTarget: "receiver",
		SinkTarget:   "iterator",
	}.PreRoute(pipe))

	c.Exec(plumber.UnaryRouter[Response]{
		SourceTarget: "iterator",
		SinkTarget:   "sender",
	}.PreRoute(pipe))

	if c.Error() != nil {
		panic(c.Error())
	}

	pipe.Flow(ctx, confluence.CloseInletsOnExit())

	err = ctx.Wait()
	return err
}
