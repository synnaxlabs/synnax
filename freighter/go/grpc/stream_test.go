// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc_test

import (
	"context"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	v1 "github.com/synnaxlabs/freighter/grpc/v1"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type streamServer struct {
	fgrpc.StreamServerCore[
		test.Request, *v1.Request,
		test.Response, *v1.Response,
	]
}

func (s *streamServer) Exec(
	stream v1.TestStreamService_ExecServer,
) error {
	return s.Handler(stream.Context(), stream)
}

func (s *streamServer) BindTo(reg grpc.ServiceRegistrar) {
	v1.RegisterTestStreamServiceServer(reg, s)
}

var _ = Describe("Stream", Ordered, Serial, func() {
	var (
		server     freighter.StreamServer[test.Request, test.Response]
		client     freighter.StreamClient[test.Request, test.Response]
		addr       address.Address
		grpcServer *grpc.Server
	)

	BeforeAll(func() {
		lis := MustSucceed(net.Listen("tcp", "localhost:0"))
		addr = address.Address(lis.Addr().String())

		grpcServer = grpc.NewServer()

		sServer := &streamServer{
			StreamServerCore: fgrpc.StreamServerCore[
				test.Request, *v1.Request,
				test.Response, *v1.Response,
			]{
				RequestTranslator:  requestTranslator{},
				ResponseTranslator: responseTranslator{},
				ServiceDesc:        &v1.TestStreamService_ServiceDesc,
				Internal:           true,
			},
		}
		sServer.BindTo(grpcServer)
		server = &sServer.StreamServerCore

		pool := fgrpc.NewPool(
			"",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		client = &fgrpc.StreamClient[
			test.Request, *v1.Request,
			test.Response, *v1.Response,
		]{
			RequestTranslator:  requestTranslator{},
			ResponseTranslator: responseTranslator{},
			Pool:               pool,
			ServiceDesc:        &v1.TestStreamService_ServiceDesc,
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*v1.Request, *v1.Response], error) {
				return v1.NewTestStreamServiceClient(conn).Exec(ctx)
			},
		}

		go func() {
			defer GinkgoRecover()
			Expect(grpcServer.Serve(lis)).To(Succeed())
		}()
	})

	AfterAll(func() { grpcServer.GracefulStop() })

	test.StreamSuite(func() (
		freighter.StreamServer[test.Request, test.Response],
		freighter.StreamClient[test.Request, test.Response],
		address.Address,
	) {
		return server, client, addr
	})
})
