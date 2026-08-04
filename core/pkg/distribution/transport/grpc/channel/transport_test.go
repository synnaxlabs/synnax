// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"
	"go/types"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/freighter/grpc/testutil"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
)

var _ = Describe("Transport", func() {
	Describe("Create", func() {
		It("Should round-trip a create request over the wire", func(ctx SpecContext) {
			transport.CreateServer().BindHandler(
				func(
					_ context.Context, req distchannel.CreateMessage,
				) (distchannel.CreateMessage, error) {
					return req, nil
				},
			)
			res := MustSucceed(transport.CreateClient().Send(
				ctx,
				addr,
				distchannel.CreateMessage{
					Channels: []distchannel.Channel{{Name: "alpha"}},
				},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Name).To(Equal("alpha"))
		})
	})

	Describe("Delete", func() {
		It("Should round-trip a delete request over the wire", func(ctx SpecContext) {
			var received distchannel.DeleteRequest
			transport.DeleteServer().BindHandler(
				func(
					_ context.Context, req distchannel.DeleteRequest,
				) (types.Nil, error) {
					received = req
					return types.Nil{}, nil
				},
			)
			Expect(transport.DeleteClient().Send(
				ctx,
				addr,
				distchannel.DeleteRequest{Keys: distchannel.Keys{1, 2, 3}},
			)).To(Equal(types.Nil{}))
			Expect(received.Keys).To(Equal(distchannel.Keys{1, 2, 3}))
		})
	})

	Describe("Rename", func() {
		It("Should round-trip a rename request over the wire", func(ctx SpecContext) {
			var received distchannel.RenameRequest
			transport.RenameServer().BindHandler(
				func(
					_ context.Context, req distchannel.RenameRequest,
				) (types.Nil, error) {
					received = req
					return types.Nil{}, nil
				},
			)
			Expect(transport.RenameClient().Send(
				ctx,
				addr,
				distchannel.RenameRequest{
					Renames: map[distchannel.Key]string{1: "beta", 2: "gamma"},
				},
			)).To(Equal(types.Nil{}))
			Expect(received.Renames).To(Equal(
				map[distchannel.Key]string{1: "beta", 2: "gamma"}),
			)
		})
	})

	// Use is exercised against an isolated transport and server so the registered
	// middleware does not leak into the shared transport used by the other specs.
	Describe("Use", func() {
		It(
			"Should apply middleware to both the client and server endpoints",
			func(ctx SpecContext) {
				var t channel.Transport
				useAddr := StartServer(
					func(reg grpc.ServiceRegistrar, pool *fgrpc.Pool) {
						t = channel.New(pool)
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

				t.CreateServer().BindHandler(
					func(
						_ context.Context, req distchannel.CreateMessage,
					) (distchannel.CreateMessage, error) {
						return req, nil
					},
				)
				res := MustSucceed(t.CreateClient().Send(
					ctx,
					useAddr,
					distchannel.CreateMessage{
						Channels: []distchannel.Channel{{Name: "alpha"}},
					},
				))
				Expect(res.Channels).To(HaveLen(1))
				Expect(clientCalls.Load()).To(Equal(int32(1)))
				Expect(serverCalls.Load()).To(Equal(int32(1)))
			},
		)
	})
})
