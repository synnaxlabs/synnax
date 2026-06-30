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
	"go/types"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/freighter/grpc/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	transportgrpc "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func openPool() *fgrpc.Pool {
	return DeferClose(fgrpc.OpenPool(
		"",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	))
}

var _ = Describe("Transport", func() {
	Describe("Construction", func() {
		It("Should bundle the channel and framer transports", func() {
			t := transportgrpc.New(openPool())
			Expect(t.Channel()).ToNot(BeNil())
			Expect(t.Framer()).ToNot(BeNil())
		})

		It("Should register both the channel and framer services on bind", func() {
			t := transportgrpc.New(openPool())
			srv := grpc.NewServer()
			t.BindTo(srv)
			info := srv.GetServiceInfo()
			Expect(info).To(HaveKey(HavePrefix("distribution.channel.pb.")))
			Expect(info).To(HaveKey(HavePrefix("synnax.distribution.framer.")))
		})

		It("Should report the gRPC protocol and protobuf encoding", func() {
			Expect(transportgrpc.New(openPool()).Report()).To(Equal(fgrpc.Reporter.Report()))
		})
	})

	Describe("Wire round-trip", func() {
		It("Should round-trip a channel create through the bundled transport", func(ctx SpecContext) {
			transport.Channel().CreateServer().BindHandler(
				func(_ context.Context, req channel.CreateMessage) (channel.CreateMessage, error) {
					return req, nil
				},
			)
			res := MustSucceed(transport.Channel().CreateClient().Send(
				ctx,
				addr,
				channel.CreateMessage{Channels: []channel.Channel{{Name: "alpha"}}},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Name).To(Equal("alpha"))
		})

		It("Should round-trip a framer delete through the bundled transport", func(ctx SpecContext) {
			var received deleter.Request
			transport.Framer().Deleter().Server().BindHandler(
				func(_ context.Context, req deleter.Request) (types.Nil, error) {
					received = req
					return types.Nil{}, nil
				},
			)
			Expect(transport.Framer().Deleter().Client().Send(
				ctx,
				addr,
				deleter.Request{Keys: channel.Keys{1, 2, 3}},
			)).To(Equal(types.Nil{}))
			Expect(received.Keys).To(Equal(channel.Keys{1, 2, 3}))
		})
	})

	// Use is exercised against an isolated transport and server so the registered
	// middleware does not leak into the shared transport used by the other specs.
	Describe("Use", func() {
		It("Should apply middleware to both the channel and framer transports", func(ctx SpecContext) {
			var t transportgrpc.Transport
			useAddr := StartServer(func(reg grpc.ServiceRegistrar, pool *fgrpc.Pool) {
				t = transportgrpc.New(pool)
				t.BindTo(reg)
			})

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

			t.Channel().CreateServer().BindHandler(
				func(_ context.Context, req channel.CreateMessage) (channel.CreateMessage, error) {
					return req, nil
				},
			)
			MustSucceed(t.Channel().CreateClient().Send(
				ctx,
				useAddr,
				channel.CreateMessage{Channels: []channel.Channel{{Name: "alpha"}}},
			))

			t.Framer().Deleter().Server().BindHandler(
				func(_ context.Context, _ deleter.Request) (types.Nil, error) {
					return types.Nil{}, nil
				},
			)
			Expect(t.Framer().Deleter().Client().Send(
				ctx,
				useAddr,
				deleter.Request{Keys: channel.Keys{1}},
			)).To(Equal(types.Nil{}))

			Expect(clientCalls.Load()).To(Equal(int32(2)))
			Expect(serverCalls.Load()).To(Equal(int32(2)))
		})
	})
})
