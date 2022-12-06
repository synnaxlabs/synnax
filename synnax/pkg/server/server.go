package server

import (
	"context"
	"crypto/tls"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"net"

	"github.com/cockroachdb/cmux"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type Config struct {
	ListenAddress address.Address
	Security      SecurityConfig
	Logger        *zap.Logger
	Branches      []Branch
}

type SecurityConfig struct {
	Insecure *bool
	TLS      *tls.Config
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{
		Security: SecurityConfig{
			Insecure: config.BoolPointer(false),
		},
	}
)

func (c Config) Override(other Config) Config {
	c.ListenAddress = override.String(c.ListenAddress, other.ListenAddress)
	c.Security.Insecure = override.Nil(c.Security.Insecure, other.Security.Insecure)
	c.Security.TLS = override.Nil(c.Security.TLS, other.Security.TLS)
	c.Logger = override.Nil(c.Logger, other.Logger)
	c.Branches = override.Slice(c.Branches, other.Branches)
	return c
}

func (c Config) Validate() error {
	v := validate.New("server")
	validate.NotEmptyString(v, "listenAddress", c.ListenAddress)
	validate.NotNil(v, "logger", c.Logger)
	return v.Error()
}

type Server struct {
	Config
	lis net.Listener
}

func New(configs ...Config) (*Server, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	return &Server{Config: cfg}, err
}

func (s *Server) Start(_ context.Context) (err error) {
	// explicitly ignore the context cancellation function here,
	// as our goroutines will cancel the context when they exit.
	sCtx, _ := signal.Background(signal.WithLogger(s.Logger))
	s.lis, err = net.Listen("tcp", s.ListenAddress.PortString())
	if err != nil {
		return err
	}

	m := cmux.New(s.lis)
	listeners := make([]net.Listener, len(s.Branches))
	for i, b := range s.Branches {
		listeners[i] = m.Match(b.Match()...)
	}

	bc := BranchConfig{}
	for i, b := range s.Branches {
		b := b
		i := i
		sCtx.Go(func(ctx context.Context) error {
			bc.Lis = listeners[i]
			return b.Serve(bc)
		}, signal.WithKey(b.Key()))
	}
	return filterCloseError(m.Serve())
}

func (s *Server) Stop() error {
	for _, b := range s.Branches {
		b.Stop()
	}
	return nil
}

func filterCloseError(err error) error {
	if errors.Is(err, cmux.ErrListenerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}
