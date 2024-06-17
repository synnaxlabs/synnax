// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createCalcOffset(offset int) func(context.Context, telem.TimeStamp, telem.TimeStamp) (int64, telem.TimeStamp, error) {
	return func(
		ctx context.Context,
		domainStart telem.TimeStamp,
		ts telem.TimeStamp,
	) (int64, telem.TimeStamp, error) {
		return int64(offset), ts, nil
	}
}

var _ = Describe("Delete", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, func() {
			var (
				db      *domain.DB
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeEach(func() {
				fs, cleanUp = makeFS()
				db = MustSucceed(domain.Open(domain.Config{FS: fs}))
				Expect(domain.Write(ctx, db, (10 * telem.SecondTS).SpanRange(10*telem.Second), []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})).To(Succeed())
				Expect(domain.Write(ctx, db, (20 * telem.SecondTS).SpanRange(10*telem.Second), []byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29})).To(Succeed())
				Expect(domain.Write(ctx, db, (30 * telem.SecondTS).SpanRange(10*telem.Second), []byte{30, 31, 32, 33, 34, 35, 36, 37, 38, 39})).To(Succeed())
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})
			Context("Multiple Pointers", func() {
				DescribeTable("Deleting time range", func(
					tr telem.TimeRange,
					startOffset int,
					endOffset int,
					firstTr telem.TimeRange,
					firstData []byte,
					secondTr telem.TimeRange,
					secondData []byte,
				) {
					Expect(db.Delete(ctx, createCalcOffset(startOffset), createCalcOffset(endOffset), tr, telem.Density(1))).To(Succeed())
					iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal(firstTr))
					r := MustSucceed(iter.NewReader(ctx))
					p := make([]byte, len(firstData))
					_, err := r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:len(firstData)]).To(Equal(firstData))
					Expect(r.Close()).To(Succeed())
					Expect(iter.Next()).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal(secondTr))
					r = MustSucceed(iter.NewReader(ctx))
					p = make([]byte, len(secondData))
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:len(secondData)]).To(Equal(secondData))
					Expect(iter.Close()).To(Succeed())
					Expect(r.Close()).To(Succeed())
				},
					Entry("across two pointers", (13*telem.SecondTS).Range(23*telem.SecondTS), 3, 3, (10*telem.SecondTS).Range(13*telem.SecondTS), []byte{10, 11, 12}, (23*telem.SecondTS).Range(30*telem.SecondTS), []byte{23, 24, 25, 26, 27, 28}),
					Entry("start at start of pointer", (10*telem.SecondTS).Range(23*telem.SecondTS), 0, 3, (23*telem.SecondTS).Range(30*telem.SecondTS), []byte{23, 24, 25, 26, 27, 28, 29}, (30*telem.SecondTS).Range(40*telem.SecondTS), []byte{30, 31, 32, 33, 34, 35, 36, 37, 38, 39}),
					Entry("end at end of pointer", (13*telem.SecondTS).Range(20*telem.SecondTS), 3, 0, (10*telem.SecondTS).Range(13*telem.SecondTS), []byte{10, 11, 12}, (20*telem.SecondTS).Range(30*telem.SecondTS), []byte{20, 21, 22, 23, 24, 25, 26, 27, 28}))

				It("Should delete with the end being end of db", func() {
					Expect(db.Delete(ctx, createCalcOffset(2), createCalcOffset(10), (12 * telem.SecondTS).Range(40*telem.SecondTS), telem.Density(1))).To(Succeed())
					iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).Range(12 * telem.SecondTS)))
					r := MustSucceed(iter.NewReader(ctx))
					p := make([]byte, 2)
					_, err := r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:2]).To(Equal([]byte{10, 11}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should delete nothing", func() {
					Expect(db.Delete(ctx, createCalcOffset(4), createCalcOffset(4), (24 * telem.SecondTS).Range(24*telem.SecondTS), telem.Density(1))).To(Succeed())
					iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).Range(20 * telem.SecondTS)))
					r := MustSucceed(iter.NewReader(ctx))
					p := make([]byte, 10)
					_, err := r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:10]).To(Equal([]byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeTrue())
					r = MustSucceed(iter.NewReader(ctx))
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:10]).To(Equal([]byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29}))
					Expect(iter.Close()).To(Succeed())
					Expect(r.Close()).To(Succeed())
				})

				It("Should delete the entire db", func() {
					Expect(db.Delete(ctx, createCalcOffset(0), createCalcOffset(10), (10 * telem.SecondTS).Range(40*telem.SecondTS), telem.Density(1))).To(Succeed())
					iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should delete multiple pointers", func() {
					Expect(db.Delete(ctx, createCalcOffset(2), createCalcOffset(3), (12 * telem.SecondTS).Range(13*telem.SecondTS), telem.Density(1))).To(Succeed())
					// at this point, pointer 0 splits into two: 10, 11 / 13, 14, 15, 16, ..., 19
					Expect(db.Delete(ctx, createCalcOffset(1), createCalcOffset(2), (11 * telem.SecondTS).Range(15*telem.SecondTS), telem.Density(1))).To(Succeed())
					// 10 / 15, 16, ..., 19
					Expect(db.Delete(ctx, createCalcOffset(2), createCalcOffset(4), (17 * telem.SecondTS).Range(19*telem.SecondTS), telem.Density(1))).To(Succeed())
					// 10 / 15, 16 / 19
					iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).Range(11 * telem.SecondTS)))
					r := MustSucceed(iter.NewReader(ctx))
					p := make([]byte, 1)
					_, err := r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:1]).To(Equal([]byte{10}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((15 * telem.SecondTS).Range(17 * telem.SecondTS)))
					r = MustSucceed(iter.NewReader(ctx))
					p = make([]byte, 2)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:2]).To(Equal([]byte{15, 16}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((19 * telem.SecondTS).Range(20 * telem.SecondTS)))
					r = MustSucceed(iter.NewReader(ctx))
					p = make([]byte, 1)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:1]).To(Equal([]byte{19}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Next()).To(BeTrue())
					Expect(iter.TimeRange()).To(Equal((20 * telem.SecondTS).Range(30 * telem.SecondTS)))
					r = MustSucceed(iter.NewReader(ctx))
					p = make([]byte, 10)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p[:10]).To(Equal([]byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29}))
					Expect(r.Close()).To(Succeed())

					Expect(iter.Close()).To(Succeed())
				})

				It("Should delete multiple pointers that add up to the whole db", func() {
					Expect(db.Delete(ctx, createCalcOffset(2), createCalcOffset(3), (12 * telem.SecondTS).Range(23*telem.SecondTS), telem.Density(1))).To(Succeed())
					Expect(db.Delete(ctx, createCalcOffset(3), createCalcOffset(5), (26 * telem.SecondTS).Range(28*telem.SecondTS), telem.Density(1))).To(Succeed())
					Expect(db.Delete(ctx, createCalcOffset(0), createCalcOffset(5), (30 * telem.SecondTS).Range(35*telem.SecondTS), telem.Density(1))).To(Succeed())
					Expect(db.Delete(ctx, createCalcOffset(0), createCalcOffset(0), (28 * telem.SecondTS).Range(30*telem.SecondTS), telem.Density(1))).To(Succeed())
					Expect(db.Delete(ctx, createCalcOffset(0), createCalcOffset(0), (23 * telem.SecondTS).Range(30*telem.SecondTS), telem.Density(1))).To(Succeed())
					Expect(db.Delete(ctx, createCalcOffset(0), createCalcOffset(0), (35 * telem.SecondTS).Range(40*telem.SecondTS), telem.Density(1))).To(Succeed())
					Expect(db.Delete(ctx, createCalcOffset(0), createCalcOffset(0), (10 * telem.SecondTS).Range(12*telem.SecondTS), telem.Density(1))).To(Succeed())
					iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})

					Expect(iter.SeekFirst(ctx)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})
			Context("Edge cases", func() {
				It("Should not return an error when the start pointer is 1 greater than the end pointer and the offsets are 0 and full, respectively", func() {
					err := db.Delete(ctx, createCalcOffset(0), createCalcOffset(10), telem.TimeRange{Start: 40 * telem.SecondTS, End: 39 * telem.SecondTS}, telem.Density(1))
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should return errors when the startOffset is after the endOffset for same pointer deletion", func() {
					err := db.Delete(ctx, createCalcOffset(7), createCalcOffset(5), telem.TimeRange{Start: 26 * telem.SecondTS, End: 25 * telem.SecondTS}, telem.Density(1))
					Expect(err).To(MatchError(ContainSubstring("deletion start offset 7 is after end offset 5")))
				})

				It("Should return an error when the db is closed", func() {
					db2 := MustSucceed(domain.Open(domain.Config{FS: fs}))
					Expect(db2.Close()).To(Succeed())
					Expect(db2.Delete(ctx, createCalcOffset(0), createCalcOffset(0), telem.TimeRangeMin, telem.Density(1))).To(HaveOccurredAs(core.EntityClosed("domain.db")))
				})
			})
		})
	}
})
