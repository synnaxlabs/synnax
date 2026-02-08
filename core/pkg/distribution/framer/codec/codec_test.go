// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
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
			frame.NewMulti(
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
			frame.NewMulti(
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
			frame.NewMulti(
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
			frame.NewMulti(
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
			frame.NewMulti(
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
			frame.NewMulti(
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
			frame.NewMulti(
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
			frame.NewMulti(
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
			frame.NewMulti(
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
			frame.NewMulti(
				channel.Keys{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[uint8](1, 2, 3),
					telem.NewSeriesVariableV("cat", "dog"),
					MustSucceed(telem.NewSeriesJSONV(
						map[string]any{"key": "value"},
						map[string]any{"key": "value2"},
					)),
				},
			),
		),
		Entry("Multiple Series for the Same Channel",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			frame.NewMulti(
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
			s2 := telem.NewSeries(float32Data)
			s2.TimeRange = telem.NewRangeSeconds(3, 5)
			s2.Alignment = 10
			s3 := telem.NewSeriesVariableV("cat", "dog", "rabbit", "frog")
			s3.TimeRange = telem.NewRangeSeconds(1, 5)
			s3.Alignment = 5
			s4 := telem.MakeSeries(telem.Uint8T, 5000)
			s4.Alignment = cesium.LeadingAlignment(5000, 5)
			s4.TimeRange = telem.NewRangeSeconds(9999999, 999999999)
			originalFrame := frame.NewMulti(
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
			fr := frame.NewUnary(1, telem.NewSeriesSecondsTSV(1, 2, 3))
			encoded, err := c.Encode(ctx, fr)
			Expect(encoded).To(HaveLen(0))
			Expect(err).To(HaveOccurredAs(validate.ErrValidation))
		})
	})

	Describe("Dynamic Codec", Ordered, func() {
		var (
			builder    *mock.Cluster
			channelSvc *channel.Service
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
			fr := frame.NewMulti(
				channel.Keys{dataCh.Key(), idxCh.Key()},
				[]telem.Series{
					telem.NewSeriesV[float32](1, 2, 3, 4),
					telem.NewSeriesSecondsTSV(1, 2, 3, 4),
				},
			)
			encoded := MustSucceed(codec.Encode(ctx, fr))
			decoded := MustSucceed(codec.Decode(encoded))
			Expect(fr.Frame).To(telem.MatchFrame(decoded.Frame))
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

			frame1 := frame.NewUnary(idxCh.Key(), telem.NewSeriesSecondsTSV(1, 2, 3))
			encoded := MustSucceed(encoder.Encode(ctx, frame1))
			decoded := MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame).To(telem.MatchFrame(frame1.Frame))

			By("Correctly using the previous encoding state when the two codecs are out of sync")
			Expect(decoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())

			encoded = MustSucceed(encoder.Encode(ctx, frame1))
			decoded = MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame).To(telem.MatchFrame(frame1.Frame))

			By("Correctly using he most up to date state after the codec are in sync again")
			Expect(encoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
			encoded = MustSucceed(encoder.Encode(ctx, frame1))
			decoded = MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame.SeriesSlice()).To(HaveLen(0))

			frame2 := frame.NewUnary(dataCh.Key(), telem.NewSeriesV[float32](1, 2, 3, 4))
			encoded = MustSucceed(encoder.Encode(ctx, frame2))
			decoded = MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame).To(telem.MatchFrame(frame2.Frame))
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

					frame1 := frame.NewUnary(
						idxCh.Key(),
						telem.NewSeriesSecondsTSV(1, 2, 3),
					)
					encoded := MustSucceed(encoder.Encode(ctx, frame1))
					decoded := MustSucceed(decoder.Decode(encoded))
					Expect(decoded.Frame).To(telem.MatchFrame(frame1.Frame))

					Expect(decoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
					Expect(encoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
					delayedFrame1 := frame.NewUnary(
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

					frame1 := frame.NewMulti(
						keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(1, 2, 3),
							telem.NewSeriesV[float32](1, 2, 3),
						},
					)
					encoded := MustSucceed(encoder.Encode(ctx, frame1))
					decoded := MustSucceed(decoder.Decode(encoded))
					Expect(decoded.Frame).To(telem.MatchFrame(frame1.Frame))

					Expect(decoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
					Expect(encoder.Update(ctx, []channel.Key{dataCh.Key()})).To(Succeed())
					delayedFrame1 := frame.NewMulti(
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

			largeFrame := frame.NewMulti(
				channel.Keys{5, 3, 1, 4, 2},
				[]telem.Series{
					telem.NewSeriesV(1.1, 2.2, 3.3),
					telem.NewSeriesV[int64](100, 200, 300),
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[uint8](10, 20, 30),
					telem.NewSeriesV[float32](4.4, 5.5, 6.6),
				},
			)
			encoded1 := MustSucceed(codec.Encode(ctx, largeFrame))
			decoded1 := MustSucceed(codec.Decode(encoded1))
			Expect(largeFrame.Frame).To(telem.MatchFrame(decoded1.Frame))

			smallFrame := frame.NewMulti(
				channel.Keys{2, 4},
				[]telem.Series{
					telem.NewSeriesV[float32](7.7, 8.8),
					telem.NewSeriesV[uint8](40, 50),
				},
			)
			encoded2 := MustSucceed(codec.Encode(ctx, smallFrame))
			decoded2 := MustSucceed(codec.Decode(encoded2))
			Expect(smallFrame.Frame).To(telem.MatchFrame(decoded2.Frame))

			anotherLargeFrame := frame.NewMulti(
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

			emptyFrame := frame.Frame{}
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
			frame := frame.NewMulti(
				channel.Keys{20, 10, 30, 10, 20, 30, 10},
				[]telem.Series{
					telem.NewSeriesV(1.1, 2.2),                  // channel 20
					telem.NewSeriesV[int32](100, 200, 300),      // channel 10
					telem.NewSeriesV[uint8](5, 6, 7),            // channel 30
					telem.NewSeriesV[int32](400, 500),           // channel 10
					telem.NewSeriesV(3.3, 4.4, 5.5),             // channel 20
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

			frame := frame.NewMulti(
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

			frame1 := frame.NewMulti(
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

			emptyFrame := frame.Frame{}
			encoded2 := MustSucceed(codec.Encode(ctx, emptyFrame))
			decoded2 := MustSucceed(codec.Decode(encoded2))
			Expect(decoded2.Empty()).To(BeTrue())

			frame3 := frame.NewMulti(
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

			multiFrame := frame.NewMulti(
				channel.Keys{300, 100, 200},
				[]telem.Series{
					telem.NewSeriesVariableV("hello", "world"),
					telem.NewSeriesV[int64](1000, 2000, 3000),
					telem.NewSeriesV(1.111, 2.222),
				},
			)
			encoded1 := MustSucceed(codec.Encode(ctx, multiFrame))
			decoded1 := MustSucceed(codec.Decode(encoded1))
			Expect(multiFrame.Frame).To(telem.MatchFrame(decoded1.Frame))

			singleFrame := frame.NewUnary(200, telem.NewSeriesV(9.999))
			encoded2 := MustSucceed(codec.Encode(ctx, singleFrame))
			decoded2 := MustSucceed(codec.Decode(encoded2))
			Expect(singleFrame.Frame).To(telem.MatchFrame(decoded2.Frame))
		})
	})

	Describe("Alignment Compression", func() {
		It("Should merge two contiguous series for the same channel", func() {
			keys := channel.Keys{1}
			dataTypes := []telem.DataType{telem.Int32T}
			codec := codec.NewStatic(keys, dataTypes)

			// Create two series with contiguous alignments
			// Series 1: alignment 0, length 3 -> bounds [0, 3)
			// Series 2: alignment 3, length 2 -> bounds [3, 5)
			// These should merge into one series
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.Alignment = 0
			s2 := telem.NewSeriesV[int32](4, 5)
			s2.Alignment = 3

			frame := frame.NewMulti(
				channel.Keys{1, 1},
				[]telem.Series{s1, s2},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			// After merging, we should have only one series
			Expect(decoded.Count()).To(Equal(1))

			// Verify the data is correct (concatenated)
			series := decoded.Get(1)
			Expect(len(series.Series)).To(Equal(1))
			mergedData := telem.UnmarshalSlice[int32](series.Series[0].Data, telem.Int32T)
			Expect(mergedData).To(Equal([]int32{1, 2, 3, 4, 5}))

			// Verify alignment is from the first series
			Expect(series.Series[0].Alignment).To(Equal(telem.Alignment(0)))
		})

		It("Should merge three contiguous series for the same channel", func() {
			keys := channel.Keys{1}
			dataTypes := []telem.DataType{telem.Uint8T}
			codec := codec.NewStatic(keys, dataTypes)

			s1 := telem.NewSeriesV[uint8](1, 2)
			s1.Alignment = 0
			s2 := telem.NewSeriesV[uint8](3, 4, 5)
			s2.Alignment = 2
			s3 := telem.NewSeriesV[uint8](6)
			s3.Alignment = 5

			frame := frame.NewMulti(
				channel.Keys{1, 1, 1},
				[]telem.Series{s1, s2, s3},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			Expect(decoded.Count()).To(Equal(1))
			series := decoded.Get(1)
			Expect(len(series.Series)).To(Equal(1))
			mergedData := telem.UnmarshalSlice[uint8](series.Series[0].Data, telem.Uint8T)
			Expect(mergedData).To(Equal([]uint8{1, 2, 3, 4, 5, 6}))
		})

		It("Should not merge non-contiguous series for the same channel", func() {
			keys := channel.Keys{1}
			dataTypes := []telem.DataType{telem.Int32T}
			codec := codec.NewStatic(keys, dataTypes)

			// Gap between series: s1 ends at 3, s2 starts at 5
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.Alignment = 0
			s2 := telem.NewSeriesV[int32](4, 5)
			s2.Alignment = 5 // Gap! Previous ends at 3

			frame := frame.NewMulti(
				channel.Keys{1, 1},
				[]telem.Series{s1, s2},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			// Should have two separate series
			Expect(decoded.Count()).To(Equal(2))
			series := decoded.Get(1)
			Expect(len(series.Series)).To(Equal(2))
		})

		It("Should handle mixed contiguous and non-contiguous series", func() {
			keys := channel.Keys{1}
			dataTypes := []telem.DataType{telem.Int32T}
			codec := codec.NewStatic(keys, dataTypes)

			// s1 and s2 are contiguous (merge)
			// s3 has gap (don't merge)
			// s4 continues s3 (don't merge with s1+s2, but keep separate)
			s1 := telem.NewSeriesV[int32](1, 2)
			s1.Alignment = 0
			s2 := telem.NewSeriesV[int32](3, 4)
			s2.Alignment = 2
			s3 := telem.NewSeriesV[int32](5)
			s3.Alignment = 10 // Gap!
			s4 := telem.NewSeriesV[int32](6)
			s4.Alignment = 11 // Contiguous with s3

			frame := frame.NewMulti(
				channel.Keys{1, 1, 1, 1},
				[]telem.Series{s1, s2, s3, s4},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			// Should have 2 merged series: [s1+s2] and [s3+s4]
			Expect(decoded.Count()).To(Equal(2))
			series := decoded.Get(1)
			Expect(len(series.Series)).To(Equal(2))

			// First merged series should be [1, 2, 3, 4]
			firstData := telem.UnmarshalSlice[int32](series.Series[0].Data, telem.Int32T)
			Expect(firstData).To(Equal([]int32{1, 2, 3, 4}))

			// Second merged series should be [5, 6]
			secondData := telem.UnmarshalSlice[int32](series.Series[1].Data, telem.Int32T)
			Expect(secondData).To(Equal([]int32{5, 6}))
		})

		It("Should merge series for multiple channels independently", func() {
			keys := channel.Keys{1, 2}
			dataTypes := []telem.DataType{telem.Int32T, telem.Float32T}
			codec := codec.NewStatic(keys, dataTypes)

			// Channel 1: two contiguous series
			s1Ch1 := telem.NewSeriesV[int32](1, 2)
			s1Ch1.Alignment = 0
			s2Ch1 := telem.NewSeriesV[int32](3, 4)
			s2Ch1.Alignment = 2

			// Channel 2: two contiguous series
			s1Ch2 := telem.NewSeriesV[float32](1.1, 2.2, 3.3)
			s1Ch2.Alignment = 5
			s2Ch2 := telem.NewSeriesV[float32](4.4)
			s2Ch2.Alignment = 8

			frame := frame.NewMulti(
				channel.Keys{1, 1, 2, 2},
				[]telem.Series{s1Ch1, s2Ch1, s1Ch2, s2Ch2},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			// Should have 2 series total (one per channel)
			Expect(decoded.Count()).To(Equal(2))

			// Channel 1 should have merged series
			ch1Series := decoded.Get(1)
			Expect(len(ch1Series.Series)).To(Equal(1))
			ch1Data := telem.UnmarshalSlice[int32](ch1Series.Series[0].Data, telem.Int32T)
			Expect(ch1Data).To(Equal([]int32{1, 2, 3, 4}))

			// Channel 2 should have merged series
			ch2Series := decoded.Get(2)
			Expect(len(ch2Series.Series)).To(Equal(1))
			ch2Data := telem.UnmarshalSlice[float32](ch2Series.Series[0].Data, telem.Float32T)
			Expect(ch2Data).To(Equal([]float32{1.1, 2.2, 3.3, 4.4}))
		})

		It("Should merge series with zero alignments", func() {
			keys := channel.Keys{1}
			dataTypes := []telem.DataType{telem.Int32T}
			codec := codec.NewStatic(keys, dataTypes)

			// All zero alignments are considered contiguous
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.Alignment = 0
			s2 := telem.NewSeriesV[int32](4, 5)
			s2.Alignment = 0 // Both zero, should still merge if data is contiguous

			frame := frame.NewMulti(
				channel.Keys{1, 1},
				[]telem.Series{s1, s2},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			// Zero alignments: s1 has bounds [0, 3), s2 has bounds [0, 2)
			// These are NOT contiguous (s2 starts at 0, not 3)
			// So they should NOT merge
			Expect(decoded.Count()).To(Equal(2))
		})

		It("Should handle time range extension when merging", func() {
			keys := channel.Keys{1}
			dataTypes := []telem.DataType{telem.Int32T}
			codec := codec.NewStatic(keys, dataTypes)

			s1 := telem.NewSeriesV[int32](1, 2)
			s1.Alignment = 0
			s1.TimeRange = telem.TimeRange{
				Start: telem.TimeStamp(100),
				End:   telem.TimeStamp(200),
			}

			s2 := telem.NewSeriesV[int32](3, 4)
			s2.Alignment = 2
			s2.TimeRange = telem.TimeRange{
				Start: telem.TimeStamp(200),
				End:   telem.TimeStamp(300),
			}

			frame := frame.NewMulti(
				channel.Keys{1, 1},
				[]telem.Series{s1, s2},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			Expect(decoded.Count()).To(Equal(1))
			series := decoded.Get(1)
			Expect(len(series.Series)).To(Equal(1))

			// Time range should span both series
			mergedSeries := series.Series[0]
			Expect(mergedSeries.TimeRange.Start).To(Equal(telem.TimeStamp(100)))
			Expect(mergedSeries.TimeRange.End).To(Equal(telem.TimeStamp(300)))
		})

		It("Should preserve variable-density types when merging", func() {
			keys := channel.Keys{1}
			dataTypes := []telem.DataType{telem.StringT}
			codec := codec.NewStatic(keys, dataTypes)

			s1 := telem.NewSeriesVariableV("hello", "world")
			s1.Alignment = 0

			s2 := telem.NewSeriesVariableV("foo")
			s2.Alignment = 2

			frame := frame.NewMulti(
				channel.Keys{1, 1},
				[]telem.Series{s1, s2},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			Expect(decoded.Count()).To(Equal(1))
			series := decoded.Get(1)
			Expect(len(series.Series)).To(Equal(1))

			// Data should be concatenated correctly
			mergedStrings := telem.UnmarshalVariable[string](series.Series[0].Data)
			Expect(mergedStrings).To(Equal([]string{"hello", "world", "foo"}))
		})
	})
})

func BenchmarkEncode(b *testing.B) {
	dataTypes := []telem.DataType{"int32"}
	keys := channel.Keys{1}
	fr := frame.NewMulti(
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
	fr := frame.NewMulti(
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
		fr        = frame.NewMulti(
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
	encoded, err := json.Marshal(frame.NewMulti(
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

// Benchmark alignment compression with single series (no benefit expected)
func BenchmarkAlignmentCompression_SingleSeries(b *testing.B) {
	keys := channel.Keys{1}
	dataTypes := []telem.DataType{telem.Int32T}
	frame := frame.NewUnary(1, telem.NewSeriesV[int32](1, 2, 3, 4, 5))

	b.Run("Enabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes)
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})

	b.Run("Disabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes, codec.DisableAlignmentCompression())
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})
}

// Benchmark alignment compression with two contiguous series
func BenchmarkAlignmentCompression_TwoContiguous(b *testing.B) {
	keys := channel.Keys{1}
	dataTypes := []telem.DataType{telem.Int32T}

	s1 := telem.NewSeriesV[int32](1, 2, 3)
	s1.Alignment = 0
	s2 := telem.NewSeriesV[int32](4, 5, 6)
	s2.Alignment = 3

	frame := frame.NewMulti(channel.Keys{1, 1}, []telem.Series{s1, s2})

	b.Run("Enabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes)
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})

	b.Run("Disabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes, codec.DisableAlignmentCompression())
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})
}

// Benchmark alignment compression with many contiguous series (best case)
func BenchmarkAlignmentCompression_ManyContiguous(b *testing.B) {
	keys := channel.Keys{1}
	dataTypes := []telem.DataType{telem.Int32T}

	// Create 100 small contiguous series
	seriesKeys := make(channel.Keys, 100)
	seriesList := make([]telem.Series, 100)
	for i := 0; i < 100; i++ {
		seriesKeys[i] = 1
		s := telem.NewSeriesV(int32(i*10), int32(i*10+1), int32(i*10+2))
		s.Alignment = telem.Alignment(i * 3)
		seriesList[i] = s
	}

	frame := frame.NewMulti(seriesKeys, seriesList)

	b.Run("Enabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes)
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})

	b.Run("Disabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes, codec.DisableAlignmentCompression())
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})
}

// Benchmark alignment compression with mixed contiguous/non-contiguous
func BenchmarkAlignmentCompression_MixedContiguity(b *testing.B) {
	keys := channel.Keys{1}
	dataTypes := []telem.DataType{telem.Int32T}

	// Create 50 series: alternating contiguous groups and gaps
	seriesKeys := make(channel.Keys, 50)
	seriesList := make([]telem.Series, 50)
	alignment := telem.Alignment(0)
	for i := 0; i < 50; i++ {
		seriesKeys[i] = 1
		s := telem.NewSeriesV(int32(i*10), int32(i*10+1))
		s.Alignment = alignment
		seriesList[i] = s

		// Every 5 series, add a gap
		if (i+1)%5 == 0 {
			alignment += 10 // Gap
		} else {
			alignment += 2 // Contiguous
		}
	}

	frame := frame.NewMulti(seriesKeys, seriesList)

	b.Run("Enabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes)
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})

	b.Run("Disabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes, codec.DisableAlignmentCompression())
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})
}

// Benchmark alignment compression with multiple channels
func BenchmarkAlignmentCompression_MultiChannel(b *testing.B) {
	keys := channel.Keys{1, 2, 3}
	dataTypes := []telem.DataType{telem.Int32T, telem.Float32T, telem.Uint8T}

	// Create contiguous series for each channel
	seriesKeys := make(channel.Keys, 60) // 20 series per channel
	seriesList := make([]telem.Series, 60)

	for ch := 0; ch < 3; ch++ {
		alignment := telem.Alignment(ch * 100)
		for i := 0; i < 20; i++ {
			idx := ch*20 + i
			seriesKeys[idx] = channel.Key(ch + 1)

			var s telem.Series
			switch ch {
			case 0:
				s = telem.NewSeriesV(int32(i), int32(i+1))
			case 1:
				s = telem.NewSeriesV(float32(i), float32(i+1))
			case 2:
				s = telem.NewSeriesV(uint8(i), uint8(i+1))
			}
			s.Alignment = alignment
			alignment += 2
			seriesList[idx] = s
		}
	}

	frame := frame.NewMulti(seriesKeys, seriesList)

	b.Run("Enabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes)
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})

	b.Run("Disabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes, codec.DisableAlignmentCompression())
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})
}

// Benchmark bandwidth savings - measure encoded size
func BenchmarkAlignmentCompression_BandwidthSavings(b *testing.B) {
	keys := channel.Keys{1}
	dataTypes := []telem.DataType{telem.Int32T}

	// Create 100 small contiguous series
	seriesKeys := make(channel.Keys, 100)
	seriesList := make([]telem.Series, 100)
	for i := 0; i < 100; i++ {
		seriesKeys[i] = 1
		s := telem.NewSeriesV(int32(i*10), int32(i*10+1), int32(i*10+2))
		s.Alignment = telem.Alignment(i * 3)
		seriesList[i] = s
	}

	frame := frame.NewMulti(seriesKeys, seriesList)

	b.Run("Enabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes)
		encoded, err := cd.Encode(b.Context(), frame)
		if err != nil {
			b.Fatalf("failed to encode: %v", err)
		}
		b.ReportMetric(float64(len(encoded)), "bytes")
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})

	b.Run("Disabled", func(b *testing.B) {
		cd := codec.NewStatic(keys, dataTypes, codec.DisableAlignmentCompression())
		encoded, err := cd.Encode(b.Context(), frame)
		if err != nil {
			b.Fatalf("failed to encode: %v", err)
		}
		b.ReportMetric(float64(len(encoded)), "bytes")
		for b.Loop() {
			if _, err := cd.Encode(b.Context(), frame); err != nil {
				b.Fatalf("failed to encode: %v", err)
			}
		}
	})
}
