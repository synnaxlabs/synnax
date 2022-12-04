package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
)

var _ = Describe("Rate", func() {
	DescribeTable("Distance", func(tr telem.TimeRange, expected index.DistanceApproximation) {
		idx := index.Rate{Rate: 1 * telem.Hz, Logger: zap.NewNop()}
		actual := MustSucceed(idx.Distance(tr, true))
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
		Entry("TimeRange - inexact start exact end",
			(1500*telem.MillisecondTS).Range(5*telem.SecondTS),
			index.Between[int64](3, 4),
		),
		Entry("TimeRange - inexact start inexact end",
			(3500*telem.MillisecondTS).Range(6500*telem.MillisecondTS),
			index.Between[int64](2, 4),
		),
	)
	Describe("Stamp", func() {
		DescribeTable("Distance", func(ts telem.TimeStamp, dist int, expected index.TimeStampApproximation) {
			idx := index.Rate{Rate: 1 * telem.Hz, Logger: zap.NewNop()}
			actual := MustSucceed(idx.Stamp(ts, int64(dist), true))
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
