// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Codec", func() {
	DescribeTable("Encode + Decode", func(
		channels channel.Keys,
		dataTypes []telem.DataType,
		fr framer.Frame,
	) {
		cdc := codec.NewStatic(channels, dataTypes)
		encoded := MustSucceed(cdc.Encode(nil, fr))
		decoded := MustSucceed(cdc.Decode(encoded))
		Expect(fr.Frame).To(telem.MatchFrame(decoded.Frame))
	},
		Entry("Empty Frame", channel.Keys{}, []telem.DataType{}, framer.Frame{}),
		Entry("All Channels Present, In Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Int64T, telem.Float32T, telem.Float64T},
			core.MultiFrame(
				channel.Keys{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[float32](4, 5, 6),
					telem.NewSeriesV[float64](7, 8, 9),
				},
			),
		),
		Entry("All Channels Present, Out of Order",
			channel.Keys{3, 1, 2},
			[]telem.DataType{telem.Float64T, telem.Int64T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{2, 3, 1},
				[]telem.Series{
					telem.NewSeriesV[float32](3, 2, 1),
					telem.NewSeriesV[float64](1, 2, 3),
					telem.NewSeriesV[int64](5, 6, 7),
				},
			),
		),
		Entry("Some Channels Present, In Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			core.MultiFrame(
				channel.Keys{1, 3},
				[]telem.Series{
					telem.NewSeriesV[uint8](1, 2, 3),
					telem.NewSeriesV[float64](7, 8, 9),
				},
			),
		),
		Entry("Some Channels Present, Out of Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			core.MultiFrame(
				channel.Keys{3, 1},
				[]telem.Series{
					telem.NewSeriesV[float64](7, 8, 9),
					telem.NewSeriesV[uint8](1, 2, 3),
				},
			),
		),
		Entry("All Same Time Range",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2},
				[]telem.Series{
					{
						DataType:  telem.Uint8T,
						Data:      []byte{1},
						TimeRange: telem.TimeStamp(0).SpanRange(5),
					},
					{
						DataType:  telem.Float32T,
						Data:      []byte{1, 2, 3, 4},
						TimeRange: telem.TimeStamp(0).SpanRange(5),
					},
				},
			),
		),
		Entry("Different Time Ranges",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2},
				[]telem.Series{
					{
						DataType:  telem.Uint8T,
						Data:      []byte{1},
						TimeRange: telem.TimeStamp(0).SpanRange(5),
					},
					{
						DataType:  telem.Float32T,
						Data:      []byte{1, 2, 3, 4},
						TimeRange: telem.TimeStamp(5).SpanRange(5),
					},
				},
			),
		),
		Entry("Partial Present, Different Lengths",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			core.MultiFrame(
				channel.Keys{1, 3},
				[]telem.Series{
					telem.NewSeriesV[uint8](1),
					telem.NewSeriesV[float64](1, 2, 3, 4),
				},
			),
		),
		Entry("Same Alignments",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2},
				[]telem.Series{
					{
						DataType:  telem.Uint8T,
						Data:      []byte{1},
						Alignment: 5,
					},
					{
						DataType:  telem.Float32T,
						Data:      []byte{1, 2, 3, 4},
						Alignment: 5,
					},
				},
			),
		),
		Entry("Different Alignments",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2},
				[]telem.Series{
					{
						DataType:  telem.Uint8T,
						Data:      []byte{1},
						Alignment: 5,
					},
					{
						DataType:  telem.Float32T,
						Data:      []byte{1, 2, 3, 4},
						Alignment: 10,
					},
				},
			),
		),
		Entry("Variable Data Types",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.StringT, telem.JSONT},
			core.MultiFrame(
				channel.Keys{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[uint8](1, 2, 3),
					telem.NewStringsV("cat", "dog"),
					telem.NewStaticJSONV(
						map[string]any{"key": "value"},
						map[string]any{"key": "value2"},
					),
				},
			),
		),
		Entry("Multiple Series for the Same Channel",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2, 2, 1, 2},
				[]telem.Series{
					telem.NewSeriesV[uint8](1, 2, 3),
					telem.NewSeriesV[float32](1, 2, 3),
					telem.NewSeriesV[float32](5, 6, 7),
					telem.NewSeriesV[uint8](4, 5, 6),
					telem.NewSeriesV[float32](42.1, 42.3, 69.1),
				},
			),
		),
	)

	Describe("Complex Frames", func() {
		It("Should correctly serialize and deserialize a complex frame", func() {
			keys := channel.Keys{1, 2, 3, 4}
			dataTypes := []telem.DataType{"int32", "float32", "string", "uint8"}
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.TimeRange = telem.NewSecondsRange(1, 12)
			s1.Alignment = 7
			float32Data := make([]float32, 5000)
			for i := range float32Data {
				float32Data[i] = 1.234 + float32(i)*rand.Float32()
			}
			s2 := telem.NewSeries[float32](float32Data)
			s2.TimeRange = telem.NewSecondsRange(3, 5)
			s2.Alignment = 10
			s3 := telem.NewStringsV("cat", "dog", "rabbit", "frog")
			s3.TimeRange = telem.NewSecondsRange(1, 5)
			s3.Alignment = 5
			s4 := telem.MakeSeries(telem.Uint8T, 5000)
			s4.Alignment = telem.LeadingAlignment(5000, 5)
			s4.TimeRange = telem.NewSecondsRange(9999999, 999999999)
			originalFrame := core.MultiFrame(
				keys,
				[]telem.Series{s1, s2, s3, s4},
			)

			cdc := codec.NewStatic(keys, dataTypes)
			encoded := MustSucceed(cdc.Encode(ctx, originalFrame))
			decoded := MustSucceed(cdc.Decode(encoded))
			Expect(originalFrame.Frame).To(telem.MatchFrame(decoded.Frame))
		})
	})

	Describe("Error Handling", func() {
		It("Should return a validation error when a channel in a frame was not provided to the codec", func() {
			c := codec.NewStatic(
				[]channel.Key{1, 2, 3},
				[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			)
			fr := core.UnaryFrame(4, telem.NewSecondsTSV(1, 2, 3))
			encoded, err := c.Encode(ctx, fr)
			Expect(encoded).To(HaveLen(0))
			Expect(err).To(HaveOccurredAs(validate.Error))
		})

		It("Should return a validation error when a series has the wrong data type", func() {
			c := codec.NewStatic(
				[]channel.Key{1},
				[]telem.DataType{telem.Uint8T},
			)
			fr := core.UnaryFrame(1, telem.NewSecondsTSV(1, 2, 3))
			encoded, err := c.Encode(ctx, fr)
			Expect(encoded).To(HaveLen(0))
			Expect(err).To(HaveOccurredAs(validate.Error))
		})
	})

	Describe("Dynamic Codec", Ordered, func() {
		var (
			builder    *mock.Builder
			channelSvc channel.Service
			idxCh      channel.Channel
			dataCh     channel.Channel
		)
		BeforeAll(func() {
			builder = mock.NewBuilder()
			dist := builder.New(ctx)
			channelSvc = dist.Channel
			w := dist.Channel.NewWriter(nil)
			idxCh = channel.Channel{
				DataType: telem.TimeStampT,
				Name:     "time",
				IsIndex:  true,
			}
			Expect(w.Create(ctx, &idxCh)).To(Succeed())
			dataCh = channel.Channel{
				Name:       "data",
				DataType:   telem.Float32T,
				LocalIndex: idxCh.Key().LocalKey(),
			}
			Expect(w.Create(ctx, &dataCh)).To(Succeed())
		})
		AfterAll(func() {
			Expect(builder.Close()).To(Succeed())
			Expect(builder.Cleanup()).To(Succeed())
		})
		ShouldNotLeakGoroutinesDuringEach()

		It("Should allow the caller to update the list of channels", func() {
			codec := codec.NewDynamic(channelSvc)
			Expect(codec.Update(ctx, []channel.Key{dataCh.Key(), idxCh.Key()})).To(Succeed())
			fr := core.MultiFrame(
				channel.Keys{dataCh.Key(), idxCh.Key()},
				[]telem.Series{
					telem.NewSeriesV[float32](1, 2, 3, 4),
					telem.NewSecondsTSV(1, 2, 3, 4),
				},
			)
			encoded := MustSucceed(codec.Encode(ctx, fr))
			decoded := MustSucceed(codec.Decode(encoded))
			Expect(fr.Frame).To(telem.MatchFrame[channel.Key](decoded.Frame))
		})

		It("Should panic if the codec is not initialized", func() {
			codec := codec.NewDynamic(nil)
			Expect(func() {
				fr := framer.Frame{}
				_, _ = codec.Encode(ctx, fr)
			}).To(Panic())
		})

		It("Should use the correct encode/decode state even if the codecs are out of sync", func() {
			encoder := codec.NewDynamic(channelSvc)
			decoder := codec.NewDynamic(channelSvc)
			By("Correctly encoding and decoding when the two codecs are in sync")
			Expect(decoder.Update(ctx, []channel.Key{idxCh.Key()})).To(Succeed())
			Expect(encoder.Update(ctx, []channel.Key{idxCh.Key()})).To(Succeed())

			frame1 := core.UnaryFrame(idxCh.Key(), telem.NewSecondsTSV(1, 2, 3))
			encoded := MustSucceed(encoder.Encode(ctx, frame1))
			decoded := MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame).To(telem.MatchFrame[channel.Key](frame1.Frame))

			By("Correctly using the previous encoding state when the two codecs are out of sync")
			Expect(decoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())

			encoded = MustSucceed(encoder.Encode(ctx, frame1))
			decoded = MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame).To(telem.MatchFrame[channel.Key](frame1.Frame))

			By("Correctly using he most up to date state after the codec are in sync again")
			Expect(encoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
			_, err := encoder.Encode(ctx, frame1)
			Expect(err).To(HaveOccurredAs(validate.Error))
			frame2 := core.UnaryFrame(dataCh.Key(), telem.NewSeriesV[float32](1, 2, 3, 4))
			encoded = MustSucceed(encoder.Encode(ctx, frame2))
			decoded = MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame).To(telem.MatchFrame[channel.Key](frame2.Frame))
		})
	})
})

func BenchmarkEncode(b *testing.B) {
	dataTypes := []telem.DataType{"int32"}
	keys := channel.Keys{1}
	fr := core.MultiFrame(
		keys,
		[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
	)
	cd := codec.NewStatic(keys, dataTypes)
	w := bytes.NewBuffer(nil)
	if err := cd.EncodeStream(nil, w, fr); err != nil {
		b.Fatalf("failed to encode stream: %v", err)
	}
	for range b.N {
		if err := cd.EncodeStream(nil, w, fr); err != nil {
			b.Fatalf("failed to encode stream: %v", err)
		}
		w.Reset()
	}
}

func BenchmarkJSONEncode(b *testing.B) {
	keys := channel.Keys{1}
	fr := core.MultiFrame(
		keys,
		[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
	)
	for range b.N {
		if _, err := json.Marshal(fr); err != nil {
			b.Fatalf("failed to encode stream: %v", err)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	var (
		dataTypes = []telem.DataType{"int32"}
		keys      = channel.Keys{1}
		fr        = core.MultiFrame(
			keys,
			[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
		)
		cd         = codec.NewStatic(keys, dataTypes)
		encoded, _ = cd.Encode(nil, fr)
		r          = bytes.NewReader(encoded)
	)
	for range b.N {
		if _, err := r.Seek(0, 0); err != nil {
			b.Fatalf("failed to seek: %v", err)
		}
		if fr, err := cd.DecodeStream(r); err != nil || fr.Empty() {
			b.Fatalf("failed to decode stream: %v", err)
		}
	}
}

func BenchmarkJSONDecode(b *testing.B) {
	keys := channel.Keys{1}
	encoded, err := json.Marshal(core.MultiFrame(
		keys,
		[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
	))
	if err != nil {
		b.Fatalf("failed to encode stream: %v", err)
	}
	var v framer.Frame
	for range b.N {
		if err := json.Unmarshal(encoded, &v); err != nil {
			b.Fatalf("failed to decode stream: %v", err)
		}
	}
}
