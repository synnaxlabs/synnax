package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
)

var _ = Describe("Ranger", func() {
	var (
		db  *ranger.DB
		idx index.Index
	)
	BeforeEach(func() {
		db = MustSucceed(ranger.Open(ranger.Config{FS: fs.NewMem()}))
		idx = &index.Ranger{DB: db, Logger: zap.NewNop()}
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Distance", func() {
		Context("Continuous", func() {
			BeforeEach(func() {
				Expect(ranger.Write(
					db,
					(1 * telem.SecondTS).Range(20*telem.SecondTS+1),
					telem.NewSecondsTSV(1, 2, 3, 5, 7, 9, 15, 19, 20).Data,
				)).To(Succeed())
			})
			DescribeTable("Continuous",
				func(
					tr telem.TimeRange,
					expected index.DistanceApproximation,
					expectedErr error,
				) {
					actual, err := idx.Distance(tr /*continuous*/, true)
					if expectedErr != nil {
						Expect(err).To(HaveOccurredAs(expectedErr))
					} else {
						Expect(err).To(BeNil())
					}
					Expect(actual).To(Equal(expected))
				},
				Entry("Zero zero",
					telem.TimeRangeZero,
					index.Exactly[int64](0),
					index.ErrDiscontinuous,
				),
				Entry("Empty range - exact stamp",
					(1*telem.SecondTS).SpanRange(0),
					index.Exactly[int64](0),
					nil,
				),
				Entry("Empty range - inexact stamp",
					(4*telem.SecondTS).SpanRange(0),
					index.Exactly[int64](0),
					nil,
				),
				Entry("Both in range - exact start, exact end",
					(2*telem.SecondTS).SpanRange(3*telem.Second),
					index.Exactly[int64](2),
					nil,
				),
				Entry("Both in range - exact start, inexact end",
					(2*telem.SecondTS).SpanRange(4*telem.Second),
					index.Between[int64](2, 3),
					nil,
				),
				Entry("Both in range - inexact start, exact end",
					(4*telem.SecondTS).SpanRange(3*telem.Second),
					index.Between[int64](1, 2),
					nil,
				),
				Entry("Both in range - inexact start, inexact end",
					(4*telem.SecondTS).SpanRange(4*telem.Second),
					index.Between[int64](1, 3),
					nil,
				),
				Entry("Start at range start - exact end",
					(1*telem.SecondTS).SpanRange(1*telem.Second),
					index.Exactly[int64](1),
					nil,
				),
				Entry("Start exactly range end - end after range end",
					(20*telem.SecondTS).SpanRange(1*telem.Second),
					index.Exactly[int64](0),
					index.ErrDiscontinuous,
				),
				Entry("Start just beyond range end - end after range end",
					(20*telem.SecondTS+1).SpanRange(1*telem.Second),
					index.Exactly[int64](0),
					index.ErrDiscontinuous,
				),
				Entry("Start just before range end - end after range end",
					(20*telem.SecondTS-1).SpanRange(1*telem.Second),
					index.Exactly[int64](0),
					index.ErrDiscontinuous,
				),
				Entry("Start just before range end - end at range end",
					(20*telem.SecondTS-1).SpanRange(1),
					index.Between[int64](0, 1),
					nil,
				),
				Entry("Start exactly at range start - end exactly at range end",
					(1*telem.SecondTS).SpanRange(19*telem.Second),
					index.Exactly[int64](8),
					nil,
				),
				Entry("Start just before range start - end just after range end",
					(1*telem.SecondTS-1).Range(20*telem.SecondTS+1),
					index.Exactly[int64](0),
					index.ErrDiscontinuous,
				),
			)
		})
	})
	Describe("Stamp", func() {
		Context("Continuous", func() {
			BeforeEach(func() {
				Expect(ranger.Write(
					db,
					(1 * telem.SecondTS).SpanRange(19*telem.Second+1),
					telem.NewSecondsTSV(1, 2, 3, 5, 7, 9, 15, 19, 20).Data,
				)).To(Succeed())
			})
			DescribeTable("Continuous", func(
				start telem.TimeStamp,
				distance int,
				expected index.TimeStampApproximation,
				expectedErr error,
			) {
				actual, err := idx.Stamp(start, int64(distance))
				if expectedErr != nil {
					Expect(err).To(HaveOccurredAs(expectedErr))
				} else {
					Expect(err).To(BeNil())
				}
				Expect(actual).To(Equal(expected))
			},
				Entry("Zero zero",
					0*telem.SecondTS,
					0,
					index.Exactly[telem.TimeStamp](0),
					index.ErrDiscontinuous,
				),
				Entry("Empty range",
					1*telem.SecondTS,
					0,
					index.Exactly(1*telem.SecondTS),
					nil,
				),
				Entry("Ref in range and exact, distance in range",
					2*telem.SecondTS,
					3,
					index.Exactly(7*telem.SecondTS),
					nil,
				),
				Entry("Ref in range and exact, distance out of range",
					2*telem.SecondTS,
					20,
					index.Exactly[telem.TimeStamp](0),
					index.ErrDiscontinuous,
				),
				Entry("Ref in range and inexact",
					4*telem.SecondTS,
					3,
					index.Between[telem.TimeStamp](9*telem.SecondTS, 15*telem.SecondTS),
					nil,
				),
			)
		})
	})

})
