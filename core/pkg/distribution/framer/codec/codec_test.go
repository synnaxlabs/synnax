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
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"
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
		ctx SpecContext,
		channels channel.Keys,
		dataTypes []telem.DataType,
		fr framer.Frame,
	) {
		cdc := codec.NewStatic(channels, dataTypes)
		encoded := MustSucceed(cdc.Encode(ctx, fr))
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
					telem.NewSeriesV("cat", "dog"),
					MustSucceed(telem.NewJSONSeriesV(
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
		It("Should correctly serialize and deserialize a complex frame", func(ctx SpecContext) {
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
			s3 := telem.NewSeriesV("cat", "dog", "rabbit", "frog")
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

	Describe("Multiple Contiguous Series Per Channel (throttle pattern)", func() {
		It("Should preserve alignment when a channel has multiple contiguous series", func(ctx SpecContext) {
			keys := channel.Keys{1, 2}
			dataTypes := []telem.DataType{telem.Int64T, telem.Int64T}
			// Channel 1 carries two contiguous series in the same leading domain, which
			// is what the accumulation throttle produces when it batches several writes
			// into one streamed frame.
			s1a := telem.NewSeriesV[int64](1, 2)
			s1a.Alignment = cesium.LeadingAlignment(4, 100) // [100, 102)
			s1b := telem.NewSeriesV[int64](3, 4)
			s1b.Alignment = cesium.LeadingAlignment(4, 102) // [102, 104)
			s2 := telem.NewSeriesV[int64](5, 6)
			s2.Alignment = cesium.LeadingAlignment(4, 100)
			fr := frame.NewMulti(channel.Keys{1, 1, 2}, []telem.Series{s1a, s1b, s2})

			cdc := codec.NewStatic(keys, dataTypes)
			decoded := MustSucceed(cdc.Decode(MustSucceed(cdc.Encode(ctx, fr))))

			var ch1 []telem.Series
			for k, s := range decoded.Entries() {
				if k == 1 {
					ch1 = append(ch1, s)
				}
			}
			slices.SortFunc(ch1, func(a, b telem.Series) int {
				switch {
				case a.Alignment < b.Alignment:
					return -1
				case a.Alignment > b.Alignment:
					return 1
				default:
					return 0
				}
			})
			// The decoded series for channel 1 must cover [100, 104) with no overlap
			// and no gap. A backward-overlap bug shows up as prev.Upper > next.Lower.
			Expect(ch1[0].AlignmentBounds().Lower).To(Equal(cesium.LeadingAlignment(4, 100)))
			var total int64
			for i, s := range ch1 {
				total += s.Len()
				if i > 0 {
					Expect(ch1[i-1].AlignmentBounds().Upper).To(
						Equal(s.AlignmentBounds().Lower),
					)
				}
			}
			Expect(total).To(Equal(int64(4)))
			Expect(ch1[len(ch1)-1].AlignmentBounds().Upper).To(
				Equal(cesium.LeadingAlignment(4, 104)),
			)
		})
	})

	Describe("Error Handling", func() {
		It("Should return a validation error when a series has the wrong data type", func(ctx SpecContext) {
			c := codec.NewStatic(
				[]channel.Key{1},
				[]telem.DataType{telem.Uint8T},
			)
			fr := frame.NewUnary(1, telem.NewSeriesSecondsTSV(1, 2, 3))
			encoded, err := c.Encode(ctx, fr)
			Expect(encoded).To(BeEmpty())
			Expect(err).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Int64 / TimeStamp Equivalence", func() {
		It("Should accept an int64 series for a timestamp channel", func(ctx SpecContext) {
			c := codec.NewStatic(
				[]channel.Key{1},
				[]telem.DataType{telem.TimeStampT},
			)
			fr := frame.NewUnary(1, telem.NewSeriesV[int64](1778020940471336961))
			encoded := MustSucceed(c.Encode(ctx, fr))
			Expect(encoded).ToNot(BeEmpty())
		})

		It("Should accept a timestamp series for an int64 channel", func(ctx SpecContext) {
			c := codec.NewStatic(
				[]channel.Key{1},
				[]telem.DataType{telem.Int64T},
			)
			fr := frame.NewUnary(1, telem.NewSeriesSecondsTSV(1, 2, 3))
			encoded := MustSucceed(c.Encode(ctx, fr))
			Expect(encoded).ToNot(BeEmpty())
		})
	})

	Describe("Dynamic Codec", Ordered, func() {
		ShouldNotLeakGoroutinesPerSpec()
		var (
			node       mock.Node
			channelSvc *channel.Service
			idxCh      channel.Channel
			dataCh     channel.Channel
		)
		BeforeAll(func(ctx SpecContext) {
			node = mock.MustOpenNode(ctx)
			channelSvc = node.Channel
			w := node.Channel.NewWriter(nil)
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

		It("Should allow the caller to update the list of channels", func(ctx SpecContext) {
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

			It("Should return true if update has been called on the codec at least once", func(ctx SpecContext) {
				codec := codec.NewDynamic(channelSvc)
				Expect(codec.Update(ctx, []channel.Key{dataCh.Key(), idxCh.Key()})).To(Succeed())
				Expect(codec.Initialized()).To(BeTrue())
			})
		})

		It("Should not mutate the caller's keys slice when updating", func(ctx SpecContext) {
			c := codec.NewDynamic(channelSvc)
			keys := []channel.Key{dataCh.Key(), idxCh.Key()}
			original := make([]channel.Key, len(keys))
			copy(original, keys)
			Expect(c.Update(ctx, keys)).To(Succeed())
			Expect(keys).To(Equal(original))
		})

		It("Should panic if the codec is not initialized", func(ctx SpecContext) {
			codec := codec.NewDynamic(nil)
			Expect(func() {
				fr := framer.Frame{}
				_, _ = codec.Encode(ctx, fr)
			}).To(Panic())
		})

		It("Should use the correct encode/decode state even if the codecs are out of sync", func(ctx SpecContext) {
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
			Expect(decoded.Frame.SeriesSlice()).To(BeEmpty())

			frame2 := frame.NewUnary(dataCh.Key(), telem.NewSeriesV[float32](1, 2, 3, 4))
			encoded = MustSucceed(encoder.Encode(ctx, frame2))
			decoded = MustSucceed(decoder.Decode(encoded))
			Expect(decoded.Frame).To(telem.MatchFrame(frame2.Frame))
		})

		It("Should preserve alignment across rapid key churn (schematic pattern)", func(ctx SpecContext) {
			enc := codec.NewDynamic(channelSvc)
			dec := codec.NewDynamic(channelSvc)
			only := channel.Keys{dataCh.Key()}
			both := channel.Keys{dataCh.Key(), idxCh.Key()}
			Expect(enc.Update(ctx, only)).To(Succeed())
			Expect(dec.Update(ctx, only)).To(Succeed())

			var (
				prevEnd telem.Alignment
				hasPrev bool
				sample  uint32
			)
			for i := range 60 {
				// Churn the streamed key set the way mounting/unmounting schematic
				// symbols does. The decoder is updated a beat before the encoder so a
				// frame is sometimes encoded under a key set the decoder has already
				// moved past (the seqNum-backlog path).
				switch i % 4 {
				case 0:
					Expect(dec.Update(ctx, both)).To(Succeed())
					Expect(enc.Update(ctx, both)).To(Succeed())
				case 2:
					Expect(dec.Update(ctx, only)).To(Succeed())
				case 3:
					Expect(enc.Update(ctx, only)).To(Succeed())
				}
				s := telem.NewSeriesV[float32](float32(i))
				s.Alignment = cesium.LeadingAlignment(1, sample)
				fr := frame.NewUnary(dataCh.Key(), s)
				decoded := MustSucceed(dec.Decode(MustSucceed(enc.Encode(ctx, fr))))
				for k, ds := range decoded.Entries() {
					if k != dataCh.Key() {
						continue
					}
					if hasPrev {
						Expect(ds.AlignmentBounds().Lower).To(Equal(prevEnd))
					}
					prevEnd = ds.AlignmentBounds().Upper
					hasPrev = true
				}
				sample++
			}
		})

		// This test is a regression that ensures the codec is designed to handle
		// race conditions between the encoding side and an upstream go-routines
		// producing frames. Even if an upstream routine passes a frame to the encoder
		// that contains keys that are not in the current state, they should be properly
		// ignored.
		Describe("Delayed Frames", func() {
			Context("Empty Result", func() {
				It("Should work correctly when a 'delayed' frame is provided ot the codec", func(ctx SpecContext) {
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
					Expect(decoded.Frame.KeysSlice()).To(BeEmpty())
				})
			})

			Context("Non-Empty Result", func() {
				It("Should work correctly when a 'delayed' frame is provided ot the codec", func(ctx SpecContext) {
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
		It("Should correctly handle encoding frames of varying sizes sequentially", func(ctx SpecContext) {
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
		It("Should correctly sort and encode frames with duplicate channel keys", func(ctx SpecContext) {
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
			Expect(ch10Series.Series).To(HaveLen(3))
			ch20Series := decoded.Get(20)
			Expect(ch20Series.Series).To(HaveLen(2))
			ch30Series := decoded.Get(30)
			Expect(ch30Series.Series).To(HaveLen(2))

			Expect(frame.Frame).To(telem.MatchFrame(decoded.Frame))
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle frames with very large channel key values", func(ctx SpecContext) {
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

		It("Should handle encoding after an empty frame (sorter reset edge case)", func(ctx SpecContext) {
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

		It("Should handle single channel frame after multi-channel frame", func(ctx SpecContext) {
			keys := channel.Keys{100, 200, 300}
			dataTypes := []telem.DataType{telem.Int64T, telem.Float64T, telem.StringT}
			codec := codec.NewStatic(keys, dataTypes)

			multiFrame := frame.NewMulti(
				channel.Keys{300, 100, 200},
				[]telem.Series{
					telem.NewSeriesV("hello", "world"),
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
		It("Should merge two contiguous series for the same channel", func(ctx SpecContext) {
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
			Expect(series.Series).To(HaveLen(1))
			mergedData := telem.UnmarshalSeries[int32](series.Series[0])
			Expect(mergedData).To(Equal([]int32{1, 2, 3, 4, 5}))

			// Verify alignment is from the first series
			Expect(series.Series[0].Alignment).To(Equal(telem.Alignment(0)))
		})

		It("Should merge three contiguous series for the same channel", func(ctx SpecContext) {
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
			Expect(series.Series).To(HaveLen(1))
			mergedData := telem.UnmarshalSeries[uint8](series.Series[0])
			Expect(mergedData).To(Equal([]uint8{1, 2, 3, 4, 5, 6}))
		})

		It("Should not merge non-contiguous series for the same channel", func(ctx SpecContext) {
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
			Expect(series.Series).To(HaveLen(2))
		})

		It("Should handle mixed contiguous and non-contiguous series", func(ctx SpecContext) {
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
			Expect(series.Series).To(HaveLen(2))

			// First merged series should be [1, 2, 3, 4]
			firstData := telem.UnmarshalSeries[int32](series.Series[0])
			Expect(firstData).To(Equal([]int32{1, 2, 3, 4}))

			// Second merged series should be [5, 6]
			secondData := telem.UnmarshalSeries[int32](series.Series[1])
			Expect(secondData).To(Equal([]int32{5, 6}))
		})

		It("Should merge series for multiple channels independently", func(ctx SpecContext) {
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
			Expect(ch1Series.Series).To(HaveLen(1))
			ch1Data := telem.UnmarshalSeries[int32](ch1Series.Series[0])
			Expect(ch1Data).To(Equal([]int32{1, 2, 3, 4}))

			// Channel 2 should have merged series
			ch2Series := decoded.Get(2)
			Expect(ch2Series.Series).To(HaveLen(1))
			ch2Data := telem.UnmarshalSeries[float32](ch2Series.Series[0])
			Expect(ch2Data).To(Equal([]float32{1.1, 2.2, 3.3, 4.4}))
		})

		It("Should merge series with zero alignments", func(ctx SpecContext) {
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

		It("Should handle time range extension when merging", func(ctx SpecContext) {
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
			Expect(series.Series).To(HaveLen(1))

			// Time range should span both series
			mergedSeries := series.Series[0]
			Expect(mergedSeries.TimeRange.Start).To(Equal(telem.TimeStamp(100)))
			Expect(mergedSeries.TimeRange.End).To(Equal(telem.TimeStamp(300)))
		})

		It("Should preserve variable-density types when merging", func(ctx SpecContext) {
			keys := channel.Keys{1}
			dataTypes := []telem.DataType{telem.StringT}
			codec := codec.NewStatic(keys, dataTypes)

			s1 := telem.NewSeriesV("hello", "world")
			s1.Alignment = 0

			s2 := telem.NewSeriesV("foo")
			s2.Alignment = 2

			frame := frame.NewMulti(
				channel.Keys{1, 1},
				[]telem.Series{s1, s2},
			)

			encoded := MustSucceed(codec.Encode(ctx, frame))
			decoded := MustSucceed(codec.Decode(encoded))

			Expect(decoded.Count()).To(Equal(1))
			series := decoded.Get(1)
			Expect(series.Series).To(HaveLen(1))

			// Data should be concatenated correctly
			mergedStrings := telem.UnmarshalSeries[string](series.Series[0])
			Expect(mergedStrings).To(Equal([]string{"hello", "world", "foo"}))
		})
	})

	Describe("All Channels Present Flag", func() {
		It("Should not set the flag when the merged series count matches the state but the keys do not correspond 1-to-1", func(ctx SpecContext) {
			// Regression for SY-3556: multi-domain iterator frames produce multiple
			// series for a single channel. When that count happens to equal the number
			// of channels in the codec state, the encoder used to set
			// allChannelsPresent=true and skip per-series keys, causing the decoder to
			// read each series under the wrong state key.
			keys := channel.Keys{1, 2}
			dataTypes := []telem.DataType{telem.Int32T, telem.Float64T}
			cd := codec.NewStatic(keys, dataTypes)

			// Two series but BOTH for channel 2 (channel 1 has no data in this frame).
			// len(merged) == len(state.keys) == 2, but the keys don't match the state
			// ordering.
			s1 := telem.NewSeriesV[float64](10)
			s1.Alignment = 0
			s2 := telem.NewSeriesV[float64](20, 21)
			s2.Alignment = 100 // gap forces non-merged
			fr := frame.NewMulti(channel.Keys{2, 2}, []telem.Series{s1, s2})

			encoded := MustSucceed(cd.Encode(ctx, fr))
			decoded := MustSucceed(cd.Decode(encoded))

			Expect(decoded.KeysSlice()).To(Equal([]channel.Key{2, 2}))
			Expect(decoded.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[float64](10)))
			Expect(decoded.SeriesAt(1)).To(telem.MatchSeriesData(telem.NewSeriesV[float64](20, 21)))
		})

		It("Should round-trip a multi-domain frame that spans every state channel", func(ctx SpecContext) {
			// The iterator's typical "across-domains" shape: each channel appears twice
			// (one series per domain). len(merged) = 4 != 2 so allChannelsPresent must
			// be false even before this fix, but the decoder still has to walk
			// per-series keys correctly.
			keys := channel.Keys{1, 2}
			dataTypes := []telem.DataType{telem.TimeStampT, telem.Float64T}
			cd := codec.NewStatic(keys, dataTypes)

			idxA := telem.NewSeriesSecondsTSV(3)
			idxA.Alignment = 1
			datA := telem.NewSeriesV[float64](3)
			datA.Alignment = 1
			idxB := telem.NewSeriesSecondsTSV(101, 102)
			idxB.Alignment = 100
			datB := telem.NewSeriesV[float64](10, 11)
			datB.Alignment = 100

			fr := frame.NewMulti(
				channel.Keys{1, 2, 1, 2},
				[]telem.Series{idxA, datA, idxB, datB},
			)
			encoded := MustSucceed(cd.Encode(ctx, fr))
			decoded := MustSucceed(cd.Decode(encoded))

			Expect(decoded.Count()).To(Equal(4))
			idx := decoded.Get(1)
			Expect(idx.Series).To(HaveLen(2))
			Expect(idx.Series[0]).To(telem.MatchSeriesData(idxA))
			Expect(idx.Series[1]).To(telem.MatchSeriesData(idxB))
			dat := decoded.Get(2)
			Expect(dat.Series).To(HaveLen(2))
			Expect(dat.Series[0]).To(telem.MatchSeriesData(datA))
			Expect(dat.Series[1]).To(telem.MatchSeriesData(datB))
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
	for i := range 100 {
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
	for i := range 50 {
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

	for ch := range 3 {
		alignment := telem.Alignment(ch * 100)
		for i := range 20 {
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
	for i := range 100 {
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

// makeStreamerFrame builds a "streamer-shaped" frame for a codec state of
// numChannels channels (Float32). With present=numChannels every state channel
// is in the frame and the new allChannelsPresent ordering check (SY-3556) walks
// every key — worst case for the added work. With present<numChannels the
// length check fails and the ordering loop is skipped.
func makeStreamerFrame(numChannels, present, samplesPerSeries int) (
	channel.Keys, []telem.DataType, framer.Frame,
) {
	stateKeys := make(channel.Keys, numChannels)
	dataTypes := make([]telem.DataType, numChannels)
	for i := range numChannels {
		stateKeys[i] = channel.Key(i + 1)
		dataTypes[i] = telem.Float32T
	}

	frameKeys := make(channel.Keys, present)
	seriesList := make([]telem.Series, present)
	data := make([]float32, samplesPerSeries)
	for i := range samplesPerSeries {
		data[i] = float32(i)
	}
	for i := range present {
		frameKeys[i] = stateKeys[i]
		seriesList[i] = telem.NewSeries(data)
	}
	return stateKeys, dataTypes, frame.NewMulti(frameKeys, seriesList)
}

// makeIteratorFrame builds an "iterator-shaped" frame spanning multiple time
// domains: each of numChannels channels appears numDomains times (one series
// per domain). This is the exact shape the iterator codec path now produces
// and the shape that exposed the allChannelsPresent regression SY-3556 fixes.
func makeIteratorFrame(numChannels, numDomains, samplesPerDomain int) (
	channel.Keys, []telem.DataType, framer.Frame,
) {
	stateKeys := make(channel.Keys, numChannels)
	dataTypes := make([]telem.DataType, numChannels)
	for i := range numChannels {
		stateKeys[i] = channel.Key(i + 1)
		dataTypes[i] = telem.Float32T
	}

	totalSeries := numChannels * numDomains
	frameKeys := make(channel.Keys, totalSeries)
	seriesList := make([]telem.Series, totalSeries)
	data := make([]float32, samplesPerDomain)
	for i := range samplesPerDomain {
		data[i] = float32(i)
	}
	idx := 0
	for d := range numDomains {
		alignment := telem.Alignment(d * 1_000_000)
		tr := telem.TimeStamp(d).SpanRange(telem.Second)
		for c := range numChannels {
			frameKeys[idx] = stateKeys[c]
			s := telem.NewSeries(data)
			s.Alignment = alignment
			s.TimeRange = tr
			seriesList[idx] = s
			idx++
		}
	}
	return stateKeys, dataTypes, frame.NewMulti(frameKeys, seriesList)
}

func BenchmarkStreamerFrame_Encode(b *testing.B) {
	for _, nc := range []int{1, 8, 64, 256} {
		b.Run(fmt.Sprintf("channels=%d/allFull", nc), func(b *testing.B) {
			keys, dataTypes, fr := makeStreamerFrame(nc, nc, 100)
			cd := codec.NewStatic(keys, dataTypes)
			w := bytes.NewBuffer(nil)
			b.ReportAllocs()
			for b.Loop() {
				if err := cd.EncodeStream(b.Context(), w, fr); err != nil {
					b.Fatalf("encode: %v", err)
				}
				w.Reset()
			}
		})
		if nc > 1 {
			b.Run(fmt.Sprintf("channels=%d/partial", nc), func(b *testing.B) {
				keys, dataTypes, fr := makeStreamerFrame(nc, nc-1, 100)
				cd := codec.NewStatic(keys, dataTypes)
				w := bytes.NewBuffer(nil)
				b.ReportAllocs()
				for b.Loop() {
					if err := cd.EncodeStream(b.Context(), w, fr); err != nil {
						b.Fatalf("encode: %v", err)
					}
					w.Reset()
				}
			})
		}
	}
}

func BenchmarkStreamerFrame_Decode(b *testing.B) {
	for _, nc := range []int{1, 8, 64, 256} {
		b.Run(fmt.Sprintf("channels=%d/allFull", nc), func(b *testing.B) {
			keys, dataTypes, fr := makeStreamerFrame(nc, nc, 100)
			cd := codec.NewStatic(keys, dataTypes)
			encoded, err := cd.Encode(b.Context(), fr)
			if err != nil {
				b.Fatalf("encode: %v", err)
			}
			r := bytes.NewReader(encoded)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := r.Seek(0, 0); err != nil {
					b.Fatalf("seek: %v", err)
				}
				if _, err := cd.DecodeStream(r); err != nil {
					b.Fatalf("decode: %v", err)
				}
			}
		})
	}
}

func BenchmarkIteratorFrame_Encode(b *testing.B) {
	cases := []struct {
		channels, domains, samples int
	}{
		{1, 1, 100},
		{1, 10, 100},
		{8, 1, 100},
		{8, 10, 100},
		{64, 10, 100},
		{8, 1, 10000},
	}
	for _, c := range cases {
		name := fmt.Sprintf("channels=%d/domains=%d/samples=%d", c.channels, c.domains, c.samples)
		b.Run(name, func(b *testing.B) {
			keys, dataTypes, fr := makeIteratorFrame(c.channels, c.domains, c.samples)
			cd := codec.NewStatic(keys, dataTypes)
			w := bytes.NewBuffer(nil)
			if err := cd.EncodeStream(b.Context(), w, fr); err != nil {
				b.Fatalf("encode: %v", err)
			}
			b.ReportMetric(float64(w.Len()), "bytes")
			b.ReportAllocs()
			for b.Loop() {
				w.Reset()
				if err := cd.EncodeStream(b.Context(), w, fr); err != nil {
					b.Fatalf("encode: %v", err)
				}
			}
		})
	}
}

func BenchmarkIteratorFrame_Decode(b *testing.B) {
	cases := []struct {
		channels, domains, samples int
	}{
		{1, 1, 100},
		{1, 10, 100},
		{8, 10, 100},
		{64, 10, 100},
	}
	for _, c := range cases {
		name := fmt.Sprintf("channels=%d/domains=%d/samples=%d", c.channels, c.domains, c.samples)
		b.Run(name, func(b *testing.B) {
			keys, dataTypes, fr := makeIteratorFrame(c.channels, c.domains, c.samples)
			cd := codec.NewStatic(keys, dataTypes)
			encoded, err := cd.Encode(b.Context(), fr)
			if err != nil {
				b.Fatalf("encode: %v", err)
			}
			r := bytes.NewReader(encoded)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := r.Seek(0, 0); err != nil {
					b.Fatalf("seek: %v", err)
				}
				if _, err := cd.DecodeStream(r); err != nil {
					b.Fatalf("decode: %v", err)
				}
			}
		})
	}
}

// BenchmarkIteratorFrame_RoundTrip mirrors the wire path for an iterator data
// response: encode on the server, decode on the client. SY-3556 promotes this
// from JSON/protobuf to the high-performance codec — round-trip cost here is
// what the PR is trading for.
func BenchmarkIteratorFrame_RoundTrip(b *testing.B) {
	keys, dataTypes, fr := makeIteratorFrame(8, 10, 100)
	enc := codec.NewStatic(keys, dataTypes)
	dec := codec.NewStatic(keys, dataTypes)
	w := bytes.NewBuffer(nil)
	b.ReportAllocs()
	for b.Loop() {
		w.Reset()
		if err := enc.EncodeStream(b.Context(), w, fr); err != nil {
			b.Fatalf("encode: %v", err)
		}
		r := bytes.NewReader(w.Bytes())
		if _, err := dec.DecodeStream(r); err != nil {
			b.Fatalf("decode: %v", err)
		}
	}
}
