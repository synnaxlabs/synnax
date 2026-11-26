// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/payload"
)

var _ = Describe("CheckDefault", func() {
	DescribeTable("Should accept a default that its field can hold",
		func(v icd.JSONFieldVariant) {
			Expect(icd.CheckDefault(icd.JSONField{Variant: v})).To(Succeed())
		},
		Entry("no default", icd.FloatJSONField{Scale: 1, Max: new(0.0)}),
		Entry("a scaled default at the max", icd.FloatJSONField{
			Scale: 0.5, Max: new(50.0), Default: new(payload.Float(100)),
		}),
		Entry("a default with an offset", icd.IntegerJSONField{
			Scale: 1, Offset: -10, Max: new(0.0), Default: new(payload.Integer(10)),
		}),
		Entry("a default that float error puts above the max", icd.FloatJSONField{
			Scale: 0.1, Max: new(0.3), Default: new(payload.Float(3)),
		}),
		Entry("a default that float error puts below the min", icd.UnsignedJSONField{
			Scale: 0.3, Min: new(0.9), Default: new(payload.Unsigned(3)),
		}),
		Entry("base64 text", icd.BytesJSONField{Default: new("AQI=")}),
		Entry("valid JSON", icd.JSONJSONField{Default: new(`{"a":1}`)}),
		Entry("a NaN float", icd.FloatJSONField{
			Scale: 1, Default: new(payload.Float(math.NaN())),
		}),
	)

	DescribeTable("Should refuse a default that its field cannot hold",
		func(v icd.JSONFieldVariant, expected string) {
			Expect(icd.CheckDefault(icd.JSONField{Variant: v})).To(MatchError(expected))
		},
		Entry("an infinite float",
			icd.FloatJSONField{Scale: 1, Default: new(payload.Float(math.Inf(1)))},
			"cannot be infinite: validation error",
		),
		Entry("text that is not base64",
			icd.BytesJSONField{Default: new("a*b=")},
			"must be base64: validation error",
		),
		Entry("text that is not valid JSON",
			icd.JSONJSONField{Default: new("{")},
			"must be valid JSON: validation error",
		),
		Entry("an unscaled default above the max",
			icd.FloatJSONField{
				Scale: 1, Max: new(100.0), Default: new(payload.Float(120)),
			},
			"120 is above the max 100: validation error",
		),
		Entry("a scaled default above the max",
			icd.IntegerJSONField{
				Scale: 10, Max: new(100.0), Default: new(payload.Integer(12)),
			},
			"12 decodes to 120, which is above the max 100: validation error",
		),
		Entry("a default with an offset below the min",
			icd.FloatJSONField{
				Scale: 1, Offset: 5, Min: new(0.0), Default: new(payload.Float(-6)),
			},
			"-6 decodes to -1, which is below the min 0: validation error",
		),
		Entry("a default just above the max",
			icd.FloatJSONField{
				Scale: 0.1, Max: new(0.3), Default: new(payload.Float(3.000001)),
			},
			"3.000001 decodes to 0.30000010000000005, which is above the max 0.3: "+
				"validation error",
		),
	)
})

// binaryOf returns the base of a binary field of n bits with base b.
func binaryOf(n uint8, b icd.BaseField) icd.BinaryBaseField {
	return icd.BinaryBaseField{BaseField: b, BitLength: n}
}

var _ = Describe("CheckBinaryDefault", func() {
	DescribeTable("Should accept a default that its field can hold",
		func(v icd.BinaryFieldVariant) {
			Expect(icd.CheckBinaryDefault(icd.BinaryField{Variant: v})).To(Succeed())
		},
		Entry("no default", icd.FloatBinaryField{
			BinaryBaseField: binaryOf(32, icd.BaseField{Scale: 1}),
		}),
		Entry("the largest 32-bit float", icd.FloatBinaryField{
			BinaryBaseField: binaryOf(32, icd.BaseField{Scale: 1}),
			Default:         new(payload.Float(math.MaxFloat32)),
		}),
		Entry("the lowest value of an 8-bit integer", icd.IntegerBinaryField{
			BinaryBaseField: binaryOf(8, icd.BaseField{Scale: 1}),
			Default:         new(payload.Integer(-128)),
		}),
		Entry("the highest value of an 8-bit unsigned", icd.UnsignedBinaryField{
			BinaryBaseField: binaryOf(8, icd.BaseField{Scale: 1}),
			Default:         new(payload.Unsigned(255)),
		}),
		Entry("a scaled default at the max", icd.UnsignedBinaryField{
			BinaryBaseField: binaryOf(8, icd.BaseField{Scale: 10, Max: new(100.0)}),
			Default:         new(payload.Unsigned(10)),
		}),
	)

	DescribeTable("Should refuse a default that its field cannot hold",
		func(v icd.BinaryFieldVariant, expected string) {
			Expect(icd.CheckBinaryDefault(icd.BinaryField{Variant: v})).
				To(MatchError(expected))
		},
		Entry("an infinite float",
			icd.FloatBinaryField{
				BinaryBaseField: binaryOf(64, icd.BaseField{Scale: 1}),
				Default:         new(payload.Float(math.Inf(-1))),
			},
			"cannot be infinite: validation error",
		),
		Entry("a float that overflows 32 bits",
			icd.FloatBinaryField{
				BinaryBaseField: binaryOf(32, icd.BaseField{Scale: 1}),
				Default:         new(payload.Float(1e39)),
			},
			"1e+39 does not fit a 32-bit float: validation error",
		),
		Entry("an integer above its bits",
			icd.IntegerBinaryField{
				BinaryBaseField: binaryOf(8, icd.BaseField{Scale: 1}),
				Default:         new(payload.Integer(128)),
			},
			"128 does not fit the field: validation error",
		),
		Entry("an integer below its bits",
			icd.IntegerBinaryField{
				BinaryBaseField: binaryOf(8, icd.BaseField{Scale: 1}),
				Default:         new(payload.Integer(-129)),
			},
			"-129 does not fit the field: validation error",
		),
		Entry("an unsigned above its bits",
			icd.UnsignedBinaryField{
				BinaryBaseField: binaryOf(8, icd.BaseField{Scale: 1}),
				Default:         new(payload.Unsigned(256)),
			},
			"256 does not fit the field: validation error",
		),
		Entry("a scaled default above the max",
			icd.IntegerBinaryField{
				BinaryBaseField: binaryOf(
					16,
					icd.BaseField{Scale: 10, Max: new(100.0)},
				),
				Default: new(payload.Integer(12)),
			},
			"12 decodes to 120, which is above the max 100: validation error",
		),
		Entry("a float default below the min",
			icd.FloatBinaryField{
				BinaryBaseField: binaryOf(64, icd.BaseField{Scale: 1, Min: new(0.0)}),
				Default:         new(payload.Float(-1)),
			},
			"-1 is below the min 0: validation error",
		),
	)
})

var _ = Describe("CheckTextDefault", func() {
	text := func(b icd.BaseField, d *payload.Float) icd.TextField {
		return icd.TextField{Variant: icd.DelimitedTextField{
			TextBaseField: icd.TextBaseField{BaseField: b, Default: d},
		}}
	}

	DescribeTable("Should accept a default that its field can hold",
		func(f icd.TextField) { Expect(icd.CheckTextDefault(f)).To(Succeed()) },
		Entry("no default", text(icd.BaseField{Scale: 1, Max: new(0.0)}, nil)),
		Entry("a scaled default at the max",
			text(icd.BaseField{Scale: 0.5, Max: new(50.0)}, new(payload.Float(100))),
		),
	)

	DescribeTable("Should refuse a default that its field cannot hold",
		func(f icd.TextField, expected string) {
			Expect(icd.CheckTextDefault(f)).To(MatchError(expected))
		},
		Entry("an infinite default",
			text(icd.BaseField{Scale: 1}, new(payload.Float(math.Inf(1)))),
			"cannot be infinite: validation error",
		),
		Entry("a default with an offset below the min",
			text(
				icd.BaseField{Scale: 1, Offset: 5, Min: new(0.0)},
				new(payload.Float(-6)),
			),
			"-6 decodes to -1, which is below the min 0: validation error",
		),
	)
})
