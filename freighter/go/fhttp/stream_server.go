package fhttp

import (
	"context"
	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
	"go.uber.org/zap"
	"time"
)

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
	if err := s.MiddlewareCollector.Exec(
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
		})); err != nil {
		return encodeAndWrite(c, ecd, ferrors.Encode(err))
	}
	return nil
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
