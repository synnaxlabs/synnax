// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rate", func() {
	DescribeTable("Distance", func(tr telem.TimeRange, expected index.DistanceApproximation) {
		idx := index.Rate{Rate: 1 * telem.Hz}
		actual, _ := MustSucceed2(idx.Distance(ctx, tr, true))
		Expect(actual).To(Equal(expected))
	},
		Entry("Zero zero",
			telem.TimeRangeZero,
			index.Exactly[int64](0),
		),
		Entry("Empty range - exact stamp",
			(1*telem.SecondTS).SpanRange(0),
			index.Exactly[int64](0),
		),
		Entry("TimeRange - exact start exact end",
			(1*telem.SecondTS).Range(2*telem.SecondTS),
			index.Exactly[int64](1),
		),
		Entry("TimeRange - exact start inexact end",
			(1*telem.SecondTS).Range(2500*telem.MillisecondTS),
			index.Between[int64](1, 2),
		),
		Entry("TimeRange - inexact start inexact end",
			(3500*telem.MillisecondTS).Range(6500*telem.MillisecondTS),
			index.Between[int64](2, 4),
		),
		Entry("Timerange - inexact start exact end",
			(3999*telem.MillisecondTS).Range(6000*telem.MillisecondTS),
			index.Between[int64](2, 3),
		),
	)
	Describe("Stamp", func() {
		DescribeTable("Distance", func(ts telem.TimeStamp, dist int, expected index.TimeStampApproximation) {
			idx := index.Rate{Rate: 1 * telem.Hz}
			actual := MustSucceed(idx.Stamp(ctx, ts, int64(dist), true))
			Expect(actual).To(Equal(expected))
		},
			Entry("Zero zero",
				telem.TimeStamp(0),
				0,
				index.Exactly[telem.TimeStamp](0),
			),
			Entry("Exact start",
				1*telem.SecondTS,
				2,
				index.Exactly[telem.TimeStamp](3*telem.SecondTS),
			),
			Entry("Exact end",
				1500*telem.MillisecondTS,
				2,
				index.Between[telem.TimeStamp](3*telem.SecondTS, 4*telem.SecondTS),
			),
		)
	})
})
