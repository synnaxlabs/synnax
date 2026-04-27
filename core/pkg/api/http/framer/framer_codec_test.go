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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	httpframer "github.com/synnaxlabs/synnax/pkg/api/http/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FramerCodec", func() {
	Describe("Frame Write Request", func() {
		It("Should encode and decode single channel int32", func(ctx SpecContext) {
			keys := channel.Keys{1}
			v := httpframer.Codec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			req := framer.WriterRequest{
				Command: writer.CommandWrite,
				Frame:   frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			}
			msg := fhttp.WSMessage[framer.WriterRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[framer.WriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Command).To(Equal(writer.CommandWrite))
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1}))
			Expect(resMsg.Payload.Frame.Count()).To(Equal(1))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should encode and decode multiple channels", func(ctx SpecContext) {
			keys := channel.Keys{1, 2, 3}
			v := httpframer.Codec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32", "float32", "uint64"}),
				LowerPerfCodec: json.Codec,
			}
			req := framer.WriterRequest{
				Command: writer.CommandWrite,
				Frame: frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[float32](1.1, 2.2),
					telem.NewSeriesV[uint64](100, 200),
				}),
			}
			msg := fhttp.WSMessage[framer.WriterRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[framer.WriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1, 2, 3}))
			Expect(resMsg.Payload.Frame.Count()).To(Equal(3))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2)))
			Expect(resMsg.Payload.Frame.SeriesAt(1)).To(telem.MatchSeriesData(telem.NewSeriesV[float32](1.1, 2.2)))
			Expect(resMsg.Payload.Frame.SeriesAt(2)).To(telem.MatchSeriesData(telem.NewSeriesV[uint64](100, 200)))
		})

		It("Should encode and decode open command", func(ctx SpecContext) {
			channels := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			Expect(dist.Channel.CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			cdec := codec.NewDynamic(dist.Channel)
			v := httpframer.Codec{
				Codec:          cdec,
				LowerPerfCodec: json.Codec,
			}
			req := framer.WriterRequest{Command: writer.CommandOpen, Config: framer.WriterConfig{Keys: keys}}
			msg := fhttp.WSMessage[framer.WriterRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.WriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(cdec.Initialized()).To(BeTrue())
			Expect(resMsg.Payload.Command).To(Equal(writer.CommandOpen))
			Expect(resMsg.Payload.Config.Keys).To(Equal(keys))
		})

		It("Should encode and decode open message", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			msg := fhttp.WSMessage[framer.WriterRequest]{Type: fhttp.WSMessageTypeOpen}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.WriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeOpen))
		})

		It("Should encode and decode close message", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			msg := fhttp.WSMessage[framer.WriterRequest]{Type: fhttp.WSMessageTypeClose}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.WriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeClose))
		})
	})

	Describe("Frame Writer Response", func() {
		It("Should encode and decode response", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.WriterResponse{Command: writer.CommandWrite, Authorized: true}
			msg := fhttp.WSMessage[framer.WriterResponse]{Type: fhttp.WSMessageTypeData, Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.WriterResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Command).To(Equal(writer.CommandWrite))
			Expect(resMsg.Payload.Authorized).To(BeTrue())
		})
	})

	Describe("Frame Streamer Request", func() {
		It("Should encode and decode request", func(ctx SpecContext) {
			channels := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			Expect(dist.Channel.CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			cdec := codec.NewDynamic(dist.Channel)
			v := httpframer.Codec{
				Codec:          cdec,
				LowerPerfCodec: json.Codec,
			}
			req := framer.StreamerRequest{Keys: keys}
			msg := fhttp.WSMessage[framer.StreamerRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.StreamerRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Keys).To(Equal(keys))
		})
	})

	Describe("Frame Stream Response", func() {
		It("Should encode and decode single channel int32", func(ctx SpecContext) {
			keys := channel.Keys{1}
			v := httpframer.Codec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.StreamerResponse{
				Frame: frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			}
			msg := fhttp.WSMessage[framer.StreamerResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[framer.StreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1}))
			Expect(resMsg.Payload.Frame.Count()).To(Equal(1))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should encode and decode multiple channels", func(ctx SpecContext) {
			keys := channel.Keys{1, 2}
			v := httpframer.Codec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"float64", "int64"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.StreamerResponse{
				Frame: frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV(1.5, 2.5, 3.5),
					telem.NewSeriesV[int64](1000, 2000, 3000),
				}),
			}
			msg := fhttp.WSMessage[framer.StreamerResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.StreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1, 2}))
			Expect(resMsg.Payload.Frame.Count()).To(Equal(2))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV(1.5, 2.5, 3.5)))
			Expect(resMsg.Payload.Frame.SeriesAt(1)).To(telem.MatchSeriesData(telem.NewSeriesV[int64](1000, 2000, 3000)))
		})
		It("Should encode and decode empty frame", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.StreamerResponse{Frame: frame.Frame{}}
			msg := fhttp.WSMessage[framer.StreamerResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.StreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Frame.Empty()).To(BeTrue())
		})

		It("Should encode and decode open message", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			msg := fhttp.WSMessage[framer.StreamerResponse]{Type: fhttp.WSMessageTypeOpen}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.StreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeOpen))
		})

		It("Should encode and decode close message", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			msg := fhttp.WSMessage[framer.StreamerResponse]{Type: fhttp.WSMessageTypeClose}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.StreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeClose))
		})
	})

	Describe("Frame Iterator Request", func() {
		It("Should encode and decode an open request and update the codec", func(ctx SpecContext) {
			channels := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			Expect(dist.Channel.CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			cdec := codec.NewDynamic(dist.Channel)
			v := httpframer.Codec{
				Codec:          cdec,
				LowerPerfCodec: json.Codec,
			}
			req := framer.IteratorRequest{Keys: keys}
			msg := fhttp.WSMessage[framer.IteratorRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.IteratorRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(cdec.Initialized()).To(BeTrue())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Keys).To(Equal(keys))
		})

		It("Should not call Update when the request has no keys", func(ctx SpecContext) {
			cdec := codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"})
			v := httpframer.Codec{
				Codec:          cdec,
				LowerPerfCodec: json.Codec,
			}
			req := framer.IteratorRequest{
				Command: iterator.CommandNext,
				Span:    telem.Second,
			}
			msg := fhttp.WSMessage[framer.IteratorRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.IteratorRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Payload.Command).To(Equal(iterator.CommandNext))
			Expect(resMsg.Payload.Span).To(Equal(telem.Second))
		})
	})

	Describe("Frame Iterator Response", func() {
		It("Should binary-encode a data variant response carrying a frame", func(ctx SpecContext) {
			keys := channel.Keys{1}
			v := httpframer.Codec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Command: iterator.CommandNext,
				Frame:   frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			}
			msg := fhttp.WSMessage[framer.IteratorResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[framer.IteratorResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Variant).To(Equal(iterator.ResponseVariantData))
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1}))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should JSON-encode an ack variant response", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.IteratorResponse{
				Variant: iterator.ResponseVariantAck,
				Command: iterator.CommandNext,
				Ack:     true,
				SeqNum:  42,
			}
			msg := fhttp.WSMessage[framer.IteratorResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(254)))
			var resMsg fhttp.WSMessage[framer.IteratorResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Payload.Variant).To(Equal(iterator.ResponseVariantAck))
			Expect(resMsg.Payload.Ack).To(BeTrue())
			Expect(resMsg.Payload.Command).To(Equal(iterator.CommandNext))
			Expect(resMsg.Payload.SeqNum).To(Equal(42))
		})

		It("Should JSON-encode a data variant response with an empty frame", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Frame:   frame.Frame{},
			}
			msg := fhttp.WSMessage[framer.IteratorResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(254)))
			var resMsg fhttp.WSMessage[framer.IteratorResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Payload.Variant).To(Equal(iterator.ResponseVariantData))
			Expect(resMsg.Payload.Frame.Empty()).To(BeTrue())
		})

		It("Should encode and decode an open message", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			msg := fhttp.WSMessage[framer.IteratorResponse]{Type: fhttp.WSMessageTypeOpen}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.IteratorResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeOpen))
		})

		It("Should encode and decode a close message", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			msg := fhttp.WSMessage[framer.IteratorResponse]{Type: fhttp.WSMessageTypeClose}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.IteratorResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeClose))
		})

		It("Should drop non-frame metadata on the binary path", func(ctx SpecContext) {
			keys := channel.Keys{1}
			v := httpframer.Codec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Command: iterator.CommandNext,
				Ack:     true,
				SeqNum:  99,
				Frame:   frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			}
			msg := fhttp.WSMessage[framer.IteratorResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[framer.IteratorResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			// On the binary data path only Variant and Frame survive.
			// Command, Ack, and SeqNum are intentionally not on the wire.
			Expect(resMsg.Payload.Variant).To(Equal(iterator.ResponseVariantData))
			Expect(resMsg.Payload.Command).To(Equal(iterator.Command(0)))
			Expect(resMsg.Payload.Ack).To(BeFalse())
			Expect(resMsg.Payload.SeqNum).To(Equal(0))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})
	})

	Describe("Error paths and edge cases", func() {
		It("Should return an error from EncodeStream when the value type is not recognized", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			Expect(v.Encode(ctx, "not a websocket message")).Error().To(MatchError(ContainSubstring("incompatible type")))
		})

		It("Should return an error from DecodeStream when the target type is not recognized", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			var target string
			Expect(v.Decode(ctx, []byte{0xFE}, &target)).To(MatchError(ContainSubstring("incompatible type")))
		})

		It("Should return an error from decodeWriteResponse when given a high-perf prefix", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			var resMsg fhttp.WSMessage[framer.WriterResponse]
			Expect(v.Decode(ctx, []byte{0xFF}, &resMsg)).To(MatchError(ContainSubstring("unexpected high performance codec special character")))
		})

		It("Should keep the codec initialized across non-open iterator requests", func(ctx SpecContext) {
			channels := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			Expect(dist.Channel.CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			cdec := codec.NewDynamic(dist.Channel)
			v := httpframer.Codec{Codec: cdec, LowerPerfCodec: json.Codec}

			openReq := fhttp.WSMessage[framer.IteratorRequest]{
				Type:    "data",
				Payload: framer.IteratorRequest{Keys: keys},
			}
			encOpen := MustSucceed(v.Encode(ctx, openReq))
			var decOpen fhttp.WSMessage[framer.IteratorRequest]
			Expect(v.Decode(ctx, encOpen, &decOpen)).To(Succeed())
			Expect(cdec.Initialized()).To(BeTrue())

			nextReq := fhttp.WSMessage[framer.IteratorRequest]{
				Type:    "data",
				Payload: framer.IteratorRequest{Command: iterator.CommandNext, Span: telem.Second},
			}
			encNext := MustSucceed(v.Encode(ctx, nextReq))
			var decNext fhttp.WSMessage[framer.IteratorRequest]
			Expect(v.Decode(ctx, encNext, &decNext)).To(Succeed())
			Expect(cdec.Initialized()).To(BeTrue())
			Expect(decNext.Payload.Command).To(Equal(iterator.CommandNext))
		})

		It("Should round-trip an iterator data response after a control message", func(ctx SpecContext) {
			keys := channel.Keys{1}
			v := httpframer.Codec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			openMsg := fhttp.WSMessage[framer.IteratorResponse]{Type: fhttp.WSMessageTypeOpen}
			MustSucceed(v.Encode(ctx, openMsg))

			res := framer.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Frame:   frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](7, 8)}),
			}
			dataMsg := fhttp.WSMessage[framer.IteratorResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, dataMsg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var dec fhttp.WSMessage[framer.IteratorResponse]
			Expect(v.Decode(ctx, encoded, &dec)).To(Succeed())
			Expect(dec.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](7, 8)))
		})

		It("Should round-trip a streamer response with an empty frame as JSON", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			res := framer.StreamerResponse{Frame: frame.Frame{}}
			msg := fhttp.WSMessage[framer.StreamerResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(254)))
			var dec fhttp.WSMessage[framer.StreamerResponse]
			Expect(v.Decode(ctx, encoded, &dec)).To(Succeed())
			Expect(dec.Payload.Frame.Empty()).To(BeTrue())
		})

		It("Should round-trip a writer SetAuthority command via the JSON path", func(ctx SpecContext) {
			v := httpframer.Codec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: json.Codec,
			}
			req := framer.WriterRequest{
				Command: writer.CommandSetAuthority,
				Config:  framer.WriterConfig{Keys: channel.Keys{1}, Authorities: []uint32{255}},
			}
			msg := fhttp.WSMessage[framer.WriterRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(254)))
			var dec fhttp.WSMessage[framer.WriterRequest]
			Expect(v.Decode(ctx, encoded, &dec)).To(Succeed())
			Expect(dec.Payload.Command).To(Equal(writer.CommandSetAuthority))
			Expect(dec.Payload.Config.Authorities).To(Equal([]uint32{255}))
		})
	})
})
