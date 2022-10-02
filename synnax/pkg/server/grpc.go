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

func (g *GRPCBranch) Key() string { return "grpc" }

func (g *GRPCBranch) Serve(cfg BranchConfig) error {
	lis := cfg.Mux.Match(cmux.Any())
	var opts []grpc.ServerOption
	if cfg.TLS != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(cfg.TLS)))
	}
	g.server = grpc.NewServer(opts...)
	for _, t := range g.Transports {
		t.BindTo(g.server)
	}
	return filterCloseError(g.server.Serve(lis))
}

func (g *GRPCBranch) Stop() { g.server.GracefulStop() }
