// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"context"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	grpctestutil "github.com/synnaxlabs/freighter/grpc/testutil"
	v1 "github.com/synnaxlabs/freighter/grpc/v1"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
)

type echoServer struct {
	v1.UnimplementedTestUnaryServiceServer
}

func (echoServer) Exec(_ context.Context, req *v1.Request) (*v1.Response, error) {
	return &v1.Response{Id: req.Id + 1, Message: req.Message}, nil
}

var _ = Describe("StartServer", func() {
	It("Should expose the address, pool, and server it created", func() {
		srv := grpctestutil.StartServer(func(grpc.ServiceRegistrar, *fgrpc.Pool) {})
		Expect(srv.Address).To(HavePrefix("127.0.0.1:"))
		Expect(srv.Pool).ToNot(BeNil())
		Expect(srv.Server).ToNot(BeNil())
	})

	It("Should invoke bind with the same server and a usable pool", func() {
		var (
			gotReg  grpc.ServiceRegistrar
			gotPool *fgrpc.Pool
		)
		srv := grpctestutil.StartServer(func(reg grpc.ServiceRegistrar, pool *fgrpc.Pool) {
			gotReg, gotPool = reg, pool
		})
		Expect(gotReg).To(BeIdenticalTo(grpc.ServiceRegistrar(srv.Server)))
		Expect(gotPool).To(BeIdenticalTo(srv.Pool))
	})

	It("Should serve registered services reachable through the pool", func(ctx SpecContext) {
		srv := grpctestutil.StartServer(func(reg grpc.ServiceRegistrar, _ *fgrpc.Pool) {
			v1.RegisterTestUnaryServiceServer(reg, echoServer{})
		})
		conn := MustSucceed(srv.Pool.Acquire(srv.Address))
		res := MustSucceed(v1.NewTestUnaryServiceClient(conn).Exec(
			ctx,
			&v1.Request{Id: 41, Message: "hello"},
		))
		Expect(res.Id).To(Equal(int32(42)))
		Expect(res.Message).To(Equal("hello"))
	})

	It("Should forward server options to the underlying gRPC server", func(ctx SpecContext) {
		var intercepted atomic.Bool
		srv := grpctestutil.StartServer(
			func(reg grpc.ServiceRegistrar, _ *fgrpc.Pool) {
				v1.RegisterTestUnaryServiceServer(reg, echoServer{})
			},
			grpc.ChainUnaryInterceptor(func(
				ctx context.Context,
				req any,
				_ *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler,
			) (any, error) {
				intercepted.Store(true)
				return handler(ctx, req)
			}),
		)
		conn := MustSucceed(srv.Pool.Acquire(srv.Address))
		MustSucceed(v1.NewTestUnaryServiceClient(conn).Exec(ctx, &v1.Request{Id: 1}))
		Expect(intercepted.Load()).To(BeTrue())
	})
})
