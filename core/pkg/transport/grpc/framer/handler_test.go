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
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	grpcframer "github.com/synnaxlabs/synnax/pkg/transport/grpc/framer"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	telempb "github.com/synnaxlabs/x/telem/pb"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type deleteServer = fgrpc.UnaryServer[
	framer.DeleteRequest,
	*grpcframer.DeleteRequest,
	struct{},
	*emptypb.Empty,
]

func createVirtualChannels(ctx context.Context, dt telem.DataType, n int) channel.Keys {
	GinkgoHelper()
	chs := make([]channel.Channel, n)
	for i := range chs {
		chs[i] = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: dt,
			Virtual:  true,
		}
	}
	Expect(channelSvc.NewWriter(nil).CreateMany(ctx, &chs)).To(Succeed())
	return channel.KeysFromChannels(chs)
}

// newClientCodec returns a codec that matches the one the server builds for keys.
func newClientCodec(ctx context.Context, keys channel.Keys) *codec.Codec {
	GinkgoHelper()
	cdc := codec.NewDynamic(apiChannelSvc)
	Expect(cdc.Update(ctx, keys)).To(Succeed())
	return cdc
}

// serve makes srv receive count requests and then send responses. It returns the
// requests it received and the error that stopped it from receiving, if any.
func serve[RQ, RS freighter.Payload](
	srv freighter.StreamServer[RQ, RS],
	count int,
	responses ...RS,
) (<-chan RQ, <-chan error) {
	var (
		requests = make(chan RQ, count)
		errs     = make(chan error, 1)
	)
	srv.BindHandler(func(_ context.Context, s freighter.ServerStream[RQ, RS]) error {
		for range count {
			req, err := s.Receive()
			if err != nil {
				errs <- err
				return nil
			}
			requests <- req
		}
		for _, res := range responses {
			if err := s.Send(res); err != nil {
				return err
			}
		}
		return nil
	})
	return requests, errs
}

// exchange sends requests on stream, closes the send direction, and returns the count
// responses the server sends before it ends the stream.
func exchange[RQ, RS any](
	stream grpc.BidiStreamingClient[RQ, RS],
	count int,
	requests ...*RQ,
) []*RS {
	GinkgoHelper()
	for _, req := range requests {
		Expect(stream.Send(req)).To(Succeed())
	}
	Expect(stream.CloseSend()).To(Succeed())
	responses := make([]*RS, count)
	for i := range responses {
		responses[i] = MustSucceed(stream.Recv())
	}
	Expect(stream.Recv()).Error().To(MatchError(io.EOF))
	return responses
}

var _ = Describe("gRPC Framer Transport", func() {
	Describe("New", func() {
		It("Should bind every framer server onto the API transport", func() {
			Expect(bindable).To(HaveLen(4))
			Expect(bindable).To(ContainElements(
				transport.FrameWriter,
				transport.FrameIterator,
				transport.FrameStreamer,
				transport.FrameDelete,
			))
		})
	})

	Describe("Writer", func() {
		var client grpcframer.FrameWriterServiceClient
		BeforeEach(func() { client = grpcframer.NewFrameWriterServiceClient(conn) })

		It("Should decode a codec buffer after an open command", func(
			ctx SpecContext,
		) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			requests, _ := serve(transport.FrameWriter, 2)
			buf := MustSucceed(newClientCodec(ctx, keys).Encode(
				ctx,
				frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			))
			exchange(
				MustSucceed(client.Exec(ctx)),
				0,
				&grpcframer.WriterRequest{
					Command: int32(framer.WriterCommandOpen),
					Config:  &grpcframer.WriterConfig{Keys: keys.Uint32()},
				},
				&grpcframer.WriterRequest{
					Command: int32(framer.WriterCommandWrite),
					Buffer:  buf,
				},
			)
			var open, write framer.WriterRequest
			Eventually(requests).Should(Receive(&open))
			Expect(open.Command).To(Equal(framer.WriterCommandOpen))
			Expect(open.Config.Keys).To(Equal(keys))
			Eventually(requests).Should(Receive(&write))
			Expect(write.Command).To(Equal(framer.WriterCommandWrite))
			Expect(channel.Keys(write.Frame.KeysSlice())).To(Equal(keys))
			Expect(
				write.Frame.SeriesAt(0),
			).To(MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should decode a protobuf frame", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			requests, _ := serve(transport.FrameWriter, 1)
			fr := frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](4, 5, 6)})
			exchange(
				MustSucceed(client.Exec(ctx)),
				0,
				&grpcframer.WriterRequest{
					Command: int32(framer.WriterCommandWrite),
					Frame:   MustSucceed(telempb.FrameToPB(fr.Frame)),
				},
			)
			var write framer.WriterRequest
			Eventually(requests).Should(Receive(&write))
			Expect(channel.Keys(write.Frame.KeysSlice())).To(Equal(keys))
			Expect(
				write.Frame.SeriesAt(0),
			).To(MatchSeriesData(telem.NewSeriesV[int32](4, 5, 6)))
		})

		It("Should fail to receive a buffer that does not decode", func(
			ctx SpecContext,
		) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			_, errs := serve(transport.FrameWriter, 2)
			exchange(
				MustSucceed(client.Exec(ctx)),
				0,
				&grpcframer.WriterRequest{
					Command: int32(framer.WriterCommandOpen),
					Config:  &grpcframer.WriterConfig{Keys: keys.Uint32()},
				},
				&grpcframer.WriterRequest{
					Command: int32(framer.WriterCommandWrite),
					Buffer:  []byte{0x01, 0x02},
				},
			)
			Eventually(errs).Should(Receive(MatchError(io.ErrUnexpectedEOF)))
		})

		It("Should fail to receive an open command for a missing channel", func(
			ctx SpecContext,
		) {
			_, errs := serve(transport.FrameWriter, 1)
			exchange(
				MustSucceed(client.Exec(ctx)),
				0,
				&grpcframer.WriterRequest{
					Command: int32(framer.WriterCommandOpen),
					Config: &grpcframer.WriterConfig{
						Keys: []uint32{uint32(distchannel.NewKey(1, 999999))},
					},
				},
			)
			Eventually(errs).Should(Receive(MatchError(query.ErrNotFound)))
		})

		It("Should send the command and end of a response", func(ctx SpecContext) {
			serve(transport.FrameWriter, 1, framer.WriterResponse{
				Command: framer.WriterCommandCommit,
				End:     telem.TimeStamp(123),
			})
			res := exchange(
				MustSucceed(client.Exec(ctx)),
				1,
				&grpcframer.WriterRequest{Command: int32(framer.WriterCommandCommit)},
			)[0]
			Expect(res.Command).To(Equal(int32(framer.WriterCommandCommit)))
			Expect(res.End).To(Equal(int64(123)))
		})
	})

	Describe("Streamer", func() {
		var client grpcframer.FrameStreamerServiceClient
		BeforeEach(func() { client = grpcframer.NewFrameStreamerServiceClient(conn) })

		It("Should encode frames with the codec once a request names keys", func(
			ctx SpecContext,
		) {
			keys := createVirtualChannels(ctx, telem.Int64T, 1)
			requests, _ := serve(transport.FrameStreamer, 1, framer.StreamerResponse{
				Frame: frame.NewMulti(
					keys,
					[]telem.Series{telem.NewSeriesV[int64](10, 20, 30)},
				),
			})
			res := exchange(
				MustSucceed(client.Exec(ctx)),
				1,
				&grpcframer.StreamerRequest{Keys: keys.Uint32()},
			)[0]
			var req framer.StreamerRequest
			Eventually(requests).Should(Receive(&req))
			Expect(req.Keys).To(Equal(keys))
			Expect(res.Frame).To(BeNil())
			fr := MustSucceed(newClientCodec(ctx, keys).Decode(res.Buffer))
			Expect(channel.Keys(fr.KeysSlice())).To(Equal(keys))
			Expect(
				fr.SeriesAt(0),
			).To(MatchSeriesData(telem.NewSeriesV[int64](10, 20, 30)))
		})

		It("Should keep the codec when a later request names no keys", func(
			ctx SpecContext,
		) {
			keys := createVirtualChannels(ctx, telem.Int64T, 1)
			serve(transport.FrameStreamer, 2, framer.StreamerResponse{
				Frame: frame.NewMulti(
					keys,
					[]telem.Series{telem.NewSeriesV[int64](1, 2, 3)},
				),
			})
			res := exchange(
				MustSucceed(client.Exec(ctx)),
				1,
				&grpcframer.StreamerRequest{Keys: keys.Uint32()},
				&grpcframer.StreamerRequest{},
			)[0]
			fr := MustSucceed(newClientCodec(ctx, keys).Decode(res.Buffer))
			Expect(
				fr.SeriesAt(0),
			).To(MatchSeriesData(telem.NewSeriesV[int64](1, 2, 3)))
		})

		It("Should send a protobuf frame before a request names keys", func(
			ctx SpecContext,
		) {
			keys := createVirtualChannels(ctx, telem.Int64T, 1)
			serve(transport.FrameStreamer, 1, framer.StreamerResponse{
				Frame: frame.NewMulti(
					keys,
					[]telem.Series{telem.NewSeriesV[int64](7, 8, 9)},
				),
			})
			res := exchange(
				MustSucceed(client.Exec(ctx)),
				1,
				&grpcframer.StreamerRequest{},
			)[0]
			Expect(res.Buffer).To(BeEmpty())
			fr := MustSucceed(telempb.FrameFromPB[channel.Key](res.Frame))
			Expect(fr.KeysSlice()).To(Equal([]channel.Key(keys)))
			Expect(
				fr.SeriesAt(0),
			).To(MatchSeriesData(telem.NewSeriesV[int64](7, 8, 9)))
		})

		It("Should receive a request for an ack", func(ctx SpecContext) {
			requests, _ := serve(transport.FrameStreamer, 1)
			exchange(
				MustSucceed(client.Exec(ctx)),
				0,
				&grpcframer.StreamerRequest{AckRequested: true},
			)
			var req framer.StreamerRequest
			Eventually(requests).Should(Receive(&req))
			Expect(req.AckRequested).To(BeTrue())
		})

		DescribeTable(
			"Should send an ack",
			func(ctx SpecContext, keyed bool) {
				var req grpcframer.StreamerRequest
				if keyed {
					req.Keys = createVirtualChannels(ctx, telem.Int64T, 1).Uint32()
				}
				serve(transport.FrameStreamer, 1, framer.StreamerResponse{Ack: true})
				res := exchange(MustSucceed(client.Exec(ctx)), 1, &req)[0]
				Expect(res.Ack).To(BeTrue())
			},
			Entry("through the protobuf frame", false),
			Entry("through the codec", true),
		)
	})

	Describe("Iterator", func() {
		var client grpcframer.FrameIteratorServiceClient
		BeforeEach(func() { client = grpcframer.NewFrameIteratorServiceClient(conn) })

		openRequest := func(keys channel.Keys) *grpcframer.IteratorRequest {
			GinkgoHelper()
			return &grpcframer.IteratorRequest{
				Command:   int32(framer.IteratorCommandSeekFirst),
				Keys:      keys.Uint32(),
				ChunkSize: 100,
				Range:     MustSucceed(telempb.TimeRangeToPB(telem.TimeRangeMax)),
			}
		}

		It("Should encode data with the codec after a request names keys", func(
			ctx SpecContext,
		) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			requests, _ := serve(transport.FrameIterator, 1, framer.IteratorResponse{
				Variant: framer.IteratorResponseVariantData,
				Command: framer.IteratorCommandNext,
				Frame: frame.NewMulti(
					keys,
					[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
				),
			})
			res := exchange(MustSucceed(client.Exec(ctx)), 1, openRequest(keys))[0]
			var req framer.IteratorRequest
			Eventually(requests).Should(Receive(&req))
			Expect(req.Command).To(Equal(framer.IteratorCommandSeekFirst))
			Expect(req.Keys).To(Equal(keys))
			Expect(req.ChunkSize).To(Equal(int64(100)))
			Expect(req.Bounds).To(Equal(telem.TimeRangeMax))
			Expect(res.Variant).To(Equal(int32(framer.IteratorResponseVariantData)))
			Expect(res.Frame).To(BeNil())
			fr := MustSucceed(newClientCodec(ctx, keys).Decode(res.Buffer))
			Expect(channel.Keys(fr.KeysSlice())).To(Equal(keys))
			Expect(
				fr.SeriesAt(0),
			).To(MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should send an ack without a buffer", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			serve(transport.FrameIterator, 1, framer.IteratorResponse{
				Variant: framer.IteratorResponseVariantAck,
				Command: framer.IteratorCommandNext,
				Ack:     true,
				SeqNum:  7,
			})
			res := exchange(MustSucceed(client.Exec(ctx)), 1, openRequest(keys))[0]
			Expect(res.Buffer).To(BeEmpty())
			Expect(res.Variant).To(Equal(int32(framer.IteratorResponseVariantAck)))
			Expect(res.Ack).To(BeTrue())
			Expect(res.SeqNum).To(Equal(int32(7)))
		})

		It("Should send empty data without a buffer", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			serve(transport.FrameIterator, 1, framer.IteratorResponse{
				Variant: framer.IteratorResponseVariantData,
				Command: framer.IteratorCommandNext,
			})
			res := exchange(MustSucceed(client.Exec(ctx)), 1, openRequest(keys))[0]
			Expect(res.Buffer).To(BeEmpty())
			Expect(MustSucceed(telempb.FrameFromPB[channel.Key](res.Frame)).Empty()).
				To(BeTrue())
		})

		It("Should send a protobuf frame when no request names keys", func(
			ctx SpecContext,
		) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			requests, _ := serve(transport.FrameIterator, 1, framer.IteratorResponse{
				Variant: framer.IteratorResponseVariantData,
				Command: framer.IteratorCommandNext,
				Frame: frame.NewMulti(
					keys,
					[]telem.Series{telem.NewSeriesV[int32](4, 5, 6)},
				),
			})
			res := exchange(
				MustSucceed(client.Exec(ctx)),
				1,
				&grpcframer.IteratorRequest{
					Command: int32(framer.IteratorCommandNext),
					Span:    int64(telem.Second),
				},
			)[0]
			var req framer.IteratorRequest
			Eventually(requests).Should(Receive(&req))
			Expect(req.Command).To(Equal(framer.IteratorCommandNext))
			Expect(req.Span).To(Equal(telem.Second))
			Expect(res.Buffer).To(BeEmpty())
			fr := MustSucceed(telempb.FrameFromPB[channel.Key](res.Frame))
			Expect(fr.KeysSlice()).To(Equal([]channel.Key(keys)))
			Expect(
				fr.SeriesAt(0),
			).To(MatchSeriesData(telem.NewSeriesV[int32](4, 5, 6)))
		})
	})

	Describe("Delete", func() {
		It("Should translate the keys, names, and bounds", func(ctx SpecContext) {
			t := transport.FrameDelete.(*deleteServer).RequestTranslator
			pb := &grpcframer.DeleteRequest{
				Keys:   []uint32{1, 2},
				Names:  []string{"a", "b"},
				Bounds: &telempb.TimeRange{Start: 1, End: 5},
			}
			Expect(MustSucceed(t.Backward(ctx, pb))).To(Equal(framer.DeleteRequest{
				Keys:   channel.Keys{1, 2},
				Names:  []string{"a", "b"},
				Bounds: telem.TimeRange{Start: 1, End: 5},
			}))
		})
	})
})
