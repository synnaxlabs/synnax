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
	"fmt"
	"math/bits"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	path     string
	dataType uint32
	count    uint64
	data     []byte
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
			Expect(le.Uint32(meta)).To(BeZero())
			meta = meta[4:]
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

var _ = Describe("TDMS", func() {
	Describe("TDMSEncoder", func() {
		It("Should emit the TDMS media type", func() {
			Expect(framer.TDMSEncoder.ContentType()).To(Equal("application/x-tdms"))
		})

		It("Should declare the file, its groups, and its channels first", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			b := MustSucceed(
				framer.TDMSEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(parseTDMS(b)).To(Equal([]tdmsSegment{{
				toc: tdmsTocHeader,
				objects: []tdmsObject{
					{path: "/"},
					{path: tdmsPath(index.Name)},
					{path: tdmsPath(index.Name, index.Name)},
					{path: tdmsPath(index.Name, channels[0].Name)},
				},
			}}))
		})

		It("Should write each frame as a segment of its samples", func(
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
			objects := segments[1].objects
			Expect(objects).To(HaveLen(3))
			Expect(objects[0].path).To(Equal(tdmsPath(index.Name, index.Name)))
			Expect(objects[0].dataType).To(Equal(uint32(0x44)))
			Expect(objects[0].count).To(Equal(uint64(3)))
			Expect(objects[1]).To(Equal(tdmsObject{
				path:     tdmsPath(index.Name, channels[0].Name),
				dataType: 0x09,
				count:    3,
				data:     floats.Data,
			}))
			Expect(objects[2]).To(Equal(tdmsObject{
				path:     tdmsPath(index.Name, channels[1].Name),
				dataType: 0x04,
				count:    3,
				data:     ints.Data,
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
				object := parseTDMS(b)[1].objects[1]
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
			objects := parseTDMS(b)[1].objects
			Expect(tdmsStrings(objects[1])).To(Equal([]string{"héllo", ""}))
			Expect(tdmsStrings(objects[2])).To(Equal([]string{`{"a":1}`, "[]"}))
			Expect(tdmsStrings(objects[3])).To(Equal([]string{
				id.String(),
				uuid.Nil().String(),
			}))
			Expect(tdmsStrings(objects[4])).To(Equal([]string{"AP8=", "AQ=="}))
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
				{path: tdmsPath(firstIndex.Name)},
				{path: tdmsPath(secondIndex.Name)},
				{path: tdmsPath(firstIndex.Name, firstIndex.Name)},
				{path: tdmsPath(firstIndex.Name, first[0].Name)},
				{path: tdmsPath(secondIndex.Name, secondIndex.Name)},
				{path: tdmsPath(secondIndex.Name, second[0].Name)},
			}))
			firstPath := tdmsPath(firstIndex.Name, first[0].Name)
			Expect(tdmsChannelData(segments, firstPath)).To(Equal(tdmsObject{
				path:     firstPath,
				dataType: 0x09,
				count:    2,
				data:     telem.NewSeriesV[float32](1, 2).Data,
			}))
			secondPath := tdmsPath(secondIndex.Name, second[0].Name)
			Expect(tdmsChannelData(segments, secondPath)).To(Equal(tdmsObject{
				path:     secondPath,
				dataType: 0x02,
				count:    1,
				data:     telem.NewSeriesV[int16](-3).Data,
			}))
			secondIndexPath := tdmsPath(secondIndex.Name, secondIndex.Name)
			Expect(tdmsChannelData(segments, secondIndexPath).data).
				To(Equal(tdmsTimeStamp(0, 2082844802)))
		})

		It("Should write every sample of a read that spans many frames", func(
			ctx SpecContext,
		) {
			const samples = 20_000
			index, channels := createIndexed(ctx, telem.Float64T, telem.StringT)
			var (
				timestamps = make([]telem.TimeStamp, samples)
				floats     = make([]float64, samples)
				strs       = make([]string, samples)
				text       []byte
			)
			for i := range samples {
				timestamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
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
						svcframer.IteratorConfig{ChunkSize: 1000},
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
				tdmsPath(index.Name, channels[1].Name),
			).data).To(Equal(text))
			Expect(decodeTDMSTimeStamps(tdmsChannelData(
				segments,
				tdmsPath(index.Name, index.Name),
			))).To(Equal(timestamps))
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
				{path: tdmsPath(index.Name)},
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
