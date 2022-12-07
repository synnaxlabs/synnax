package server

import (
	"github.com/cockroachdb/cmux"
	"github.com/synnaxlabs/freighter/fgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCBranch struct {
	Transports []fgrpc.BindableTransport
	server     *grpc.Server
}

func (g *GRPCBranch) Matchers() []cmux.Matcher {
	return []cmux.Matcher{cmux.Any()}
}

func (g *GRPCBranch) Key() string { return "grpc" }

func (g *GRPCBranch) Serve(cfg BranchConfig) error {
	var opts []grpc.ServerOption
	if *cfg.Security.Insecure {
		opts = append(opts, grpc.Creds(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.Creds(fgrpc.NewMuxCredentials(cfg.Logger)))
	}
	g.server = grpc.NewServer(opts...)
	for _, t := range g.Transports {
		t.BindTo(g.server, fgrpc.MTLSMiddleware(cfg.Security.CAName))
	}
	return filterCloserError(g.server.Serve(cfg.Lis))
}

func (g *GRPCBranch) Stop() { g.server.GracefulStop() }
