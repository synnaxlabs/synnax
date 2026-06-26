// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides helpers for standing up in-process gRPC servers in tests
// that exercise freighter gRPC transports.
package testutil

import (
	"net"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Server is an in-process gRPC server started by StartServer, bundled with the address
// it listens on and a freighter connection pool dialed against it.
type Server struct {
	// Address is the ephemeral localhost address the server listens on.
	Address address.Address
	// Pool is a freighter gRPC connection pool dialed against the server with insecure
	// transport credentials. It is closed automatically when the spec completes.
	Pool *fgrpc.Pool
	// Server is the underlying gRPC server. It is gracefully stopped automatically when
	// the spec completes.
	Server *grpc.Server
}

// StartServer starts a gRPC server listening on an ephemeral localhost port and opens a
// freighter connection pool dialed against it with insecure transport credentials. bind
// registers the transports under test, receiving the server as a grpc.ServiceRegistrar
// and the pool so that a transport acting as both client and server can be constructed
// from it. Any opts are forwarded to grpc.NewServer.
//
// The server is served in a background goroutine. StartServer registers Ginkgo cleanup
// that gracefully stops the server and then closes the pool when the current spec
// completes, so it must be called from within a Ginkgo spec or lifecycle hook.
func StartServer(
	bind func(reg grpc.ServiceRegistrar, pool *fgrpc.Pool),
	opts ...grpc.ServerOption,
) Server {
	lis := testutil.MustSucceed(net.Listen("tcp", "localhost:0"))
	srv := grpc.NewServer(opts...)
	pool := testutil.DeferClose(fgrpc.OpenPool(
		"",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	))
	bind(srv, pool)
	go func() {
		defer ginkgo.GinkgoRecover()
		// Serve returns ErrServerStopped when GracefulStop wins the race against a
		// spec that finishes before the server fully starts. That is the cleanup we
		// asked for, not a failure.
		if err := srv.Serve(lis); !errors.Is(err, grpc.ErrServerStopped) {
			gomega.Expect(err).To(gomega.Succeed())
		}
	}()
	ginkgo.DeferCleanup(srv.GracefulStop)
	return Server{
		Address: address.Address(lis.Addr().String()),
		Pool:    pool,
		Server:  srv,
	}
}
