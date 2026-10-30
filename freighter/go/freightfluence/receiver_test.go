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
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

type errReceiver struct {
	recvErr error
}

func (e *errReceiver) Receive() (int, error) { return 0, e.recvErr }

// blockingReceiver closes entered when Receive is called, then blocks until release
// closes.
type blockingReceiver struct {
	entered chan struct{}
	release chan struct{}
}

func (b *blockingReceiver) Receive() (int, error) {
	close(b.entered)
	<-b.release
	return 0, freighter.EOF
}

var _ = Describe("Receiver", func() {
	var (
		server freighter.StreamServer[int, int]
		client freighter.StreamClient[int, int]
	)
	BeforeEach(func() {
		server, client = testutil.NewStreamPair[int, int]()
	})
	Describe("Receiver", func() {
		It("Should operate correctly", func(ctx SpecContext) {
			var receivedValues []int
			receiverStream := confluence.NewStream[int](10)
			server.BindHandler(
				func(ctx context.Context, server freighter.ServerStream[int, int]) error {
					sCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					receiver := &freightfluence.Receiver[int]{Receiver: server}
					receiver.OutTo(receiverStream)
					receiver.Flow(sCtx, confluence.CloseOutputInletsOnExit())
					By("Receiving values from the input server")
					receivedValues = append(receivedValues, <-receiverStream.Outlet())
					return sCtx.Wait()
				},
			)
			stream := MustSucceed(client.Stream(ctx, "localhost:0"))
			Expect(stream.Send(1)).To(Succeed())
			Expect(stream.CloseSend()).To(Succeed())
			By("Closing the network pipe on return")
			Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
			Expect(receivedValues).To(Equal([]int{1}))
			By("Closing the receive server on exit")
			Eventually(receiverStream.Outlet()).Should(BeClosed())
		})
		It("Should exit the receiver on context cancellation", func(ctx SpecContext) {
			receiverStream := confluence.NewStream[int](10)
			server.BindHandler(
				func(ctx context.Context, server freighter.ServerStream[int, int]) error {
					sCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					receiver := &freightfluence.Receiver[int]{Receiver: server}
					receiver.OutTo(receiverStream)
					receiver.Flow(sCtx, confluence.CloseOutputInletsOnExit())
					By("Receiving values from the input server")
					return sCtx.Wait()
				},
			)
			cancelCtx, cancel := context.WithCancel(ctx)
			stream := MustSucceed(client.Stream(cancelCtx, "localhost:0"))
			Expect(stream.Send(1)).To(Succeed())
			By("Closing the network pipe")
			v := <-receiverStream.Outlet()
			Expect(v).To(Equal(1))
			cancel()
			Expect(stream.Receive()).Error().To(MatchError(context.Canceled))
			By("Closing the receive server on exit")
			Eventually(receiverStream.Outlet()).Should(BeClosed())
		})
	})
	Describe("Stream Closure", func() {
		It(
			"Should not treat ErrStreamClosed as a routine failure",
			func(ctx SpecContext) {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				mockReceiver := &errReceiver{recvErr: freighter.ErrStreamClosed}
				receiver := &freightfluence.Receiver[int]{Receiver: mockReceiver}
				outputStream := confluence.NewStream[int](1)
				receiver.OutTo(outputStream)
				receiver.Flow(
					sCtx,
					confluence.CloseOutputInletsOnExit(),
					confluence.CancelOnFail(),
				)
				Expect(sCtx.Wait()).To(MatchError(context.Canceled))
				Eventually(outputStream.Outlet()).Should(BeClosed())
			},
		)
		It(
			"Should not treat TransformReceiver ErrStreamClosed as a routine failure",
			func(ctx SpecContext) {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				mockReceiver := &errReceiver{recvErr: freighter.ErrStreamClosed}
				receiver := &freightfluence.TransformReceiver[int, int]{
					Receiver:  mockReceiver,
					Transform: func(_ context.Context, v int) (int, bool, error) { return v, true, nil },
				}
				outputStream := confluence.NewStream[int](1)
				receiver.OutTo(outputStream)
				receiver.Flow(
					sCtx,
					confluence.CloseOutputInletsOnExit(),
					confluence.CancelOnFail(),
				)
				Expect(sCtx.Wait()).To(MatchError(context.Canceled))
				Eventually(outputStream.Outlet()).Should(BeClosed())
			},
		)
	})
	DescribeTable(
		"Should exit when its context is canceled while Receive blocks",
		func(
			ctx SpecContext,
			newSource func(freighter.StreamReceiver[int]) confluence.Source[int],
		) {
			stream := &blockingReceiver{
				entered: make(chan struct{}),
				release: make(chan struct{}),
			}
			DeferCleanup(func() { close(stream.release) })
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			source := newSource(stream)
			outputStream := confluence.NewStream[int](1)
			source.OutTo(outputStream)
			source.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(stream.entered).Should(BeClosed())
			cancel()
			waited := make(chan error, 1)
			go func() { waited <- sCtx.Wait() }()
			Eventually(waited).Should(Receive(MatchError(context.Canceled)))
			Eventually(outputStream.Outlet()).Should(BeClosed())
		},
		Entry("Receiver", func(r freighter.StreamReceiver[int]) confluence.Source[int] {
			return &freightfluence.Receiver[int]{Receiver: r}
		}),
		Entry(
			"TransformReceiver",
			func(r freighter.StreamReceiver[int]) confluence.Source[int] {
				return &freightfluence.TransformReceiver[int, int]{
					Receiver: r,
					Transform: func(_ context.Context, v int) (int, bool, error) {
						return v, true, nil
					},
				}
			},
		),
	)
	Describe("TransformReceiver", func() {
		It(
			"It should transform values before sending them through the channel",
			func(ctx SpecContext) {
				var receivedValues []int
				receiverStream := confluence.NewStream[int](10)
				server.BindHandler(
					func(ctx context.Context, server freighter.ServerStream[int, int]) error {
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
						receivedValues = append(
							receivedValues,
							<-receiverStream.Outlet(),
						)
						return sCtx.Wait()
					},
				)
				stream := MustSucceed(client.Stream(ctx, "localhost:0"))
				Expect(stream.Send(1)).To(Succeed())
				Expect(stream.CloseSend()).To(Succeed())
				By("Closing the network pipe on return")
				Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
				Expect(receivedValues).To(Equal([]int{2}))
				By("Closing the receive server on exit")
				Eventually(receiverStream.Outlet()).Should(BeClosed())
			},
		)
	})
})
