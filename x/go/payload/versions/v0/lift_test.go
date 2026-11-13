// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/payload/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Lift", func() {
	base := v0.BaseWriteField{Key: "s", Pointer: "/s"}

	DescribeTable("LiftStaticWriteField should take the kind of the stored Go type",
		func(value any, expected v0.WriteFieldVariant) {
			Expect(MustSucceed(v0.LiftStaticWriteField(base, value)).Variant).
				To(Equal(expected))
		},
		Entry("whole float64", 12.0,
			v0.IntegerWriteField{BaseWriteField: base, Value: 12}),
		Entry("whole float64 at 2^53", float64(1<<53),
			v0.IntegerWriteField{BaseWriteField: base, Value: 1 << 53}),
		Entry("whole float64 past 2^53", float64(1<<54),
			v0.FloatWriteField{BaseWriteField: base, Value: 1 << 54}),
		Entry("fractional float64", 2.5,
			v0.FloatWriteField{BaseWriteField: base, Value: 2.5}),
		Entry("float32", float32(0.5),
			v0.FloatWriteField{BaseWriteField: base, Value: 0.5}),
		Entry("int", -7, v0.IntegerWriteField{BaseWriteField: base, Value: -7}),
		Entry("int8", int8(-4), v0.IntegerWriteField{BaseWriteField: base, Value: -4}),
		Entry("int64", int64(math.MinInt64),
			v0.IntegerWriteField{BaseWriteField: base, Value: math.MinInt64}),
		Entry("uint", uint(7), v0.UnsignedWriteField{BaseWriteField: base, Value: 7}),
		Entry("uint8", uint8(4), v0.UnsignedWriteField{BaseWriteField: base, Value: 4}),
		Entry("uint64", uint64(math.MaxUint64),
			v0.UnsignedWriteField{BaseWriteField: base, Value: math.MaxUint64}),
		Entry("string", "on", v0.StringWriteField{BaseWriteField: base, Value: "on"}),
		Entry("boolean", false,
			v0.BooleanWriteField{BaseWriteField: base, Value: false}),
		Entry("null", nil, v0.JsonWriteField{BaseWriteField: base, Value: "null"}),
		Entry("object", map[string]any{"a": 1.0},
			v0.JsonWriteField{BaseWriteField: base, Value: `{"a":1}`}),
		Entry("array", []any{"a", true},
			v0.JsonWriteField{BaseWriteField: base, Value: `["a",true]`}),
	)

	DescribeTable(
		"LiftGeneratedWriteField should lift a known generator",
		func(
			generator v0.GeneratorType,
			format *v0.TimeFormat,
			expected v0.WriteFieldVariant,
		) {
			f, ok := v0.LiftGeneratedWriteField(base, generator, format)
			Expect(ok).To(BeTrue())
			Expect(f.Variant).To(Equal(expected))
		},
		Entry(
			"uuid",
			v0.GeneratorTypeUUID,
			nil,
			v0.UuidWriteField{BaseWriteField: base},
		),
		Entry("timestamp with no time format", v0.GeneratorTypeTimestamp, nil,
			v0.TimestampWriteField{
				BaseWriteField: base,
				TimeFormat:     v0.TimeFormatISO8601,
			}),
		Entry("timestamp with a time format", v0.GeneratorTypeTimestamp,
			new(v0.TimeFormatUnixUs),
			v0.TimestampWriteField{
				BaseWriteField: base,
				TimeFormat:     v0.TimeFormatUnixUs,
			}),
	)

	It("LiftGeneratedWriteField should not lift an unknown generator", func() {
		_, ok := v0.LiftGeneratedWriteField(base, "id", nil)
		Expect(ok).To(BeFalse())
	})
})
