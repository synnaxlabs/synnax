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
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// delimitedWith returns a delimited text field named name at position, with base giving
// its scale, min, max, and default.
func delimitedWith(name string, position uint32, base icd.TextBaseField) icd.TextField {
	base.Key, base.Name = uuid.New(), name
	if base.Scale == 0 {
		base.Scale = 1
	}
	return icd.TextField{Variant: icd.DelimitedTextField{
		TextBaseField: base,
		Position:      position,
	}}
}

var _ = Describe("TextEncoder", func() {
	DescribeTable("Should encode the cases of a fixture", func(path string) {
		f := readEncodeFixture(path)
		m := f.Message.textMessage()
		if f.Error != "" {
			Expect(codec.NewTextEncoder(m)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(f.Error),
			))
			return
		}
		f.checkCases(MustSucceed(codec.NewTextEncoder(m)), m.Bindings)
	}, fixtureEntries("encode", payload.FormatText))

	It("Should fail to encode a nil sample whose field has no default", func() {
		enc := MustSucceed(
			codec.NewTextEncoder(textMessage("cmd", "", delimited("a", 0))),
		)
		Expect(enc.Encode(nil, [][]byte{nil})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(
				"message cmd: field a: no value and no default: validation error",
			),
		))
	})

	It("Should accept a field at the last position", func() {
		Expect(codec.NewTextEncoder(
			textMessage("cmd", "", delimited("a", 65535)),
		)).Error().ToNot(HaveOccurred())
	})

	It("Should encode without allocating once its buffer has grown", func() {
		m := textMessage("cmd", "$SET", delimited("a", 1))
		m.Bindings[0].DataType = telem.Int64T
		m.Payload.Fields = append(m.Payload.Fields, icd.TextField{
			Variant: icd.TaggedTextField{
				TextBaseField: icd.TextBaseField{
					BaseField: icd.BaseField{Key: uuid.New(), Name: "t", Scale: 1},
					Default:   new(payload.Float(2.5)),
				},
				Tag: "T=",
			},
		})
		enc := MustSucceed(codec.NewTextEncoder(m))
		samples := [][]byte{telem.NewSeriesV[int64](-123456789).Data}
		var (
			line []byte
			err  error
		)
		encode := func() { line, err = enc.Encode(line[:0], samples) }
		encode()
		Expect(testing.AllocsPerRun(100, encode)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
		Expect(string(line)).To(Equal("$SET,-123456789,T=2.5"))
	})

	It("Should check a sample of a binding as Encode does", func() {
		enc := MustSucceed(codec.NewTextEncoder(textMessage("cmd", "",
			delimited("a", 0),
			delimitedWith("b", 1, icd.TextBaseField{Default: new(payload.Float(1))}),
		)))
		Expect(enc.Check(0, telem.NewSeriesV(2.5).Data)).To(Succeed())
		Expect(enc.Check(1, telem.NewSeriesV(2.5).Data)).To(Succeed())
		Expect(enc.Check(1, telem.NewSeriesV(math.NaN()).Data)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message cmd: field b: NaN is not finite: validation error"),
		))
	})

	It("Should be ready when each field has a sample or a default", func() {
		enc := MustSucceed(codec.NewTextEncoder(textMessage("cmd", "",
			delimited("a", 0),
			delimitedWith("b", 1, icd.TextBaseField{Default: new(payload.Float(1))}),
		)))
		Expect(enc.Ready([][]byte{nil, nil})).To(BeFalse())
		Expect(enc.Ready([][]byte{float64Sample(2), nil})).To(BeTrue())
	})

	Describe("Limits", func() {
		var (
			a = delimitedWith("a", 0, icd.TextBaseField{Min: new(0.0), Max: new(10.0)})
			s = delimitedWith("s", 1, icd.TextBaseField{Scale: 0.1, Max: new(10.0)})
		)

		DescribeTable("Should check a sample against the min and max of its field, "+
			"before the scale and offset come off",
			func(binding int, sample []byte, message string) {
				enc := MustSucceed(codec.NewTextEncoder(textMessage("cmd", "", a, s)))
				if message == "" {
					Expect(enc.Check(binding, sample)).To(Succeed())
					return
				}
				Expect(enc.Check(binding, sample)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(message+": validation error"),
				))
			},
			Entry("at the min", 0, telem.NewSeriesV(0.0).Data, ""),
			Entry("at the max", 0, telem.NewSeriesV(10.0).Data, ""),
			Entry("a scaled sample", 1, telem.NewSeriesV(5.0).Data, ""),
			Entry("above the max", 0, telem.NewSeriesV(10.5).Data,
				"message cmd: field a: 10.5 is above the max 10"),
			Entry("below the min", 0, telem.NewSeriesV(-1.0).Data,
				"message cmd: field a: -1 is below the min 0"),
			Entry("not finite", 0, telem.NewSeriesV(math.Inf(1)).Data,
				"message cmd: field a: Infinity is not finite"),
		)

		It("Should encode a sample outside the min and max of its field", func() {
			enc := MustSucceed(codec.NewTextEncoder(textMessage("cmd", "", a, s)))
			line := MustSucceed(enc.Encode(nil, [][]byte{
				telem.NewSeriesV(10.5).Data,
				telem.NewSeriesV(20.0).Data,
			}))
			Expect(string(line)).To(Equal("10.5,200"))
		})

		It("Should check an integer sample against the min and max of its field",
			func() {
				m := textMessage("cmd", "", a)
				m.Bindings[0].DataType = telem.Int64T
				enc := MustSucceed(codec.NewTextEncoder(m))
				Expect(enc.Check(0, telem.NewSeriesV[int64](11).Data)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError("message cmd: field a: 11 is above the max 10: "+
						"validation error"),
				))
			},
		)

		It("Should take a float32 sample at a limit at float32 precision as at the "+
			"limit, and refuse one past it with its exact value", func() {
			m := textMessage("cmd", "",
				delimitedWith("a", 0, icd.TextBaseField{Max: new(0.1)}),
				delimitedWith("b", 1, icd.TextBaseField{Min: new(3.3)}),
			)
			for i := range m.Bindings {
				m.Bindings[i].DataType = telem.Float32T
			}
			enc := MustSucceed(codec.NewTextEncoder(m))
			f32 := func(v float32) []byte { return telem.NewSeriesV(v).Data }
			Expect(enc.Check(0, f32(0.1))).To(Succeed())
			Expect(enc.Check(1, f32(3.3))).To(Succeed())
			Expect(enc.Check(0, f32(math.Nextafter32(0.1, 1)))).To(MatchError(
				"message cmd: field a: 0.10000000894069672 is above the max 0.1: " +
					"validation error",
			))
			Expect(enc.Check(1, f32(math.Nextafter32(3.3, 0)))).To(MatchError(
				"message cmd: field b: 3.299999713897705 is below the min 3.3: " +
					"validation error",
			))
		})
	})
})
