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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Transport", func() {
	Describe("Writer", func() {
		// The WriterRequest proto carries no SeqNum field (it is assigned server-side),
		// so we round-trip the Command on the request and assert the SeqNum the server
		// sets on the response.
		It("Should round-trip a request over the wire", func(ctx SpecContext) {
			transport.Writer().Server().BindHandler(func(
				ctx context.Context,
				srv freighter.ServerStream[writer.Request, writer.Response],
			) error {
				req, err := srv.Receive()
				if err != nil {
					return err
				}
				return srv.Send(writer.Response{SeqNum: 42, Command: req.Command})
			})
			stream := MustSucceed(transport.Writer().Client().Stream(ctx, addr))
			Expect(stream.Send(writer.Request{Command: writer.CommandWrite})).To(Succeed())
			res := MustSucceed(stream.Receive())
			Expect(res.SeqNum).To(Equal(42))
			Expect(res.Command).To(Equal(writer.CommandWrite))
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	Describe("Iterator", func() {
		It("Should round-trip a request over the wire", func(ctx SpecContext) {
			transport.Iterator().Server().BindHandler(func(
				ctx context.Context,
				srv freighter.ServerStream[iterator.Request, iterator.Response],
			) error {
				req, err := srv.Receive()
				if err != nil {
					return err
				}
				return srv.Send(iterator.Response{SeqNum: req.SeqNum})
			})
			stream := MustSucceed(transport.Iterator().Client().Stream(ctx, addr))
			Expect(stream.Send(iterator.Request{SeqNum: 42})).To(Succeed())
			Expect(MustSucceed(stream.Receive()).SeqNum).To(Equal(42))
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	Describe("Relay", func() {
		It("Should round-trip a request over the wire", func(ctx SpecContext) {
			transport.Relay().Server().BindHandler(func(
				ctx context.Context,
				srv freighter.ServerStream[relay.Request, relay.Response],
			) error {
				if _, err := srv.Receive(); err != nil {
					return err
				}
				return srv.Send(relay.Response{})
			})
			stream := MustSucceed(transport.Relay().Client().Stream(ctx, addr))
			Expect(stream.Send(relay.Request{})).To(Succeed())
			MustSucceed(stream.Receive())
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	Describe("Wiring", func() {
		It("Should wire all stream transport accessors", func() {
			Expect(transport.Writer().Client()).ToNot(BeNil())
			Expect(transport.Writer().Server()).ToNot(BeNil())
			Expect(transport.Iterator().Client()).ToNot(BeNil())
			Expect(transport.Iterator().Server()).ToNot(BeNil())
			Expect(transport.Relay().Client()).ToNot(BeNil())
			Expect(transport.Relay().Server()).ToNot(BeNil())
		})

		// BindTo does not register the deleter gRPC service, so the deleter cannot be
		// round-tripped over the wire here. We only assert it is wired structurally.
		It("Should wire the deleter unary transport", func() {
			Expect(transport.Deleter().Client()).ToNot(BeNil())
			Expect(transport.Deleter().Server()).ToNot(BeNil())
		})
	})
})
