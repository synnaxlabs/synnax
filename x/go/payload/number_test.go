// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Number", func() {
	DescribeTable(
		"should convert a number that the kind holds exactly",
		func(n payload.Number, k payload.Kind, expected string) {
			v, ok := n.To(k)
			Expect(ok).To(BeTrue())
			Expect(v.Kind()).To(Equal(k))
			Expect(v.String()).To(Equal(expected))
		},
		Entry("a whole float to an integer",
			payload.FloatNumber(-3), payload.KindInteger, "-3"),
		Entry("a whole float to an unsigned integer",
			payload.FloatNumber(7), payload.KindUnsigned, "7"),
		Entry("an integer to a float",
			payload.IntegerNumber(1<<53), payload.KindFloat, "9.007199254740992e+15"),
		Entry("an unsigned integer to an integer",
			payload.UnsignedNumber(math.MaxInt64), payload.KindInteger,
			"9223372036854775807"),
		Entry("a NaN to a float",
			payload.FloatNumber(payload.Float(math.NaN())), payload.KindFloat, "NaN"),
	)

	DescribeTable(
		"should not convert a number that the kind does not hold exactly",
		func(n payload.Number, k payload.Kind) {
			_, ok := n.To(k)
			Expect(ok).To(BeFalse())
		},
		Entry(
			"a fraction to an integer",
			payload.FloatNumber(1.5),
			payload.KindInteger,
		),
		Entry("a negative to an unsigned integer",
			payload.IntegerNumber(-1), payload.KindUnsigned),
		Entry("a NaN to an integer",
			payload.FloatNumber(payload.Float(math.NaN())), payload.KindInteger),
		Entry("an infinity to an unsigned integer",
			payload.FloatNumber(payload.Float(math.Inf(1))), payload.KindUnsigned),
		Entry("an integer past 2^53 to a float",
			payload.IntegerNumber(1<<53+1), payload.KindFloat),
		Entry("an unsigned integer past int64 to an integer",
			payload.UnsignedNumber(math.MaxUint64), payload.KindInteger),
		Entry("a number to a string", payload.FloatNumber(1), payload.KindString),
	)

	DescribeTable(
		"should report whether a data type holds a number",
		func(n payload.Number, dt telem.DataType, fits bool) {
			Expect(n.Fits(dt)).To(Equal(fits))
		},
		Entry("int8 at its maximum", payload.IntegerNumber(127), telem.Int8T, true),
		Entry("int8 past its maximum", payload.IntegerNumber(128), telem.Int8T, false),
		Entry("int8 at its minimum", payload.IntegerNumber(-128), telem.Int8T, true),
		Entry("int8 past its minimum", payload.IntegerNumber(-129), telem.Int8T, false),
		Entry("uint16 at its maximum",
			payload.UnsignedNumber(65535), telem.Uint16T, true),
		Entry("uint16 past its maximum",
			payload.UnsignedNumber(65536), telem.Uint16T, false),
		Entry("uint64 at its maximum",
			payload.UnsignedNumber(math.MaxUint64), telem.Uint64T, true),
		Entry(
			"a fraction on an integer type",
			payload.FloatNumber(0.5),
			telem.Int32T,
			false,
		),
		Entry(
			"float32 past its range",
			payload.FloatNumber(1e39),
			telem.Float32T,
			false,
		),
		Entry("a NaN on float32",
			payload.FloatNumber(payload.Float(math.NaN())), telem.Float32T, true),
		Entry("a large integer on float64",
			payload.UnsignedNumber(math.MaxUint64), telem.Float64T, true),
		Entry(
			"a number on a string type",
			payload.FloatNumber(1),
			telem.StringT,
			false,
		),
	)

	DescribeTable(
		"should append a number as one sample of a data type",
		func(n payload.Number, dt telem.DataType, expected telem.Series) {
			Expect(n.Append(nil, dt)).To(Equal(expected.Data))
		},
		Entry("int8", payload.IntegerNumber(-2), telem.Int8T,
			telem.NewSeriesV[int8](-2)),
		Entry("uint16", payload.FloatNumber(300), telem.Uint16T,
			telem.NewSeriesV[uint16](300)),
		Entry("int32", payload.UnsignedNumber(70000), telem.Int32T,
			telem.NewSeriesV[int32](70000)),
		Entry("uint64", payload.UnsignedNumber(math.MaxUint64), telem.Uint64T,
			telem.NewSeriesV[uint64](math.MaxUint64)),
		Entry("float32", payload.IntegerNumber(3), telem.Float32T,
			telem.NewSeriesV[float32](3)),
		Entry("float64", payload.FloatNumber(2.5), telem.Float64T,
			telem.NewSeriesV(2.5)),
	)
})
