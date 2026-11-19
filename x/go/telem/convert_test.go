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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Convert", func() {
	DescribeTable("Should convert the samples of a series to a data type",
		func(s telem.Series, dt telem.DataType, expected telem.Series) {
			Expect(MustSucceed(telem.Convert(s, dt))).To(telem.MatchSeries(expected))
		},
		Entry("int16 to float64",
			telem.NewSeriesV[int16](-1, 300),
			telem.Float64T,
			telem.NewSeriesV[float64](-1, 300),
		),
		Entry("float64 to int32, truncating",
			telem.NewSeriesV(1.9, -1.9),
			telem.Int32T,
			telem.NewSeriesV[int32](1, -1),
		),
		Entry("int64 to float64, losing integers above 2^53",
			telem.NewSeriesV[int64](1<<53+1),
			telem.Float64T,
			telem.NewSeriesV[float64](1<<53),
		),
		Entry("uint8 to boolean",
			telem.NewSeriesV[uint8](0, 2),
			telem.BooleanT,
			telem.NewSeriesV(false, true),
		),
		Entry("boolean to float32",
			telem.NewSeriesV(true, false),
			telem.Float32T,
			telem.NewSeriesV[float32](1, 0),
		),
		Entry("timestamp to int64",
			telem.NewSeriesV[telem.TimeStamp](-1, telem.SecondTS),
			telem.Int64T,
			telem.NewSeriesV[int64](-1, int64(telem.SecondTS)),
		),
		Entry("float32 to timestamp",
			telem.NewSeriesV[float32](2),
			telem.TimestampT,
			telem.NewSeriesV[telem.TimeStamp](2),
		),
	)

	It("Should keep every other field of the series", func() {
		s := telem.NewSeriesV[int16](1, 2)
		s.Alignment, s.AlignmentMultiple, s.Clock = 10, 2, 7
		s.TimeRange = telem.TimeRange{Start: 1, End: 3}
		expected := telem.NewSeriesV[float32](1, 2)
		expected.Alignment, expected.AlignmentMultiple, expected.Clock = 10, 2, 7
		expected.TimeRange = s.TimeRange
		Expect(MustSucceed(telem.Convert(s, telem.Float32T))).
			To(telem.MatchSeries(expected))
	})

	It("Should return a series of the data type unchanged", func() {
		s := telem.NewSeriesV[int16](1, 2)
		converted := MustSucceed(telem.Convert(s, telem.Int16T))
		Expect(&converted.Data[0]).To(BeIdenticalTo(&s.Data[0]))
	})

	DescribeTable("Should fail between text and numbers",
		func(s telem.Series, dt telem.DataType, msg string) {
			Expect(telem.Convert(s, dt)).Error().To(MatchError(msg))
		},
		Entry("string to float32",
			telem.NewSeriesV("a"),
			telem.Float32T,
			"cannot convert a string series to float32",
		),
		Entry("float32 to string",
			telem.NewSeriesV[float32](1),
			telem.StringT,
			"cannot convert a float32 series to string",
		),
		Entry("uuid to int64",
			telem.Series{DataType: telem.UUIDT, Data: make([]byte, 16)},
			telem.Int64T,
			"cannot convert a uuid series to int64",
		),
	)
})
