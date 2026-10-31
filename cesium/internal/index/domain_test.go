// Copyright 2026 Synnax Labs, Inc.
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
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Domain", func() {
	for fsName, openFS := range FileSystems {
		Describe("FS:"+fsName, Ordered, func() {
			var (
				db  *domain.DB
				idx *index.Domain
				fs  xfs.FS
			)
			BeforeEach(func() {
				fs = openFS()
				db = MustSucceed(
					domain.Open(domain.Config{FS: fs, Instrumentation: PanicLogger()}),
				)
				idx = &index.Domain{DB: db}
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
			})
			// openRolledOver opens a domain DB whose file size holds 3 timestamps, so a
			// commit past it rolls over into an effectively contiguous domain. It holds
			// [10 11 16 17], [18 19 20 22], and [25 26], then [30 32 33] after a gap.
			openRolledOver := func(ctx SpecContext) *domain.DB {
				GinkgoHelper()
				db := MustOpen(domain.Open(domain.Config{
					FS:              MustSucceed(fs.Sub("rolled_over")),
					Instrumentation: PanicLogger(),
					FileSize:        telem.TimestampT.Density().Size(3),
				}))
				w := MustSucceed(
					db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS}),
				)
				MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10, 11, 16, 17).Data))
				Expect(w.Commit(ctx, 17*telem.SecondTS+1)).To(Succeed())
				MustSucceed(w.Write(telem.NewSeriesSecondsTSV(18, 19, 20, 22).Data))
				Expect(w.Commit(ctx, 22*telem.SecondTS+1)).To(Succeed())
				MustSucceed(w.Write(telem.NewSeriesSecondsTSV(25, 26).Data))
				Expect(w.Commit(ctx, 26*telem.SecondTS+1)).To(Succeed())
				Expect(w.Close()).To(Succeed())
				Expect(testutil.WriteDomain(
					ctx,
					db,
					(30 * telem.SecondTS).Range(33*telem.SecondTS+1),
					telem.NewSeriesSecondsTSV(30, 32, 33).Data,
				)).To(Succeed())
				return db
			}

			Describe("Distance", func() {
				Context("Continuous", func() {
					BeforeEach(func(ctx SpecContext) {
						Expect(testutil.WriteDomain(
							ctx,
							db,
							(1 * telem.SecondTS).Range(20*telem.SecondTS+1),
							telem.NewSeriesSecondsTSV(
								1,
								2,
								3,
								5,
								7,
								9,
								15,
								19,
								20,
							).Data,
						)).To(Succeed())
					})
					DescribeTable(
						"Continuous",
						func(ctx SpecContext,
							tr telem.TimeRange,
							expected index.Approximation[int64],
							expectedErr error,
						) {
							actual, err := idx.Distance(ctx, tr /*continuous*/, true)
							if expectedErr != nil {
								Expect(err).To(MatchError(expectedErr))
							} else {
								Expect(err).ToNot(HaveOccurred())
							}
							Expect(actual.Approximation).To(Equal(expected))
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
						Entry(
							"Start just before range start - end just after range end",
							(1*telem.SecondTS-1).Range(20*telem.SecondTS+1),
							index.Exactly[int64](0),
							index.ErrDiscontinuous,
						),
					)
				})

				Context("Discontinuous", func() {
					BeforeEach(func(ctx SpecContext) {
						Expect(testutil.WriteDomain(
							ctx,
							db,
							(1 * telem.SecondTS).Range(20*telem.SecondTS+1),
							telem.NewSeriesSecondsTSV(
								1,
								2,
								3,
								5,
								7,
								9,
								15,
								19,
								20,
							).Data,
						)).To(Succeed())
						Expect(testutil.WriteDomain(
							ctx,
							db,
							(25 * telem.SecondTS).Range(30*telem.SecondTS+1),
							telem.NewSeriesSecondsTSV(25, 26, 28, 30).Data,
						)).To(Succeed())
						Expect(testutil.WriteDomain(
							ctx,
							db,
							(40 * telem.SecondTS).Range(43*telem.SecondTS+1),
							telem.NewSeriesSecondsTSV(40, 42, 43).Data,
						)).To(Succeed())
					})
					DescribeTable(
						"Discontinuous",
						func(ctx SpecContext,
							tr telem.TimeRange,
							expected index.Approximation[int64],
							err error,
						) {
							actual, e := idx.Distance(ctx, tr, false)
							if err == nil {
								Expect(actual.Approximation).To(Equal(expected))
							} else {
								Expect(e).To(MatchError(err))
							}
						},
						Entry("Zero zero",
							telem.TimeRangeZero,
							index.Exactly[int64](0),
							index.ErrDiscontinuous,
						),
						Entry("Exact, start and end equal",
							(27*telem.SecondTS).SpanRange(0),
							index.Exactly[int64](0),
							nil,
						),
						Entry("Exact, start in domain, end not in domain",
							(15*telem.SecondTS).Range(22*telem.SecondTS),
							index.Exactly[int64](3),
							nil,
						),
						Entry("Inexact, start in domain, end not in domain",
							(14*telem.SecondTS).Range(22*telem.SecondTS),
							index.Between[int64](3, 4),
							nil,
						),
						Entry(
							"Exact, start in domain, end not in domain (after a domain)",
							(15*telem.SecondTS).Range(35*telem.SecondTS),
							index.Exactly[int64](7),
							nil,
						),
						Entry(
							"Inexact, start in domain end not in domain (after a domain)",
							(14*telem.SecondTS).Range(35*telem.SecondTS),
							index.Between[int64](7, 8),
							nil,
						),
						Entry("Exact, start in domain, end in domain",
							(15*telem.SecondTS).Range(42*telem.SecondTS),
							index.Exactly[int64](8),
							nil,
						),
						Entry(
							"End inexact, start in domain, end in domain",
							(15*telem.SecondTS).Range(
								42*telem.SecondTS+500*telem.MillisecondTS,
							),
							index.Between[int64](8, 9),
							nil,
						),
						Entry("Start inexact, start in domain, end in domain",
							(14*telem.SecondTS).Range(42*telem.SecondTS),
							index.Between[int64](8, 9),
							nil,
						),
						Entry(
							"Both inexact, start in domain, end in domain",
							(14*telem.SecondTS).Range(
								42*telem.SecondTS+500*telem.MillisecondTS,
							),
							index.Between[int64](8, 10),
							nil,
						),
						Entry("End exact, start not in domain, end in first domain",
							(-1*telem.SecondTS).Range(5*telem.SecondTS),
							index.Between[int64](3, 4),
							nil,
						),
						Entry("End inexact, start not in domain, end in first domain",
							(-1*telem.SecondTS).Range(6*telem.SecondTS),
							index.Between[int64](3, 5),
							nil,
						),
						Entry("End exact, start not in domain, end not in first domain",
							(-1*telem.SecondTS).Range(26*telem.SecondTS),
							index.Between[int64](10, 11),
							nil,
						),
						Entry(
							"End inexact, start not in domain, end not in first domain",
							(-1*telem.SecondTS).Range(27*telem.SecondTS),
							index.Between[int64](10, 12),
							nil,
						),
					)
				})

				Context("Quasi Continuous (Many Continuous Domains)", func() {
					var idx2 *index.Domain
					BeforeEach(func(ctx SpecContext) {
						idx2 = &index.Domain{DB: openRolledOver(ctx)}
					})
					DescribeTable("effectively continuous", func(ctx SpecContext,
						tr telem.TimeRange,
						expected index.Approximation[int64],
						err error,
					) {
						actual, e := idx2.Distance(ctx, tr, true)
						if err == nil {
							Expect(actual.Approximation).To(Equal(expected))
						} else {
							Expect(e).To(MatchError(err))
						}
					},
						Entry("exact exact",
							(19*telem.SecondTS).Range(22*telem.SecondTS),
							index.Exactly[int64](2),
							nil,
						),
						Entry("exact exact - between effectively continuous domains",
							(16*telem.SecondTS).Range(26*telem.SecondTS),
							index.Exactly[int64](7),
							nil,
						),
						Entry("exact exact - out of domain",
							(16*telem.SecondTS).Range(32*telem.SecondTS),
							index.Exactly[int64](0),
							index.ErrDiscontinuous,
						),
						Entry(
							"inexact",
							(12*telem.SecondTS).Range(
								25*telem.SecondTS+500*telem.MillisecondTS,
							),
							index.Between[int64](6, 8),
							nil,
						),
					)
				})
			})

			Describe("EndAfter", func() {
				BeforeEach(func(ctx SpecContext) {
					Expect(testutil.WriteDomain(
						ctx,
						db,
						(1 * telem.SecondTS).Range(5*telem.SecondTS+1),
						telem.NewSeriesSecondsTSV(1, 2, 3, 5).Data,
					)).To(Succeed())
					Expect(testutil.WriteDomain(
						ctx,
						db,
						(10 * telem.SecondTS).Range(12*telem.SecondTS+1),
						telem.NewSeriesSecondsTSV(10, 11, 12).Data,
					)).To(Succeed())
				})
				DescribeTable(
					"returns the end of the range holding n samples",
					func(
						ctx SpecContext,
						ref telem.TimeStamp,
						n int64,
						expected telem.TimeStamp,
					) {
						Expect(idx.EndAfter(ctx, ref, n)).To(Equal(expected))
					},
					Entry(
						"zero samples, after ref",
						1*telem.SecondTS,
						int64(0),
						1*telem.SecondTS+1,
					),
					Entry(
						"inside the first domain",
						1*telem.SecondTS,
						int64(2),
						3*telem.SecondTS,
					),
					Entry(
						"from a stamp between samples",
						4*telem.SecondTS,
						int64(1),
						10*telem.SecondTS,
					),
					Entry(
						"across the gap between domains",
						2*telem.SecondTS,
						int64(4),
						11*telem.SecondTS,
					),
					Entry(
						"from a stamp inside the gap",
						7*telem.SecondTS,
						int64(2),
						12*telem.SecondTS,
					),
					Entry(
						"before the first domain",
						telem.TimeStamp(0),
						int64(1),
						2*telem.SecondTS,
					),
					Entry(
						"past the last sample",
						1*telem.SecondTS,
						int64(7),
						telem.TimeStampMax,
					),
					Entry(
						"after every domain",
						20*telem.SecondTS,
						int64(1),
						telem.TimeStampMax,
					),
				)
				Context("Equal timestamps", func() {
					BeforeEach(func(ctx SpecContext) {
						Expect(testutil.WriteDomain(
							ctx,
							db,
							(19 * telem.SecondTS).Range(21*telem.SecondTS+1),
							telem.NewSeriesSecondsTSV(19, 20, 20, 20, 21).Data,
						)).To(Succeed())
					})
					DescribeTable(
						"ends after the whole run of equal timestamps",
						func(
							ctx SpecContext,
							ref telem.TimeStamp,
							n int64,
							expected telem.TimeStamp,
						) {
							Expect(idx.EndAfter(ctx, ref, n)).To(Equal(expected))
						},
						Entry(
							"inside the run",
							20*telem.SecondTS,
							int64(1),
							20*telem.SecondTS+1,
						),
						Entry(
							"past the run",
							20*telem.SecondTS,
							int64(2),
							21*telem.SecondTS,
						),
					)
				})
			})

			Describe("AppendSegments", func() {
				var rolledOver *index.Domain
				BeforeEach(func(ctx SpecContext) {
					rolledOver = &index.Domain{DB: openRolledOver(ctx)}
				})
				var (
					first  = (10 * telem.SecondTS).Range(17*telem.SecondTS + 1)
					second = (17*telem.SecondTS + 1).Range(22*telem.SecondTS + 1)
					third  = (22*telem.SecondTS + 1).Range(26*telem.SecondTS + 1)
				)
				DescribeTable(
					"Splitting a run",
					func(
						ctx SpecContext,
						ts telem.TimeStamp,
						start, end int64,
						expected []index.Segment,
					) {
						Expect(
							rolledOver.AppendSegments(ctx, nil, ts, start, end),
						).To(Equal(expected))
					},
					Entry("Should return one segment for samples inside a domain",
						10*telem.SecondTS, int64(0), int64(4),
						[]index.Segment{
							{
								TimeRange: first,
								Alignment: telem.NewAlignment(0, 0),
								Len:       4,
							},
						},
					),
					Entry("Should split samples at every domain boundary",
						10*telem.SecondTS, int64(0), int64(10),
						[]index.Segment{
							{
								TimeRange: first,
								Alignment: telem.NewAlignment(0, 0),
								Len:       4,
							},
							{
								TimeRange: second,
								Alignment: telem.NewAlignment(1, 0),
								Len:       4,
							},
							{
								TimeRange: third,
								Alignment: telem.NewAlignment(2, 0),
								Len:       2,
							},
						},
					),
					Entry("Should start a run partway through a domain",
						16*telem.SecondTS, int64(0), int64(3),
						[]index.Segment{
							{
								TimeRange: first,
								Alignment: telem.NewAlignment(0, 2),
								Len:       2,
							},
							{
								TimeRange: second,
								Alignment: telem.NewAlignment(1, 0),
								Len:       1,
							},
						},
					),
					Entry(
						"Should start a run at the first sample after an inexact stamp",
						12*telem.SecondTS,
						int64(0),
						int64(1),
						[]index.Segment{
							{
								TimeRange: first,
								Alignment: telem.NewAlignment(0, 2),
								Len:       1,
							},
						},
					),
					Entry("Should start a run at a domain that begins at the stamp",
						17*telem.SecondTS+1, int64(0), int64(2),
						[]index.Segment{
							{
								TimeRange: second,
								Alignment: telem.NewAlignment(1, 0),
								Len:       2,
							},
						},
					),
					Entry("Should skip to the samples of a later domain",
						10*telem.SecondTS, int64(5), int64(7),
						[]index.Segment{
							{
								TimeRange: second,
								Alignment: telem.NewAlignment(1, 1),
								Len:       2,
							},
						},
					),
					Entry("Should split samples that straddle a boundary",
						10*telem.SecondTS, int64(3), int64(5),
						[]index.Segment{
							{
								TimeRange: first,
								Alignment: telem.NewAlignment(0, 3),
								Len:       1,
							},
							{
								TimeRange: second,
								Alignment: telem.NewAlignment(1, 0),
								Len:       1,
							},
						},
					),
					Entry("Should return no segments for an empty range",
						10*telem.SecondTS, int64(3), int64(3),
						[]index.Segment(nil),
					),
				)
				It("Should append to the given segments", func(ctx SpecContext) {
					dst := []index.Segment{{Len: 7}}
					Expect(
						rolledOver.AppendSegments(ctx, dst, 26*telem.SecondTS, 0, 1),
					).To(Equal(
						[]index.Segment{
							{Len: 7},
							{
								TimeRange: third,
								Alignment: telem.NewAlignment(2, 1),
								Len:       1,
							},
						},
					))
				})
				DescribeTable(
					"Discontinuous runs",
					func(ctx SpecContext, ts telem.TimeStamp, end int64, msg string) {
						Expect(
							rolledOver.AppendSegments(ctx, nil, ts, 0, end),
						).Error().To(SatisfyAll(
							MatchError(index.ErrDiscontinuous),
							MatchError(ContainSubstring(msg)),
						))
					},
					Entry(
						"Should fail when the run passes the effective domain",
						10*telem.SecondTS,
						int64(11),
						"position 11 in continuous index of length 10",
					),
					Entry(
						"Should fail when no domain holds the stamp",
						40*telem.SecondTS,
						int64(1),
						"does not exist in the index",
					),
				)
			})

			Describe("Stamp", func() {
				Context("Forward", func() {
					Context("Continuous", func() {
						BeforeEach(func(ctx SpecContext) {
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(1 * telem.SecondTS).SpanRange(19*telem.Second+1),
								telem.NewSeriesSecondsTSV(
									1,
									2,
									3,
									5,
									7,
									9,
									15,
									19,
								).Data,
							)).To(Succeed())
						})
						DescribeTable("Continuous", func(ctx SpecContext,
							start telem.TimeStamp,
							distance int,
							expected index.TimeStampApproximation,
							expectedErr error,
						) {
							actual, err := idx.Stamp(ctx, start, int64(distance), true)
							if expectedErr != nil {
								Expect(err).To(MatchError(expectedErr))
							} else {
								Expect(err).ToNot(HaveOccurred())
							}
							Expect(actual).To(Equal(expected))
						},
							Entry("Zero zero",
								0*telem.SecondTS,
								0,
								index.Exactly[telem.TimeStamp](0),
								domain.ErrRangeNotFound,
							),
							Entry("Empty range, exact",
								19*telem.SecondTS,
								0,
								index.Exactly(19*telem.SecondTS),
								nil,
							),
							Entry("Empty range, inexact",
								4*telem.SecondTS,
								0,
								index.Between(0*telem.SecondTS, 5*telem.SecondTS),
								nil,
							),
							Entry("Empty range, end",
								21*telem.SecondTS,
								0,
								index.Approximation[telem.TimeStamp]{},
								domain.ErrRangeNotFound,
							),
							Entry("Ref in range and exact, distance in range",
								2*telem.SecondTS,
								3,
								index.Exactly(7*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and exact, distance out of range",
								19*telem.SecondTS,
								4,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry("Ref in range and inexact",
								4*telem.SecondTS,
								3,
								index.Between(9*telem.SecondTS, 15*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and inexact, distance on the edge",
								4*telem.SecondTS,
								6,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
						)
					})

					Context("Quasi-Continuous (Many Continuous domains)", func() {
						BeforeEach(func(ctx SpecContext) {
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(1 * telem.SecondTS).Range(19*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									1,
									2,
									3,
									5,
									7,
									9,
									15,
									19,
								).Data,
							)).To(Succeed())

							Expect(testutil.WriteDomain(
								ctx,
								db,
								(19*telem.SecondTS + 1).Range(26*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(20, 21, 22, 23, 25, 26).Data,
							)).To(Succeed())

							Expect(testutil.WriteDomain(
								ctx,
								db,
								(26*telem.SecondTS + 1).Range(35*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									27,
									29,
									30,
									31,
									32,
									34,
									35,
								).Data,
							)).To(Succeed())

							Expect(testutil.WriteDomain(
								ctx,
								db,
								(40 * telem.SecondTS).Range(45*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(40, 41, 45).Data,
							)).To(Succeed())
						})
						DescribeTable(
							"Quasi-continuous",
							func(ctx SpecContext,
								start telem.TimeStamp,
								distance int,
								expected index.TimeStampApproximation,
								expectedErr error,
							) {
								actual, err := idx.Stamp(
									ctx,
									start,
									int64(distance),
									true,
								)
								if expectedErr != nil {
									Expect(err).To(MatchError(expectedErr))
								} else {
									Expect(err).ToNot(HaveOccurred())
								}
								Expect(actual).To(Equal(expected))
							},
							Entry("Empty range",
								19*telem.SecondTS+1,
								0,
								index.Between(0, 20*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and exact, distance in range",
								2*telem.SecondTS,
								13,
								index.Exactly(27*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and exact, distance out of range",
								2*telem.SecondTS,
								20,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry("Ref in range and exact, distance out of range",
								2*telem.SecondTS,
								40,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry(
								"Ref in range an exact, distance at the end of domain",
								2*telem.SecondTS,
								12,
								index.Exactly(26*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref in range and exact, distance at the start of domain",
								2*telem.SecondTS,
								13,
								index.Exactly(27*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and inexact",
								4*telem.SecondTS,
								12,
								index.Between(27*telem.SecondTS, 29*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref in range and inexact, and end is between two domains",
								4*telem.SecondTS,
								11,
								index.Between(26*telem.SecondTS, 27*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and inexact, distance out of range",
								4*telem.SecondTS,
								17,
								index.Between(34*telem.SecondTS, 35*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref in range and exact, distance partially out of range",
								4*telem.SecondTS,
								18,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry(
								"Ref in range and exact, distance totally out of range",
								10*telem.SecondTS,
								50,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry("Ref between two domains, distance in range",
								19*telem.SecondTS+500*telem.MillisecondTS,
								1,
								index.Between(20*telem.SecondTS, 21*telem.SecondTS),
								nil,
							),
							Entry("Ref between two domains, distance 0",
								19*telem.SecondTS+500*telem.MillisecondTS,
								0,
								index.Between(0*telem.SecondTS, 20*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref between two domains, distance between two domains",
								19*telem.SecondTS+500*telem.MillisecondTS,
								6,
								index.Between(26*telem.SecondTS, 27*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref between two domains, distance partially out of range",
								19*telem.SecondTS+500*telem.MillisecondTS,
								13,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
						)
					})

					Specify(
						"Quasi-Continuous Without Ending Domain",
						func(ctx SpecContext) {
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(1 * telem.SecondTS).Range(19*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									1,
									2,
									3,
									5,
									7,
									9,
									15,
									19,
								).Data,
							)).To(Succeed())

							Expect(testutil.WriteDomain(
								ctx,
								db,
								(19*telem.SecondTS + 1).Range(26*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(20, 21, 22, 23, 25, 26).Data,
							)).To(Succeed())

							Expect(testutil.WriteDomain(
								ctx,
								db,
								(26*telem.SecondTS + 1).Range(35*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									27,
									29,
									30,
									31,
									32,
									34,
									35,
								).Data,
							)).To(Succeed())

							Expect(
								MustSucceed(idx.Stamp(ctx, 25*telem.SecondTS, 8, true)),
							).To(Equal(index.Exactly(35 * telem.SecondTS)))
							_, err := idx.Stamp(ctx, 25*telem.SecondTS, 9, true)
							Expect(err).To(MatchError(index.ErrDiscontinuous))
							approx := MustSucceed(
								idx.Stamp(ctx, 24*telem.SecondTS, 8, true),
							)
							Expect(
								approx,
							).To(Equal(index.Between(34*telem.SecondTS, 35*telem.SecondTS)))
							_, err = idx.Stamp(ctx, 24*telem.SecondTS, 9, true)
							Expect(err).To(MatchError(index.ErrDiscontinuous))
						},
					)

					Context("Discontinuous", func() {
						BeforeEach(func(ctx SpecContext) {
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(1 * telem.SecondTS).Range(20*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									1,
									2,
									3,
									5,
									7,
									9,
									15,
									19,
									20,
								).Data,
							)).To(Succeed())
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(30 * telem.SecondTS).Range(40*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									30,
									31,
									32,
									33,
									34,
									35,
									36,
									37,
									38,
									39,
									40,
								).Data,
							)).To(Succeed())
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(55 * telem.SecondTS).Range(65*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									55,
									56,
									57,
									58,
									59,
									60,
									61,
									62,
									63,
									64,
									65,
								).Data,
							)).To(Succeed())
						})
						DescribeTable("Discontinuous", func(ctx SpecContext,
							start telem.TimeStamp,
							distance int,
							expected index.TimeStampApproximation,
							expectedErr error,
						) {
							actual, err := idx.Stamp(ctx, start, int64(distance), false)
							if expectedErr != nil {
								Expect(err).To(MatchError(expectedErr))
							} else {
								Expect(err).ToNot(HaveOccurred())
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

				Context("Backward", func() {
					Context("Continuous", func() {
						BeforeEach(func(ctx SpecContext) {
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(1 * telem.SecondTS).Range(19*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									1,
									2,
									3,
									5,
									7,
									9,
									15,
									19,
								).Data,
							)).To(Succeed())
						})
						DescribeTable("Continuous", func(ctx SpecContext,
							start telem.TimeStamp,
							distance int,
							expected index.TimeStampApproximation,
							expectedErr error,
						) {
							actual, err := idx.Stamp(ctx, start, int64(distance), true)
							if expectedErr != nil {
								Expect(err).To(MatchError(expectedErr))
							} else {
								Expect(err).ToNot(HaveOccurred())
							}
							Expect(actual).To(Equal(expected))
						},
							Entry("Ref in range and exact, distance in range",
								7*telem.SecondTS,
								-3,
								index.Exactly(2*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and exact, distance out of range",
								1*telem.SecondTS,
								-4,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry("Ref in range and inexact",
								4*telem.SecondTS,
								-2,
								index.Between(1*telem.SecondTS, 2*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and inexact, distance on the edge",
								4*telem.SecondTS,
								-6,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry("Ref at end, distance in range",
								19*telem.SecondTS+1,
								-3,
								index.Exactly(7*telem.SecondTS),
								nil,
							),
						)
					})

					Context("Quasi-Continuous (Many Continuous domains)", func() {
						BeforeEach(func(ctx SpecContext) {
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(1 * telem.SecondTS).Range(19*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									1,
									2,
									3,
									5,
									7,
									9,
									15,
									19,
								).Data,
							)).To(Succeed())

							Expect(testutil.WriteDomain(
								ctx,
								db,
								(19*telem.SecondTS + 1).Range(26*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(20, 21, 22, 23, 25, 26).Data,
							)).To(Succeed())

							Expect(testutil.WriteDomain(
								ctx,
								db,
								(26*telem.SecondTS + 1).Range(35*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									27,
									29,
									30,
									31,
									32,
									34,
									35,
								).Data,
							)).To(Succeed())

							Expect(testutil.WriteDomain(
								ctx,
								db,
								(40 * telem.SecondTS).Range(45*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(40, 41, 45).Data,
							)).To(Succeed())
						})
						DescribeTable(
							"Quasi-continuous",
							func(ctx SpecContext,
								start telem.TimeStamp,
								distance int,
								expected index.TimeStampApproximation,
								expectedErr error,
							) {
								actual, err := idx.Stamp(
									ctx,
									start,
									int64(distance),
									true,
								)
								if expectedErr != nil {
									Expect(err).To(MatchError(expectedErr))
								} else {
									Expect(err).ToNot(HaveOccurred())
								}
								Expect(actual).To(Equal(expected))
							},
							Entry("Empty range",
								19*telem.SecondTS+1,
								0,
								index.Between(0, 20*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and exact, distance in range",
								27*telem.SecondTS,
								-13,
								index.Exactly(2*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and exact, distance out of range",
								2*telem.SecondTS,
								-20,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry("Ref in range and exact, distance out of range",
								2*telem.SecondTS,
								-40,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry(
								"Ref in range an exact, distance at the end of domain",
								27*telem.SecondTS,
								-12,
								index.Exactly(3*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref in range and exact, distance at the start of domain",
								27*telem.SecondTS,
								-14,
								index.Exactly(1*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and inexact",
								28*telem.SecondTS,
								-12,
								index.Between(3*telem.SecondTS, 5*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref in range and inexact, and end is between two domains",
								28*telem.SecondTS,
								-7,
								index.Between(19*telem.SecondTS, 20*telem.SecondTS),
								nil,
							),
							Entry("Ref in range and inexact, distance out of range",
								33*telem.SecondTS,
								-17,
								index.Between(2*telem.SecondTS, 3*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref in range and exact, distance partially out of range",
								33*telem.SecondTS,
								-19,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry(
								"Ref in range and exact, distance totally out of range",
								35*telem.SecondTS,
								-50,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
							Entry("Ref between two domains, distance in range",
								19*telem.SecondTS+500*telem.MillisecondTS,
								-1,
								index.Between(15*telem.SecondTS, 19*telem.SecondTS),
								nil,
							),
							Entry("Ref between two domains, distance 0",
								19*telem.SecondTS+500*telem.MillisecondTS,
								0,
								index.Between(0*telem.SecondTS, 20*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref between two domains, distance between two domains",
								26*telem.SecondTS+500*telem.MillisecondTS,
								-6,
								index.Between(19*telem.SecondTS, 20*telem.SecondTS),
								nil,
							),
							Entry(
								"Ref between two domains, distance partially out of range",
								26*telem.SecondTS+500*telem.MillisecondTS,
								-14,
								index.Exactly[telem.TimeStamp](0),
								index.ErrDiscontinuous,
							),
						)
					})

					Context("Discontinuous", func() {
						BeforeEach(func(ctx SpecContext) {
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(1 * telem.SecondTS).Range(20*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									1,
									2,
									3,
									5,
									7,
									9,
									15,
									19,
									20,
								).Data,
							)).To(Succeed())
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(30 * telem.SecondTS).Range(40*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									30,
									31,
									32,
									33,
									34,
									35,
									36,
									37,
									38,
									39,
									40,
								).Data,
							)).To(Succeed())
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(55 * telem.SecondTS).Range(65*telem.SecondTS+1),
								telem.NewSeriesSecondsTSV(
									55,
									56,
									57,
									58,
									59,
									60,
									61,
									62,
									63,
									64,
									65,
								).Data,
							)).To(Succeed())
						})
						DescribeTable("Discontinuous", func(ctx SpecContext,
							start telem.TimeStamp,
							distance int,
							expected index.TimeStampApproximation,
							expectedErr error,
						) {
							actual, err := idx.Stamp(ctx, start, int64(distance), false)
							if expectedErr != nil {
								Expect(err).To(MatchError(expectedErr))
							} else {
								Expect(err).ToNot(HaveOccurred())
							}
							Expect(actual).To(Equal(expected))
						},
							Entry("Crossing TimeRange",
								37*telem.SecondTS,
								-15,
								index.Exactly(2*telem.SecondTS),
								nil,
							),
							Entry("Crossing Multiple Ranges",
								63*telem.SecondTS,
								-27,
								index.Exactly(2*telem.SecondTS),
								nil,
							),
							Entry("Start of first TimeRange",
								1*telem.SecondTS,
								-1,
								index.Between(telem.TimeStampMin, 1*telem.SecondTS),
								nil,
							),
							Entry("Before All Ranges",
								1*telem.SecondTS,
								-500,
								index.Between(telem.TimeStampMin, 1*telem.SecondTS),
								nil,
							),
							Entry("Exactly at Start of Second TimeRange",
								30*telem.SecondTS,
								-6,
								index.Exactly(5*telem.SecondTS),
								nil,
							),
							Entry("Landing on the first sample of a previous TimeRange",
								31*telem.SecondTS+500*telem.MillisecondTS,
								-2,
								index.Between(20*telem.SecondTS, 30*telem.SecondTS),
								nil,
							),
							Entry("Lower bound before the first sample",
								31*telem.SecondTS+500*telem.MillisecondTS,
								-11,
								index.Between(telem.TimeStampMin, 1*telem.SecondTS),
								nil,
							),
						)
					})
				})
			})
		})
	}
})
