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
	"github.com/google/uuid"
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
		Entry("float64 without an exponent", telem.NewSeriesV(1e21), "1"+zeros(21)),
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

	It("Should append after the existing content of dst", func() {
		s := telem.NewSeriesV[int32](42)
		text := MustSucceed(telem.AppendSampleText([]byte("x="), s.DataType, s.At(0)))
		Expect(string(text)).To(Equal("x=42"))
	})

	It("Should return an error for a data type with no text form", func() {
		Expect(telem.AppendSampleText(nil, telem.UnknownT, nil)).Error().
			To(MatchError(ContainSubstring("has no text form")))
	})
})

func zeros(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = '0'
	}
	return string(b)
}
