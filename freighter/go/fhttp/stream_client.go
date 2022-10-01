package fhttp

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	"go.uber.org/zap"
	"go/types"
	"net/http"
)

var (
	_ freighter.StreamClient[any, types.Nil] = (*streamClient[any, types.Nil])(nil)
	_ freighter.ClientStream[any, types.Nil] = (*clientStream[any, types.Nil])(nil)
	_ config.Config[ClientFactoryConfig]     = ClientFactoryConfig{}
)

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
