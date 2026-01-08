// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"github.com/cockroachdb/cmux"
	"github.com/synnaxlabs/freighter/fgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCBranch is a Branch that serves gRPC traffic.
type GRPCBranch struct {
	server *grpc.Server
	// Transports is a list of bindable transports that the Branch will serve.
	Transports []fgrpc.BindableTransport
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
	for _, t := range g.Transports {
		t.BindTo(g.server)
	}
	return g.server.Serve(ctx.Lis)
}

// Stop implements Branch. Stop is safe to call even if Serve has not been called.
func (g *GRPCBranch) Stop() { g.server.Stop() }

func (g *GRPCBranch) credentials(ctx BranchContext) grpc.ServerOption {
	if *ctx.Security.Insecure {
		return grpc.Creds(insecure.NewCredentials())
	}
	// If we're running in secure mode, use mux credentials that pass TLS handshake
	// information from the TLS multiplexer to the grpc server, which allows
	// our services that need mTLS to validate against it.
	return grpc.Creds(&fgrpc.MuxCredentials{Instrumentation: ctx.Instrumentation})
}
