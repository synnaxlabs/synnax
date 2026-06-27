// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	"context"
	"go/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	leaseholder = address.Address("leaseholder")
	gateway     = address.Address("gateway")
)

var _ = Describe("Transport", func() {
	var (
		net    *mock.Network
		server distribution.Transport
		client distribution.Transport
	)
	BeforeEach(func() {
		net = mock.NewNetwork()
		server = net.New(leaseholder, 1)
		client = net.New(gateway, 1)
	})

	It("Should provision a transport that implements distribution.Transport", func() {
		Expect(server.Channel()).ToNot(BeNil())
		Expect(server.Framer()).ToNot(BeNil())
	})

	It("Should route a channel create request through the bundled transport", func(ctx SpecContext) {
		server.Channel().CreateServer().BindHandler(
			func(
				_ context.Context, req channel.CreateMessage,
			) (channel.CreateMessage, error) {
				return req, nil
			},
		)
		res := MustSucceed(client.Channel().CreateClient().Send(
			ctx,
			leaseholder,
			channel.CreateMessage{Channels: []channel.Channel{{Name: "alpha"}}},
		))
		Expect(res.Channels).To(HaveLen(1))
		Expect(res.Channels[0].Name).To(Equal("alpha"))
	})

	It("Should route a framer delete request through the bundled transport", func(ctx SpecContext) {
		var received channel.Keys
		server.Framer().Deleter().Server().BindHandler(
			func(_ context.Context, req deleter.Request) (types.Nil, error) {
				received = req.Keys
				return types.Nil{}, nil
			},
		)
		Expect(client.Framer().Deleter().Client().Send(
			ctx,
			leaseholder,
			deleter.Request{Keys: channel.Keys{9}},
		)).To(Equal(types.Nil{}))
		Expect(received).To(Equal(channel.Keys{9}))
	})

})
