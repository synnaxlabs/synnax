package fhttp

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	"go.uber.org/zap"
	"go/types"
	"io"
	"net/http"
	"time"
)

var (
	_ freighter.StreamClient[any, types.Nil] = (*streamClient[any, types.Nil])(nil)
	_ freighter.ClientStream[any, types.Nil] = (*clientStream[any, types.Nil])(nil)
	_ config.Config[ClientFactoryConfig]     = ClientFactoryConfig{}
)

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

type streamClient[RQ, RS freighter.Payload] struct {
	logger *zap.SugaredLogger
	ecd    httputil.EncoderDecoder
	dialer ws.Dialer
	freighter.Reporter
	freighter.MiddlewareCollector
}

func (s *streamClient[RQ, RS]) Report() alamos.Report {
	r := streamReporter
	r.Encodings = []string{s.ecd.ContentType()}
	return r.Report()
}

func (s *streamClient[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (stream freighter.ClientStream[RQ, RS], _ error) {
	return stream, s.MiddlewareCollector.Exec(
		ctx,
		freighter.MD{Target: target, Protocol: s.Reporter.Protocol, Params: make(freighter.Params)},
		freighter.FinalizerFunc(func(ctx context.Context, md freighter.MD) error {
			md.Params["Content-Type"] = s.ecd.ContentType()
			conn, res, err := s.dialer.DialContext(ctx, "ws://"+target.String(), mdToHeaders(md))
			if err != nil {
				return err
			}
			if res.StatusCode != fiber.StatusSwitchingProtocols {
				return roacherrors.New("[ws] - unable to upgrade connection")
			}
			stream = &clientStream[RQ, RS]{core: newCore[RS, RQ](ctx, conn, s.ecd, s.logger)}
			return nil
		}),
	)
}

type clientStream[RQ, RS freighter.Payload] struct {
	core[RS, RQ]
	sendClosed bool
}

// Send implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) Send(req RQ) error {
	if s.peerClosed != nil {
		return freighter.EOF
	}

	if s.sendClosed {
		return freighter.StreamClosed
	}

	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}

	if err := s.core.send(message[RQ]{Type: messageTypeData, Payload: req}); err != nil {
		close(s.contextC)
		return freighter.EOF
	}

	return nil
}

// Receive implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) Receive() (res RS, err error) {
	if s.peerClosed != nil {
		return res, s.peerClosed
	}

	if s.ctx.Err() != nil {
		return res, s.ctx.Err()
	}

	msg, err := s.core.receive()

	// A close message means the server handler exited.
	if msg.Type == closeMessage {
		close(s.contextC)
		s.peerClosed = ferrors.Decode(msg.Err)
		return res, s.peerClosed
	}

	if isRemoteContextCancellation(err) {
		return res, s.cancelStream()
	}

	return msg.Payload, err
}

// CloseSend implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) CloseSend() error {
	if s.peerClosed != nil || s.sendClosed {
		return nil
	}
	s.sendClosed = true
	return s.core.send(message[RQ]{Type: closeMessage})
}

type clientDialer struct {
	wrapped      ws.Dialer
	resolvedConn *ws.Conn
}

func mdToHeaders(md freighter.MD) http.Header {
	headers := http.Header{}
	for k, v := range md.Params {
		if vStr, ok := v.(string); ok {
			headers[k] = []string{vStr}
		}
	}
	return headers
}

type streamServer[RQ, RS freighter.Payload] struct {
	freighter.Reporter
	freighter.MiddlewareCollector
	logger  *zap.SugaredLogger
	path    string
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error
}

func (s *streamServer[RQ, RS]) BindHandler(
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

func (s *streamServer[RQ, RS]) fiberHandler(c *fiber.Ctx) error {
	if !fiberws.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}
	c.Accepts(httputil.SupportedContentTypes()...)
	ecd, err := httputil.DetermineEncoderDecoder(c.Get(fiber.HeaderContentType))
	if err != nil {
		return err
	}
	err = s.MiddlewareCollector.Exec(
		c.Context(),
		parseRequestParams(c, address.Address(s.path)),
		freighter.FinalizerFunc(func(ctx context.Context, _ freighter.MD) error {
			return fiberws.New(func(c *fiberws.Conn) {
				if err := func() error {
					stream := &serverStream[RQ, RS]{core: newCore[RQ, RS](context.TODO(), c.Conn, ecd, zap.S())}

					errPayload := ferrors.Encode(s.handler(stream.ctx, stream))
					if errPayload.Type == ferrors.Nil {
						errPayload = ferrors.Encode(freighter.EOF)
					}

					if stream.ctx.Err() != nil {
						return stream.ctx.Err()
					}

					if err := stream.send(message[RS]{Type: closeMessage, Err: errPayload}); err != nil {
						return err
					}

					stream.peerClosed = freighter.StreamClosed

					clientCloseAck := make(chan struct{})

					if err := stream.conn.WriteMessage(
						ws.CloseMessage,
						ws.FormatCloseMessage(ws.CloseNormalClosure, ""),
					); err != nil {
						return err
					}

					go func() {
						defer close(clientCloseAck)
						for {
							_, err := stream.receive()
							if ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
								return
							}
							if err != nil {
								s.logger.Errorw("expected normal closure, received error instead", "error", err)
								return
							}
						}
					}()

					select {
					case <-stream.ctx.Done():
						break
					case <-time.After(500 * time.Millisecond):
						s.logger.Warnw("timed out waiting for client to acknowledge closure")
						break
					case <-clientCloseAck:
						break
					}
					close(stream.contextC)
					stream.cancel()
					return nil
				}(); err != nil {
					s.logger.Errorw("stream server handler error", "error", err)
				}
			})(c)
		}))
	fErr := ferrors.Encode(err)
	if fErr.Type == ferrors.Nil {
		return nil
	}
	c.Status(fiber.StatusBadRequest)
	return encodeAndWrite(c, ecd, fErr)
}

type serverStream[RQ, RS freighter.Payload] struct{ core[RQ, RS] }

// Receive implements the freighter.ServerStream interface.
func (s *serverStream[RQ, RS]) Receive() (req RQ, err error) {
	if s.peerClosed != nil {
		return req, s.peerClosed
	}

	if s.ctx.Err() != nil {
		return req, s.ctx.Err()
	}

	msg, err := s.core.receive()

	// A close message means the client called CloseSend.
	if msg.Type == closeMessage {
		s.peerClosed = freighter.EOF
		return req, s.peerClosed
	}

	if isRemoteContextCancellation(err) {
		return req, s.cancelStream()
	}

	return msg.Payload, err
}

// Send implements the freighter.ServerStream interface.
func (s *serverStream[RQ, RS]) Send(res RS) error {
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	return s.core.send(message[RS]{Payload: res, Type: messageTypeData})
}

func isRemoteContextCancellation(err error) bool {
	return ws.IsCloseError(err, ws.CloseGoingAway)
}
