// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"math"
	"math/rand/v2"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AppendSampleText", func() {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	DescribeTable(
		"Should append the text form of a sample",
		func(s telem.Series, expected string) {
			text := MustSucceed(telem.AppendSampleText(nil, s.DataType, s.At(0)))
			Expect(string(text)).To(Equal(expected))
		},
		Entry("float64", telem.NewSeriesV(-1.5), "-1.5"),
		Entry("float64 below 1e21", telem.NewSeriesV(1e20), "100000000000000000000"),
		Entry("float64 from 1e21", telem.NewSeriesV(1e21), "1e+21"),
		Entry("float64 at 1e-6", telem.NewSeriesV(1e-6), "0.000001"),
		Entry("float64 below 1e-6", telem.NewSeriesV(1.5e-7), "1.5e-7"),
		Entry(
			"float64 maximum",
			telem.NewSeriesV(math.MaxFloat64),
			"1.7976931348623157e+308",
		),
		Entry("float64 smallest", telem.NewSeriesV(5e-324), "5e-324"),
		Entry("negative zero", telem.NewSeriesV(math.Copysign(0, -1)), "-0"),
		Entry("NaN", telem.NewSeriesV(math.NaN()), "NaN"),
		Entry("infinity", telem.NewSeriesV(math.Inf(1)), "Infinity"),
		Entry("negative infinity", telem.NewSeriesV(math.Inf(-1)), "-Infinity"),
		Entry(
			"float32 infinity",
			telem.NewSeriesV(float32(math.Inf(1))),
			"Infinity",
		),
		Entry("float32 with an exponent", telem.NewSeriesV[float32](1e30), "1e+30"),
		Entry("float32 at its own precision", telem.NewSeriesV[float32](0.1), "0.1"),
		Entry("int64", telem.NewSeriesV[int64](-9e18), "-9000000000000000000"),
		Entry("int32", telem.NewSeriesV[int32](-7), "-7"),
		Entry("int16", telem.NewSeriesV[int16](-7), "-7"),
		Entry("int8", telem.NewSeriesV[int8](-7), "-7"),
		Entry("uint64", telem.NewSeriesV[uint64](1<<63), "9223372036854775808"),
		Entry("uint32", telem.NewSeriesV[uint32](7), "7"),
		Entry("uint16", telem.NewSeriesV[uint16](7), "7"),
		Entry("uint8", telem.NewSeriesV[uint8](255), "255"),
		Entry("boolean", telem.NewSeriesV(true), "1"),
		Entry("timestamp", telem.NewSeriesSecondsTSV(2), "2000000000"),
		Entry("uuid", telem.NewSeriesV(id), id.String()),
		Entry("string", telem.NewSeriesV("a,b"), "a,b"),
		Entry(
			"json",
			MustSucceed(telem.NewJSONSeriesV(map[string]int{"a": 1})),
			`{"a":1}`,
		),
		Entry("bytes", telem.NewSeriesV([]byte{0, 255}), "AP8="),
	)

	It("Should append floats that parse back to the same bits", func() {
		r := rand.New(rand.NewPCG(1, 2))
		for range 100_000 {
			v64 := math.Float64frombits(r.Uint64())
			v32 := math.Float32frombits(r.Uint32())
			if math.IsNaN(v64) || math.IsNaN(float64(v32)) {
				continue
			}
			s64 := telem.NewSeriesV(v64)
			text := MustSucceed(telem.AppendSampleText(nil, s64.DataType, s64.At(0)))
			parsed := MustSucceed(strconv.ParseFloat(string(text), 64))
			Expect(math.Float64bits(parsed)).To(Equal(math.Float64bits(v64)))
			s32 := telem.NewSeriesV(v32)
			text = MustSucceed(telem.AppendSampleText(nil, s32.DataType, s32.At(0)))
			parsed = MustSucceed(strconv.ParseFloat(string(text), 32))
			Expect(math.Float32bits(float32(parsed))).To(Equal(math.Float32bits(v32)))
		}
	})

	It("Should append after the existing content of dst", func() {
		s := telem.NewSeriesV[int32](42)
		text := MustSucceed(telem.AppendSampleText([]byte("x="), s.DataType, s.At(0)))
		Expect(string(text)).To(Equal("x=42"))
	})

	It("Should return an error for a data type with no text form", func() {
		Expect(telem.AppendSampleText([]byte("a"), telem.UnknownT, nil)).Error().
			To(MatchError(ContainSubstring("has no text form")))
	})
})
