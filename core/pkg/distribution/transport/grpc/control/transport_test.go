// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	"context"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/freighter/grpc/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distcontrol "github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/control"
	xcontrol "github.com/synnaxlabs/x/control"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
)

func state(key channel.Key, subject string) distcontrol.State {
	return distcontrol.State{
		Subject:   xcontrol.Subject{Key: subject, Name: subject},
		Resource:  key,
		Authority: xcontrol.AuthorityAbsolute,
	}
}

var _ = Describe("Transport", func() {
	Describe("Retrieve", func() {
		It("Should round-trip a retrieve over the wire", func(ctx SpecContext) {
			var received distcontrol.RetrieveRequest
			transport.RetrieveServer().BindHandler(func(
				_ context.Context,
				req distcontrol.RetrieveRequest,
			) (distcontrol.RetrieveResponse, error) {
				received = req
				return distcontrol.RetrieveResponse{
					States: []distcontrol.State{state(1, "writer-1")},
				}, nil
			})
			res := MustSucceed(transport.RetrieveClient().Send(
				ctx,
				addr,
				distcontrol.RetrieveRequest{Keys: channel.Keys{1, 2}},
			))
			Expect(received.Keys).To(Equal(channel.Keys{1, 2}))
			Expect(res.States).To(ConsistOf(state(1, "writer-1")))
		})

		It("Should round-trip a retrieve for every channel", func(ctx SpecContext) {
			var received distcontrol.RetrieveRequest
			transport.RetrieveServer().BindHandler(func(
				_ context.Context,
				req distcontrol.RetrieveRequest,
			) (distcontrol.RetrieveResponse, error) {
				received = req
				return distcontrol.RetrieveResponse{}, nil
			})
			res := MustSucceed(transport.RetrieveClient().Send(
				ctx,
				addr,
				distcontrol.RetrieveRequest{},
			))
			Expect(received.Keys).To(BeEmpty())
			Expect(res.States).To(BeEmpty())
		})

		It("Should propagate a handler error to the client", func(ctx SpecContext) {
			transport.RetrieveServer().BindHandler(func(
				_ context.Context,
				_ distcontrol.RetrieveRequest,
			) (distcontrol.RetrieveResponse, error) {
				return distcontrol.RetrieveResponse{}, context.Canceled
			})
			Expect(transport.RetrieveClient().
				Send(ctx, addr, distcontrol.RetrieveRequest{}),
			).Error().To(MatchError(context.Canceled))
		})
	})

	Describe("Subscribe", func() {
		It("Should stream successive snapshots over the wire", func(ctx SpecContext) {
			transport.SubscribeServer().BindHandler(func(
				_ context.Context,
				srv distcontrol.SubscribeStream,
			) error {
				if err := srv.Send(distcontrol.SubscribeResponse{
					States: []distcontrol.State{state(1, "writer-1")},
				}); err != nil {
					return err
				}
				return srv.Send(distcontrol.SubscribeResponse{
					States: []distcontrol.State{
						state(1, "writer-1"),
						state(2, "writer-2"),
					},
				})
			})
			stream := MustSucceed(transport.SubscribeClient().Stream(ctx, addr))
			Expect(MustSucceed(stream.Receive()).States).To(ConsistOf(
				state(1, "writer-1"),
			))
			Expect(MustSucceed(stream.Receive()).States).To(ConsistOf(
				state(1, "writer-1"),
				state(2, "writer-2"),
			))
			Expect(stream.CloseSend()).To(Succeed())
		})

		It("Should end the stream when the handler returns", func(ctx SpecContext) {
			transport.SubscribeServer().BindHandler(func(
				_ context.Context,
				_ distcontrol.SubscribeStream,
			) error {
				return nil
			})
			stream := MustSucceed(transport.SubscribeClient().Stream(ctx, addr))
			Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	// Use is exercised against an isolated transport and server so the registered
	// middleware does not leak into the shared transport used by the other specs.
	Describe("Use", func() {
		It(
			"Should apply middleware to the unary and stream endpoints",
			func(ctx SpecContext) {
				var t control.Transport
				useAddr := StartServer(
					func(reg grpc.ServiceRegistrar, pool *fgrpc.Pool) {
						t = control.New(pool)
						t.BindTo(reg)
					},
				)

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

				t.RetrieveServer().BindHandler(func(
					_ context.Context,
					_ distcontrol.RetrieveRequest,
				) (distcontrol.RetrieveResponse, error) {
					return distcontrol.RetrieveResponse{}, nil
				})
				MustSucceed(t.RetrieveClient().
					Send(ctx, useAddr, distcontrol.RetrieveRequest{}))

				t.SubscribeServer().BindHandler(func(
					_ context.Context,
					srv distcontrol.SubscribeStream,
				) error {
					return srv.Send(distcontrol.SubscribeResponse{})
				})
				stream := MustSucceed(t.SubscribeClient().Stream(ctx, useAddr))
				MustSucceed(stream.Receive())
				Expect(stream.CloseSend()).To(Succeed())

				Eventually(func() int32 { return clientCalls.Load() }).
					Should(Equal(int32(2)))
				Eventually(func() int32 { return serverCalls.Load() }).
					Should(Equal(int32(2)))
			},
		)
	})
})
