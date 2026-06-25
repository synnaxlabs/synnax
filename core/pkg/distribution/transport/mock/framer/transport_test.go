// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"context"
	"go/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distframer "github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	leaseholder = address.Address("leaseholder")
	gateway     = address.Address("gateway")
)

var _ = Describe("Transport", func() {
	var (
		net    *framer.Network
		server distframer.Transport
		client distframer.Transport
	)
	BeforeEach(func() {
		net = framer.NewNetwork()
		server = net.New(leaseholder, 1)
		client = net.New(gateway, 1)
	})

	It("Should round-trip a request through the streaming writer transport", func(ctx SpecContext) {
		server.Writer().Server().BindHandler(
			func(
				_ context.Context,
				srv freighter.ServerStream[writer.Request, writer.Response],
			) error {
				req, err := srv.Receive()
				if err != nil {
					return err
				}
				return srv.Send(writer.Response{SeqNum: req.SeqNum})
			},
		)
		stream := MustSucceed(client.Writer().Client().Stream(ctx, leaseholder))
		Expect(stream.Send(writer.Request{
			SeqNum:  42,
			Command: writer.CommandWrite,
		})).To(Succeed())
		Expect(MustSucceed(stream.Receive()).SeqNum).To(Equal(42))
		Expect(stream.CloseSend()).To(Succeed())
	})

	It("Should round-trip a request through the streaming iterator transport", func(ctx SpecContext) {
		server.Iterator().Server().BindHandler(
			func(
				_ context.Context,
				srv freighter.ServerStream[iterator.Request, iterator.Response],
			) error {
				req, err := srv.Receive()
				if err != nil {
					return err
				}
				return srv.Send(iterator.Response{SeqNum: req.SeqNum})
			},
		)
		stream := MustSucceed(client.Iterator().Client().Stream(ctx, leaseholder))
		Expect(stream.Send(iterator.Request{SeqNum: 7})).To(Succeed())
		Expect(MustSucceed(stream.Receive()).SeqNum).To(Equal(7))
		Expect(stream.CloseSend()).To(Succeed())
	})

	It("Should round-trip a request through the streaming relay transport", func(ctx SpecContext) {
		server.Relay().Server().BindHandler(
			func(
				_ context.Context,
				srv freighter.ServerStream[relay.Request, relay.Response],
			) error {
				if _, err := srv.Receive(); err != nil {
					return err
				}
				return srv.Send(relay.Response{Group: 43})
			},
		)
		stream := MustSucceed(client.Relay().Client().Stream(ctx, leaseholder))
		Expect(stream.Send(relay.Request{
			Keys: channel.Keys{4, 5},
		})).To(Succeed())
		relayRes := MustSucceed(stream.Receive())
		Expect(relayRes.Group).To(Equal(uint32(43)))
		Expect(stream.CloseSend()).To(Succeed())
	})

	It("Should round-trip a request through the unary deleter transport", func(ctx SpecContext) {
		var received channel.Keys
		server.Deleter().Server().BindHandler(
			func(
				_ context.Context,
				req deleter.Request,
			) (types.Nil, error) {
				received = req.Keys
				return types.Nil{}, nil
			},
		)
		Expect(client.Deleter().Client().Send(ctx, leaseholder, deleter.Request{
			Keys: channel.Keys{4, 5},
		})).To(Equal(types.Nil{}))
		Expect(received).To(Equal(channel.Keys{4, 5}))
	})
})
