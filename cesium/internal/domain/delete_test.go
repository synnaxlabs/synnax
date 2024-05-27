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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

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
					startPosition int,
					endPosition int,
					startOffset int,
					endOffset int,
					firstTr telem.TimeRange,
					firstData []byte,
					secondTr telem.TimeRange,
					secondData []byte,
				) {
					Expect(db.Delete(ctx, startPosition, endPosition, int64(startOffset), int64(endOffset), tr)).To(Succeed())
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
					Entry("across two pointers", (13*telem.SecondTS).Range(23*telem.SecondTS), 0, 1, 3, 7, (10*telem.SecondTS).Range(13*telem.SecondTS), []byte{10, 11, 12}, (23*telem.SecondTS).Range(30*telem.SecondTS), []byte{23, 24, 25, 26, 27, 28}),
					Entry("start at start of pointer", (10*telem.SecondTS).Range(23*telem.SecondTS), 0, 1, 0, 7, (23*telem.SecondTS).Range(30*telem.SecondTS), []byte{23, 24, 25, 26, 27, 28, 29}, (30*telem.SecondTS).Range(40*telem.SecondTS), []byte{30, 31, 32, 33, 34, 35, 36, 37, 38, 39}),
					Entry("end at end of pointer", (13*telem.SecondTS).Range(20*telem.SecondTS), 0, 1, 3, 10, (10*telem.SecondTS).Range(13*telem.SecondTS), []byte{10, 11, 12}, (20*telem.SecondTS).Range(30*telem.SecondTS), []byte{20, 21, 22, 23, 24, 25, 26, 27, 28}))

				It("Should delete with the end being end of db", func() {
					Expect(db.Delete(ctx, 0, 2, 2, 0, (12 * telem.SecondTS).Range(40*telem.SecondTS))).To(Succeed())
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
					Expect(db.Delete(ctx, 1, 1, 4, 6, (24 * telem.SecondTS).Range(24*telem.SecondTS))).To(Succeed())
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
					Expect(db.Delete(ctx, 0, 2, 0, 0, (10 * telem.SecondTS).Range(40*telem.SecondTS))).To(Succeed())
					iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(iter.SeekFirst(ctx)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should delete multiple pointers", func() {
					Expect(db.Delete(ctx, 0, 0, 2, 7, (12 * telem.SecondTS).Range(13*telem.SecondTS))).To(Succeed())
					// at this point, pointer 0 splits into two: 10, 11 / 13, 14, 15, 16, ..., 19
					Expect(db.Delete(ctx, 0, 1, 1, 5, (11 * telem.SecondTS).Range(15*telem.SecondTS))).To(Succeed())
					// 10 / 15, 16, ..., 19
					Expect(db.Delete(ctx, 1, 1, 2, 1, (17 * telem.SecondTS).Range(19*telem.SecondTS))).To(Succeed())
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
					Expect(db.Delete(ctx, 0, 1, 2, 7, (12 * telem.SecondTS).Range(23*telem.SecondTS))).To(Succeed())
					Expect(db.Delete(ctx, 1, 1, 3, 2, (26 * telem.SecondTS).Range(28*telem.SecondTS))).To(Succeed())
					Expect(db.Delete(ctx, 3, 3, 0, 5, (30 * telem.SecondTS).Range(35*telem.SecondTS))).To(Succeed())
					Expect(db.Delete(ctx, 2, 2, 0, 0, (28 * telem.SecondTS).Range(30*telem.SecondTS))).To(Succeed())
					Expect(db.Delete(ctx, 1, 1, 0, 0, (23 * telem.SecondTS).Range(30*telem.SecondTS))).To(Succeed())
					Expect(db.Delete(ctx, 1, 1, 0, 0, (35 * telem.SecondTS).Range(40*telem.SecondTS))).To(Succeed())
					Expect(db.Delete(ctx, 0, 0, 0, 0, (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())
					iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})

					Expect(iter.SeekFirst(ctx)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})
			Context("Edge cases", func() {
				It("Should return errors when the start pointer is invalid", func() {
					err := db.Delete(ctx, -1, 1, 2, 7, telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("starting at invalid domain position -1"))
				})

				It("Should return errors when the end pointer is invalid", func() {
					err := db.Delete(ctx, 1, 3, 2, 7, telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("ending at invalid domain position 3 for length 3"))
				})

				It("Should return errors when the start pointer is greater than the end pointer", func() {
					err := db.Delete(ctx, 2, 1, 2, 7, telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("deletion start domain 2 is greater than deletion end domain 1"))
				})

				It("Should not return an error when the start pointer is 1 greater than the end pointer and the offsets are 0 and full, respectively", func() {
					err := db.Delete(ctx, 2, 1, 0, 0, telem.TimeRange{Start: 40 * telem.SecondTS, End: 39 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should clip the offset to the end if the start offset is beyond the length", func() {
					err := db.Delete(ctx, 0, 1, 15, 7, telem.TimeRange{Start: 20 * telem.SecondTS, End: 23 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())

					i := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})

					Expect(i.SeekFirst(ctx)).To(BeTrue())
					Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(20 * telem.SecondTS)))
					r := MustSucceed(i.NewReader(ctx))
					p := make([]byte, 10)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p).To(Equal([]byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					Expect(i.TimeRange()).To(Equal((23 * telem.SecondTS).Range(30 * telem.SecondTS)))
					r = MustSucceed(i.NewReader(ctx))
					p = make([]byte, 7)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p).To(Equal([]byte{23, 24, 25, 26, 27, 28, 29}))
					Expect(r.Close()).To(Succeed())
					Expect(i.Close()).To(Succeed())
				})

				It("Should clip the offset to the start if the end offset is beyond the length", func() {
					err := db.Delete(ctx, 0, 1, 2, 15, telem.TimeRange{Start: 12 * telem.SecondTS, End: 20 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())

					i := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})

					Expect(i.SeekFirst(ctx)).To(BeTrue())
					Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(12 * telem.SecondTS)))
					r := MustSucceed(i.NewReader(ctx))
					p := make([]byte, 2)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p).To(Equal([]byte{10, 11}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					Expect(i.TimeRange()).To(Equal((20 * telem.SecondTS).Range(30 * telem.SecondTS)))
					r = MustSucceed(i.NewReader(ctx))
					p = make([]byte, 10)
					_, err = r.ReadAt(p, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(p).To(Equal([]byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29}))
					Expect(r.Close()).To(Succeed())
					Expect(i.Close()).To(Succeed())
				})

				It("Should return errors when the startOffset is after the endOffset for same pointer deletion", func() {
					err := db.Delete(ctx, 0, 0, 6, 5, telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("deletion start offset 6 is after end offset 5"))
				})

			})
		})
	}
})
