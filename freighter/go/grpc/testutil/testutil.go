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
	"context"
	"net"
	"sync/atomic"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/stats"
)

// connBeginHandler reports whether the server has accepted a connection.
type connBeginHandler struct{ began *atomic.Bool }

var _ stats.Handler = connBeginHandler{}

func (connBeginHandler) TagRPC(
	ctx context.Context,
	_ *stats.RPCTagInfo,
) context.Context {
	return ctx
}

func (connBeginHandler) HandleRPC(context.Context, stats.RPCStats) {}

func (connBeginHandler) TagConn(
	ctx context.Context,
	_ *stats.ConnTagInfo,
) context.Context {
	return ctx
}

func (h connBeginHandler) HandleConn(_ context.Context, s stats.ConnStats) {
	if _, ok := s.(*stats.ConnBegin); ok {
		h.began.Store(true)
	}
}

// StartServer starts a gRPC server listening on an ephemeral localhost port and opens a
// freighter connection pool dialed against it with insecure transport credentials. bind
// registers the transports under test, receiving the server as a grpc.ServiceRegistrar
// and the pool so that a transport acting as both client and server can be constructed
// from it. Any opts are forwarded to grpc.NewServer. It returns the ephemeral localhost
// address the server listens on.
//
// The server is served in a background goroutine. StartServer registers Ginkgo cleanup
// that gracefully stops the server and then closes the pool when the current spec
// completes, so it must be called from within a Ginkgo spec or lifecycle hook.
//
// Before returning, StartServer dials and waits for both ends of the pooled connection
// to come up. The pool dials lazily and caches the connection for the rest of the spec,
// so pre-warming it here makes the connection part of the caller's goroutine baseline:
// callers that invoke StartServer from a BeforeAll keep per-spec goroutine-leak checks
// passing, because no spec is the one that first registers the long-lived connection.
// The server end is awaited separately because the client reports ready as soon as it
// reads the server preface, which the server writes before it starts its per-connection
// goroutines.
func StartServer(
	bind func(grpc.ServiceRegistrar, *fgrpc.Pool),
	opts ...grpc.ServerOption,
) address.Address {
	ginkgo.GinkgoHelper()
	lis := testutil.MustSucceed(net.Listen("tcp", "localhost:0"))
	addr := address.Address(lis.Addr().String())
	var began atomic.Bool
	srv := grpc.NewServer(
		append(opts, grpc.StatsHandler(connBeginHandler{began: &began}))...,
	)
	pool := testutil.DeferClose(fgrpc.OpenPool(
		"",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	))
	bind(srv, pool)
	go func() {
		defer ginkgo.GinkgoRecover()
		// Serve returns ErrServerStopped when GracefulStop wins the race against a spec
		// that finishes before the server fully starts. That is the cleanup we asked
		// for, not a failure.
		if err := srv.Serve(lis); !errors.Is(err, grpc.ErrServerStopped) {
			gomega.Expect(err).To(gomega.Succeed())
		}
	}()
	ginkgo.DeferCleanup(srv.GracefulStop)
	conn := testutil.MustSucceed(pool.Acquire(addr))
	conn.Connect()
	gomega.Eventually(conn.GetState).Should(gomega.Equal(connectivity.Ready))
	gomega.Eventually(began.Load).Should(gomega.BeTrue())
	return addr
}
