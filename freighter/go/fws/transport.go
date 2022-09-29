package fws

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	override "github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"go/types"
)

var (
	_ freighter.StreamTransport[any, types.Nil] = (*Transport[any, types.Nil])(nil)
	_ config.Config[TransportConfig]            = TransportConfig{}
)

type TransportConfig struct {
	EncoderDecoder httputil.EncoderDecoder
	DialMiddleware []DialMiddleware
	Logger         *zap.SugaredLogger
}

func (t TransportConfig) client() ClientConfig {
	return ClientConfig{
		EncoderDecoder: t.EncoderDecoder,
		DialMiddleware: t.DialMiddleware,
		Logger:         t.Logger,
	}
}

func (t TransportConfig) server() ServerConfig {
	return ServerConfig{
		EncoderDecoder: t.EncoderDecoder,
		Logger:         t.Logger,
	}
}

func (t TransportConfig) Validate() error {
	v := validate.New("[ws.Transport]")
	v.Exec(t.client().Validate)
	v.Exec(t.server().Validate)
	return v.Error()
}

func (t TransportConfig) Override(other TransportConfig) TransportConfig {
	t.EncoderDecoder = override.Nil(t.EncoderDecoder, other.EncoderDecoder)
	t.Logger = override.Nil(t.Logger, other.Logger)
	t.DialMiddleware = override.Slice(t.DialMiddleware, other.DialMiddleware)
	return t
}

var DefaultTransportConfig = TransportConfig{
	EncoderDecoder: DefaultClientConfig.EncoderDecoder,
	Logger:         DefaultClientConfig.Logger,
	DialMiddleware: DefaultClientConfig.DialMiddleware,
}

type Transport[RQ, RS freighter.Payload] struct {
	Server[RQ, RS]
	Client[RQ, RS]
	path string
}

func New[RQ, RS freighter.Payload](configs ...TransportConfig) (*Transport[RQ, RS], error) {
	cfg, err := config.OverrideAndValidate(DefaultTransportConfig, configs...)
	if err != nil {
		return nil, err
	}
	client, err := NewClient[RQ, RS](cfg.client())
	if err != nil {
		return nil, err
	}
	server, err := NewServer[RQ, RS](cfg.server())
	if err != nil {
		return nil, err
	}
	return &Transport[RQ, RS]{Server: *server, Client: *client}, nil
}

// Report implements the freighter.Transport interface.
func (s *Transport[RQ, RS]) Report() alamos.Report { return s.Client.Report() }

func (s *Transport[RQ, RS]) BindTo(r fiber.Router, path string) {
	s.path = path
	s.Server.BindTo(r, s.path)
}

// Transport implements the freighter.StreamTransport interface.
func (s *Transport[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	return s.Client.Stream(ctx, address.Address("resolvedConn://"+target.String()+s.path))
}
