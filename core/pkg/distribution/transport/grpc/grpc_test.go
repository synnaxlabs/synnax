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
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	transportgrpc "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Transport", func() {
	Describe("Construction", func() {
		It("Should bundle the channel and framer transports", func() {
			pool := fgrpc.NewPool(
				"",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			t := transportgrpc.New(pool)
			Expect(t.Channel()).ToNot(BeNil())
			Expect(t.Framer()).ToNot(BeNil())
			Expect(t.BindableTransports()).To(HaveLen(2))
		})
	})

	Describe("Wire round-trip", Ordered, Serial, func() {
		var (
			t          transportgrpc.Transport
			addr       address.Address
			grpcServer *grpc.Server
		)

		BeforeAll(func() {
			lis := MustSucceed(net.Listen("tcp", "localhost:0"))
			addr = address.Address(lis.Addr().String())

			pool := fgrpc.NewPool(
				"",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			t = transportgrpc.New(pool)

			grpcServer = grpc.NewServer()
			for _, bt := range t.BindableTransports() {
				bt.BindTo(grpcServer)
			}

			go func() {
				defer GinkgoRecover()
				Expect(grpcServer.Serve(lis)).To(Succeed())
			}()
			DeferCleanup(grpcServer.GracefulStop)
		})

		It("Should round-trip a channel create through the bundled transport", func(ctx SpecContext) {
			t.Channel().CreateServer().BindHandler(
				func(
					_ context.Context, req channel.CreateMessage,
				) (channel.CreateMessage, error) {
					return req, nil
				},
			)
			res := MustSucceed(t.Channel().CreateClient().Send(
				ctx,
				addr,
				channel.CreateMessage{Channels: []channel.Channel{{Name: "alpha"}}},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Name).To(Equal("alpha"))
		})
	})
})
