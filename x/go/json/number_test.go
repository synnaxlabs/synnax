// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// integer is the result of ParsedNumber.Int64 or ParsedNumber.Uint64.
type integer[T int64 | uint64] struct {
	value T
	ok    bool
}

var _ = Describe("ParseNumber", func() {
	DescribeTable("Should hold a number as an integer only when it is one exactly",
		func(text string, i integer[int64], u integer[uint64], f float64) {
			n := MustSucceed(xjson.ParseNumber([]byte(text)))
			v, ok := n.Int64()
			Expect(integer[int64]{v, ok}).To(Equal(i))
			w, ok := n.Uint64()
			Expect(integer[uint64]{w, ok}).To(Equal(u))
			Expect(n.Float64()).To(Equal(f))
		},
		Entry("a small integer", "3", integer[int64]{3, true},
			integer[uint64]{3, true}, 3.0),
		Entry("a whole decimal", "3.0", integer[int64]{3, true},
			integer[uint64]{3, true}, 3.0),
		Entry("a negative integer", "-3", integer[int64]{-3, true},
			integer[uint64]{}, -3.0),
		Entry("an integer past the precision of a float", "9007199254740993",
			integer[int64]{past2To53, true}, integer[uint64]{past2To53, true},
			9007199254740992.0),
		Entry("a fraction that a float rounds away", "9007199254740993.5",
			integer[int64]{}, integer[uint64]{}, 9007199254740994.0),
		Entry("an integer past an int64", "18446744073709551615", integer[int64]{},
			integer[uint64]{math.MaxUint64, true}, 18446744073709551615.0),
		Entry("the smallest int64", "-9223372036854775808",
			integer[int64]{math.MinInt64, true}, integer[uint64]{},
			-9223372036854775808.0),
		Entry("an integer below an int64", "-9223372036854775809", integer[int64]{},
			integer[uint64]{}, -9223372036854775808.0),
		Entry("an integer past a uint64", "1e20", integer[int64]{}, integer[uint64]{},
			1e20),
		Entry("negative zero", "-0.0", integer[int64]{0, true},
			integer[uint64]{0, true}, math.Copysign(0, -1)),
		Entry("a fraction", "2.5", integer[int64]{}, integer[uint64]{}, 2.5),
	)

	DescribeTable("Should reject text that is not a number",
		func(text, message string) {
			Expect(xjson.ParseNumber([]byte(text))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry("a word", "x", `invalid number "x"`),
		Entry("a number past a float64", "1e999", `invalid number "1e999"`),
	)
})

var _ = Describe("AppendFloat", func() {
	DescribeTable("Should append the shortest JSON number that round trips",
		func(v float64, expected string) {
			Expect(xjson.AppendFloat([]byte("x"), v)).To(Equal([]byte("x" + expected)))
		},
		Entry("a whole number", 40.0, "40"),
		Entry("a fraction", 0.1, "0.1"),
		Entry("a large number", 1e21, "1e+21"),
		Entry("a small number", 1e-9, "1e-9"),
	)

	DescribeTable(
		"Should reject a float that JSON cannot hold",
		func(v float64, expected string) {
			Expect(xjson.AppendFloat(nil, v)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
		},
		Entry("NaN", math.NaN(), "NaN is not a JSON number: validation error"),
		Entry(
			"infinity",
			math.Inf(1),
			"Infinity is not a JSON number: validation error",
		),
	)
})
