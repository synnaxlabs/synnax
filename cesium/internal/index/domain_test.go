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
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Domain", func() {
	var (
		db  *domain.DB
		idx index.Index
	)
	BeforeEach(func() {
		db = MustSucceed(domain.Open(domain.Config{FS: fs.NewMem()}))
		idx = &index.Domain{DB: db}
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Distance", func() {
		Context("Continuous", func() {
			BeforeEach(func() {
				Expect(domain.Write(
					ctx,
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
					actual, err := idx.Distance(ctx, tr /*continuous*/, true)
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
				Expect(domain.Write(
					ctx,
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
				actual, err := idx.Stamp(ctx, start, int64(distance), true)
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
		Context("Discontinuous", func() {
			BeforeEach(func() {
				Expect(domain.Write(
					ctx,
					db,
					(1 * telem.SecondTS).Range(20*telem.SecondTS+1),
					telem.NewSecondsTSV(1, 2, 3, 5, 7, 9, 15, 19, 20).Data,
				)).To(Succeed())
				Expect(domain.Write(
					ctx,
					db,
					(30 * telem.SecondTS).Range(40*telem.SecondTS+1),
					telem.NewSecondsTSV(30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40).Data,
				))
				Expect(domain.Write(
					ctx,
					db,
					(55 * telem.SecondTS).Range(65*telem.SecondTS+1),
					telem.NewSecondsTSV(55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65).Data,
				))
			})
			DescribeTable("Discontinuous", func(
				start telem.TimeStamp,
				distance int,
				expected index.TimeStampApproximation,
				expectedErr error,
			) {
				actual, err := idx.Stamp(ctx, start, int64(distance), false)
				if expectedErr != nil {
					Expect(err).To(HaveOccurredAs(expectedErr))
				} else {
					Expect(err).To(BeNil())
				}
				Expect(actual).To(Equal(expected))
			},
				Entry("Crossing TimeRange",
					2*telem.SecondTS,
					15,
					index.Exactly(37*telem.SecondTS),
					nil,
				),
				Entry("Crossing Multiple Ranges",
					2*telem.SecondTS,
					27,
					index.Exactly(63*telem.SecondTS),
					nil,
				),
				Entry("End of last TimeRange",
					2*telem.SecondTS,
					30,
					index.Between(65*telem.SecondTS+1, telem.TimeStampMax),
					nil,
				),
				Entry("After All Ranges",
					2*telem.SecondTS,
					500,
					index.Between(65*telem.SecondTS+1, telem.TimeStampMax),
					nil,
				),
				Entry("Exactly at End of First TimeRange",
					5*telem.SecondTS,
					6,
					index.Exactly(30*telem.SecondTS),
					nil,
				),
			)
		})
	})

})
