package fws

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/httputil"
	"go.uber.org/zap"
	"go/types"
)

var (
	_ freighter.StreamTransport[any, types.Nil] = (*Stream[any, types.Nil])(nil)
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

// Stream implements the freighter.StreamTransport interface.
func (s *Stream[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	return s.Client.Stream(ctx, address.Address("resolvedConn://"+target.String()+s.path))
}
