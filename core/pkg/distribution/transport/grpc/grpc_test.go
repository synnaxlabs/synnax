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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	transportgrpc "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Transport", func() {
	Describe("Construction", func() {
		It("Should bundle the channel and framer transports", func() {
			pool := DeferClose(fgrpc.OpenPool(
				"",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			))
			t := transportgrpc.New(pool)
			Expect(t.Channel()).ToNot(BeNil())
			Expect(t.Framer()).ToNot(BeNil())
			Expect(t.BindableTransports()).To(HaveLen(2))
		})
	})

	Describe("Wire round-trip", func() {
		It("Should round-trip a channel create through the bundled transport", func(ctx SpecContext) {
			transport.Channel().CreateServer().BindHandler(
				func(
					_ context.Context, req channel.CreateMessage,
				) (channel.CreateMessage, error) {
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
	})
})
