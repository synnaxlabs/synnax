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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Unary", Ordered, Serial, func() {
	var (
		server     freighter.UnaryServer[test.Request, test.Response]
		client     freighter.UnaryClient[test.Request, test.Response]
		addr       address.Address
		grpcServer *grpc.Server
	)

	BeforeAll(func() {
		lis, err := net.Listen("tcp", "localhost:0")
		Expect(err).ToNot(HaveOccurred())
		addr = address.Address(lis.Addr().String())

		grpcServer = grpc.NewServer()

		uServer := &fgrpc.UnaryServer[
			test.Request, *v1.Request,
			test.Response, *v1.Response,
		]{
			RequestTranslator:  requestTranslator{},
			ResponseTranslator: responseTranslator{},
			ServiceDesc:        &v1.TestUnaryService_ServiceDesc,
			Internal:           true,
		}
		uServer.BindTo(grpcServer)
		server = uServer

		pool := fgrpc.NewPool(
			"",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		client = &fgrpc.UnaryClient[
			test.Request, *v1.Request,
			test.Response, *v1.Response,
		]{
			RequestTranslator:  requestTranslator{},
			ResponseTranslator: responseTranslator{},
			Pool:               pool,
			ServiceDesc:        &v1.TestUnaryService_ServiceDesc,
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *v1.Request,
			) (*v1.Response, error) {
				return v1.NewTestUnaryServiceClient(conn).Exec(ctx, req)
			},
		}

		go func() {
			defer GinkgoRecover()
			Expect(grpcServer.Serve(lis)).To(Succeed())
		}()
	})

	AfterAll(func() { grpcServer.GracefulStop() })

	test.UnarySuite(func() (
		freighter.UnaryServer[test.Request, test.Response],
		freighter.UnaryClient[test.Request, test.Response],
		address.Address,
	) {
		return server, client, addr
	})
})
