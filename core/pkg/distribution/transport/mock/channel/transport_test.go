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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	mockchannel "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/channel"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	leaseholder = address.Address("leaseholder")
	gateway     = address.Address("gateway")
)

var _ = Describe("Transport", func() {
	var (
		net    *mockchannel.Network
		server channel.Transport
		client channel.Transport
	)
	BeforeEach(func() {
		net = mockchannel.NewNetwork()
		server = net.New(leaseholder)
		client = net.New(gateway)
	})

	It("Should round-trip a create request to the leaseholder", func(ctx SpecContext) {
		server.CreateServer().BindHandler(
			func(_ context.Context, req channel.CreateMessage) (channel.CreateMessage, error) {
				for i := range req.Channels {
					req.Channels[i].Name += "-created"
				}
				return req, nil
			},
		)
		res := MustSucceed(client.CreateClient().Send(ctx, leaseholder, channel.CreateMessage{
			Channels: []channel.Channel{{Name: "alpha"}, {Name: "beta"}},
		}))
		Expect(res.Channels).To(HaveLen(2))
		Expect(res.Channels[0].Name).To(Equal("alpha-created"))
		Expect(res.Channels[1].Name).To(Equal("beta-created"))
	})

	It("Should round-trip a delete request to the leaseholder", func(ctx SpecContext) {
		var received channel.Keys
		server.DeleteServer().BindHandler(
			func(_ context.Context, req channel.DeleteRequest) (types.Nil, error) {
				received = req.Keys
				return types.Nil{}, nil
			},
		)
		MustSucceed(client.DeleteClient().Send(ctx, leaseholder, channel.DeleteRequest{
			Keys: channel.Keys{1, 2, 3},
		}))
		Expect(received).To(Equal(channel.Keys{1, 2, 3}))
	})

	It("Should round-trip a rename request to the leaseholder", func(ctx SpecContext) {
		var received channel.RenameRequest
		server.RenameServer().BindHandler(
			func(_ context.Context, req channel.RenameRequest) (types.Nil, error) {
				received = req
				return types.Nil{}, nil
			},
		)
		MustSucceed(client.RenameClient().Send(ctx, leaseholder, channel.RenameRequest{
			Keys:  channel.Keys{7},
			Names: []string{"renamed"},
		}))
		Expect(received.Keys).To(Equal(channel.Keys{7}))
		Expect(received.Names).To(Equal([]string{"renamed"}))
	})

	It("Should return a target-not-found error for an unbound leaseholder", func(ctx SpecContext) {
		Expect(client.CreateClient().Send(ctx, "nonexistent", channel.CreateMessage{})).
			Error().To(MatchError(ContainSubstring("not found")))
	})

	It("Should expose non-nil clients and servers for every operation", func() {
		Expect(server.CreateClient()).ToNot(BeNil())
		Expect(server.CreateServer()).ToNot(BeNil())
		Expect(server.DeleteClient()).ToNot(BeNil())
		Expect(server.DeleteServer()).ToNot(BeNil())
		Expect(server.RenameClient()).ToNot(BeNil())
		Expect(server.RenameServer()).ToNot(BeNil())
	})
})
