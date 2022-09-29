package fws

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	"github.com/synnaxlabs/x/middleware"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"go/types"
	"net/http"
)

var (
	_ freighter.StreamTransportClient[any, types.Nil] = (*Client[any, types.Nil])(nil)
	_ freighter.ClientStream[any, types.Nil]          = (*clientStream[any, types.Nil])(nil)
	_ config.Config[ClientConfig]                     = ClientConfig{}
)

type (
	DialRequest struct {
		Target address.Address
		Header http.Header
	}
	DialMiddleware = middleware.Middleware[DialRequest, *http.Response]
)

type ClientConfig struct {
	EncoderDecoder httputil.EncoderDecoder
	DialMiddleware []DialMiddleware
	Logger         *zap.SugaredLogger
}

func (c ClientConfig) Validate() error {
	v := validate.New("[ws.Client]")
	validate.NotNil(v, "EncoderDecoder", c.EncoderDecoder)
	return v.Error()
}

func (c ClientConfig) Override(other ClientConfig) ClientConfig {
	c.EncoderDecoder = override.Nil(c.EncoderDecoder, other.EncoderDecoder)
	c.Logger = override.Nil(c.Logger, other.Logger)
	return c
}

var DefaultClientConfig = ClientConfig{
	EncoderDecoder: httputil.MsgPackEncoderDecoder,
	Logger:         zap.S(),
}

type Client[RQ, RS freighter.Payload] struct {
	ClientConfig
	dialer ws.Dialer
	freighter.Reporter
}

func NewClient[RQ, RS freighter.Payload](configs ...ClientConfig) (*Client[RQ, RS], error) {
	cfg, err := config.OverrideAndValidate(DefaultClientConfig, configs...)
	return &Client[RQ, RS]{ClientConfig: cfg, Reporter: reporter}, err
}

func (s *Client[RQ, RS]) Report() alamos.Report {
	r := reporter
	r.Encodings = []string{s.EncoderDecoder.ContentType()}
	return r.Report()
}

func (s *Client[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	header := http.Header{}
	header.Set(fiber.HeaderContentType, s.EncoderDecoder.ContentType())
	req := DialRequest{Target: target, Header: header}
	dialer := clientDialer{wrapped: s.dialer}
	_, err := middleware.ExecSequentially[DialRequest, *http.Response](ctx, req, append(s.DialMiddleware, dialer))
	return newClientStream[RQ, RS](ctx, s.EncoderDecoder, dialer.resolvedConn), err
}

type clientStream[RQ, RS freighter.Payload] struct {
	core[RS, RQ]
}

func newClientStream[RQ, RS freighter.Payload](
	ctx context.Context,
	ecd binary.EncoderDecoder,
	conn *ws.Conn,
) *clientStream[RQ, RS] {
	return &clientStream[RQ, RS]{core: newCore[RS, RQ](ctx, ecd, conn)}
}

// Send implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) Send(req RQ) error {
	if s.peerClosed != nil {
		return freighter.EOF
	}

	if s.closed {
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
	if s.peerClosed != nil || s.closed {
		return nil
	}
	s.closed = true
	return s.core.send(message[RQ]{Type: closeMessage})
}

type clientDialer struct {
	wrapped      ws.Dialer
	resolvedConn *ws.Conn
}

var _ DialMiddleware = clientDialer{}

func (c clientDialer) Handle(
	ctx context.Context,
	req DialRequest,
	_ func() (*http.Response, error),
) (*http.Response, error) {
	_ws, res, err := c.wrapped.DialContext(ctx, req.Target.String(), req.Header)
	if err != nil {
		return res, err
	}
	if res.StatusCode != fiber.StatusSwitchingProtocols {
		return res, roacherrors.New("[ws] - unable to upgrade connection")
	}
	c.resolvedConn = _ws
	return res, nil
}
