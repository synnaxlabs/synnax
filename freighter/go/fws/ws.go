package fws

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/ferrors"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/httputil"
	roacherrors "github.com/cockroachdb/errors"
	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
	"go/types"
	"io"
	"log"
	"net/http"
	"time"
)

func New[RQ, RS freighter.Payload](
	ecd httputil.EncoderDecoder,
	logger *zap.SugaredLogger,
) *Stream[RQ, RS] {
	return &Stream[RQ, RS]{
		Server: *NewServer[RQ, RS](ecd, logger),
		Client: *NewClient[RQ, RS](ecd),
	}
}

type Stream[RQ, RS freighter.Payload] struct {
	Server[RQ, RS]
	Client[RQ, RS]
	path string
}

var reporter = freighter.Reporter{
	Protocol:  "websocket",
	Encodings: httputil.SupportedContentTypes(),
}

// Report implements the freighter.Transport interface.
func (s *Stream[RQ, RS]) Report() alamos.Report { return s.Client.Report() }

func (s *Stream[RQ, RS]) BindTo(r fiber.Router, path string) {
	s.path = path
	s.Server.BindTo(r, s.path)
}

func (s *Stream[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	return s.Client.Stream(ctx, address.Address("ws://"+target.String()+s.path))
}

var (
	_ freighter.StreamTransport[any, types.Nil]       = (*Stream[any, types.Nil])(nil)
	_ freighter.StreamTransportServer[any, types.Nil] = (*Server[any, types.Nil])(nil)
	_ freighter.StreamTransportClient[any, types.Nil] = (*Client[any, types.Nil])(nil)
	_ freighter.ClientStream[any, types.Nil]          = (*clientStream[any, types.Nil])(nil)
	_ freighter.ServerStream[any, types.Nil]          = (*serverStream[any, types.Nil])(nil)
)

func NewClient[RQ, RS freighter.Payload](ecd httputil.EncoderDecoder) *Client[RQ, RS] {
	return &Client[RQ, RS]{ecd: ecd, Reporter: reporter}
}

func NewServer[RQ, RS freighter.Payload](ecd httputil.EncoderDecoder, logger *zap.SugaredLogger) *Server[RQ, RS] {
	return &Server[RQ, RS]{ecd: ecd, logger: logger, Reporter: reporter}
}

type Server[RQ, RS freighter.Payload] struct {
	ecd     httputil.EncoderDecoder
	logger  *zap.SugaredLogger
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error
	freighter.Reporter
}

func (s *Server[RQ, RS]) BindHandler(
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

func (s *Server[RQ, RS]) BindTo(router fiber.Router, path string) {
	router.Get(path, func(c *fiber.Ctx) error {
		c.Accepts(httputil.SupportedContentTypes()...)
		if !fiberws.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		ecd, err := s.determineEncoderDecoder(c)
		if err != nil {
			return err
		}
		return fiberws.New(func(conn *fiberws.Conn) {
			err := s.exec(newServerStream[RQ, RS](ecd, conn.Conn))
			if err != nil && !roacherrors.Is(err, context.Canceled) {
				s.logger.Errorw("error executing server stream", "error", err)
			}
		})(c)
	})
}

func (s *Server[RQ, RS]) exec(stream *serverStream[RQ, RS]) error {
	err := s.handler(stream.ctx, stream)

	errPayload := ferrors.Encode(err)

	if errPayload.Type == ferrors.Nil {
		errPayload = ferrors.Encode(freighter.EOF)
	}

	if stream.ctx.Err() != nil {
		return stream.ctx.Err()
	}

	if err := stream.send(message[RS]{Type: closeMessage, Err: errPayload}); err != nil {
		return err
	}

	stream.clientClosed = freighter.StreamClosed

	closed := make(chan struct{}, 1)
	go func() {
		defer close(closed)
		for {
			_, err := stream.receive()
			if ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
				return
			}
			if err != nil {
				s.logger.Debugw("expected normal closure, received error instead", "error", err)
				return
			}
		}
	}()

	if err := stream.conn.WriteMessage(
		ws.CloseMessage,
		ws.FormatCloseMessage(ws.CloseNormalClosure, ""),
	); err != nil {
		return err
	}

	select {
	case <-stream.ctx.Done():
		break
	case <-time.After(500 * time.Millisecond):
		break
	case <-closed:
		break
	}

	close(stream.contextC)
	stream.cancel()

	return stream.conn.Close()
}

func (s *Server[RQ, RS]) determineEncoderDecoder(c *fiber.Ctx) (httputil.EncoderDecoder, error) {
	ct := c.Get("Content-Type")
	if s.ecd != nil {
		if s.ecd.ContentType() == ct {
			return s.ecd, nil
		}
		return nil, roacherrors.Newf("[freighter] - unsupported content type")
	}
	return httputil.DetermineEncoderDecoder(ct)
}

type Client[RQ, RS freighter.Payload] struct {
	dialer ws.Dialer
	ecd    httputil.EncoderDecoder
	freighter.Reporter
}

func (s *Client[RQ, RS]) Report() alamos.Report {
	r := reporter
	r.Encodings = []string{s.ecd.ContentType()}
	return r.Report()
}

func (s *Client[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	header := http.Header{}
	header.Set("Content-Type", s.ecd.ContentType())
	conn, res, err := s.dialer.DialContext(ctx, target.String(), header)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 101 {
		return nil, roacherrors.New("[ws] - unable to upgrade connection")
	}
	return newClientStream[RQ, RS](ctx, s.ecd, conn), nil
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

func newClientStream[RQ, RS freighter.Payload](
	ctx context.Context,
	ecd binary.EncoderDecoder,
	conn *ws.Conn,
) *clientStream[RQ, RS] {
	return &clientStream[RQ, RS]{base: newBase[RS, RQ](ctx, ecd, conn)}
}

func newServerStream[RQ, RS freighter.Payload](
	ecd binary.EncoderDecoder,
	conn *ws.Conn,
) *serverStream[RQ, RS] {
	return &serverStream[RQ, RS]{base: newBase[RQ, RS](context.TODO(), ecd, conn)}
}

func newBase[RQ, RS freighter.Payload](
	ctx context.Context,
	ecd binary.EncoderDecoder,
	conn *ws.Conn,
) base[RQ, RS] {
	ctx, cancel := context.WithCancel(ctx)
	b := base[RQ, RS]{
		ctx:      ctx,
		cancel:   cancel,
		conn:     conn,
		ecd:      ecd,
		contextC: make(chan struct{}),
	}
	go b.listenForContextCancellation()
	return b
}

type clientStream[RQ, RS freighter.Payload] struct {
	base[RS, RQ]
	serverClosed error
	sendClosed   bool
}

type serverStream[RQ, RS freighter.Payload] struct {
	base[RQ, RS]
	clientClosed error
	closed       bool
}

type base[I, O freighter.Payload] struct {
	ctx      context.Context
	cancel   context.CancelFunc
	contextC chan struct{}
	conn     *ws.Conn
	ecd      binary.EncoderDecoder
}

// |||||| CLIENT SEND - SERVER RECEIVE ||||||

// Send implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) Send(req RQ) error {
	if s.serverClosed != nil {
		return freighter.EOF
	}

	if s.sendClosed {
		return freighter.StreamClosed
	}

	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}

	if err := s.base.send(message[RQ]{Type: messageTypeData, Payload: req}); err != nil {
		close(s.contextC)
		return freighter.EOF
	}

	return nil
}

// Receive implements the freighter.ServerStream interface.
func (s *serverStream[RQ, RS]) Receive() (req RQ, err error) {
	if s.clientClosed != nil {
		return req, s.clientClosed
	}

	if s.ctx.Err() != nil {
		return req, s.ctx.Err()
	}

	msg, err := s.base.receive()

	// A close message means the client called CloseSend.
	if msg.Type == closeMessage {
		s.clientClosed = freighter.EOF
		return req, s.clientClosed
	}

	// A close error with code close going away means the client context
	// was cancelled and we should cancel the server context.
	if ws.IsCloseError(err, ws.CloseGoingAway) {
		close(s.contextC)
		s.clientClosed = context.Canceled
		s.cancel()
		return req, s.clientClosed
	}

	return msg.Payload, err
}

// |||||| SERVER SEND - CLIENT RECEIVE ||||||

// Send implements the freighter.ServerStream interface.
func (s *serverStream[RQ, RS]) Send(res RS) error {
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	return s.base.send(message[RS]{Payload: res, Type: messageTypeData})
}

// Receive implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) Receive() (res RS, err error) {
	if s.serverClosed != nil {
		return res, s.serverClosed
	}

	if s.ctx.Err() != nil {
		return res, s.ctx.Err()
	}

	msg, err := s.base.receive()

	// A close message means the server handler exited.
	if msg.Type == closeMessage {
		close(s.contextC)
		s.serverClosed = ferrors.Decode(msg.Err)
		return res, s.serverClosed
	}

	if ws.IsCloseError(err, ws.CloseGoingAway) {
		close(s.contextC)
		s.serverClosed = context.Canceled
		s.cancel()
		return res, s.serverClosed
	}

	return msg.Payload, err
}

func (s *base[I, O]) send(msg message[O]) error {
	b, err := s.ecd.Encode(msg)
	if err != nil {
		return err
	}
	return s.conn.WriteMessage(ws.BinaryMessage, b)
}

func (s *base[I, O]) receive() (msg message[I], err error) {
	var r io.Reader
	_, r, err = s.conn.NextReader()
	if err != nil {
		return msg, err
	}
	return msg, s.ecd.DecodeStream(r, &msg)
}

func (s *base[I, O]) listenForContextCancellation() {
	select {
	case <-s.contextC:
		return
	case <-s.ctx.Done():
		if err := s.conn.WriteControl(
			ws.CloseMessage,
			ws.FormatCloseMessage(ws.CloseGoingAway, ""),
			time.Now().Add(time.Second),
		); err != nil && !roacherrors.Is(err, ws.ErrCloseSent) {
			log.Printf("error sending close message: %v \n", err)
		}
	}
}

// CloseSend implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) CloseSend() error {
	if s.serverClosed != nil || s.sendClosed {
		return nil
	}
	s.sendClosed = true
	return s.base.send(message[RQ]{Type: closeMessage})
}
