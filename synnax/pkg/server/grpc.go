package server

import (
	"github.com/cockroachdb/cmux"
	"github.com/synnaxlabs/freighter/fgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GRPCBranch struct {
	Transports []fgrpc.BindableTransport
	server     *grpc.Server
}

func (g *GRPCBranch) Match() []cmux.Matcher {
	return []cmux.Matcher{cmux.Any()}
}

func (g *GRPCBranch) Key() string { return "grpc" }

func (g *GRPCBranch) Serve(cfg BranchConfig) error {
	var opts []grpc.ServerOption
	if !*cfg.Security.Insecure {
		opts = append(opts, grpc.Creds(credentials.NewTLS(cfg.Security.TLS)))
	}
	g.server = grpc.NewServer(opts...)
	for _, t := range g.Transports {
		t.BindTo(g.server)
	}
	return filterCloseError(g.server.Serve(cfg.Lis))
}

func (g *GRPCBranch) Stop() { g.server.GracefulStop() }
