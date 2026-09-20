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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Stream", Ordered, Serial, func() {
	var (
		server *mock.StreamServer[test.Request, test.Response]
		client *mock.StreamClient[test.Request, test.Response]
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		server, client = mock.NewStreamPair[test.Request, test.Response](11, 11)
	})

	test.StreamSuite(func() (
		freighter.StreamServer[test.Request, test.Response],
		freighter.StreamClient[test.Request, test.Response],
		address.Address,
	) {
		return server, client, "localhost:0"
	})

	It("Should release the handler when the client stops receiving", func(
		ctx SpecContext,
	) {
		ShouldNotLeakGoroutines()
		abandoned, dialer := mock.NewStreamPair[test.Request, test.Response](1, 1)
		returned := make(chan struct{})
		abandoned.BindHandler(func(
			_ context.Context,
			stream freighter.ServerStream[test.Request, test.Response],
		) error {
			defer close(returned)
			if err := stream.Send(test.Response{ID: 1}); err != nil {
				return err
			}
			return stream.Send(test.Response{ID: 2})
		})
		streamCtx, cancel := context.WithCancel(ctx)
		stream := MustSucceed(dialer.Stream(streamCtx, "localhost:0"))
		Expect(stream.Receive()).To(Equal(test.Response{ID: 1}))
		// The client leaves the second response in the buffer, so the closing error the
		// handler emits on return has nowhere to go until the client cancels.
		Eventually(returned).Should(BeClosed())
		cancel()
	})
})

// Go picks at random between the ready cases of a select, so a cancelled context and
// a ready channel case each win about half the time. Repeat the race so a regression
// cannot pass by luck.
const cancelRaceAttempts = 50

var _ = Describe("Context cancellation", func() {
	// openBlocked dials a stream whose handler parks until the stream context is
	// cancelled, so the spec drives every transition itself.
	openBlocked := func(
		ctx SpecContext,
		buffers ...int,
	) (freighter.ClientStream[test.Request, test.Response], context.CancelFunc) {
		GinkgoHelper()
		server, client := mock.NewStreamPair[test.Request, test.Response](buffers...)
		server.BindHandler(func(
			handlerCtx context.Context,
			_ freighter.ServerStream[test.Request, test.Response],
		) error {
			<-handlerCtx.Done()
			return nil
		})
		streamCtx, cancel := context.WithCancel(ctx)
		return MustSucceed(client.Stream(streamCtx, "localhost:0")), cancel
	}

	It(
		"Should return the context error from Receive instead of a buffered response",
		func(
			ctx SpecContext,
		) {
			receiveAfterCancel := func() {
				GinkgoHelper()
				server, client := mock.NewStreamPair[test.Request, test.Response](1, 1)
				buffered := make(chan struct{})
				server.BindHandler(func(
					handlerCtx context.Context,
					stream freighter.ServerStream[test.Request, test.Response],
				) error {
					defer GinkgoRecover()
					Expect(stream.Send(test.Response{ID: 1})).To(Succeed())
					close(buffered)
					<-handlerCtx.Done()
					return nil
				})
				streamCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				stream := MustSucceed(client.Stream(streamCtx, "localhost:0"))
				// Send returns once the response sits in the buffer, so the buffered
				// response and the cancellation are both ready when Receive runs.
				Eventually(buffered).Should(BeClosed())
				cancel()
				Expect(stream.Receive()).Error().To(MatchError(context.Canceled))
			}
			for range cancelRaceAttempts {
				receiveAfterCancel()
			}
		},
	)

	It(
		"Should return the context error from Send instead of buffering the request",
		func(
			ctx SpecContext,
		) {
			for range cancelRaceAttempts {
				stream, cancel := openBlocked(ctx, 1, 1)
				cancel()
				Expect(
					stream.Send(test.Request{ID: 1}),
				).To(MatchError(context.Canceled))
			}
		},
	)

	It("Should return from CloseSend when the request buffer is full", func(
		ctx SpecContext,
	) {
		stream, cancel := openBlocked(ctx, 1, 1)
		Expect(stream.Send(test.Request{ID: 1})).To(Succeed())
		cancel()
		closed := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(closed)
			Expect(stream.CloseSend()).To(Succeed())
		}()
		Eventually(closed).Should(BeClosed())
	})
})

var _ = Describe("Address", func() {
	It("Should return the address the network assigned to the server", func() {
		net := mock.NewNetwork[test.Request, test.Response]()
		Expect(net.StreamServer("localhost:1234").Address()).
			To(Equal(address.Address("localhost:1234")))
		Expect(net.StreamServer("").Address()).ToNot(BeEmpty())
	})

	It("Should be empty for a server that is not on a network", func() {
		server, _ := mock.NewStreamPair[test.Request, test.Response]()
		Expect(server.Address()).To(BeEmpty())
	})
})
