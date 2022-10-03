package server

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/cockroachdb/cmux"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type Config struct {
	ListenAddress address.Address
	Security      struct {
		TLS *tls.Config
	}
	Logger   *zap.Logger
	Branches []Branch
}

type Server struct {
	Config
	lis net.Listener
}

func New(cfg Config) *Server {
	return &Server{Config: cfg}
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
	bc := BranchConfig{
		TLS: s.Security.TLS,
	}
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
