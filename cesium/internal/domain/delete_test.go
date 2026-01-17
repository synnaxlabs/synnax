// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func fixedOffset(offset telem.Size) domain.OffsetResolver {
	return func(
		ctx context.Context,
		_ telem.TimeStamp,
		ts telem.TimeStamp,
	) (telem.Size, telem.TimeStamp, error) {
		return offset, ts, nil
	}
}

func fixedOffsetAndTimeStamp(offset telem.Size, ts telem.TimeStamp) domain.OffsetResolver {
	return func(
		_ context.Context,
		_ telem.TimeStamp,
		_ telem.TimeStamp,
	) (telem.Size, telem.TimeStamp, error) {
		return offset, ts, nil
	}
}

var _ = Describe("Delete", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, func() {
			var (
				db      *domain.DB
				fs      fs.FS
				cleanUp func() error
				density = telem.Uint8T.Density()
			)
			BeforeEach(func() {
				fs, cleanUp = makeFS()
				db = MustSucceed(domain.Open(domain.Config{FS: fs, Instrumentation: PanicLogger()}))

			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})

			Context("Single Pointer", func() {
				JustBeforeEach(func() {
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).SpanRange(10*telem.Second), []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})).To(Succeed())
				})

				DescribeTable("Delete Entire Pointer",
					func(startOffset, endOffset int, timeRange telem.TimeRange) {
						Expect(db.Delete(
							ctx,
							timeRange,
							fixedOffset(density.Size(int64(startOffset))),
							fixedOffset(density.Size(int64(endOffset))),
						)).To(Succeed())
						b := MustSucceed(domain.Read(ctx, db, telem.TimeRangeMax))
						Expect(b).To(BeNil())
					},
					Entry("if the time range end is beyond the domain end", 0, 0, (10*telem.SecondTS).Range(20*telem.SecondTS)),
					Entry("if the time range start is before the domain start and end is after it", 0, 0, telem.TimeRangeMax),
					Entry("if the time range start is before the domain start and end is after it with offset", 0, 10, (10*telem.SecondTS).Range(12*telem.SecondTS)),
				)

				DescribeTable("Should not delete anything under various offset conditions",
					func(startOffset, endOffset int, timeRange telem.TimeRange) {
						Expect(db.Delete(
							ctx,
							timeRange,
							fixedOffset(density.Size(int64(endOffset))),
							fixedOffset(density.Size(int64(startOffset))),
						)).To(Succeed())
						b := MustSucceed(domain.Read(ctx, db, telem.TimeRangeMax))
						Expect(b).To(Equal([]byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}))
						i := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(20 * telem.SecondTS)))
						Expect(i.Close()).To(Succeed())
					},
					Entry("when there is only one pointer and both offsets are 0",
						0, 0, (10*telem.SecondTS).Range(19*telem.SecondTS)),
					Entry("when both offsets are the same (non-zero)",
						5, 5, (10*telem.SecondTS).Range(12*telem.SecondTS)),
					Entry("when both offsets are much larger than the size of the domain",
						100, 100, (10*telem.SecondTS).Range(12*telem.SecondTS)),
				)

				It("Should correctly modify the time range of a domain when a new start is provided", func() {
					Expect(db.Delete(
						ctx,
						(10 * telem.SecondTS).Range(12*telem.SecondTS),
						fixedOffsetAndTimeStamp(0, 200*telem.SecondTS),
						fixedOffset(1),
					)).To(Succeed())
					b := MustSucceed(domain.Read(ctx, db, telem.TimeRangeMax))
					Expect(b).To(Equal([]byte{11, 12, 13, 14, 15, 16, 17, 18, 19}))
					i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
					Expect(i.SeekFirst(ctx)).To(BeTrue())
					Expect(i.TimeRange()).To(Equal((12 * telem.SecondTS).Range(20 * telem.SecondTS)))
					Expect(i.Close()).To(Succeed())
				})
			})

			Context("Multiple Pointers", func() {
				JustBeforeEach(func() {
					Expect(domain.Write(
						ctx,
						db,
						(10 * telem.SecondTS).SpanRange(10*telem.Second),
						[]byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
					)).To(Succeed())
					Expect(domain.Write(
						ctx,
						db,
						(22 * telem.SecondTS).SpanRange(8*telem.Second),
						[]byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29},
					)).To(Succeed())
					Expect(domain.Write(
						ctx,
						db,
						(30 * telem.SecondTS).SpanRange(10*telem.Second),
						[]byte{30, 31, 32, 33, 34, 35, 36, 37, 38, 39},
					)).To(Succeed())
				})
				type MultiPointerSpec struct {
					TimeRange                       telem.TimeRange
					StartOffset, EndOffset          int64
					FirstTimeRange, SecondTimeRange telem.TimeRange
					FirstData, SecondData           []byte
				}
				DescribeTable("Basic, continuous deletion of time range", func(
					cfg MultiPointerSpec,
				) {
					Expect(db.Delete(
						ctx,
						cfg.TimeRange,
						fixedOffset(density.Size(cfg.StartOffset)),
						fixedOffset(density.Size(cfg.EndOffset)),
					)).To(Succeed())
					iter := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal(cfg.FirstTimeRange))
					r := MustSucceed(iter.OpenReader(ctx))
					p := make([]byte, len(cfg.FirstData))
					_, err := r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:len(cfg.FirstData)]).To(Equal(cfg.FirstData))
					Expect(r.Close()).To(Succeed())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal(cfg.SecondTimeRange))
					r = MustSucceed(iter.OpenReader(ctx))
					p = make([]byte, len(cfg.SecondData))
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:len(cfg.SecondData)]).To(Equal(cfg.SecondData))
					Expect(iter.Close()).To(Succeed())
					Expect(r.Close()).To(Succeed())
				},
					Entry("across two pointers",
						MultiPointerSpec{
							TimeRange:       (13 * telem.SecondTS).Range(23 * telem.SecondTS),
							StartOffset:     3,
							EndOffset:       3,
							FirstTimeRange:  (10 * telem.SecondTS).Range(13 * telem.SecondTS),
							FirstData:       []byte{10, 11, 12},
							SecondTimeRange: (23 * telem.SecondTS).Range(30 * telem.SecondTS),
							SecondData:      []byte{23, 24, 25, 26, 27, 28},
						},
					),
					Entry("start at start of pointer",
						MultiPointerSpec{
							TimeRange:       (10 * telem.SecondTS).Range(23 * telem.SecondTS),
							StartOffset:     0,
							EndOffset:       3,
							FirstTimeRange:  (23 * telem.SecondTS).Range(30 * telem.SecondTS),
							FirstData:       []byte{23, 24, 25, 26, 27, 28, 29},
							SecondTimeRange: (30 * telem.SecondTS).Range(40 * telem.SecondTS),
							SecondData:      []byte{30, 31, 32, 33, 34, 35, 36, 37, 38, 39},
						}),
					Entry("start just before start of pointer",
						MultiPointerSpec{
							TimeRange:       (9 * telem.SecondTS).Range(23 * telem.SecondTS),
							StartOffset:     0,
							EndOffset:       3,
							FirstTimeRange:  (23 * telem.SecondTS).Range(30 * telem.SecondTS),
							FirstData:       []byte{23, 24, 25, 26, 27, 28, 29},
							SecondTimeRange: (30 * telem.SecondTS).Range(40 * telem.SecondTS),
							SecondData:      []byte{30, 31, 32, 33, 34, 35, 36, 37, 38, 39},
						}),
					Entry("end at end of pointer",
						MultiPointerSpec{
							TimeRange:       (13 * telem.SecondTS).Range(20 * telem.SecondTS),
							StartOffset:     3,
							EndOffset:       0,
							FirstTimeRange:  (10 * telem.SecondTS).Range(13 * telem.SecondTS),
							FirstData:       []byte{10, 11, 12},
							SecondTimeRange: (22 * telem.SecondTS).Range(30 * telem.SecondTS),
							SecondData:      []byte{20, 21, 22, 23, 24, 25, 26, 27, 28},
						}),
					Entry(
						"end just after end of pointer",
						MultiPointerSpec{
							TimeRange:       (13 * telem.SecondTS).Range(21 * telem.SecondTS),
							StartOffset:     3,
							EndOffset:       0,
							FirstTimeRange:  (10 * telem.SecondTS).Range(13 * telem.SecondTS),
							FirstData:       []byte{10, 11, 12},
							SecondTimeRange: (22 * telem.SecondTS).Range(30 * telem.SecondTS),
							SecondData:      []byte{20, 21, 22, 23, 24, 25, 26, 27, 28},
						}),
					Entry("end before first pointer",
						MultiPointerSpec{
							TimeRange:       (1 * telem.SecondTS).Range(3 * telem.SecondTS),
							StartOffset:     0,
							EndOffset:       0,
							FirstTimeRange:  (10 * telem.SecondTS).Range(20 * telem.SecondTS),
							FirstData:       []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
							SecondTimeRange: (22 * telem.SecondTS).Range(30 * telem.SecondTS),
							SecondData:      []byte{20, 21, 22, 23, 24, 25, 26, 27, 28},
						},
					),
				)

				It("Should delete with the end being end of db", func() {
					Expect(db.Delete(
						ctx,
						(12 * telem.SecondTS).Range(40*telem.SecondTS),
						fixedOffset(2),
						fixedOffset(10),
					)).To(Succeed())
					iter := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).Range(12 * telem.SecondTS)))
					r := MustSucceed(iter.OpenReader(ctx))
					p := make([]byte, 2)
					_, err := r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:2]).To(Equal([]byte{10, 11}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should delete nothing", func() {
					Expect(db.Delete(
						ctx,
						(24 * telem.SecondTS).Range(24*telem.SecondTS),
						fixedOffset(4),
						fixedOffset(4),
					)).To(Succeed())
					iter := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).Range(20 * telem.SecondTS)))
					r := MustSucceed(iter.OpenReader(ctx))
					p := make([]byte, 10)
					_, err := r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:10]).To(Equal([]byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeTrue())
					r = MustSucceed(iter.OpenReader(ctx))
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:10]).To(Equal([]byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29}))
					Expect(iter.Close()).To(Succeed())
					Expect(r.Close()).To(Succeed())
				})

				It("Should delete the entire db", func() {
					Expect(db.Delete(
						ctx,
						(10 * telem.SecondTS).Range(40*telem.SecondTS),
						fixedOffset(0),
						fixedOffset(10),
					)).To(Succeed())
					iter := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should delete multiple pointers", func() {
					Expect(db.Delete(
						ctx,
						(12 * telem.SecondTS).Range(13*telem.SecondTS),
						fixedOffset(2),
						fixedOffset(3),
					)).To(Succeed())
					// at this point, pointer 0 splits into two: 10, 11 / 13, 14, 15, 16, ..., 19
					Expect(db.Delete(
						ctx,
						(11 * telem.SecondTS).Range(15*telem.SecondTS),
						fixedOffset(1),
						fixedOffset(2),
					)).To(Succeed())
					// 10 / 15, 16, ..., 19
					Expect(db.Delete(
						ctx,
						(17 * telem.SecondTS).Range(19*telem.SecondTS),
						fixedOffset(2),
						fixedOffset(4),
					)).To(Succeed())
					// 10 / 15, 16 / 19
					iter := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).Range(11 * telem.SecondTS)))
					r := MustSucceed(iter.OpenReader(ctx))
					p := make([]byte, 1)
					_, err := r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:1]).To(Equal([]byte{10}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((15 * telem.SecondTS).Range(17 * telem.SecondTS)))
					r = MustSucceed(iter.OpenReader(ctx))
					p = make([]byte, 2)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:2]).To(Equal([]byte{15, 16}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((19 * telem.SecondTS).Range(20 * telem.SecondTS)))
					r = MustSucceed(iter.OpenReader(ctx))
					p = make([]byte, 1)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:1]).To(Equal([]byte{19}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((22 * telem.SecondTS).Range(30 * telem.SecondTS)))
					r = MustSucceed(iter.OpenReader(ctx))
					p = make([]byte, 10)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:10]).To(Equal([]byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Close()).To(Succeed())
				})

				It("Should delete multiple pointers that add up to the whole db", func() {
					Expect(db.Delete(
						ctx,
						(12 * telem.SecondTS).Range(23*telem.SecondTS),
						fixedOffset(2),
						fixedOffset(3),
					)).To(Succeed())
					Expect(db.Delete(
						ctx,
						(26 * telem.SecondTS).Range(28*telem.SecondTS),
						fixedOffset(3),
						fixedOffset(5),
					)).To(Succeed())
					Expect(db.Delete(
						ctx,
						(30 * telem.SecondTS).Range(35*telem.SecondTS),
						fixedOffset(0),
						fixedOffset(5),
					)).To(Succeed())
					Expect(db.Delete(
						ctx,
						(28 * telem.SecondTS).Range(30*telem.SecondTS),
						fixedOffset(0),
						fixedOffset(0),
					)).To(Succeed())
					Expect(db.Delete(
						ctx,
						(23 * telem.SecondTS).Range(30*telem.SecondTS),
						fixedOffset(0),
						fixedOffset(0),
					)).To(Succeed())
					Expect(db.Delete(
						ctx,
						(35 * telem.SecondTS).Range(40*telem.SecondTS),
						fixedOffset(0),
						fixedOffset(0),
					)).To(Succeed())
					Expect(db.Delete(
						ctx,
						(10 * telem.SecondTS).Range(12*telem.SecondTS),
						fixedOffset(0),
						fixedOffset(0),
					)).To(Succeed())
					iter := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})

					Expect(iter.SeekFirst(ctx)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})

			Context("Edge cases", func() {
				Context("With Data", func() {
					JustBeforeEach(func() {
						Expect(domain.Write(ctx, db, (10 * telem.SecondTS).SpanRange(10*telem.Second), []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})).To(Succeed())
						Expect(domain.Write(ctx, db, (22 * telem.SecondTS).SpanRange(8*telem.Second), []byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29})).To(Succeed())
						Expect(domain.Write(ctx, db, (30 * telem.SecondTS).SpanRange(10*telem.Second), []byte{30, 31, 32, 33, 34, 35, 36, 37, 38, 39})).To(Succeed())
					})

					It("Should not return an error when the start pointer is 1 greater than the end pointer and the offsets are 0 and full, respectively", func() {
						err := db.Delete(
							ctx,
							telem.TimeRange{Start: 40 * telem.SecondTS, End: 39 * telem.SecondTS},
							fixedOffset(0),
							fixedOffset(10),
						)
						Expect(err).ToNot(HaveOccurred())
					})

					It("Should return errors when the startOffset is after the endOffset for same pointer deletion", func() {
						err := db.Delete(
							ctx,
							telem.TimeRange{Start: 26 * telem.SecondTS, End: 25 * telem.SecondTS},
							fixedOffset(7),
							fixedOffset(5),
						)
						Expect(err).To(MatchError(ContainSubstring("deletion start offset 7 is after end offset 5")))
					})

					It("Should return an error when the db is closed", func() {
						db2 := MustSucceed(domain.Open(domain.Config{FS: fs, Instrumentation: PanicLogger()}))
						Expect(db2.Close()).To(Succeed())
						Expect(db2.Delete(
							ctx,
							telem.TimeRangeMin,
							fixedOffset(0),
							fixedOffset(0),
						)).To(HaveOccurredAs(resource.NewClosedError("domain.db")))
					})
				})

				Context("Without Data", func() {
					It("Should not delete anything", func() {
						Expect(db.Delete(
							ctx,
							telem.TimeRangeMax,
							fixedOffset(0),
							fixedOffset(0),
						)).To(Succeed())
						iter := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
						Expect(iter.SeekFirst(ctx)).To(BeFalse())
						Expect(iter.Close()).To(Succeed())
					})
				})
			})
		})
	}
})
