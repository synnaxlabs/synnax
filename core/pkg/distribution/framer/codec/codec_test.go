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
	"context"
	"encoding/json"
	"math/rand"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
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
		encoded := MustSucceed(cdc.Encode(context.Background(), fr))
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
					telem.NewSeriesStringsV("cat", "dog"),
					telem.NewSeriesStaticJSONV(
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
			s1.TimeRange = telem.NewRangeSeconds(1, 12)
			s1.Alignment = 7
			float32Data := make([]float32, 5000)
			for i := range float32Data {
				float32Data[i] = 1.234 + float32(i)*rand.Float32()
			}
			s2 := telem.NewSeries[float32](float32Data)
			s2.TimeRange = telem.NewRangeSeconds(3, 5)
			s2.Alignment = 10
			s3 := telem.NewSeriesStringsV("cat", "dog", "rabbit", "frog")
			s3.TimeRange = telem.NewRangeSeconds(1, 5)
			s3.Alignment = 5
			s4 := telem.MakeSeries(telem.Uint8T, 5000)
			s4.Alignment = cesium.LeadingAlignment(5000, 5)
			s4.TimeRange = telem.NewRangeSeconds(9999999, 999999999)
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
		It("Should return a validation error when a series has the wrong data type", func() {
			c := codec.NewStatic(
				[]channel.Key{1},
				[]telem.DataType{telem.Uint8T},
			)
			fr := core.UnaryFrame(1, telem.NewSeriesSecondsTSV(1, 2, 3))
			encoded, err := c.Encode(ctx, fr)
			Expect(encoded).To(HaveLen(0))
			Expect(err).To(HaveOccurredAs(validate.Error))
		})
	})

	Describe("Dynamic Codec", Ordered, func() {
		var (
			builder    *mock.Cluster
			channelSvc channel.Service
			idxCh      channel.Channel
			dataCh     channel.Channel
		)
		BeforeAll(func() {
			builder = mock.NewCluster()
			dist := builder.Provision(ctx)
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
		})
		ShouldNotLeakGoroutinesBeforeEach()

		It("Should allow the caller to update the list of channels", func() {
			codec := codec.NewDynamic(channelSvc)
			Expect(codec.Update(ctx, []channel.Key{dataCh.Key(), idxCh.Key()})).To(Succeed())
			fr := core.MultiFrame(
				channel.Keys{dataCh.Key(), idxCh.Key()},
				[]telem.Series{
					telem.NewSeriesV[float32](1, 2, 3, 4),
					telem.NewSeriesSecondsTSV(1, 2, 3, 4),
				},
			)
			encoded := MustSucceed(codec.Encode(ctx, fr))
			decoded := MustSucceed(codec.Decode(encoded))
			Expect(fr.Frame).To(telem.MatchFrame[channel.Key](decoded.Frame))
		})

		Describe("Initialized", func() {
			It("Should return false if update has not been called on the codec at least once", func() {
				codec := codec.NewDynamic(channelSvc)
				Expect(codec.Initialized()).To(BeFalse())
			})

			It("Should return true if update has been called on the codec at least once", func() {
				codec := codec.NewDynamic(channelSvc)
				Expect(codec.Update(ctx, []channel.Key{dataCh.Key(), idxCh.Key()})).To(Succeed())
				Expect(codec.Initialized()).To(BeTrue())
			})
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

			frame1 := core.UnaryFrame(idxCh.Key(), telem.NewSeriesSecondsTSV(1, 2, 3))
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
			encoded = MustSucceed(encoder.Encode(ctx, frame1))
			decoded = MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame.SeriesSlice()).To(HaveLen(0))

			frame2 := core.UnaryFrame(dataCh.Key(), telem.NewSeriesV[float32](1, 2, 3, 4))
			encoded = MustSucceed(encoder.Encode(ctx, frame2))
			decoded = MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame).To(telem.MatchFrame[channel.Key](frame2.Frame))
		})

		// This test is a regression that ensures the codec is designed to handle
		// race conditions between the encoding side and an upstream go-routines
		// producing frames. Even if an upstream routine passes a frame to the encoder
		// that contains keys that are not in the current state, they should be properly
		// ignored.
		Describe("Delayed Frames", func() {
			Context("Empty Result", func() {
				It("Should work correctly when a 'delayed' frame is provided ot the codec", func() {
					encoder := codec.NewDynamic(channelSvc)
					decoder := codec.NewDynamic(channelSvc)
					By("Correctly encoding and decoding when the two codecs are in sync")
					Expect(decoder.Update(ctx, []channel.Key{idxCh.Key()})).To(Succeed())
					Expect(encoder.Update(ctx, []channel.Key{idxCh.Key()})).To(Succeed())

					frame1 := core.UnaryFrame(
						idxCh.Key(),
						telem.NewSeriesSecondsTSV(1, 2, 3),
					)
					encoded := MustSucceed(encoder.Encode(ctx, frame1))
					decoded := MustSucceed(decoder.Decode(encoded))
					Expect(decoded.Frame).To(telem.MatchFrame[channel.Key](frame1.Frame))

					Expect(decoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
					Expect(encoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
					delayedFrame1 := core.UnaryFrame(
						idxCh.Key(),
						telem.NewSeriesV[float32](1, 2, 3, 4),
					)
					encoded = MustSucceed(encoder.Encode(ctx, delayedFrame1))
					decoded = MustSucceed(decoder.Decode(encoded))
					Expect(decoded.Frame.KeysSlice()).To(HaveLen(0))
				})
			})

			Context("Non-Empty Result", func() {
				It("Should work correctly when a 'delayed' frame is provided ot the codec", func() {
					encoder := codec.NewDynamic(channelSvc)
					decoder := codec.NewDynamic(channelSvc)
					By("Correctly encoding and decoding when the two codecs are in sync")
					keys := []channel.Key{idxCh.Key(), dataCh.Key()}
					Expect(decoder.Update(ctx, keys)).To(Succeed())
					Expect(encoder.Update(ctx, keys)).To(Succeed())

					frame1 := core.MultiFrame(
						keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(1, 2, 3),
							telem.NewSeriesV[float32](1, 2, 3),
						},
					)
					encoded := MustSucceed(encoder.Encode(ctx, frame1))
					decoded := MustSucceed(decoder.Decode(encoded))
					Expect(decoded.Frame).To(telem.MatchFrame[channel.Key](frame1.Frame))

					Expect(decoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
					Expect(encoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
					delayedFrame1 := core.MultiFrame(
						keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(1, 2, 3),
							telem.NewSeriesV[float32](1, 2, 3),
						},
					)
					encoded = MustSucceed(encoder.Encode(ctx, delayedFrame1))
					decoded = MustSucceed(decoder.Decode(encoded))
					Expect(decoded.Frame.KeysSlice()).To(HaveLen(1))
				})
			})
		})
	})

	Describe("Sorter Reuse", func() {
		It("Should correctly handle encoding frames of varying sizes sequentially", func() {
			keys := channel.Keys{1, 2, 3, 4, 5}
			dataTypes := []telem.DataType{telem.Int32T, telem.Float32T, telem.Int64T, telem.Uint8T, telem.Float64T}
			codec := codec.NewStatic(keys, dataTypes)

			largeFrame := core.MultiFrame(
				channel.Keys{5, 3, 1, 4, 2},
				[]telem.Series{
					telem.NewSeriesV[float64](1.1, 2.2, 3.3),
					telem.NewSeriesV[int64](100, 200, 300),
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[uint8](10, 20, 30),
					telem.NewSeriesV[float32](4.4, 5.5, 6.6),
				},
			)
			encoded1 := MustSucceed(codec.Encode(ctx, largeFrame))
			decoded1 := MustSucceed(codec.Decode(encoded1))
			Expect(largeFrame.Frame).To(telem.MatchFrame(decoded1.Frame))

			smallFrame := core.MultiFrame(
				channel.Keys{2, 4},
				[]telem.Series{
					telem.NewSeriesV[float32](7.7, 8.8),
					telem.NewSeriesV[uint8](40, 50),
				},
			)
			encoded2 := MustSucceed(codec.Encode(ctx, smallFrame))
			decoded2 := MustSucceed(codec.Decode(encoded2))
			Expect(smallFrame.Frame).To(telem.MatchFrame(decoded2.Frame))

			anotherLargeFrame := core.MultiFrame(
				channel.Keys{4, 2, 1, 3},
				[]telem.Series{
					telem.NewSeriesV[uint8](60, 70, 80, 90),
					telem.NewSeriesV[float32](9.9, 10.10),
					telem.NewSeriesV[int32](4, 5, 6, 7, 8),
					telem.NewSeriesV[int64](400, 500),
				},
			)
			encoded3 := MustSucceed(codec.Encode(ctx, anotherLargeFrame))
			decoded3 := MustSucceed(codec.Decode(encoded3))
			Expect(anotherLargeFrame.Frame).To(telem.MatchFrame(decoded3.Frame))

			emptyFrame := core.Frame{}
			encoded4 := MustSucceed(codec.Encode(ctx, emptyFrame))
			decoded4 := MustSucceed(codec.Decode(encoded4))
			Expect(emptyFrame.Frame).To(telem.MatchFrame(decoded4.Frame))
		})
	})

	Describe("Duplicate Channel Keys Sorting", func() {
		It("Should correctly sort and encode frames with duplicate channel keys", func() {
			keys := channel.Keys{10, 20, 30}
			dataTypes := []telem.DataType{telem.Int32T, telem.Float64T, telem.Uint8T}
			codec := codec.NewStatic(keys, dataTypes)

			// Create frame with multiple series for the same channels in random order
			frame := core.MultiFrame(
				channel.Keys{20, 10, 30, 10, 20, 30, 10},
				[]telem.Series{
					telem.NewSeriesV[float64](1.1, 2.2),         // channel 20
					telem.NewSeriesV[int32](100, 200, 300),      // channel 10
					telem.NewSeriesV[uint8](5, 6, 7),            // channel 30
					telem.NewSeriesV[int32](400, 500),           // channel 10
					telem.NewSeriesV[float64](3.3, 4.4, 5.5),    // channel 20
					telem.NewSeriesV[uint8](8, 9),               // channel 30
					telem.NewSeriesV[int32](600, 700, 800, 900), // channel 10
				},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			Expect(decoded.Count()).To(Equal(7))

			ch10Series := decoded.Get(10)
			Expect(len(ch10Series.Series)).To(Equal(3))
			ch20Series := decoded.Get(20)
			Expect(len(ch20Series.Series)).To(Equal(2))
			ch30Series := decoded.Get(30)
			Expect(len(ch30Series.Series)).To(Equal(2))

			Expect(frame.Frame).To(telem.MatchFrame(decoded.Frame))
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle frames with very large channel key values", func() {
			keys := channel.Keys{channel.Key(^uint32(0)), channel.Key(^uint32(0) - 1), channel.Key(1)}
			dataTypes := []telem.DataType{telem.Int32T, telem.Float32T, telem.Uint64T}
			codec := codec.NewStatic(keys, dataTypes)

			frame := core.MultiFrame(
				channel.Keys{channel.Key(^uint32(0) - 1), channel.Key(^uint32(0)), channel.Key(1)},
				[]telem.Series{
					telem.NewSeriesV[float32](1.1, 2.2, 3.3),
					telem.NewSeriesV[int32](10, 20, 30),
					telem.NewSeriesV[uint64](100, 200, 300),
				},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))
			Expect(frame.Frame).To(telem.MatchFrame(decoded.Frame))
		})

		It("Should handle encoding after an empty frame (sorter reset edge case)", func() {
			keys := channel.Keys{5, 10, 15}
			dataTypes := []telem.DataType{telem.Int32T, telem.Float32T, telem.Uint8T}
			codec := codec.NewStatic(keys, dataTypes)

			frame1 := core.MultiFrame(
				channel.Keys{15, 5, 10},
				[]telem.Series{
					telem.NewSeriesV[uint8](1, 2, 3),
					telem.NewSeriesV[int32](10, 20),
					telem.NewSeriesV[float32](1.5, 2.5, 3.5),
				},
			)
			encoded1 := MustSucceed(codec.Encode(ctx, frame1))
			decoded1 := MustSucceed(codec.Decode(encoded1))
			Expect(frame1.Frame).To(telem.MatchFrame(decoded1.Frame))

			emptyFrame := core.Frame{}
			encoded2 := MustSucceed(codec.Encode(ctx, emptyFrame))
			decoded2 := MustSucceed(codec.Decode(encoded2))
			Expect(decoded2.Empty()).To(BeTrue())

			frame3 := core.MultiFrame(
				channel.Keys{10, 5},
				[]telem.Series{
					telem.NewSeriesV[float32](4.5, 5.5),
					telem.NewSeriesV[int32](30, 40, 50),
				},
			)
			encoded3 := MustSucceed(codec.Encode(ctx, frame3))
			decoded3 := MustSucceed(codec.Decode(encoded3))
			Expect(frame3.Frame).To(telem.MatchFrame(decoded3.Frame))
		})

		It("Should handle single channel frame after multi-channel frame", func() {
			keys := channel.Keys{100, 200, 300}
			dataTypes := []telem.DataType{telem.Int64T, telem.Float64T, telem.StringT}
			codec := codec.NewStatic(keys, dataTypes)

			multiFrame := core.MultiFrame(
				channel.Keys{300, 100, 200},
				[]telem.Series{
					telem.NewSeriesStringsV("hello", "world"),
					telem.NewSeriesV[int64](1000, 2000, 3000),
					telem.NewSeriesV[float64](1.111, 2.222),
				},
			)
			encoded1 := MustSucceed(codec.Encode(ctx, multiFrame))
			decoded1 := MustSucceed(codec.Decode(encoded1))
			Expect(multiFrame.Frame).To(telem.MatchFrame(decoded1.Frame))

			singleFrame := core.UnaryFrame(200, telem.NewSeriesV[float64](9.999))
			encoded2 := MustSucceed(codec.Encode(ctx, singleFrame))
			decoded2 := MustSucceed(codec.Decode(encoded2))
			Expect(singleFrame.Frame).To(telem.MatchFrame(decoded2.Frame))
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
	if err := cd.EncodeStream(b.Context(), w, fr); err != nil {
		b.Fatalf("failed to encode stream: %v", err)
	}
	for b.Loop() {
		if err := cd.EncodeStream(b.Context(), w, fr); err != nil {
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
	for b.Loop() {
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
		encoded, _ = cd.Encode(b.Context(), fr)
		r          = bytes.NewReader(encoded)
	)
	for b.Loop() {
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
	for b.Loop() {
		if err := json.Unmarshal(encoded, &v); err != nil {
			b.Fatalf("failed to decode stream: %v", err)
		}
	}
}
