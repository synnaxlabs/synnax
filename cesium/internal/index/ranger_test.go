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
		db = MustSucceed(ranger.Open(ranger.Config{
			FS: fs.NewMemFS(),
		}))
		idx = &index.Ranger{
			DB:     db,
			Logger: MustSucceed(zap.NewDevelopment()),
		}
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("Distance", func() {
		Context("Continuous", func() {
			BeforeEach(func() {
				ranger.Write(
					db,
					(1 * telem.SecondTS).SpanRange(19*telem.Second+1),
					telem.NewSecondsTSV(1, 2, 3, 5, 7, 9, 15, 19, 20).Data,
				)
			})
			DescribeTable("Continuous",
				func(
					tr telem.TimeRange,
					expected int,
					expectedErr error,
				) {
					n, err := idx.Distance(tr /*continuous*/, true)
					if expectedErr != nil {
						Expect(err).To(HaveOccurredAs(expectedErr))
					} else {
						Expect(err).To(BeNil())
					}
					Expect(n).To(Equal(int64(expected)))
				},
				Entry("Zero zero", telem.TimeRangeZero, 0, index.ErrDiscontinuous),
				Entry("Empty range", (1*telem.SecondTS).SpanRange(0), 0, nil),
				Entry("Start at range start", (1*telem.SecondTS).SpanRange(1*telem.Second), 0, nil),
				Entry("Start at Ranger range end", (20*telem.SecondTS).SpanRange(1*telem.Second), 0, index.ErrDiscontinuous),
				Entry("Start at exclusive Ranger range end", (20*telem.SecondTS+1).SpanRange(1*telem.Second), 0, index.ErrDiscontinuous),
				Entry("Start before ranger start", (0*telem.SecondTS).SpanRange(1*telem.Second), 0, index.ErrDiscontinuous),
				Entry("Start after exclusive range end", (21*telem.SecondTS).SpanRange(1*telem.Second), 0, index.ErrDiscontinuous),
				Entry("End at range start", (0*telem.SecondTS).SpanRange(1*telem.Second), 0, index.ErrDiscontinuous),
				Entry("End at range end", (19*telem.SecondTS).SpanRange(1*telem.Second), 0, nil),
				Entry("End at exclusive range end", (19*telem.SecondTS+1).SpanRange(1*telem.Second), 0, nil),
				Entry("End after exclusive range end", (19*telem.SecondTS+2).SpanRange(1*telem.Second), 0, index.ErrDiscontinuous),
				Entry("End before exclusive range start", (0*telem.SecondTS).SpanRange(1*telem.Second), 0, index.ErrDiscontinuous),
				Entry("Start and range start, end at range end", (1*telem.SecondTS).SpanRange(19*telem.Second), 7, nil),
				Entry("Both in range - exact start and exact end", (2*telem.SecondTS).SpanRange(2*telem.Second), 1, nil),
				Entry("Both in range - exact start  and inexact end", (2*telem.SecondTS).SpanRange(3*telem.Second), 1, nil),
				Entry("Both in range - inexact start and exact end", (4*telem.SecondTS).SpanRange(5*telem.Second), 1, nil),
				Entry("Both in range - inexact start and inexact end", (4*telem.SecondTS).SpanRange(6*telem.Second), 2, nil),
			)
		})
	})
	Describe("Stamp", func() {
		Context("Continuous", func() {
			BeforeEach(func() {
				ranger.Write(
					db,
					(1 * telem.SecondTS).SpanRange(19*telem.Second+1),
					telem.NewSecondsTSV(1, 2, 3, 5, 7, 9, 15, 19, 20).Data,
				)
			})
			DescribeTable("Continuous", func(
				start telem.TimeStamp,
				sampleCount int,
				expected telem.TimeStamp,
				expectedErr error,
			) {
				end, err := idx.Stamp(start, int64(sampleCount))
				if expectedErr != nil {
					Expect(err).To(HaveOccurredAs(expectedErr))
				} else {
					Expect(err).To(BeNil())
				}
				Expect(end).To(Equal(expected))
			},
				Entry("Zero zero", 0*telem.SecondTS, 0, 0*telem.SecondTS, index.ErrDiscontinuous),
				Entry("Empty range", 1*telem.SecondTS, 0, 1*telem.SecondTS, nil),
				Entry("Start at range start - length of range samples", 1*telem.SecondTS, 9, 0*telem.SecondTS, index.ErrDiscontinuous),
				Entry("Start at range start - length of range samples - 1", 1*telem.SecondTS, 8, 20*telem.SecondTS, nil),
				Entry("Start before range start", 0*telem.SecondTS, 1, 0*telem.SecondTS, index.ErrDiscontinuous),
				Entry("Start in range - end in range", 2*telem.SecondTS, 2, 5*telem.SecondTS, nil),
			)
		})
	})

})
