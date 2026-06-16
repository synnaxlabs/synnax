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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	v1 "github.com/synnaxlabs/freighter/grpc/v1"
	"github.com/synnaxlabs/freighter/recovery"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// A negative request ID instructs the wire-test handlers to panic, exercising the
// recovery interceptors without a dedicated control channel.
const panicID = -1

var _ = Describe("Recovery (wire)", Ordered, Serial, func() {
	var (
		unaryClient  freighter.UnaryClient[test.Request, test.Response]
		streamClient freighter.StreamClient[test.Request, test.Response]
		addr         address.Address
		grpcServer   *grpc.Server
	)

	BeforeAll(func() {
		lis := MustSucceed(net.Listen("tcp", "localhost:0"))
		addr = address.Address(lis.Addr().String())

		ins := alamos.Instrumentation{}
		grpcServer = grpc.NewServer(
			grpc.ChainUnaryInterceptor(fgrpc.RecoveryUnaryServerInterceptor(ins)),
			grpc.ChainStreamInterceptor(fgrpc.RecoveryStreamServerInterceptor(ins)),
		)

		uServer := &fgrpc.UnaryServer[
			test.Request, *v1.Request,
			test.Response, *v1.Response,
		]{
			RequestTranslator:  requestTranslator{},
			ResponseTranslator: responseTranslator{},
			ServiceDesc:        &v1.TestUnaryService_ServiceDesc,
			Internal:           true,
		}
		uServer.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
			if req.ID == panicID {
				panic("boom in unary handler")
			}
			return test.Response(req), nil
		})
		uServer.BindTo(grpcServer)

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
		sServer.BindHandler(func(
			_ context.Context,
			server freighter.ServerStream[test.Request, test.Response],
		) error {
			req, err := server.Receive()
			if err != nil {
				return err
			}
			if req.ID == panicID {
				panic("boom in stream handler")
			}
			return server.Send(test.Response{ID: req.ID + 1, Message: req.Message})
		})
		sServer.BindTo(grpcServer)

		pool := fgrpc.NewPool("", grpc.WithTransportCredentials(insecure.NewCredentials()))
		unaryClient = &fgrpc.UnaryClient[
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
		streamClient = &fgrpc.StreamClient[
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

	It("should contain a unary handler panic and keep serving", func(ctx SpecContext) {
		By("surfacing the panic to the client as a generic error")
		Expect(unaryClient.Send(ctx, addr, test.Request{ID: panicID})).
			Error().To(MatchError(ContainSubstring(recovery.ErrPanic.Error())))

		By("continuing to serve subsequent requests")
		Expect(MustSucceed(unaryClient.Send(ctx, addr, test.Request{ID: 7, Message: "ok"}))).
			To(Equal(test.Response{ID: 7, Message: "ok"}))
	})

	It("should contain a stream handler panic and keep serving", func(ctx SpecContext) {
		By("surfacing the panic on the panicking stream")
		panicStream := MustSucceed(streamClient.Stream(ctx, addr))
		Expect(panicStream.Send(test.Request{ID: panicID})).To(Succeed())
		Expect(panicStream.Receive()).
			Error().To(MatchError(ContainSubstring(recovery.ErrPanic.Error())))

		By("continuing to serve new streams")
		okStream := MustSucceed(streamClient.Stream(ctx, addr))
		Expect(okStream.Send(test.Request{ID: 5, Message: "ok"})).To(Succeed())
		Expect(MustSucceed(okStream.Receive())).
			To(Equal(test.Response{ID: 6, Message: "ok"}))
	})
})
