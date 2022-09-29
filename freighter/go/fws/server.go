package fws

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	"github.com/synnaxlabs/x/override"
	"go.uber.org/zap"
	"go/types"
	"time"
)

var (
	_ freighter.StreamTransportServer[any, types.Nil] = (*Server[any, types.Nil])(nil)
	_ freighter.ServerStream[any, types.Nil]          = (*serverStream[any, types.Nil])(nil)
	_ config.Config[ServerConfig]                     = ServerConfig{}
)

type ServerConfig struct {
	EncoderDecoder httputil.EncoderDecoder
	Logger         *zap.SugaredLogger
}

func (s ServerConfig) Validate() error { return nil }

func (s ServerConfig) Override(other ServerConfig) ServerConfig {
	s.EncoderDecoder = override.Nil(s.EncoderDecoder, other.EncoderDecoder)
	s.Logger = override.Nil(s.Logger, other.Logger)
	return s
}

var DefaultServerConfig = ServerConfig{Logger: zap.S()}

type Server[RQ, RS freighter.Payload] struct {
	ServerConfig
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error
	freighter.Reporter
}

func NewServer[RQ, RS freighter.Payload](configs ...ServerConfig) (*Server[RQ, RS], error) {
	cfg, err := config.OverrideAndValidate(DefaultServerConfig, configs...)
	return &Server[RQ, RS]{ServerConfig: cfg, Reporter: reporter}, err
}

func (s *Server[RQ, RS]) BindHandler(
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

func (s *Server[RQ, RS]) BindTo(router fiber.Router, path string) {
	router.Get(path, func(c *fiber.Ctx) error {
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
				s.Logger.Errorw("error executing server stream", "error", err)
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

	if err := stream.conn.WriteMessage(
		ws.CloseMessage,
		ws.FormatCloseMessage(ws.CloseNormalClosure, ""),
	); err != nil {
		return err
	}

	go func() {
		defer close(closed)
		for {
			_, err := stream.receive()
			if ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
				return
			}
			if err != nil {
				s.Logger.Debugw("expected normal closure, received error instead", "error", err)
				return
			}
		}
	}()

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

const queryStringContentType = "contentType"

func (s *Server[RQ, RS]) determineEncoderDecoder(c *fiber.Ctx) (httputil.EncoderDecoder, error) {
	c.Accepts(httputil.SupportedContentTypes()...)
	ct := c.Get(fiber.HeaderContentType)

	if ct == "" {
		// try to get it from the query string
		ct = c.Query(queryStringContentType)
	}

	if s.EncoderDecoder != nil {
		if s.EncoderDecoder.ContentType() == ct {
			return s.EncoderDecoder, nil
		}
		return nil, roacherrors.Newf("[freighter] - unsupported content type")
	}

	return httputil.DetermineEncoderDecoder(ct)
}

func newServerStream[RQ, RS freighter.Payload](
	ecd binary.EncoderDecoder,
	conn *ws.Conn,
) *serverStream[RQ, RS] {
	return &serverStream[RQ, RS]{core: newCore[RQ, RS](context.TODO(), ecd, conn)}
}

type serverStream[RQ, RS freighter.Payload] struct {
	core[RQ, RS]
	clientClosed error
	closed       bool
}

// Receive implements the freighter.ServerStream interface.
func (s *serverStream[RQ, RS]) Receive() (req RQ, err error) {
	if s.clientClosed != nil {
		return req, s.clientClosed
	}

	if s.ctx.Err() != nil {
		return req, s.ctx.Err()
	}

	msg, err := s.core.receive()

	// A close message means the client called CloseSend.
	if msg.Type == closeMessage {
		s.clientClosed = freighter.EOF
		return req, s.clientClosed
	}

	// A close error with code close going away means the client context
	// was cancelled, and we should cancel the server context.
	if ws.IsCloseError(err, ws.CloseGoingAway) {
		close(s.contextC)
		s.clientClosed = context.Canceled
		s.cancel()
		return req, s.clientClosed
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
