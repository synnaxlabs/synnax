// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deleter_test

import (
	"context"
	"go/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distdeleter "github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/deleter"
	"github.com/synnaxlabs/x/address"
)

const (
	leaseholder = address.Address("leaseholder")
	gateway     = address.Address("gateway")
)

var _ = Describe("Transport", func() {
	var (
		net    *deleter.Network
		server distdeleter.Transport
		client distdeleter.Transport
	)
	BeforeEach(func() {
		net = deleter.NewNetwork()
		server = net.New(leaseholder)
		client = net.New(gateway)
	})

	It(
		"Should round-trip a request through the unary transport",
		func(ctx SpecContext) {
			var received channel.Keys
			server.Server().BindHandler(
				func(_ context.Context, req distdeleter.Request) (types.Nil, error) {
					received = req.Keys
					return types.Nil{}, nil
				},
			)
			Expect(client.Client().Send(ctx, leaseholder, distdeleter.Request{
				Keys: channel.Keys{4, 5},
			})).To(Equal(types.Nil{}))
			Expect(received).To(Equal(channel.Keys{4, 5}))
		},
	)
})
