package server

import (
	"context"
	"github.com/synnaxlabs/x/signal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
)

type grpcServer struct {
	server *grpc.Server
}

func newGRPCServer(cfg Config) *grpcServer {
	srv := &grpcServer{server: grpc.NewServer(grpc.Creds(insecure.NewCredentials()))}
	return srv
}

func (g grpcServer) start(ctx signal.Context, lis net.Listener) {
	ctx.Go(func(ctx context.Context) error {
		if err := g.server.Serve(lis); !isCloseErr(err) {
			return err
		}
		return nil
	}, signal.WithKey("server.grpc"), signal.CancelOnExit())
}

func (g grpcServer) Stop() error { g.server.Stop(); return nil }
