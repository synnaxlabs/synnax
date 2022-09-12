package server

import (
	"context"
	fiberapi "github.com/synnaxlabs/synnax/pkg/api/fiber"
	"github.com/synnaxlabs/synnax/pkg/api/grpc"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/signal"
	"github.com/cockroachdb/cmux"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"net"
)

type Config struct {
	ListenAddress address.Address
	FiberAPI      fiberapi.API
	GrpcAPI       grpc.API
	Security      struct {
		Insecure bool
	}
	Logger *zap.Logger
}

type Server struct {
	config Config
	fiber  *fiberServer
	grpc   *grpcServer
}

func New(cfg Config) *Server {
	return &Server{
		config: cfg,
		fiber:  newFiberServer(cfg),
		grpc:   newGRPCServer(cfg),
	}
}

func (s *Server) Start(_ context.Context) error {
	// explicitly ignore the context cancellation function here,
	// as our goroutines will cancel the context when they exit.
	ctx, _ := signal.Background(signal.WithLogger(s.config.Logger))
	lis, err := net.Listen("tcp", s.config.ListenAddress.PortString())
	if err != nil {
		return err
	}
	m := newMux(lis)
	s.grpc.start(ctx, m.grpc)
	s.fiber.start(ctx, m.http)
	ctx.Go(func(ctx context.Context) error {
		if err := m.serve(); !isCloseErr(err) {
			return err
		}
		return nil
	}, signal.WithKey("server.mux"), signal.CancelOnExit())
	return ctx.Wait()
}

func (s *Server) Stop() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	c.Exec(s.fiber.Stop)
	c.Exec(s.grpc.Stop)
	return c.Error()
}

func isCloseErr(err error) bool {
	return errors.Is(err, cmux.ErrListenerClosed) || errors.Is(err, net.ErrClosed)
}
