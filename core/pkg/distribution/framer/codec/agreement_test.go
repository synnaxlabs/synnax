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
	"encoding/hex"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// frameBytes decodes hex with spaces between its bytes.
func frameBytes(h string) []byte {
	return MustSucceed(hex.DecodeString(strings.ReplaceAll(h, " ", "")))
}

func float32Codec() *codec.Codec {
	return codec.NewStatic(channel.Keys{1}, []telem.DataType{telem.Float32T})
}

func one(s telem.Series) framer.Frame { return frame.NewUnary(1, s) }

func invalid(msg string) OmegaMatcher {
	return SatisfyAll(
		MatchError(validate.ErrValidation),
		MatchError(msg+": validation error"),
	)
}

var _ = Describe("Agreement", func() {
	Describe("Type declaration", func() {
		It("Should declare the types in the first frame of a state only", func(
			ctx SpecContext,
		) {
			c := float32Codec()
			s := telem.NewSeriesV[float32](1)
			Expect(c.Encode(ctx, one(s))).To(
				Equal(frameBytes("BF 01000000 01 09 01000000 0000803F")),
			)
			Expect(c.Encode(ctx, one(s))).To(
				Equal(frameBytes("3F 01000000 01000000 0000803F")),
			)
		})

		It("Should set the extended bit that shipped decoders reject", func(
			ctx SpecContext,
		) {
			encoded := MustSucceed(float32Codec().Encode(ctx, framer.Frame{}))
			Expect(encoded[0] & 0x80).To(Equal(byte(0x80)))
		})

		It("Should declare the types again after an update", func(ctx SpecContext) {
			resolver := mapResolver{1: telem.Float32T}
			c := codec.NewDynamic(resolver)
			Expect(c.Update(ctx, channel.Keys{1})).To(Succeed())
			MustSucceed(c.Encode(ctx, framer.Frame{}))
			Expect(c.Update(ctx, channel.Keys{1})).To(Succeed())
			Expect(c.Encode(ctx, framer.Frame{})).To(
				Equal(frameBytes("BE 02000000 01 09 FFFFFFFF")),
			)
		})

		It("Should fail on a data type with no type code", func(ctx SpecContext) {
			c := codec.NewStatic(channel.Keys{1}, []telem.DataType{"int128"})
			Expect(c.Encode(ctx, framer.Frame{})).Error().
				To(invalid(`data type "int128" of channel 1 has no type code`))
		})
	})

	Describe("Overrides", func() {
		roundTrip := func(
			ctx SpecContext,
			enc, dec *codec.Codec,
			s ...telem.Series,
		) []telem.Series {
			GinkgoHelper()
			keys := make(channel.Keys, len(s))
			for i := range keys {
				keys[i] = 1
			}
			fr := frame.NewMulti(keys, s)
			return MustSucceed(dec.Decode(MustSucceed(enc.Encode(ctx, fr)))).
				Get(1).Series
		}

		It("Should carry a clock to every later frame of the state", func(
			ctx SpecContext,
		) {
			enc, dec := float32Codec(), float32Codec()
			s := clocked(telem.NewSeriesV[float32](1), 5)
			Expect(enc.Encode(ctx, one(s))).To(Equal(frameBytes(
				"BF 01000000 03 09 01000000 00000000 02 05000000 FFFFFFFF 0000803F",
			)))
			steady := MustSucceed(enc.Encode(ctx, one(s)))
			Expect(steady).To(Equal(frameBytes("3F 01000000 01000000 0000803F")))
			Expect(dec.Decode(frameBytes(
				"BF 01000000 03 09 01000000 00000000 02 05000000 FFFFFFFF 0000803F",
			))).Error().ToNot(HaveOccurred())
			Expect(MustSucceed(dec.Decode(steady)).Get(1).Series[0].Clock).
				To(Equal(uint32(5)))
		})

		It("Should reach every later series of the key in the frame", func(
			ctx SpecContext,
		) {
			series := []telem.Series{
				clocked(aligned(telem.NewSeriesV[float32](1), 0), 5),
				clocked(aligned(telem.NewSeriesV[float32](2), 9), 5),
				clocked(aligned(telem.NewSeriesV[float32](3), 20), 7),
			}
			decoded := roundTrip(ctx, float32Codec(), float32Codec(), series...)
			Expect(decoded).To(HaveExactElements(
				telem.MatchSeries(series[0]),
				telem.MatchSeries(series[1]),
				telem.MatchSeries(series[2]),
			))
		})

		It("Should carry a type override for its own series only", func(
			ctx SpecContext,
		) {
			enc, dec := float32Codec(), float32Codec()
			wide := telem.NewSeriesV[float64](1)
			Expect(roundTrip(ctx, enc, dec, wide)[0]).To(telem.MatchSeries(wide))
			narrow := telem.NewSeriesV[float32](2)
			Expect(roundTrip(ctx, enc, dec, narrow)[0]).To(telem.MatchSeries(narrow))
			Expect(roundTrip(ctx, enc, dec, wide)[0]).To(telem.MatchSeries(wide))
		})

		It("Should label an int64 series on a timestamp key with no override", func(
			ctx SpecContext,
		) {
			enc := codec.NewStatic(channel.Keys{1}, []telem.DataType{telem.TimestampT})
			dec := codec.NewStatic(channel.Keys{1}, []telem.DataType{telem.TimestampT})
			first := MustSucceed(enc.Encode(ctx, one(telem.NewSeriesV[int64](7))))
			Expect(
				first,
			).To(Equal(frameBytes("BF 01000000 01 0B 01000000 0700000000000000")))
			Expect(MustSucceed(dec.Decode(first)).Get(1).Series[0].DataType).
				To(Equal(telem.TimestampT))
		})

		It("Should commit nothing when a frame fails to encode", func(
			ctx SpecContext,
		) {
			c := float32Codec()
			bad := clocked(telem.Series{DataType: "int128", Data: make([]byte, 16)}, 5)
			Expect(c.Encode(ctx, one(bad))).Error().
				To(invalid(`data type "int128" of channel 1 has no type code`))
			Expect(c.Encode(ctx, one(clocked(telem.NewSeriesV[float32](1), 5)))).To(
				Equal(frameBytes(
					"BF 01000000 03 09 01000000 00000000 02 05000000 FFFFFFFF 0000803F",
				)),
			)
		})
	})

	Describe("Rejects", func() {
		// clockFrame is the first frame of a float32 key 1 whose one sample, 1.0, has
		// clock 5. Byte 5 is the extension flags, byte 6 the type code, byte 15 the
		// override kinds, and bytes 20 to 23 the next override ordinal.
		const clockFrame = "BF 01000000 03 09 01000000 00000000 02 05000000 FFFFFFFF 0000803F"

		DescribeTable(
			"Should reject a malformed frame",
			func(corrupt func([]byte) []byte, msg string) {
				dec := float32Codec()
				Expect(dec.Decode(corrupt(frameBytes(clockFrame)))).Error().
					To(invalid(msg))
				Expect(dec.Decode(frameBytes(clockFrame))).Error().To(invalid(
					"remote sent a frame for state 1 after a frame of it failed to decode",
				))
			},
			Entry("a reserved extension bit", func(b []byte) []byte {
				b[5] |= 0b0000_0100
				return b
			}, "remote sent extension flags 00000111, which hold a reserved bit"),
			Entry("a second extension byte", func(b []byte) []byte {
				b[5] |= 0b1000_0000
				return b
			}, "remote sent extension flags 10000011, which hold a reserved bit"),
			Entry("extension flags of 0", func(b []byte) []byte {
				b[5] = 0
				return b
			}, "remote sent extension flags of 0"),
			Entry("type code 0", func(b []byte) []byte {
				b[6] = 0
				return b
			}, "remote sent data type code 0, which this codec does not know"),
			Entry("an unknown type code", func(b []byte) []byte {
				b[6] = 17
				return b
			}, "remote sent data type code 17, which this codec does not know"),
			Entry("a variable type under a shared length", func(b []byte) []byte {
				b[6] = byte(telem.DataTypeCodeString)
				return b
			}, "remote sent variable data type string under a shared length"),
			Entry("override kinds of 0", func(b []byte) []byte {
				b[15] = 0
				return b
			}, "remote sent override kinds of 0 for series 0"),
			Entry("a reserved override kind", func(b []byte) []byte {
				b[15] |= 0b0000_0100
				return b
			}, "remote sent override kinds 00000110 for series 0, which hold a reserved bit"),
			Entry(
				"a next override that is not after the series",
				func(b []byte) []byte {
					telem.ByteOrder.PutUint32(b[20:24], 0)
					return b
				},
				"remote sent next override 0, which is not after series 0",
			),
			Entry("an override pending at the end of the frame", func(b []byte) []byte {
				telem.ByteOrder.PutUint32(b[20:24], 1)
				return b
			}, "remote ended the frame with the override for series 1 pending"),
			Entry("bytes after the last series", func(b []byte) []byte {
				return append(b, 0)
			}, "remote sent bytes after the last series"),
		)

		It("Should reject a second type declaration for a state", func() {
			dec := float32Codec()
			MustSucceed(dec.Decode(frameBytes(clockFrame)))
			Expect(dec.Decode(frameBytes(clockFrame))).Error().
				To(invalid("remote declared the types of state 1 twice"))
		})

		It("Should name the older codec of a remote that declares no types", func() {
			Expect(float32Codec().Decode(
				frameBytes("3F 01000000 01000000 0000803F"),
			)).Error().To(invalid(
				"remote sent no type declaration for state 1, so it runs an older " +
					"codec, which this codec does not support",
			))
		})
	})
})
