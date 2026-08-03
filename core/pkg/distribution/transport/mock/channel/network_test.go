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
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/channel"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	leaseholder = address.Address("leaseholder")
	gateway     = address.Address("gateway")
)

var _ = Describe("Transport", func() {
	var (
		net    *channel.Network
		server distchannel.Transport
		client distchannel.Transport
	)
	BeforeEach(func() {
		net = channel.NewNetwork()
		server = net.New(leaseholder)
		client = net.New(gateway)
	})

	It("Should round-trip a create request to the leaseholder", func(ctx SpecContext) {
		server.CreateServer().BindHandler(
			func(
				_ context.Context, req distchannel.CreateMessage,
			) (distchannel.CreateMessage, error) {
				for i := range req.Channels {
					req.Channels[i].Name += "-created"
				}
				return req, nil
			},
		)
		res := MustSucceed(client.CreateClient().Send(
			ctx, leaseholder,
			distchannel.CreateMessage{
				Channels: []distchannel.Channel{{Name: "alpha"}, {Name: "beta"}},
			},
		))
		Expect(res.Channels).To(HaveLen(2))
		Expect(res.Channels[0].Name).To(Equal("alpha-created"))
		Expect(res.Channels[1].Name).To(Equal("beta-created"))
	})

	It("Should round-trip a delete request to the leaseholder", func(ctx SpecContext) {
		var received distchannel.Keys
		server.DeleteServer().BindHandler(
			func(_ context.Context, req distchannel.DeleteRequest) (types.Nil, error) {
				received = req.Keys
				return types.Nil{}, nil
			},
		)
		Expect(client.DeleteClient().Send(
			ctx,
			leaseholder,
			distchannel.DeleteRequest{Keys: distchannel.Keys{1, 2, 3}},
		)).To(Equal(types.Nil{}))
		Expect(received).To(Equal(distchannel.Keys{1, 2, 3}))
	})

	It("Should round-trip a rename request to the leaseholder", func(ctx SpecContext) {
		var received distchannel.RenameRequest
		server.RenameServer().BindHandler(
			func(_ context.Context, req distchannel.RenameRequest) (types.Nil, error) {
				received = req
				return types.Nil{}, nil
			},
		)
		Expect(client.RenameClient().Send(ctx, leaseholder, distchannel.RenameRequest{
			Renames: map[distchannel.Key]string{7: "renamed"},
		})).To(Equal(types.Nil{}))
		Expect(received.Renames).To(Equal(map[distchannel.Key]string{7: "renamed"}))
	})

	It("Should return a target-not-found error for an unbound leaseholder", func(ctx SpecContext) {
		Expect(client.CreateClient().Send(ctx, "nonexistent", distchannel.CreateMessage{})).
			Error().To(MatchError(ContainSubstring("not found")))
	})
})
