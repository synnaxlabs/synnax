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
	"fmt"
	"io"
	"math"
	"runtime"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// workedExample is the first frame of a state that requests keys 6 (float32) and 8
// (uint8), where clock 5 times key 6 and key 6 carries float64.
const workedExample = "BF 01000000 0B 0901 01000000 05000000 02000000 01000000" +
	" E803000000000000 D007000000000000" +
	" 03 0A 05000000 FFFFFFFF 000000000000F03F 0000000000000040" +
	" 0100"

func workedExampleCodec() *codec.Codec {
	return codec.NewStatic(
		channel.Keys{6, 8},
		[]telem.DataType{telem.Float32T, telem.Uint8T},
	)
}

func timestamps(clock uint32, values ...telem.TimeStamp) telem.Series {
	return clocked(telem.NewSeriesV(values...), clock)
}

type keyed struct {
	key    channel.Key
	series telem.Series
}

func multi(entries ...keyed) framer.Frame {
	keys := make(channel.Keys, len(entries))
	series := make([]telem.Series, len(entries))
	for i, e := range entries {
		keys[i], series[i] = e.key, e.series
	}
	return frame.NewMulti(keys, series)
}

func expectSeries(fr framer.Frame, entries ...keyed) {
	GinkgoHelper()
	Expect(fr.Frame).To(telem.MatchFrame(multi(entries...).Frame))
}

var _ = Describe("Declared clocks", func() {
	It("Should encode and decode the worked example", func(ctx SpecContext) {
		enc, dec := workedExampleCodec(), workedExampleCodec()
		series := []keyed{
			{5, timestamps(5, 1000, 2000)},
			{6, clocked(telem.NewSeriesV(1.0, 2.0), 5)},
			{8, telem.NewSeriesV[uint8](1, 0)},
		}
		encoded := MustSucceed(enc.Encode(ctx, multi(series...)))
		Expect(encoded).To(Equal(frameBytes(workedExample)))
		Expect(encoded).To(HaveLen(68))
		expectSeries(MustSucceed(dec.Decode(encoded)), series...)

		steady := []keyed{
			{5, timestamps(5, 1000, 2000)},
			{6, clocked(telem.NewSeriesV[float32](1, 2), 5)},
			{8, telem.NewSeriesV[uint8](1, 0)},
		}
		encoded = MustSucceed(enc.Encode(ctx, multi(steady...)))
		Expect(encoded[:9]).To(Equal(frameBytes("3F 01000000 02000000")))
		Expect(encoded).To(HaveLen(9 + 26))
		expectSeries(MustSucceed(dec.Decode(encoded)), steady...)
	})

	It("Should declare nothing for data streamed without its time series", func(
		ctx SpecContext,
	) {
		c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		s := clocked(telem.NewSeriesV[float32](1), 5)
		Expect(c.Encode(ctx, multi(keyed{6, s}))).To(Equal(frameBytes(
			"BF 01000000 03 09 01000000 00000000 02 05000000 FFFFFFFF 0000803F",
		)))
		Expect(c.Encode(ctx, multi(keyed{6, s}))).To(
			Equal(frameBytes("3F 01000000 01000000 0000803F")),
		)
	})

	It("Should drop a series of another key that is not its own clock", func(
		ctx SpecContext,
	) {
		c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		decoded := MustSucceed(c.Decode(MustSucceed(c.Encode(ctx, multi(
			keyed{6, telem.NewSeriesV[float32](1)},
			keyed{9, timestamps(0, 1)},
			keyed{10, clocked(telem.NewSeriesV[float32](1), 10)},
		)))))
		Expect(decoded.KeysSlice()).To(Equal([]channel.Key{6}))
	})

	It("Should keep a declared clock while an agreed clock names it", func(
		ctx SpecContext,
	) {
		enc := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		dec := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		roundTrip := func(entries ...keyed) []byte {
			GinkgoHelper()
			encoded := MustSucceed(enc.Encode(ctx, multi(entries...)))
			expectSeries(MustSucceed(dec.Decode(encoded)), entries...)
			return encoded
		}
		roundTrip(
			keyed{5, timestamps(5, 1)},
			keyed{6, clocked(telem.NewSeriesV[float32](1), 5)},
		)
		Expect(roundTrip(keyed{6, clocked(telem.NewSeriesV[float32](2), 5)})).To(
			Equal(frameBytes("3E 01000000 01000000 06000000 00000040")),
		)
		moved := []keyed{
			{6, clocked(telem.NewSeriesV[float32](3), 7)},
			{7, timestamps(7, 3)},
		}
		move := roundTrip(moved...)
		Expect(move[5]).To(Equal(byte(0b0000_1010)))
		Expect(len(move) - len(roundTrip(moved...))).To(Equal(22))
	})

	It("Should drop a declared clock that leaves the frame unnamed", func(
		ctx SpecContext,
	) {
		c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		data := telem.NewSeriesV[float32](1)
		Expect(
			c.Encode(ctx, multi(keyed{6, data}, keyed{9, timestamps(9, 1)})),
		).To(Equal(frameBytes(
			"BF 01000000 09 09 01000000 09000000 01000000 0000803F 0100000000000000",
		)))
		Expect(c.Encode(ctx, multi(keyed{6, data}))).To(Equal(frameBytes(
			"BF 01000000 08 00000000 01000000 0000803F",
		)))
		Expect(c.Encode(ctx, multi(keyed{6, data}))).To(
			Equal(frameBytes("3F 01000000 01000000 0000803F")),
		)
	})

	It("Should carry one key on two clocks in one frame", func(ctx SpecContext) {
		enc := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		dec := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		roundTrip := func(entries ...keyed) []byte {
			GinkgoHelper()
			encoded := MustSucceed(enc.Encode(ctx, multi(entries...)))
			expectSeries(MustSucceed(dec.Decode(encoded)), entries...)
			return encoded
		}
		roundTrip(
			keyed{5, timestamps(5, 1)},
			keyed{6, clocked(telem.NewSeriesV[float32](1), 5)},
		)
		// 49 bytes in the layout of a frame with no extension, then 5 for the
		// extension flags and the first override, 9 for the clock override, and 12 for
		// the list of two clocks.
		Expect(roundTrip(
			keyed{5, timestamps(5, 2)},
			keyed{6, clocked(telem.NewSeriesV[float32](2), 5)},
			keyed{6, clocked(telem.NewSeriesV[float32](3), 7)},
			keyed{7, timestamps(7, 3)},
		)).To(HaveLen(49 + 26))
		// 21 bytes, then 9 for the extension flags and the list of one clock.
		Expect(roundTrip(
			keyed{6, clocked(telem.NewSeriesV[float32](4), 7)},
			keyed{7, timestamps(7, 4)},
		)).To(HaveLen(21 + 9))
	})

	It("Should label an int64 series under its own clock as a timestamp", func(
		ctx SpecContext,
	) {
		c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		decoded := MustSucceed(c.Decode(MustSucceed(c.Encode(ctx, multi(
			keyed{5, clocked(telem.NewSeriesV[int64](1), 5)},
			keyed{6, telem.NewSeriesV[float32](1)},
		)))))
		Expect(decoded.Get(5).Series[0].DataType).To(Equal(telem.TimestampT))
	})

	Describe("Rejects", func() {
		DescribeTable("Should reject a malformed list",
			func(corrupt func([]byte) []byte, msg string) {
				Expect(workedExampleCodec().Decode(corrupt(frameBytes(workedExample)))).
					Error().To(invalid(msg))
			},
			Entry("a requested key in the list", func(b []byte) []byte {
				telem.ByteOrder.PutUint32(b[12:16], 6)
				return b
			}, "remote declared clock 6, which the request names"),
			Entry("an override on a declared clock", func(b []byte) []byte {
				telem.ByteOrder.PutUint32(b[20:24], 0)
				return slices.Insert(b, 24, 0b0000_0010)
			}, "remote sent an override for declared clock 5"),
		)

		DescribeTable("Should reject a list that is not ascending",
			func(ctx SpecContext, first, second uint32, msg string) {
				c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
				encoded := MustSucceed(c.Encode(ctx, multi(
					keyed{5, timestamps(5, 1)},
					keyed{6, telem.NewSeriesV[float32](1)},
					keyed{7, timestamps(7, 1)},
				)))
				// Bytes 7 to 10 hold the count, then the two clocks.
				Expect(
					encoded[7:19],
				).To(Equal(frameBytes("02000000 05000000 07000000")))
				telem.ByteOrder.PutUint32(encoded[11:15], first)
				telem.ByteOrder.PutUint32(encoded[15:19], second)
				Expect(c.Decode(encoded)).Error().To(invalid(msg))
			},
			Entry("descending", uint32(7), uint32(5),
				"remote declared clock 5 after clock 7, so the list is not ascending"),
			Entry("repeated", uint32(5), uint32(5),
				"remote declared clock 5 after clock 5, so the list is not ascending"),
		)

		It("Should count each declared clock against the decode limit", func() {
			// Three series of 16, 16, and 2 bytes, and one declared clock.
			fits := 4*codec.SeriesOverhead + 34
			decoder := func(limit int64) *codec.Codec {
				return codec.NewStatic(
					channel.Keys{6, 8},
					[]telem.DataType{telem.Float32T, telem.Uint8T},
					codec.WithDecodeLimit(limit),
				)
			}
			MustSucceed(decoder(fits).Decode(frameBytes(workedExample)))
			Expect(decoder(fits - 1).Decode(frameBytes(workedExample))).Error().
				To(invalid(fmt.Sprintf(
					"frame decodes to more than the limit of %d bytes",
					fits-1,
				)))
		})

		It("Should fail without allocating a list the frame does not hold", func() {
			encoded := frameBytes(workedExample)
			telem.ByteOrder.PutUint32(encoded[8:12], math.MaxUint32)
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			Expect(workedExampleCodec().Decode(encoded)).Error().
				To(MatchError(io.ErrUnexpectedEOF))
			runtime.ReadMemStats(&after)
			Expect(after.TotalAlloc - before.TotalAlloc).To(BeNumerically("<", 8<<20))
		})
	})
})
