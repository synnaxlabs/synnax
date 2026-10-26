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
	"bytes"
	"fmt"
	"math"
	"math/bits"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

const (
	tdmsTocHeader uint32 = 0b0110
	tdmsTocData   uint32 = 0b1110
)

// tdmsSampleSizes holds the byte size of one sample of each fixed-size TDMS type.
var tdmsSampleSizes = map[uint32]uint64{
	0x01: 1, 0x02: 2, 0x03: 4, 0x04: 8,
	0x05: 1, 0x06: 2, 0x07: 4, 0x08: 8,
	0x09: 4, 0x0A: 8, 0x21: 1, 0x44: 16,
}

type tdmsObject struct {
	path       string
	properties map[string]string
	dataType   uint32
	count      uint64
	data       []byte
}

type tdmsSegment struct {
	toc     uint32
	objects []tdmsObject
}

// parseTDMS splits a TDMS file into its segments, failing on any byte that breaks the
// format.
func parseTDMS(b []byte) []tdmsSegment {
	GinkgoHelper()
	le := telem.ByteOrder
	var segments []tdmsSegment
	for len(b) > 0 {
		Expect(len(b)).To(BeNumerically(">=", 28))
		Expect(string(b[:4])).To(Equal("TDSm"))
		Expect(le.Uint32(b[8:])).To(Equal(uint32(4713)))
		seg := tdmsSegment{toc: le.Uint32(b[4:])}
		next, rawOffset := le.Uint64(b[12:]), le.Uint64(b[20:])
		meta, raw := b[28:28+rawOffset], b[28+rawOffset:28+next]
		b = b[28+next:]
		n := le.Uint32(meta)
		meta = meta[4:]
		for range n {
			size := le.Uint32(meta)
			o := tdmsObject{path: string(meta[4 : 4+size])}
			meta = meta[4+size:]
			index := le.Uint32(meta)
			meta = meta[4:]
			if index != 0xFFFFFFFF {
				o.dataType = le.Uint32(meta)
				Expect(le.Uint32(meta[4:])).To(Equal(uint32(1)))
				o.count = le.Uint64(meta[8:])
				meta = meta[16:]
				var size uint64
				if o.dataType == 0x20 {
					Expect(index).To(Equal(uint32(28)))
					size = le.Uint64(meta)
					meta = meta[8:]
				} else {
					Expect(index).To(Equal(uint32(20)))
					size = o.count * tdmsSampleSizes[o.dataType]
				}
				o.data, raw = raw[:size], raw[size:]
			}
			properties := le.Uint32(meta)
			meta = meta[4:]
			for range properties {
				if o.properties == nil {
					o.properties = map[string]string{}
				}
				size := le.Uint32(meta)
				name := string(meta[4 : 4+size])
				meta = meta[4+size:]
				Expect(le.Uint32(meta)).To(Equal(uint32(0x20)))
				size = le.Uint32(meta[4:])
				o.properties[name] = string(meta[8 : 8+size])
				meta = meta[8+size:]
			}
			seg.objects = append(seg.objects, o)
		}
		Expect(meta).To(BeEmpty())
		Expect(raw).To(BeEmpty())
		segments = append(segments, seg)
	}
	return segments
}

// tdmsStrings decodes the raw data of a string object.
func tdmsStrings(o tdmsObject) []string {
	GinkgoHelper()
	Expect(o.dataType).To(Equal(uint32(0x20)))
	text := o.data[4*o.count:]
	values := make([]string, o.count)
	var start uint32
	for i := range values {
		end := telem.ByteOrder.Uint32(o.data[4*i:])
		values[i], start = string(text[start:end]), end
	}
	return values
}

// tdmsTimeStamp encodes a fraction of a second in units of 2^-64 seconds and secs since
// the TDMS epoch as a TDMS timestamp.
func tdmsTimeStamp(fraction uint64, secs int64) []byte {
	b := telem.ByteOrder.AppendUint64(nil, fraction)
	return telem.ByteOrder.AppendUint64(b, uint64(secs))
}

// tdmsChannelData joins the samples a file holds for the object at path across all of
// its segments. It joins strings as text, each followed by a zero byte.
func tdmsChannelData(segments []tdmsSegment, path string) tdmsObject {
	GinkgoHelper()
	joined := tdmsObject{path: path}
	for _, seg := range segments {
		for _, o := range seg.objects {
			if o.path != path || o.count == 0 {
				continue
			}
			if joined.dataType != 0 {
				Expect(o.dataType).To(Equal(joined.dataType))
			}
			joined.dataType = o.dataType
			joined.count += o.count
			if o.dataType != 0x20 {
				joined.data = append(joined.data, o.data...)
				continue
			}
			for _, v := range tdmsStrings(o) {
				joined.data = append(append(joined.data, v...), 0)
			}
		}
	}
	return joined
}

// decodeTDMSTimeStamps decodes the raw data of a timestamp object the way a reader that
// rounds fractions down to nanoseconds does.
func decodeTDMSTimeStamps(o tdmsObject) []telem.TimeStamp {
	GinkgoHelper()
	Expect(o.dataType).To(Equal(uint32(0x44)))
	stamps := make([]telem.TimeStamp, o.count)
	for i := range stamps {
		b := o.data[16*i:]
		secs := int64(telem.ByteOrder.Uint64(b[8:])) - 2082844800
		nanos, _ := bits.Mul64(telem.ByteOrder.Uint64(b), 1e9)
		stamps[i] = telem.TimeStamp(secs*1e9 + int64(nanos))
	}
	return stamps
}

func tdmsPath(names ...string) string {
	path := ""
	for _, name := range names {
		path += fmt.Sprintf("/'%s'", name)
	}
	return path
}

// tdmsObjectAt returns the object at path in seg.
func tdmsObjectAt(seg tdmsSegment, path string) tdmsObject {
	GinkgoHelper()
	for _, o := range seg.objects {
		if o.path == path {
			return o
		}
	}
	Fail("segment has no object at " + path)
	return tdmsObject{}
}

// tdmsGroup returns the declaration of the group with the given name.
func tdmsGroup(name string) tdmsObject {
	return tdmsObject{
		path:       tdmsPath(name),
		properties: map[string]string{"wf_xcolumns": "one"},
	}
}

// beNaN matches a NaN float64.
func beNaN() types.GomegaMatcher { return Satisfy(math.IsNaN) }

// decodeFloat64s decodes the raw data of a float64 object.
func decodeFloat64s(o tdmsObject) []float64 {
	GinkgoHelper()
	Expect(o.dataType).To(Equal(uint32(0x0A)))
	values := make([]float64, o.count)
	for i := range values {
		values[i] = math.Float64frombits(telem.ByteOrder.Uint64(o.data[8*i:]))
	}
	return values
}

var _ = Describe("TDMS", func() {
	Describe("TDMSEncoder", func() {
		It("Should emit the TDMS media type", func() {
			Expect(framer.TDMSEncoder.ContentType()).To(Equal("application/x-tdms"))
		})

		It("Should declare the file, its groups, and its channels first", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T, telem.Int64T)
			b := MustSucceed(
				framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(parseTDMS(b)).To(Equal([]tdmsSegment{{
				toc: tdmsTocHeader,
				objects: []tdmsObject{
					{path: "/"},
					tdmsGroup(index.Name),
					tdmsGroup(channels[1].Name),
					{path: tdmsPath(index.Name, index.Name)},
					{path: tdmsPath(index.Name, channels[0].Name)},
					{path: tdmsPath(channels[1].Name, index.Name)},
					{path: tdmsPath(channels[1].Name, channels[1].Name)},
				},
			}}))
		})

		It("Should write a segment of the samples of each group", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T, telem.Int64T)
			floats := telem.NewSeriesV[float32](1.5, 2.25, 3)
			ints := telem.NewSeriesV[int64](10, 20, 30)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2, 3),
				channels,
				floats,
				ints,
			)
			b := MustSucceed(
				framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			segments := parseTDMS(b)
			Expect(segments).To(HaveLen(2))
			Expect(segments[1].toc).To(Equal(tdmsTocData))
			times := []byte(nil)
			for _, secs := range []int64{1, 2, 3} {
				times = append(times, tdmsTimeStamp(0, 2082844800+secs)...)
			}
			Expect(segments[1].objects).To(Equal([]tdmsObject{
				{
					path:     tdmsPath(index.Name, index.Name),
					dataType: 0x44,
					count:    3,
					data:     times,
				},
				{
					path:     tdmsPath(index.Name, channels[0].Name),
					dataType: 0x09,
					count:    3,
					data:     floats.Data,
				},
				{
					path:     tdmsPath(channels[1].Name, index.Name),
					dataType: 0x44,
					count:    3,
					data:     times,
				},
				{
					path:     tdmsPath(channels[1].Name, channels[1].Name),
					dataType: 0x04,
					count:    3,
					data:     ints.Data,
				},
			}))
		})

		DescribeTable(
			"Should write a timestamp from the TDMS epoch",
			func(
				ctx SpecContext,
				ts telem.TimeStamp,
				fraction uint64,
				secs int64,
			) {
				index, channels := createIndexed(ctx, telem.TimestampT)
				write(
					ctx,
					index,
					telem.NewSeriesSecondsTSV(1),
					channels,
					telem.NewSeriesV(ts),
				)
				b := MustSucceed(
					framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...)),
				)
				object := tdmsChannelData(
					parseTDMS(b),
					tdmsPath(channels[0].Name, channels[0].Name),
				)
				Expect(object.dataType).To(Equal(uint32(0x44)))
				Expect(object.data).To(Equal(tdmsTimeStamp(fraction, secs)))
				Expect(decodeTDMSTimeStamps(object)).To(Equal([]telem.TimeStamp{ts}))
			},
			Entry(
				"at the Unix epoch",
				telem.TimeStamp(0),
				uint64(0),
				int64(2082844800),
			),
			Entry(
				"half a second after the Unix epoch",
				telem.TimeStamp(500*telem.Millisecond),
				uint64(1)<<63,
				int64(2082844800),
			),
			Entry(
				"half a second before the Unix epoch",
				telem.TimeStamp(-500*telem.Millisecond),
				uint64(1)<<63,
				int64(2082844799),
			),
			Entry(
				"one nanosecond after the Unix epoch, rounding the fraction up",
				telem.TimeStamp(1),
				uint64(18446744074),
				int64(2082844800),
			),
			Entry(
				"one nanosecond before a whole second, rounding the fraction up",
				telem.TimeStamp(999_999_999),
				uint64(18446744055262807543),
				int64(2082844800),
			),
		)

		It("Should write text samples as TDMS strings", func(ctx SpecContext) {
			id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
			json := telem.NewSeriesV(`{"a":1}`, "[]")
			json.DataType = telem.JSONT
			index, channels := createIndexed(
				ctx,
				telem.StringT,
				telem.JSONT,
				telem.UUIDT,
				telem.BytesT,
			)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2),
				channels,
				telem.NewSeriesV("héllo", ""),
				json,
				telem.NewSeriesV(id, uuid.Nil()),
				telem.NewSeriesV([]byte{0, 255}, []byte{1}),
			)
			b := MustSucceed(
				framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			segment := parseTDMS(b)[1]
			textOf := func(ch svcchannel.Channel) []string {
				GinkgoHelper()
				return tdmsStrings(tdmsObjectAt(segment, tdmsPath(ch.Name, ch.Name)))
			}
			Expect(textOf(channels[0])).To(Equal([]string{"héllo", ""}))
			Expect(textOf(channels[1])).To(Equal([]string{`{"a":1}`, "[]"}))
			Expect(textOf(channels[2])).To(Equal([]string{
				id.String(),
				uuid.Nil().String(),
			}))
			Expect(textOf(channels[3])).To(Equal([]string{"AP8=", "AQ=="}))
		})

		It("Should write each index's channels to their own group", func(
			ctx SpecContext,
		) {
			firstIndex, first := createIndexed(ctx, telem.Float32T)
			secondIndex, second := createIndexed(ctx, telem.Int16T)
			write(
				ctx,
				firstIndex,
				telem.NewSeriesSecondsTSV(1, 3),
				first,
				telem.NewSeriesV[float32](1, 2),
			)
			write(
				ctx,
				secondIndex,
				telem.NewSeriesSecondsTSV(2),
				second,
				telem.NewSeriesV[int16](-3),
			)
			channels := append(first, second...)
			indexes := []svcchannel.Channel{firstIndex, secondIndex}
			b := MustSucceed(
				framer.TDMSEncoder.Encode(
					ctx,
					response(channels, indexes, svcframer.IteratorConfig{}),
				),
			)
			segments := parseTDMS(b)
			Expect(segments[0].objects).To(Equal([]tdmsObject{
				{path: "/"},
				tdmsGroup(firstIndex.Name),
				tdmsGroup(secondIndex.Name),
				tdmsGroup(second[0].Name),
				{path: tdmsPath(firstIndex.Name, firstIndex.Name)},
				{path: tdmsPath(firstIndex.Name, first[0].Name)},
				{path: tdmsPath(secondIndex.Name, secondIndex.Name)},
				{path: tdmsPath(second[0].Name, secondIndex.Name)},
				{path: tdmsPath(second[0].Name, second[0].Name)},
			}))
			firstPath := tdmsPath(firstIndex.Name, first[0].Name)
			Expect(tdmsChannelData(segments, firstPath)).To(Equal(tdmsObject{
				path:     firstPath,
				dataType: 0x09,
				count:    2,
				data:     telem.NewSeriesV[float32](1, 2).Data,
			}))
			secondPath := tdmsPath(second[0].Name, second[0].Name)
			Expect(tdmsChannelData(segments, secondPath)).To(Equal(tdmsObject{
				path:     secondPath,
				dataType: 0x02,
				count:    1,
				data:     telem.NewSeriesV[int16](-3).Data,
			}))
			for _, path := range []string{
				tdmsPath(secondIndex.Name, secondIndex.Name),
				tdmsPath(second[0].Name, secondIndex.Name),
			} {
				Expect(tdmsChannelData(segments, path).data).
					To(Equal(tdmsTimeStamp(0, 2082844802)))
			}
		})

		It("Should write every sample of a read that spans many segments", func(
			ctx SpecContext,
		) {
			const samples = 250_000
			index, channels := createIndexed(ctx, telem.Float64T, telem.StringT)
			var (
				timestamps = make([]telem.TimeStamp, samples)
				floats     = make([]float64, samples)
				strs       = make([]string, samples)
				text       []byte
			)
			for i := range samples {
				timestamps[i] = telem.TimeStamp(i+1) * 100 * telem.MicrosecondTS
				floats[i] = float64(i) / 3
				strs[i] = fmt.Sprintf("s%d", i)
				text = append(append(text, strs[i]...), 0)
			}
			write(
				ctx,
				index,
				telem.NewSeriesV(timestamps...),
				channels,
				telem.NewSeriesV(floats...),
				telem.NewSeriesV(strs...),
			)
			b := MustSucceed(
				framer.TDMSEncoder.Encode(
					ctx,
					response(
						channels,
						[]svcchannel.Channel{index},
						svcframer.IteratorConfig{ChunkSize: 10_000},
					),
				),
			)
			segments := parseTDMS(b)
			Expect(len(segments)).To(BeNumerically(">", 2))
			Expect(tdmsChannelData(
				segments,
				tdmsPath(index.Name, channels[0].Name),
			).data).To(Equal(telem.NewSeriesV(floats...).Data))
			Expect(tdmsChannelData(
				segments,
				tdmsPath(channels[1].Name, channels[1].Name),
			).data).To(Equal(text))
			for _, path := range []string{
				tdmsPath(index.Name, index.Name),
				tdmsPath(channels[1].Name, index.Name),
			} {
				Expect(decodeTDMSTimeStamps(tdmsChannelData(segments, path))).
					To(Equal(timestamps))
			}
		})

		It("Should keep each channel paired with its timestamps across segments", func(
			ctx SpecContext,
		) {
			const samples, half = 250_000, 125_000
			index, channels := createIndexed(ctx, telem.Float64T, telem.Int64T)
			var (
				timestamps = make([]telem.TimeStamp, samples)
				floats     = make([]float64, half)
				ints       = make([]int64, half)
			)
			for i := range samples {
				timestamps[i] = telem.TimeStamp(i+1) * 100 * telem.MicrosecondTS
			}
			for i := range half {
				floats[i], ints[i] = float64(i)/3, int64(i)-half
			}
			write(
				ctx,
				index,
				telem.NewSeriesV(timestamps[:half]...),
				channels[:1],
				telem.NewSeriesV(floats...),
			)
			write(
				ctx,
				index,
				telem.NewSeriesV(timestamps[half:]...),
				channels[1:],
				telem.NewSeriesV(ints...),
			)
			b := MustSucceed(
				framer.TDMSEncoder.Encode(
					ctx,
					response(
						channels,
						[]svcchannel.Channel{index},
						svcframer.IteratorConfig{ChunkSize: 10_000},
					),
				),
			)
			segments := parseTDMS(b)
			Expect(len(segments)).To(BeNumerically(">", 2))
			floatPath := tdmsPath(index.Name, channels[0].Name)
			for _, seg := range segments[1:] {
				Expect(tdmsObjectAt(seg, floatPath).count).
					To(Equal(tdmsObjectAt(seg, tdmsPath(index.Name, index.Name)).count))
			}
			values := decodeFloat64s(tdmsChannelData(segments, floatPath))
			Expect(values).To(HaveLen(samples))
			Expect(values[:half]).To(Equal(floats))
			Expect(values[half:]).To(HaveEach(beNaN()))
			Expect(decodeTDMSTimeStamps(tdmsChannelData(
				segments,
				tdmsPath(channels[1].Name, index.Name),
			))).To(Equal(timestamps[half:]))
			Expect(tdmsChannelData(
				segments,
				tdmsPath(channels[1].Name, channels[1].Name),
			).data).To(Equal(telem.NewSeriesV(ints...).Data))
		})

		It("Should write NaN where a float channel has no sample", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float64T, telem.Float32T)
			write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6), nil)
			writeData(ctx, channels[0], telem.SecondTS, telem.NewSeriesV[float64](1, 2))
			writeData(
				ctx,
				channels[0],
				telem.SecondTS*5,
				telem.NewSeriesV[float64](5, 6),
			)
			writeData(
				ctx,
				channels[1],
				telem.SecondTS*3,
				telem.NewSeriesV[float32](3),
			)
			b := MustSucceed(
				framer.TDMSEncoder.Encode(
					ctx,
					response(
						channels,
						[]svcchannel.Channel{index},
						svcframer.IteratorConfig{ChunkSize: 2},
					),
				),
			)
			segments := parseTDMS(b)
			Expect(decodeFloat64s(tdmsChannelData(
				segments,
				tdmsPath(index.Name, channels[0].Name),
			))).To(HaveExactElements(1.0, 2.0, beNaN(), beNaN(), 5.0, 6.0))
			nan := telem.ByteOrder.AppendUint32(nil, 0x7FC00000)
			Expect(tdmsChannelData(
				segments,
				tdmsPath(index.Name, channels[1].Name),
			)).To(Equal(tdmsObject{
				path:     tdmsPath(index.Name, channels[1].Name),
				dataType: 0x09,
				count:    6,
				data: slices.Concat(
					nan,
					nan,
					telem.NewSeriesV[float32](3).Data,
					nan,
					nan,
					nan,
				),
			}))
		})

		It("Should write the timestamp of each sample of a channel in its own group",
			func(ctx SpecContext) {
				index, channels := createIndexed(
					ctx,
					telem.Int32T,
					telem.StringT,
					telem.TimestampT,
				)
				write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3, 4), nil)
				writeData(
					ctx,
					channels[0],
					telem.SecondTS*3,
					telem.NewSeriesV[int32](3, 4),
				)
				writeData(ctx, channels[1], telem.SecondTS, telem.NewSeriesV("a", "b"))
				stamps := telem.NewSeriesV[telem.TimeStamp](-1, 1, telem.SecondTS)
				writeData(ctx, channels[2], telem.SecondTS*2, stamps)
				segments := parseTDMS(MustSucceed(
					framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...)),
				))
				Expect(decodeTDMSTimeStamps(tdmsChannelData(
					segments,
					tdmsPath(channels[0].Name, index.Name),
				))).To(Equal([]telem.TimeStamp{telem.SecondTS * 3, telem.SecondTS * 4}))
				Expect(tdmsChannelData(
					segments,
					tdmsPath(channels[0].Name, channels[0].Name),
				).data).To(Equal(telem.NewSeriesV[int32](3, 4).Data))
				Expect(decodeTDMSTimeStamps(tdmsChannelData(
					segments,
					tdmsPath(channels[1].Name, index.Name),
				))).To(Equal([]telem.TimeStamp{telem.SecondTS, telem.SecondTS * 2}))
				Expect(tdmsChannelData(
					segments,
					tdmsPath(channels[1].Name, channels[1].Name),
				).data).To(Equal([]byte("a\x00b\x00")))
				Expect(decodeTDMSTimeStamps(tdmsChannelData(
					segments,
					tdmsPath(channels[2].Name, index.Name),
				))).To(Equal([]telem.TimeStamp{
					telem.SecondTS * 2,
					telem.SecondTS * 3,
					telem.SecondTS * 4,
				}))
				Expect(decodeTDMSTimeStamps(tdmsChannelData(
					segments,
					tdmsPath(channels[2].Name, channels[2].Name),
				))).To(Equal([]telem.TimeStamp{-1, 1, telem.SecondTS}))
			},
		)

		It("Should pair a channel's samples with each index domain that holds them",
			func(ctx SpecContext) {
				index, channels := createIndexed(ctx, telem.Float64T, telem.Uint8T)
				write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3), nil)
				write(ctx, index, telem.NewSeriesSecondsTSV(10, 11, 12), nil)
				for _, start := range []telem.TimeStamp{2, 10} {
					writeData(
						ctx,
						channels[0],
						telem.SecondTS*start,
						telem.NewSeriesV(float64(start), float64(start+1)),
					)
					writeData(
						ctx,
						channels[1],
						telem.SecondTS*start,
						telem.NewSeriesV(uint8(start), uint8(start+1)),
					)
				}
				segments := parseTDMS(MustSucceed(
					framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...)),
				))
				Expect(decodeFloat64s(tdmsChannelData(
					segments,
					tdmsPath(index.Name, channels[0].Name),
				))).To(HaveExactElements(beNaN(), 2.0, 3.0, 10.0, 11.0, beNaN()))
				Expect(decodeTDMSTimeStamps(tdmsChannelData(
					segments,
					tdmsPath(channels[1].Name, index.Name),
				))).To(Equal([]telem.TimeStamp{
					telem.SecondTS * 2,
					telem.SecondTS * 3,
					telem.SecondTS * 10,
					telem.SecondTS * 11,
				}))
				Expect(tdmsChannelData(
					segments,
					tdmsPath(channels[1].Name, channels[1].Name),
				).data).To(Equal([]byte{2, 3, 10, 11}))
			},
		)

		It("Should pair a downsampled channel's samples with its index", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float64T, telem.Int16T)
			write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6), nil)
			writeData(
				ctx,
				channels[0],
				telem.SecondTS*2,
				telem.NewSeriesV[float64](2, 3, 4, 5, 6),
			)
			writeData(
				ctx,
				channels[1],
				telem.SecondTS*2,
				telem.NewSeriesV[int16](2, 3, 4, 5, 6),
			)
			segments := parseTDMS(MustSucceed(
				framer.TDMSEncoder.Encode(
					ctx,
					response(
						channels,
						[]svcchannel.Channel{index},
						svcframer.IteratorConfig{
							Reduction: telem.Reduction{
								Variant: telem.StrideReduction{Factor: 2},
							},
						},
					),
				),
			))
			Expect(decodeTDMSTimeStamps(tdmsChannelData(
				segments,
				tdmsPath(index.Name, index.Name),
			))).To(Equal([]telem.TimeStamp{
				telem.SecondTS,
				telem.SecondTS * 3,
				telem.SecondTS * 5,
			}))
			Expect(decodeFloat64s(tdmsChannelData(
				segments,
				tdmsPath(index.Name, channels[0].Name),
			))).To(HaveExactElements(beNaN(), 3.0, 5.0))
			Expect(decodeTDMSTimeStamps(tdmsChannelData(
				segments,
				tdmsPath(channels[1].Name, index.Name),
			))).To(Equal([]telem.TimeStamp{telem.SecondTS * 3, telem.SecondTS * 5}))
			Expect(tdmsChannelData(
				segments,
				tdmsPath(channels[1].Name, channels[1].Name),
			).data).To(Equal(telem.NewSeriesV[int16](3, 5).Data))
		})

		It("Should return an error when a sample has no index sample", func(
			ctx SpecContext,
		) {
			index, _ := createIndexed(ctx)
			write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3), nil)
			write(ctx, index, telem.NewSeriesSecondsTSV(10, 11, 12), nil)
			otherIndex, channels := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				otherIndex,
				telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8),
				nil,
			)
			writeData(
				ctx,
				channels[0],
				telem.SecondTS*6,
				telem.NewSeriesV[float32](6, 7, 8),
			)
			misindexed := channels[0]
			misindexed.LocalIndex = index.LocalKey
			Expect(framer.TDMSEncoder.Encode(
				ctx,
				response(
					[]svcchannel.Channel{misindexed},
					[]svcchannel.Channel{index},
					svcframer.IteratorConfig{},
				),
			)).Error().To(MatchError(fmt.Sprintf(
				"[api.Read] channel %s has a sample at alignment 0-5 with no "+
					"index sample",
				misindexed.Name,
			)))
		})

		It("Should write an index requested on its own", func(ctx SpecContext) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1),
				channels,
				telem.NewSeriesV[float32](1),
			)
			b := MustSucceed(
				framer.TDMSEncoder.Encode(
					ctx,
					response(
						[]svcchannel.Channel{index},
						nil,
						svcframer.IteratorConfig{},
					),
				),
			)
			segments := parseTDMS(b)
			Expect(segments[0].objects).To(Equal([]tdmsObject{
				{path: "/"},
				tdmsGroup(index.Name),
				{path: tdmsPath(index.Name, index.Name)},
			}))
			Expect(tdmsChannelData(segments, tdmsPath(index.Name, index.Name)).data).
				To(Equal(tdmsTimeStamp(0, 2082844801)))
		})

		It("Should double the quotes in a name and count its bytes", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			channels[0].Name = "it's é"
			b := MustSucceed(
				framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(parseTDMS(b)[0].objects[3].path).
				To(Equal(tdmsPath(index.Name, "it''s é")))
		})

		It("Should tell apart channels that share a name", func(ctx SpecContext) {
			index, channels := createIndexed(
				ctx,
				telem.Float32T,
				telem.Float32T,
				telem.Int64T,
			)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2),
				channels,
				telem.NewSeriesV[float32](1, 2),
				telem.NewSeriesV[float32](3, 4),
				telem.NewSeriesV[int64](5, 6),
			)
			channels[1].Name = channels[0].Name
			channels[2].Name = index.Name
			named := func(ch svcchannel.Channel) string {
				return fmt.Sprintf("%s (%s)", ch.Name, ch.Key())
			}
			var (
				indexGroup = named(index)
				ownGroup   = named(channels[2])
			)
			segments := parseTDMS(MustSucceed(framer.TDMSEncoder.Encode(
				ctx,
				response(
					channels,
					[]svcchannel.Channel{index},
					svcframer.IteratorConfig{},
				),
			)))
			Expect(segments[0].objects).To(Equal([]tdmsObject{
				{path: "/"},
				tdmsGroup(indexGroup),
				tdmsGroup(ownGroup),
				{path: tdmsPath(indexGroup, named(index))},
				{path: tdmsPath(indexGroup, named(channels[0]))},
				{path: tdmsPath(indexGroup, named(channels[1]))},
				{path: tdmsPath(ownGroup, named(index))},
				{path: tdmsPath(ownGroup, named(channels[2]))},
			}))
			for i, data := range [][]byte{
				telem.NewSeriesV[float32](1, 2).Data,
				telem.NewSeriesV[float32](3, 4).Data,
			} {
				path := tdmsPath(indexGroup, named(channels[i]))
				Expect(tdmsChannelData(segments, path).data).To(Equal(data))
			}
			Expect(tdmsChannelData(
				segments,
				tdmsPath(ownGroup, named(channels[2])),
			).data).To(Equal(telem.NewSeriesV[int64](5, 6).Data))
		})

		It("Should fail a read where two channels still take the same name", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(
				ctx,
				telem.Float32T,
				telem.Float32T,
				telem.Float32T,
			)
			channels[1].Name = channels[0].Name
			channels[2].Name = fmt.Sprintf(
				"%s (%s)",
				channels[0].Name,
				channels[0].Key(),
			)
			Expect(framer.TDMSEncoder.Encode(
				ctx,
				response(
					channels,
					[]svcchannel.Channel{index},
					svcframer.IteratorConfig{},
				),
			)).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(fmt.Sprintf(
					"channels %s and %s both take the name %s in a TDMS file",
					channels[0].Key(),
					channels[2].Key(),
					channels[2].Name,
				))),
			))
		})

		It("Should fail a read that leaves out the index of a channel", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1),
				channels,
				telem.NewSeriesV[float32](1),
			)
			Expect(framer.TDMSEncoder.Encode(
				ctx,
				response(channels, nil, svcframer.IteratorConfig{}),
			)).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(channels[0].Name)),
			))
		})

		It("Should fail a limited read before the first byte", func(ctx SpecContext) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2, 3, 4),
				channels,
				telem.NewSeriesV[float32](1, 2, 3, 4),
			)
			res := response(
				channels,
				[]svcchannel.Channel{index},
				svcframer.IteratorConfig{
					Reduction: telem.Reduction{
						Variant: telem.LimitReduction{
							Aggregation: telem.AggregationDecimate,
							PointLimit:  2,
						},
					},
				},
			)
			var buf bytes.Buffer
			Expect(framer.TDMSEncoder.EncodeStream(ctx, &buf, res)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("cannot take a limit reduction")),
			))
			Expect(buf.Len()).To(BeZero())
		})

		It("Should return an error for a data type with no TDMS form", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			channels[0].DataType = telem.UnknownT
			Expect(framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...))).
				Error().To(MatchError(ContainSubstring("has no TDMS form")))
		})

		It("Should return an error when the value is not a read response", func(
			ctx SpecContext,
		) {
			Expect(framer.TDMSEncoder.Encode(ctx, svcchannel.Channel{})).
				Error().To(MatchError(ContainSubstring("cannot encode a value")))
		})
	})
})
