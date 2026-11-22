// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package strconv_test

import (
	"math"
	"math/rand/v2"
	stdstrconv "strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/strconv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AppendFloat", func() {
	DescribeTable(
		"Should append the shortest text of a float",
		func(v float64, bitSize int, expected string) {
			Expect(string(strconv.AppendFloat(nil, v, bitSize))).To(Equal(expected))
		},
		Entry("decimal", -1.5, 64, "-1.5"),
		Entry("integer-valued", 1e10, 64, "10000000000"),
		Entry("below 1e21", 1e20, 64, "100000000000000000000"),
		Entry("from 1e21", 1e21, 64, "1e+21"),
		Entry("at 1e-6", 1e-6, 64, "0.000001"),
		Entry("below 1e-6", 1.5e-7, 64, "1.5e-7"),
		Entry("maximum", math.MaxFloat64, 64, "1.7976931348623157e+308"),
		Entry("smallest", 5e-324, 64, "5e-324"),
		Entry("zero", 0.0, 64, "0"),
		Entry("negative zero", math.Copysign(0, -1), 64, "-0"),
		Entry("NaN", math.NaN(), 64, "NaN"),
		Entry("infinity", math.Inf(1), 64, "Infinity"),
		Entry("negative infinity", math.Inf(-1), 64, "-Infinity"),
		Entry("float32 at its own precision", float64(float32(0.1)), 32, "0.1"),
		Entry("float32 with an exponent", float64(float32(1e30)), 32, "1e+30"),
		Entry("float32 at 1e-6", float64(float32(1e-6)), 32, "0.000001"),
		Entry("float32 below 1e-6", float64(float32(1.5e-7)), 32, "1.5e-7"),
		Entry(
			"float32 tie to even",
			float64(float32(0.000244140625)),
			32,
			"0.00024414062",
		),
		Entry("float32 infinity", float64(float32(math.Inf(1))), 32, "Infinity"),
		Entry(
			"float32 with fewer digits than its integer part",
			float64(float32(1e20)),
			32,
			"100000000000000000000",
		),
		Entry(
			"float64 with fewer digits than its integer part",
			1000000000000000128.0,
			64,
			"1000000000000000100",
		),
	)

	It("Should format the text it appends", func() {
		Expect(strconv.FormatFloat(1.5e-7, 64)).To(Equal("1.5e-7"))
	})

	It("Should append after the existing content of dst", func() {
		Expect(string(strconv.AppendFloat([]byte("x="), 2, 64))).To(Equal("x=2"))
	})

	It("Should append text that parses back to the same bits", func() {
		r := rand.New(rand.NewPCG(1, 2))
		for range 100_000 {
			v64 := math.Float64frombits(r.Uint64())
			v32 := math.Float32frombits(r.Uint32())
			if math.IsNaN(v64) || math.IsNaN(float64(v32)) {
				continue
			}
			text := string(strconv.AppendFloat(nil, v64, 64))
			parsed := MustSucceed(stdstrconv.ParseFloat(text, 64))
			Expect(math.Float64bits(parsed)).To(Equal(math.Float64bits(v64)))
			text = string(strconv.AppendFloat(nil, float64(v32), 32))
			parsed = MustSucceed(stdstrconv.ParseFloat(text, 32))
			Expect(math.Float32bits(float32(parsed))).To(Equal(math.Float32bits(v32)))
		}
	})
})
