// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distrelay "github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/relay"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	leaseholder = address.Address("leaseholder")
	gateway     = address.Address("gateway")
)

var _ = Describe("Transport", func() {
	var (
		net    *relay.Network
		server distrelay.Transport
		client distrelay.Transport
	)
	BeforeEach(func() {
		net = relay.NewNetwork()
		server = net.New(leaseholder, 1)
		client = net.New(gateway, 1)
	})

	It("Should round-trip a request through the streaming transport", func(ctx SpecContext) {
		server.Server().BindHandler(func(
			_ context.Context,
			srv freighter.ServerStream[distrelay.Request, distrelay.Response],
		) error {
			if _, err := srv.Receive(); err != nil {
				return err
			}
			return srv.Send(distrelay.Response{Group: 43})
		})
		stream := MustSucceed(client.Client().Stream(ctx, leaseholder))
		Expect(stream.Send(distrelay.Request{Keys: channel.Keys{4, 5}})).To(Succeed())
		Expect(MustSucceed(stream.Receive()).Group).To(Equal(uint32(43)))
		Expect(stream.CloseSend()).To(Succeed())
	})
})
