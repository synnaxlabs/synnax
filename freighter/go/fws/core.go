package fws

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	ws "github.com/fasthttp/websocket"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/httputil"
	"go.uber.org/zap"
	"io"
	"time"
)

var reporter = freighter.Reporter{
	Protocol:  "websocket",
	Encodings: httputil.SupportedContentTypes(),
}

type messageType string

const (
	messageTypeData messageType = "data"
	closeMessage    messageType = "close"
)

type message[P freighter.Payload] struct {
	Type    messageType     `json:"type" msgpack:"type"`
	Err     ferrors.Payload `json:"error" msgpack:"error"`
	Payload P               `json:"payload" msgpack:"payload"`
}

func newCore[RQ, RS freighter.Payload](
	ctx context.Context,
	conn *ws.Conn,
	ecd binary.EncoderDecoder,
	logger *zap.SugaredLogger,
) core[RQ, RS] {
	ctx, cancel := context.WithCancel(ctx)
	b := core[RQ, RS]{
		ctx:      ctx,
		cancel:   cancel,
		conn:     conn,
		ecd:      ecd,
		contextC: make(chan struct{}),
		logger:   logger,
	}
	go b.listenForContextCancellation()
	return b
}

type core[I, O freighter.Payload] struct {
	ctx        context.Context
	cancel     context.CancelFunc
	contextC   chan struct{}
	conn       *ws.Conn
	ecd        binary.EncoderDecoder
	peerClosed error
	closed     bool
	logger     *zap.SugaredLogger
}

func (c *core[I, O]) send(msg message[O]) error {
	b, err := c.ecd.Encode(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(ws.BinaryMessage, b)
}

func (c *core[I, O]) receive() (msg message[I], err error) {
	var r io.Reader
	_, r, err = c.conn.NextReader()
	if err != nil {
		return msg, err
	}
	return msg, c.ecd.DecodeStream(r, &msg)
}

func (c *core[I, O]) cancelStream() error {
	close(c.contextC)
	c.peerClosed = context.Canceled
	c.cancel()
	return c.peerClosed
}

func (c *core[I, O]) listenForContextCancellation() {
	select {
	case <-c.contextC:
		return
	case <-c.ctx.Done():
		if err := c.conn.WriteControl(
			ws.CloseMessage,
			ws.FormatCloseMessage(ws.CloseGoingAway, ""),
			time.Now().Add(time.Second),
		); err != nil && !roacherrors.Is(err, ws.ErrCloseSent) {
			c.logger.Errorf("error sending close message: %v \n", err)
		}
	}
}

func isRemoteContextCancellation(err error) bool {
	return ws.IsCloseError(err, ws.CloseGoingAway)
}
