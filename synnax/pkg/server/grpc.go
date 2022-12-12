package server

import (
	"github.com/cockroachdb/cmux"
	"github.com/synnaxlabs/freighter/fgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCBranch is a Branch that serves gRPC traffic.
type GRPCBranch struct {
	// Transports is a list of bindable transports that the Branch will serve.
	Transports []fgrpc.BindableTransport
	server     *grpc.Server
}

var _ Branch = (*GRPCBranch)(nil)

// Routing implements Branch.
func (g *GRPCBranch) Routing() BranchRouting {
	return BranchRouting{
		Policy:   ServeAlwaysPreferSecure,
		Matchers: []cmux.Matcher{cmux.Any()},
	}
}

// Key implements Branch.
func (g *GRPCBranch) Key() string { return "grpc" }

// Serve implements Branch.
func (g *GRPCBranch) Serve(ctx BranchContext) error {
	opts := []grpc.ServerOption{g.credentials(ctx)}
	g.server = grpc.NewServer(opts...)
	return filterCloserError(g.server.Serve(ctx.Lis))
}

// Stop implements Branch.
func (g *GRPCBranch) Stop() { g.server.GracefulStop() }

func (g *GRPCBranch) credentials(ctx BranchContext) grpc.ServerOption {
	if *ctx.Security.Insecure {
		return grpc.Creds(insecure.NewCredentials())
	}
	// If we're running insecure mode, use mux credentials that pass TLS handshake
	// information from the TLS multiplexer to the grpc server, which allows
	// our services that need mTLS to validate against it.
	return grpc.Creds(fgrpc.NewMuxCredentials(ctx.Logger, ctx.ServerName))
}
