// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freightfluence_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Receiver", func() {
	var (
		server freighter.StreamServer[int, int]
		client freighter.StreamClient[int, int]
	)
	BeforeEach(func() {
		server, client = fmock.NewStreamPair[int, int]()
	})
	Describe("Receiver", func() {
		It("Should operate correctly", func() {
			var receivedValues []int
			receiverStream := confluence.NewStream[int](10)
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				receiver := &freightfluence.Receiver[int]{Receiver: server}
				receiver.OutTo(receiverStream)
				receiver.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				By("Receiving values from the input server")
				receivedValues = append(receivedValues, <-receiverStream.Outlet())
				return sCtx.Wait()
			})
			stream, err := client.Stream(context.TODO(), "localhost:0")
			Expect(err).ToNot(HaveOccurred())
			Expect(stream.Send(1)).To(Succeed())
			Expect(stream.CloseSend()).To(Succeed())
			By("Closing the network pipe on return")
			_, err = stream.Receive()
			Expect(err).To(Equal(freighter.EOF))
			Expect(receivedValues).To(Equal([]int{1}))
			By("Closing the receive server on exit")
			_, ok := <-receiverStream.Outlet()
			Expect(ok).To(BeFalse())
		})
		It("Should exit the receiver on context cancellation", func() {
			receiverStream := confluence.NewStream[int](10)
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				receiver := &freightfluence.Receiver[int]{Receiver: server}
				receiver.OutTo(receiverStream)
				receiver.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				By("Receiving values from the input server")
				return sCtx.Wait()
			})
			ctx, cancel := context.WithCancel(context.TODO())
			stream, err := client.Stream(ctx, "localhost:0")
			Expect(err).ToNot(HaveOccurred())
			Expect(stream.Send(1)).To(Succeed())
			By("Closing the network pipe")
			v := <-receiverStream.Outlet()
			Expect(v).To(Equal(1))
			cancel()
			_, err = stream.Receive()
			Expect(err).To(HaveOccurredAs(context.Canceled))
			By("Closing the receive server on exit")
			_, ok := <-receiverStream.Outlet()
			Expect(ok).To(BeFalse())
		})
	})
	Describe("TransformReceiver", func() {
		It("It should transform values before sending them through the channel", func() {
			var receivedValues []int
			receiverStream := confluence.NewStream[int](10)
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				receiver := &freightfluence.TransformReceiver[int, int]{}
				receiver.Receiver = server
				receiver.OutTo(receiverStream)
				receiver.Transform = func(ctx context.Context, v int) (int, bool, error) {
					return v * 2, true, nil
				}
				receiver.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				By("Receiving values from the input server")
				receivedValues = append(receivedValues, <-receiverStream.Outlet())
				return sCtx.Wait()
			})
			stream, err := client.Stream(context.TODO(), "localhost:0")
			Expect(err).ToNot(HaveOccurred())
			Expect(stream.Send(1)).To(Succeed())
			Expect(stream.CloseSend()).To(Succeed())
			By("Closing the network pipe on return")
			_, err = stream.Receive()
			Expect(err).To(Equal(freighter.EOF))
			Expect(receivedValues).To(Equal([]int{2}))
			By("Closing the receive server on exit")
			_, ok := <-receiverStream.Outlet()
			Expect(ok).To(BeFalse())
		})
	})
})
