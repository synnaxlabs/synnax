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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distcontrol "github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/control"
	"github.com/synnaxlabs/x/address"
	xcontrol "github.com/synnaxlabs/x/control"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	leaseholder = address.Address("leaseholder")
	gateway     = address.Address("gateway")
)

var channelKeys = channel.Keys{1, 2}

var state = distcontrol.State{
	Subject:   xcontrol.Subject{Key: "writer-1", Name: "Writer 1"},
	Resource:  1,
	Authority: xcontrol.AuthorityAbsolute,
}

var _ = Describe("Transport", func() {
	var (
		net    *control.Network
		server distcontrol.Transport
		client distcontrol.Transport
	)
	BeforeEach(func() {
		net = control.NewNetwork()
		server = net.New(leaseholder)
		client = net.New(gateway)
	})

	It("Should round-trip a retrieve to the leaseholder", func(ctx SpecContext) {
		var received distcontrol.RetrieveRequest
		server.RetrieveServer().BindHandler(func(
			_ context.Context,
			req distcontrol.RetrieveRequest,
		) (distcontrol.RetrieveResponse, error) {
			received = req
			return distcontrol.RetrieveResponse{
				States: []distcontrol.State{state},
			}, nil
		})
		res := MustSucceed(client.RetrieveClient().Send(
			ctx,
			leaseholder,
			distcontrol.RetrieveRequest{Keys: channelKeys},
		))
		Expect(received.Keys).To(Equal(channelKeys))
		Expect(res.States).To(ConsistOf(state))
	})

	It("Should round-trip a subscription to the leaseholder", func(ctx SpecContext) {
		server.SubscribeServer().BindHandler(func(
			_ context.Context,
			srv distcontrol.SubscribeStream,
		) error {
			return srv.Send(distcontrol.SubscribeResponse{
				States: []distcontrol.State{state},
			})
		})
		stream := MustSucceed(client.SubscribeClient().Stream(ctx, leaseholder))
		Expect(MustSucceed(stream.Receive()).States).To(ConsistOf(state))
		Expect(stream.CloseSend()).To(Succeed())
	})

	It(
		"Should return a target-not-found error for an unbound leaseholder",
		func(ctx SpecContext) {
			Expect(client.RetrieveClient().
				Send(ctx, "nonexistent", distcontrol.RetrieveRequest{}),
			).Error().To(MatchError(ContainSubstring("not found")))
		},
	)
})
