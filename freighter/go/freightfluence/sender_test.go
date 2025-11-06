// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

func clientStreamsToSlice(
	clients map[address.Address]freighter.StreamSenderCloser[int],
) []freighter.StreamSenderCloser[int] {
	slice := make([]freighter.StreamSenderCloser[int], 0, len(clients))
	for _, client := range clients {
		slice = append(slice, client)
	}
	return slice
}

var _ = Describe("Sender", func() {
	var net *fmock.Network[int, int]
	BeforeEach(func() {
		net = fmock.NewNetwork[int, int]()
	})
	Context("Single Stream", func() {
		var (
			server         freighter.StreamServer[int, int]
			client         freighter.StreamClient[int, int]
			receiverStream *confluence.Stream[int]
			senderStream   *confluence.Stream[int]
		)
		BeforeEach(func() {
			server = net.StreamServer("", 10)
			client = net.StreamClient()
			receiverStream = confluence.NewStream[int](0)
			senderStream = confluence.NewStream[int](0)
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				receiver := &freightfluence.Receiver[int]{}
				receiver.Receiver = server
				receiver.OutTo(receiverStream)
				receiver.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				return sCtx.Wait()
			})
		})
		Describe("Sender", func() {
			It("Should operate correctly", func() {
				sCtx, cancel := signal.WithCancel(context.TODO())
				client, err := client.Stream(sCtx, "localhost:0")
				Expect(err).ToNot(HaveOccurred())
				sender := &freightfluence.Sender[int]{Sender: client}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 1
				v := <-receiverStream.Outlet()
				Expect(v).To(Equal(1))
				senderStream.Inlet() <- 2
				v = <-receiverStream.Outlet()
				Expect(v).To(Equal(2))
				cancel()
				Expect(sCtx.Wait()).To(HaveOccurredAs(context.Canceled))
				_, ok := <-receiverStream.Outlet()
				Expect(ok).To(BeFalse())
			})
		})
		Describe("TransformSender", func() {
			It("Should transform values before sending them", func() {
				sCtx, cancel := signal.WithCancel(context.TODO())
				client, err := client.Stream(sCtx, "localhost:0")
				Expect(err).ToNot(HaveOccurred())
				sender := &freightfluence.TransformSender[int, int]{}
				sender.Sender = client
				sender.Transform = func(ctx context.Context, v int) (int, bool, error) {
					return v * 2, true, nil
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 1
				v := <-receiverStream.Outlet()
				Expect(v).To(Equal(2))
				cancel()
				Expect(sCtx.Wait()).To(HaveOccurredAs(context.Canceled))
				_, ok := <-receiverStream.Outlet()
				Expect(ok).To(BeFalse())
			})
			It("Should exit when the transform returns an error", func() {
				sCtx, cancel := signal.WithCancel(context.TODO())
				defer cancel()
				client, err := client.Stream(sCtx, "localhost:0")
				Expect(err).ToNot(HaveOccurred())
				sender := &freightfluence.TransformSender[int, int]{}
				sender.Sender = client
				sender.Transform = func(ctx context.Context, v int) (int, bool, error) {
					return v * 2, true, errors.New("error")
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 1
				Expect(sCtx.Wait()).To(MatchError("error"))
			})
		})
	})
	Context("Multiple Streams", func() {
		var (
			sCtx            signal.Context
			cancel          context.CancelFunc
			nStreams        = 5
			senderStream    *confluence.Stream[int]
			receiverStreams map[address.Address]*confluence.Stream[int]
			clientSender    freightfluence.MapTargetedSender[int]
		)
		BeforeEach(func() {
			sCtx, cancel = signal.WithCancel(context.TODO())
			senderStream = confluence.NewStream[int](nStreams)
			clientTransport := net.StreamClient(1)
			clientSender = make(map[address.Address]freighter.StreamSenderCloser[int], nStreams)
			receiverStreams = make(map[address.Address]*confluence.Stream[int], nStreams)
			for range nStreams {
				stream := net.StreamServer("", 1)
				receiverStream := confluence.NewStream[int](1)
				stream.BindHandler(func(ctx context.Context, serverStream freighter.ServerStream[int, int]) error {
					serverCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					receiver := &freightfluence.Receiver[int]{Receiver: serverStream}
					receiver.OutTo(receiverStream)
					receiver.Flow(serverCtx, confluence.CloseOutputInletsOnExit())
					return serverCtx.Wait()
				})
				clientStream, err := clientTransport.Stream(sCtx, stream.Address)
				Expect(err).ToNot(HaveOccurred())
				clientSender[stream.Address] = clientStream
				receiverStreams[stream.Address] = receiverStream
			}
		})
		AfterEach(func() { cancel() })
		Describe("BatchSwitchSender", func() {
			It("Should route values to the correct stream", func() {
				sender := &freightfluence.BatchSwitchSender[int, int]{}
				sender.Senders = clientSender
				sender.Switch = func(ctx context.Context, v int, o map[address.Address]int) error {
					addr := address.Newf("localhost:%v", v)
					o[addr] = v
					addr2 := address.Newf("localhost:%v", v+1)
					o[addr2] = v + 1
					return nil
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 2
				Expect(<-receiverStreams["localhost:2"].Outlet()).To(Equal(2))
				Expect(<-receiverStreams["localhost:3"].Outlet()).To(Equal(3))
				senderStream.Close()
				Expect(sCtx.Wait()).To(Succeed())
			})
		})
		It("Should exit when the context is canceled", func() {
			sender := &freightfluence.BatchSwitchSender[int, int]{}
			sender.Senders = clientSender
			sender.Switch = func(ctx context.Context, v int, o map[address.Address]int) error {
				addr := address.Newf("localhost:%v", v)
				o[addr] = v
				return nil
			}
			sender.InFrom(senderStream)
			sender.Flow(sCtx)
			senderStream.Inlet() <- 2
			cancel()
			Expect(sCtx.Wait()).To(MatchError(context.Canceled))
		})
		It("Should exit when the switch returns an error", func() {
			sender := &freightfluence.BatchSwitchSender[int, int]{}
			sender.Senders = clientSender
			sender.Switch = func(ctx context.Context, v int, o map[address.Address]int) error {
				return errors.New("error")
			}
			sender.InFrom(senderStream)
			sender.Flow(sCtx)
			senderStream.Inlet() <- 2
			Expect(sCtx.Wait()).To(MatchError("error"))
		})
	})
})
