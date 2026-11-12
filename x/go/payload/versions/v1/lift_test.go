// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/x/payload/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Lift", func() {
	base := v1.BaseWriteField{Key: "s", Pointer: "/s"}

	DescribeTable("LiftStaticWriteField should take the kind of the stored Go type",
		func(value any, expected v1.WriteFieldVariant) {
			Expect(MustSucceed(v1.LiftStaticWriteField(base, value)).Variant).
				To(Equal(expected))
		},
		Entry("whole float64", 12.0,
			v1.IntegerWriteField{BaseWriteField: base, Value: 12}),
		Entry("whole float64 at 2^53", float64(1<<53),
			v1.IntegerWriteField{BaseWriteField: base, Value: 1 << 53}),
		Entry("whole float64 past 2^53", float64(1<<54),
			v1.FloatWriteField{BaseWriteField: base, Value: 1 << 54}),
		Entry("fractional float64", 2.5,
			v1.FloatWriteField{BaseWriteField: base, Value: 2.5}),
		Entry("float32", float32(0.5),
			v1.FloatWriteField{BaseWriteField: base, Value: 0.5}),
		Entry("int", -7, v1.IntegerWriteField{BaseWriteField: base, Value: -7}),
		Entry("int8", int8(-4), v1.IntegerWriteField{BaseWriteField: base, Value: -4}),
		Entry("int64", int64(math.MinInt64),
			v1.IntegerWriteField{BaseWriteField: base, Value: math.MinInt64}),
		Entry("uint", uint(7), v1.UnsignedWriteField{BaseWriteField: base, Value: 7}),
		Entry("uint8", uint8(4), v1.UnsignedWriteField{BaseWriteField: base, Value: 4}),
		Entry("uint64", uint64(math.MaxUint64),
			v1.UnsignedWriteField{BaseWriteField: base, Value: math.MaxUint64}),
		Entry("string", "on", v1.StringWriteField{BaseWriteField: base, Value: "on"}),
		Entry("boolean", false,
			v1.BooleanWriteField{BaseWriteField: base, Value: false}),
		Entry("null", nil, v1.JsonWriteField{BaseWriteField: base, Value: "null"}),
		Entry("object", map[string]any{"a": 1.0},
			v1.JsonWriteField{BaseWriteField: base, Value: `{"a":1}`}),
		Entry("array", []any{"a", true},
			v1.JsonWriteField{BaseWriteField: base, Value: `["a",true]`}),
	)

	DescribeTable(
		"LiftGeneratedWriteField should lift a known generator",
		func(
			generator v1.GeneratorType,
			format *v1.TimeFormat,
			expected v1.WriteFieldVariant,
		) {
			f, ok := v1.LiftGeneratedWriteField(base, generator, format)
			Expect(ok).To(BeTrue())
			Expect(f.Variant).To(Equal(expected))
		},
		Entry(
			"uuid",
			v1.GeneratorTypeUUID,
			nil,
			v1.UuidWriteField{BaseWriteField: base},
		),
		Entry("timestamp with no time format", v1.GeneratorTypeTimestamp, nil,
			v1.TimestampWriteField{
				BaseWriteField: base,
				TimeFormat:     v1.TimeFormatISO8601,
			}),
		Entry("timestamp with a time format", v1.GeneratorTypeTimestamp,
			new(v1.TimeFormatUnixUs),
			v1.TimestampWriteField{
				BaseWriteField: base,
				TimeFormat:     v1.TimeFormatUnixUs,
			}),
	)

	It("LiftGeneratedWriteField should not lift an unknown generator", func() {
		_, ok := v1.LiftGeneratedWriteField(base, "id", nil)
		Expect(ok).To(BeFalse())
	})
})
