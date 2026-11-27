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
	"math"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// bitsAt returns a little-endian base field named name, bitLength bits wide from
// startBit.
func bitsAt(name string, startBit uint16, bitLength uint8) icd.BinaryBaseField {
	return icd.BinaryBaseField{
		Key:       uuid.New(),
		Name:      name,
		Scale:     1,
		StartBit:  startBit,
		BitLength: bitLength,
		ByteOrder: icd.ByteOrderLittleEndian,
	}
}

// selectedBy returns an unsigned field named name in the byte at offset that the field
// mux selects with values.
func selectedBy(
	name string,
	offset uint16,
	mux icd.FieldKey,
	values ...int32,
) icd.BinaryField {
	b := bitsAt(name, offset*8, 8)
	b.Multiplexor, b.MultiplexValues = &mux, values
	return icd.BinaryField{Variant: icd.UnsignedBinaryField{BinaryBaseField: b}}
}

// frame returns a message named frame of fields, each bound to a channel of dt.
func frame(dt telem.DataType, fields ...icd.BinaryField) codec.BinaryMessage {
	return codec.BinaryMessage{
		Name:    "frame",
		Payload: icd.BinaryPayload{Fields: fields},
		Bindings: lo.Map(fields, func(f icd.BinaryField, _ int) codec.Binding {
			return codec.Binding{Field: f.Base().Key, DataType: dt}
		}),
	}
}

func uint8Sample(v uint8) []byte { return telem.NewSeriesV(v).Data }

var _ = Describe("BinaryEncoder", func() {
	DescribeTable("Should encode the cases of a fixture", func(path string) {
		f := readEncodeFixture(path)
		m := f.Message.binaryMessage()
		if f.Error != "" {
			Expect(codec.NewBinaryEncoder(m)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(f.Error),
			))
			return
		}
		f.checkCases(MustSucceed(codec.NewBinaryEncoder(m)), m.Bindings)
	}, fixtureEntries("encode", payload.FormatBinary))

	It("Should refuse a message with a CAN identifier", func() {
		m := frame(telem.Uint8T, byteField("v", 0))
		m.Payload.Identifier = &icd.Identifier{Variant: icd.CanIdentifier{ID: 0x100}}
		Expect(codec.NewBinaryEncoder(m)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message frame: identifier: only the Driver writes CAN "+
				"frames: validation error"),
		))
	})

	It("Should encode a scaled sample that a decoder of the same message reads back",
		func() {
			b := bitsAt("v", 0, 16)
			b.Scale, b.Offset = 0.1, -40
			m := frame(
				telem.Float64T,
				icd.BinaryField{Variant: icd.IntegerBinaryField{BinaryBaseField: b}},
			)
			enc := MustSucceed(codec.NewBinaryEncoder(m))
			data := MustSucceed(enc.Encode(nil, [][]byte{float64Sample(21.5)}))
			dec := MustSucceed(codec.NewBinary([]codec.BinaryMessage{m}))
			samples := [][][]byte{make([][]byte, 1)}
			_, ok, failures, err := dec.Decode(data, samples)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(failures).To(BeEmpty())
			Expect(math.Float64frombits(telem.ByteOrder.Uint64(samples[0][0]))).
				To(BeNumerically("~", 21.5, 1e-9))
		})

	It("Should fail to encode a nil sample whose field has no default", func() {
		enc := MustSucceed(
			codec.NewBinaryEncoder(frame(telem.Uint8T, byteField("v", 0))),
		)
		Expect(enc.Encode(nil, [][]byte{nil})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message frame: field v: no value and no default: "+
				"validation error"),
		))
	})

	It("Should encode without allocating once its payload has room", func() {
		b := bitsAt("temp", 0, 16)
		b.Scale = 0.1
		enc := MustSucceed(codec.NewBinaryEncoder(frame(
			telem.Float64T,
			icd.BinaryField{Variant: icd.IntegerBinaryField{BinaryBaseField: b}},
		)))
		samples := [][]byte{float64Sample(0.1 + 0.2)}
		data := MustSucceed(enc.Encode(nil, samples))
		var err error
		Expect(testing.AllocsPerRun(100, func() {
			data, err = enc.Encode(data[:0], samples)
		})).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte{0x03, 0x00}))
	})

	It("Should be ready when each field has a sample or a default", func() {
		w := bitsAt("w", 8, 8)
		enc := MustSucceed(codec.NewBinaryEncoder(frame(
			telem.Uint8T,
			byteField("v", 0),
			icd.BinaryField{Variant: icd.UnsignedBinaryField{
				BinaryBaseField: w,
				Default:         new(payload.Unsigned(1)),
			}},
		)))
		Expect(enc.Ready([][]byte{nil, nil})).To(BeFalse())
		Expect(enc.Ready([][]byte{uint8Sample(1), nil})).To(BeTrue())
	})

	It("Should need samples only for the multiplexor and the fields that its value "+
		"selects", func() {
		mode := byteField("mode", 0)
		enc := MustSucceed(codec.NewBinaryEncoder(frame(
			telem.Uint8T,
			mode,
			selectedBy("a", 1, mode.Base().Key, 0),
			selectedBy("b", 2, mode.Base().Key, 1),
		)))
		five := uint8Sample(5)
		Expect(enc.Ready([][]byte{nil, five, five})).To(BeFalse())
		Expect(enc.Ready([][]byte{uint8Sample(0), nil, five})).To(BeFalse())
		Expect(enc.Ready([][]byte{uint8Sample(0), five, nil})).To(BeTrue())
		Expect(enc.Ready([][]byte{uint8Sample(1), five, nil})).To(BeFalse())
	})

	Describe("Check", func() {
		check := func(enc *codec.BinaryEncoder, sample []byte, expected string) {
			GinkgoHelper()
			Expect(enc.Check(0, sample)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("message frame: field v: "+expected+": validation error"),
			))
		}

		It("Should refuse a sample outside the min and max of its field, and accept a "+
			"float32 sample that equals a limit at float32 precision", func() {
			b := bitsAt("v", 0, 32)
			b.Min, b.Max = new(-1.0), new(0.1)
			enc := MustSucceed(codec.NewBinaryEncoder(frame(
				telem.Float32T,
				icd.BinaryField{Variant: icd.FloatBinaryField{BinaryBaseField: b}},
			)))
			Expect(enc.Check(0, telem.NewSeriesV[float32](0.1).Data)).To(Succeed())
			check(enc, telem.NewSeriesV[float32](0.5).Data, "0.5 is above the max 0.1")
			check(enc, telem.NewSeriesV[float32](-2).Data, "-2 is below the min -1")
		})

		It("Should check the value that a sample rounds to", func() {
			b := bitsAt("v", 0, 8)
			b.Scale, b.Max = 5, new(13.0)
			enc := MustSucceed(codec.NewBinaryEncoder(frame(
				telem.Float64T,
				icd.BinaryField{Variant: icd.IntegerBinaryField{BinaryBaseField: b}},
			)))
			Expect(enc.Check(0, float64Sample(12))).To(Succeed())
			check(enc, float64Sample(13), "3 decodes to 15, which is above the max 13")
		})

		It("Should check the float32 value of a float64 sample for a float32 field",
			func() {
				b := bitsAt("v", 0, 32)
				b.Max = new(0.1)
				enc := MustSucceed(codec.NewBinaryEncoder(frame(
					telem.Float64T,
					icd.BinaryField{Variant: icd.FloatBinaryField{BinaryBaseField: b}},
				)))
				Expect(enc.Check(0, float64Sample(0.1))).To(Succeed())
				check(
					enc,
					float64Sample(0.10000001),
					"0.10000000894069672 is above the max 0.1",
				)
			})

		DescribeTable("Should refuse an infinite or NaN sample for a float field "+
			"with a min and a max", func(bitLength uint8) {
			b := bitsAt("v", 0, bitLength)
			b.Min, b.Max = new(-10.0), new(10.0)
			enc := MustSucceed(codec.NewBinaryEncoder(frame(
				telem.Float64T,
				icd.BinaryField{Variant: icd.FloatBinaryField{BinaryBaseField: b}},
			)))
			check(enc, float64Sample(math.Inf(1)), "Infinity is above the max 10")
			check(enc, float64Sample(math.Inf(-1)), "-Infinity is below the min -10")
			check(enc, float64Sample(math.NaN()), "NaN is outside the limits")
		}, Entry("32 bits", uint8(32)), Entry("64 bits", uint8(64)))

		It("Should accept a NaN sample for a float field with no min and no max",
			func() {
				enc := MustSucceed(codec.NewBinaryEncoder(frame(
					telem.Float64T,
					icd.BinaryField{Variant: icd.FloatBinaryField{
						BinaryBaseField: bitsAt("v", 0, 64),
					}},
				)))
				Expect(enc.Check(0, float64Sample(math.NaN()))).To(Succeed())
			})
	})
})
