// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apifra "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createVirtualChannels(ctx context.Context, dt telem.DataType, n int) channel.Keys {
	chs := make([]channel.Channel, n)
	for i := range chs {
		chs[i] = channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: dt,
			Virtual:  true,
		}
	}
	Expect(dist.Channel.CreateMany(ctx, &chs)).To(Succeed())
	return channel.KeysFromChannels(chs)
}

var _ = Describe("GRPC Framer Translators", func() {
	Describe("Frame Writer Translators", func() {
		It("Should round-trip a write request via the codec buffer", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			cdec := codec.NewDynamic(dist.Channel)
			Expect(cdec.Update(ctx, keys)).To(Succeed())

			t := frameWriterRequestTranslator{codec: cdec}
			req := apifra.WriterRequest{
				Command: writer.CommandWrite,
				Frame:   frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			}
			pb := MustSucceed(t.Forward(ctx, req))
			Expect(pb.Buffer).ToNot(BeEmpty())

			out := MustSucceed(t.Backward(ctx, pb))
			Expect(channel.Keys(out.Frame.KeysSlice())).To(Equal(keys))
			Expect(out.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should round-trip a writer response", func(ctx SpecContext) {
			t := frameWriterResponseTranslator{}
			res := apifra.WriterResponse{
				Command: writer.CommandWrite,
				End:     telem.TimeStamp(123),
			}
			pb := MustSucceed(t.Forward(ctx, res))
			out := MustSucceed(t.Backward(ctx, pb))
			Expect(out.Command).To(Equal(writer.CommandWrite))
			Expect(out.End).To(Equal(telem.TimeStamp(123)))
		})
	})

	Describe("Frame Streamer Translators", func() {
		It("Should update the codec on a request with keys", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int64T, 1)
			cdec := codec.NewDynamic(dist.Channel)
			rt := frameStreamerRequestTranslator{codec: cdec}
			pbReq := MustSucceed(rt.Forward(ctx, apifra.StreamerRequest{Keys: keys}))
			Expect(pbReq.Keys).To(Equal(keys.Uint32()))
			MustSucceed(rt.Backward(ctx, pbReq))
			Expect(cdec.Initialized()).To(BeTrue())
		})

		It("Should encode a streamer response into the buffer when the codec is initialized", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int64T, 1)
			cdec := codec.NewDynamic(dist.Channel)
			Expect(cdec.Update(ctx, keys)).To(Succeed())

			st := frameStreamerResponseTranslator{codec: cdec}
			res := apifra.StreamerResponse{
				Frame: frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int64](10, 20, 30)}),
			}
			pbRes := MustSucceed(st.Forward(ctx, res))
			Expect(pbRes.Buffer).ToNot(BeEmpty())
			Expect(pbRes.Frame).To(BeNil())

			fr := MustSucceed(cdec.Decode(pbRes.Buffer))
			Expect(channel.Keys(fr.KeysSlice())).To(Equal(keys))
			Expect(fr.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int64](10, 20, 30)))
		})
	})

	Describe("Frame Iterator Request Translator", func() {
		It("Should round-trip an open request and update the codec", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			cdec := codec.NewDynamic(dist.Channel)
			t := frameIteratorRequestTranslator{codec: cdec}
			req := apifra.IteratorRequest{
				Command:   iterator.CommandSeekFirst,
				Keys:      keys,
				ChunkSize: 100,
				Bounds:    telem.TimeRangeMax,
			}
			pb := MustSucceed(t.Forward(ctx, req))
			Expect(pb.Keys).To(Equal(keys.Uint32()))

			out := MustSucceed(t.Backward(ctx, pb))
			Expect(channel.Keys(out.Keys)).To(Equal(keys))
			Expect(out.Command).To(Equal(iterator.CommandSeekFirst))
			Expect(cdec.Initialized()).To(BeTrue())
		})

		It("Should not call Update when the request has no keys", func(ctx SpecContext) {
			cdec := codec.NewDynamic(dist.Channel)
			t := frameIteratorRequestTranslator{codec: cdec}
			pb := MustSucceed(t.Forward(ctx, apifra.IteratorRequest{
				Command: iterator.CommandNext,
				Span:    telem.Second,
			}))
			out := MustSucceed(t.Backward(ctx, pb))
			Expect(out.Command).To(Equal(iterator.CommandNext))
			Expect(out.Span).To(Equal(telem.Second))
			Expect(cdec.Initialized()).To(BeFalse())
		})
	})

	Describe("Frame Iterator Response Translator", func() {
		It("Should encode a data variant frame to the buffer when the codec is initialized", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			cdec := codec.NewDynamic(dist.Channel)
			Expect(cdec.Update(ctx, keys)).To(Succeed())

			t := frameIteratorResponseTranslator{codec: cdec}
			res := apifra.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Command: iterator.CommandNext,
				Frame:   frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			}
			pb := MustSucceed(t.Forward(ctx, res))
			Expect(pb.Buffer).ToNot(BeEmpty())
			Expect(pb.Frame).To(BeNil())

			out := MustSucceed(t.Backward(ctx, pb))
			Expect(out.Variant).To(Equal(iterator.ResponseVariantData))
			Expect(channel.Keys(out.Frame.KeysSlice())).To(Equal(keys))
			Expect(out.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should fall back to the protobuf frame for an ack variant", func(ctx SpecContext) {
			cdec := codec.NewDynamic(dist.Channel)
			Expect(cdec.Update(ctx, channel.Keys{})).To(Succeed())
			t := frameIteratorResponseTranslator{codec: cdec}
			res := apifra.IteratorResponse{
				Variant: iterator.ResponseVariantAck,
				Command: iterator.CommandNext,
				Ack:     true,
				SeqNum:  7,
			}
			pb := MustSucceed(t.Forward(ctx, res))
			Expect(pb.Buffer).To(BeEmpty())

			out := MustSucceed(t.Backward(ctx, pb))
			Expect(out.Variant).To(Equal(iterator.ResponseVariantAck))
			Expect(out.Ack).To(BeTrue())
			Expect(out.SeqNum).To(Equal(7))
		})

		It("Should fall back to the protobuf frame for an empty data response", func(ctx SpecContext) {
			keys := createVirtualChannels(ctx, telem.Int32T, 1)
			cdec := codec.NewDynamic(dist.Channel)
			Expect(cdec.Update(ctx, keys)).To(Succeed())

			t := frameIteratorResponseTranslator{codec: cdec}
			res := apifra.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Command: iterator.CommandNext,
				Frame:   frame.Frame{},
			}
			pb := MustSucceed(t.Forward(ctx, res))
			Expect(pb.Buffer).To(BeEmpty())

			out := MustSucceed(t.Backward(ctx, pb))
			Expect(out.Frame.Empty()).To(BeTrue())
		})
	})
})
