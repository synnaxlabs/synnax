// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"context"
	"net"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	framergrpc "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Transport", func() {
	Describe("Writer", func() {
		// The WriterRequest proto carries no SeqNum field (it is assigned server-side),
		// so we round-trip the Command on the request and assert the SeqNum the server
		// sets on the response.
		It("Should round-trip a request over the wire", func(ctx SpecContext) {
			transport.Writer().Server().BindHandler(func(
				ctx context.Context,
				srv freighter.ServerStream[writer.Request, writer.Response],
			) error {
				req, err := srv.Receive()
				if err != nil {
					return err
				}
				return srv.Send(writer.Response{SeqNum: 42, Command: req.Command})
			})
			stream := MustSucceed(transport.Writer().Client().Stream(ctx, addr))
			Expect(stream.Send(writer.Request{Command: writer.CommandWrite})).To(Succeed())
			res := MustSucceed(stream.Receive())
			Expect(res.SeqNum).To(Equal(42))
			Expect(res.Command).To(Equal(writer.CommandWrite))
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	Describe("Iterator", func() {
		It("Should round-trip a request over the wire", func(ctx SpecContext) {
			transport.Iterator().Server().BindHandler(func(
				ctx context.Context,
				srv freighter.ServerStream[iterator.Request, iterator.Response],
			) error {
				req, err := srv.Receive()
				if err != nil {
					return err
				}
				return srv.Send(iterator.Response{SeqNum: req.SeqNum})
			})
			stream := MustSucceed(transport.Iterator().Client().Stream(ctx, addr))
			Expect(stream.Send(iterator.Request{SeqNum: 42})).To(Succeed())
			Expect(MustSucceed(stream.Receive()).SeqNum).To(Equal(42))
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	Describe("Relay", func() {
		It("Should round-trip a request over the wire", func(ctx SpecContext) {
			transport.Relay().Server().BindHandler(func(
				ctx context.Context,
				srv freighter.ServerStream[relay.Request, relay.Response],
			) error {
				if _, err := srv.Receive(); err != nil {
					return err
				}
				return srv.Send(relay.Response{})
			})
			stream := MustSucceed(transport.Relay().Client().Stream(ctx, addr))
			Expect(stream.Send(relay.Request{})).To(Succeed())
			MustSucceed(stream.Receive())
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	Describe("Wiring", func() {
		It("Should wire all stream transport accessors", func() {
			Expect(transport.Writer().Client()).ToNot(BeNil())
			Expect(transport.Writer().Server()).ToNot(BeNil())
			Expect(transport.Iterator().Client()).ToNot(BeNil())
			Expect(transport.Iterator().Server()).ToNot(BeNil())
			Expect(transport.Relay().Client()).ToNot(BeNil())
			Expect(transport.Relay().Server()).ToNot(BeNil())
		})

		// BindTo does not register the deleter gRPC service, so the deleter cannot be
		// round-tripped over the wire here. We only assert it is wired structurally.
		It("Should wire the deleter unary transport", func() {
			Expect(transport.Deleter().Client()).ToNot(BeNil())
			Expect(transport.Deleter().Server()).ToNot(BeNil())
		})
	})

	// Use is exercised against an isolated transport and server so the registered
	// middleware does not leak into the shared transport used by the other specs.
	Describe("Use", func() {
		// framer's Use applies middleware only to the streaming client endpoints
		// (writer, iterator, relay), not the servers, so a stream round-trip invokes the
		// middleware on the client side only.
		It("Should apply middleware to the streaming client endpoints", func(ctx SpecContext) {
			lis := MustSucceed(net.Listen("tcp", "localhost:0"))
			useAddr := address.Address(lis.Addr().String())
			grpcServer := grpc.NewServer()
			pool := fgrpc.NewPool("", grpc.WithTransportCredentials(insecure.NewCredentials()))
			t := framergrpc.New(pool)
			t.BindTo(grpcServer)
			go func() {
				defer GinkgoRecover()
				Expect(grpcServer.Serve(lis)).To(Succeed())
			}()
			DeferCleanup(grpcServer.GracefulStop)

			var clientCalls, serverCalls atomic.Int32
			t.Use(freighter.MiddlewareFunc(func(
				mCtx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				switch mCtx.Role {
				case freighter.RoleClient:
					clientCalls.Add(1)
				case freighter.RoleServer:
					serverCalls.Add(1)
				}
				return next(mCtx)
			}))

			t.Writer().Server().BindHandler(func(
				_ context.Context,
				srv freighter.ServerStream[writer.Request, writer.Response],
			) error {
				if _, err := srv.Receive(); err != nil {
					return err
				}
				return srv.Send(writer.Response{})
			})
			stream := MustSucceed(t.Writer().Client().Stream(ctx, useAddr))
			Expect(stream.Send(writer.Request{Command: writer.CommandWrite})).To(Succeed())
			MustSucceed(stream.Receive())
			Expect(stream.CloseSend()).To(Succeed())

			Expect(clientCalls.Load()).To(Equal(int32(1)))
			Expect(serverCalls.Load()).To(Equal(int32(0)))
		})
	})
})
