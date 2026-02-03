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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Garbage Collection", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, func() {
			var (
				db      *domain.DB
				fs      fs.FS
				cleanUp func() error
			)
			BeforeEach(func() {
				fs, cleanUp = makeFS()
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})

			Context("Happy path - one file", func() {
				It("Should garbage collect one tombstone", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        9 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})).To(Succeed())
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 12*telem.SecondTS + 1, End: 16*telem.SecondTS + 1},
						fixedOffset(3),
						fixedOffset(7),
					)).To(Succeed())

					By("Garbage collecting and asserting the file got smaller")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(6)))

					By("Asserting that we can still write to the file")
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(28*telem.SecondTS+1), []byte{20, 21, 22, 23, 24, 25, 26, 27, 28})).To(Succeed())

					By("Asserting that the data is correct")
					i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
					Expect(i.SeekFirst(ctx)).To(BeTrue())
					Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
					r := MustSucceed(i.OpenReader(ctx))
					var buf = make([]byte, 3)
					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{10, 11, 12}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					Expect(i.TimeRange()).To(Equal((16*telem.SecondTS + 1).Range(19*telem.SecondTS + 1)))
					r = MustSucceed(i.OpenReader(ctx))
					buf = make([]byte, 3)
					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{17, 18, 19}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					Expect(i.TimeRange()).To(Equal((20 * telem.SecondTS).Range(28*telem.SecondTS + 1)))
					r = MustSucceed(i.OpenReader(ctx))
					buf = make([]byte, 9)
					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{20, 21, 22, 23, 24, 25, 26, 27, 28}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeFalse())
					Expect(i.Close()).To(Succeed())
				})

				It("Should garbage collect multiple tombstones", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        20 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})).To(Succeed())
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(23*telem.SecondTS+1), []byte{20, 21, 22, 23})).To(Succeed())
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).Range(36*telem.SecondTS+1), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed())
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 12*telem.SecondTS + 1, End: 33*telem.SecondTS + 1},
						fixedOffset(3),
						fixedOffset(4),
					)).To(Succeed())

					By("Garbage collecting and asserting the file got smaller")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(21)))
					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(6)))

					By("Asserting that we can still write to the file")
					// Opening two writers to force the first one to go to the empty
					// 2.domain. This way, the second one can go to 1.domain.
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 50 * telem.SecondTS}))
					w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 50 * telem.SecondTS}))
					_, err := w2.Write([]byte{50, 51, 52})
					Expect(err).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(9)))
					Expect(w2.Commit(ctx, 52*telem.SecondTS+1)).To(Succeed())
					Expect(w2.Close()).To(Succeed())
					Expect(w1.Close()).To(Succeed())

					By("Asserting that we can still delete data")
					Expect(db.Delete(
						ctx,
						(11 * telem.SecondTS).Range(35*telem.SecondTS),
						fixedOffset(2),
						fixedOffset(1),
					)).To(Succeed())

					By("Asserting that the data did not change", func() {
						i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(11 * telem.SecondTS)))
						r := MustSucceed(i.OpenReader(ctx))
						var buf = make([]byte, 1)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{10}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((35 * telem.SecondTS).Range(36*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 2)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{35, 36}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((50 * telem.SecondTS).Range(52*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 3)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{50, 51, 52}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})

				It("Should garbage collect multiple tombstones based on the threshold", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        (1.25 * 20) * telem.Byte,
						GCThreshold:     float32(16) / 20,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})).To(Succeed())
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(23*telem.SecondTS+1), []byte{20, 21, 22, 23})).To(Succeed())
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).Range(36*telem.SecondTS+1), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed())
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 12*telem.SecondTS + 1, End: 33*telem.SecondTS + 1},
						fixedOffset(3),
						fixedOffset(4),
					)).To(Succeed())

					By("Garbage collecting and asserting the file did not get smaller as the threshold is not reached.")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(21)))
					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(21)))

					By("Deleting more data to reach the threshold")
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 10*telem.SecondTS + 1, End: 12*telem.SecondTS + 1},
						fixedOffset(1),
						fixedOffset(3),
					)).To(Succeed())
					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(4)))

					By("Asserting that the data did not change", func() {
						i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(10*telem.SecondTS + 1)))
						r := MustSucceed(i.OpenReader(ctx))
						var buf = make([]byte, 1)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{10}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((33*telem.SecondTS + 1).Range(36*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 3)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{34, 35, 36}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})

				It("Should not garbage collect a file that is oversize but not still being written to", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        13 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (1 * telem.SecondTS).Range(7*telem.SecondTS+1), []byte{1, 2, 3, 4, 5, 6, 7})).To(Succeed())
					Expect(db.Delete(
						ctx,
						(1 * telem.SecondTS).Range(3*telem.SecondTS),
						fixedOffset(0),
						fixedOffset(2),
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS}))

					_, err := w.Write([]byte{10, 11, 12, 13, 14})
					Expect(err).ToNot(HaveOccurred())

					// Now, file 1 should be oversize
					Expect(db.GarbageCollect(ctx)).To(Succeed())

					By("Expecting that garbage collection did not occur")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(12)))

					By("Closing the writer and expecting garbage collection to occur")
					Expect(w.Commit(ctx, 14*telem.SecondTS+1)).To(Succeed())
					Expect(w.Close()).To(Succeed())

					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))

					By("Asserting that the data did not change", func() {
						i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((3 * telem.SecondTS).Range(7*telem.SecondTS + 1)))
						r := MustSucceed(i.OpenReader(ctx))
						var buf = make([]byte, 5)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{3, 4, 5, 6, 7}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(14*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 5)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{10, 11, 12, 13, 14}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})

				It("Should not garbage collect a file that has an open reader on it", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        9 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})).To(Succeed())
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 12*telem.SecondTS + 1, End: 16*telem.SecondTS + 1},
						fixedOffset(3),
						fixedOffset(7),
					)).To(Succeed())

					iter := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					r := MustSucceed(iter.OpenReader(ctx))

					By("Garbage collecting and asserting the file did not change")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(r.Close()).To(Succeed())
					Expect(iter.Close()).To(Succeed())

					By("Asserting that we can still write to the file")
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(28*telem.SecondTS+1), []byte{20, 21, 22, 23, 24, 25, 26, 27, 28})).To(Succeed())

					By("Asserting that the data is correct")
					data := MustSucceed(domain.Read(ctx, db, (20 * telem.SecondTS).Range(29*telem.SecondTS+1)))
					Expect(data).To(HaveLen(9))
					Expect(data).To(Equal([]byte{20, 21, 22, 23, 24, 25, 26, 27, 28}))

					By("Garbage collecting and asserting the file did not change")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(6)))

					Expect(db.Close()).To(Succeed())
				})
			})

			Context("Happy path - multiple files", func() {
				It("Should garbage collect multiple tombstones", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						MaxDescriptors:  4,
						FileSize:        2 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})).To(Succeed())
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(23*telem.SecondTS+1), []byte{20, 21, 22, 23})).To(Succeed())
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).Range(36*telem.SecondTS+1), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed())
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 12*telem.SecondTS + 1, End: 33*telem.SecondTS + 1},
						fixedOffset(3),
						fixedOffset(4),
					)).To(Succeed())

					By("Garbage collecting and asserting the file got smaller")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(4)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(7)))

					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(3)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(0)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(3)))

					Expect(domain.Write(ctx, db, (100 * telem.SecondTS).Range(105*telem.SecondTS+1), []byte{100, 101, 102, 103, 104, 105})).To(Succeed())
					Expect(domain.Write(ctx, db, (110 * telem.SecondTS).Range(115*telem.SecondTS+1), []byte{110, 111, 112, 113, 114, 115})).To(Succeed())

					By("Asserting that the data did not change", func() {
						i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
						r := MustSucceed(i.OpenReader(ctx))
						var buf = make([]byte, 3)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{10, 11, 12}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((33*telem.SecondTS + 1).Range(36*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 3)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{34, 35, 36}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((100 * telem.SecondTS).Range(105*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 6)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{100, 101, 102, 103, 104, 105}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((110 * telem.SecondTS).Range(115*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 6)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{110, 111, 112, 113, 114, 115}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})

				It("Should garbage collect multiple tombstones across many files", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        7 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13})).To(Succeed())             // file 1
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(25*telem.SecondTS+1), []byte{20, 21, 22, 23, 24, 25})).To(Succeed())     // file 1
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).Range(36*telem.SecondTS+1), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed()) // file 2
					Expect(domain.Write(ctx, db, (40 * telem.SecondTS).Range(43*telem.SecondTS+1), []byte{40, 41, 43})).To(Succeed())                 // file 3
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 23 * telem.SecondTS, End: 41 * telem.SecondTS},
						fixedOffset(3),
						fixedOffset(1),
					)).To(Succeed())

					By("Garbage collecting and asserting the files got smaller")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(7)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(3)))

					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(7)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(0)))
					// File 3 should not get smaller since it is not full.
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(3)))

					By("Asserting that we can still write data")
					Expect(domain.Write(ctx, db, (44 * telem.SecondTS).Range(44*telem.SecondTS+1), []byte{44})).To(Succeed())
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(4)))

					By("Asserting that the data did not change", func() {
						i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((20 * telem.SecondTS).Range(23 * telem.SecondTS)))
						r := MustSucceed(i.OpenReader(ctx))
						var buf = make([]byte, 3)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{20, 21, 22}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((41 * telem.SecondTS).Range(43*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 2)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{41, 43}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((44 * telem.SecondTS).Range(44*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 1)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{44}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})

				It("Should garbage collect tombstones based on the threshold", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        7 * telem.Byte,
						GCThreshold:     0.4,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13})).To(Succeed())             // file 1
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(25*telem.SecondTS+1), []byte{20, 21, 22, 23, 24, 25})).To(Succeed())     // file 1
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).Range(36*telem.SecondTS+1), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed()) // file 2
					Expect(domain.Write(ctx, db, (40 * telem.SecondTS).Range(43*telem.SecondTS+1), []byte{40, 41, 43})).To(Succeed())                 // file 3
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 23 * telem.SecondTS, End: 41 * telem.SecondTS},
						fixedOffset(3),
						fixedOffset(1),
					)).To(Succeed())

					By("Garbage collecting and asserting the file got smaller")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(7)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(3)))

					Expect(db.GarbageCollect(ctx)).To(Succeed())
					// file 1 should be collected (3 > 2)
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(7)))
					// file 2 should be collected (7 > 2)
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(0)))
					// file 3 should not be garbage collected (1 < 2)
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(3)))

					By("Asserting that the data did not change", func() {
						i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((20 * telem.SecondTS).Range(23 * telem.SecondTS)))
						r := MustSucceed(i.OpenReader(ctx))
						var buf = make([]byte, 3)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{20, 21, 22}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((41 * telem.SecondTS).Range(43*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 2)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{41, 43}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})
			})

			Context("Tombstone persist", func() {
				It("Should preserve the tombstones after database closure", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        7 * telem.Byte,
						GCThreshold:     0.4,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13})).To(Succeed())             // file 1
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(25*telem.SecondTS+1), []byte{20, 21, 22, 23, 24, 25})).To(Succeed())     // file 1
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).Range(36*telem.SecondTS+1), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed()) // file 2
					Expect(domain.Write(ctx, db, (40 * telem.SecondTS).Range(46*telem.SecondTS+1), []byte{40, 41, 43, 44, 45, 46})).To(Succeed())     // file 3
					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 23 * telem.SecondTS, End: 41 * telem.SecondTS},
						fixedOffset(3),
						fixedOffset(1),
					)).To(Succeed())

					By("Reopening the DB")
					Expect(db.Close()).To(Succeed())
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        7 * telem.Byte,
						GCThreshold:     0.4,
						Instrumentation: PanicLogger(),
					}))

					By("Garbage collecting and asserting the file got smaller")
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(7)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(6)))

					Expect(db.GarbageCollect(ctx)).To(Succeed())
					// file 1 should be collected (3 > 2)
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(7)))
					// file 2 should be collected (7 > 2)
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(0)))
					// file 3 should not be garbage collected (1 < 2)
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(6)))

					By("Asserting that new data would still be written")
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).Range(35*telem.SecondTS+1), []byte{30, 31, 32, 33, 34, 35})).To(Succeed())
					// It's difficult to test that the new data actually went to a
					// specific freed file – as they could go to any one of file 1, 2, 4
					// since they are all below the file size limit.

					By("Asserting that the data did not change", func() {
						i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((20 * telem.SecondTS).Range(23 * telem.SecondTS)))
						r := MustSucceed(i.OpenReader(ctx))
						var buf = make([]byte, 3)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{20, 21, 22}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((30 * telem.SecondTS).Range(35*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 6)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{30, 31, 32, 33, 34, 35}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((41 * telem.SecondTS).Range(46*telem.SecondTS + 1)))
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 5)
						MustSucceed(r.ReadAt(buf, 0))
						Expect(buf).To(Equal([]byte{41, 43, 44, 45, 46}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})
			})

			Context("Regression", func() {
				// This regression test is used to verify that when GC is run, there is
				// no readers on the old file that is still symlinking to the old file,
				// causing the reading of incorrect data.
				Specify("Reader should be recycled", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        7 * telem.Byte,
						GCThreshold:     0,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13})).To(Succeed())             // file 1
					Expect(domain.Write(ctx, db, (20 * telem.SecondTS).Range(25*telem.SecondTS+1), []byte{20, 21, 22, 23, 24, 25})).To(Succeed())     // file 1
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).Range(36*telem.SecondTS+1), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed()) // file 2
					Expect(domain.Write(ctx, db, (40 * telem.SecondTS).Range(46*telem.SecondTS+1), []byte{40, 41, 43, 44, 45, 46})).To(Succeed())     // file 3
					i := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(i.SeekLE(ctx, 43*telem.SecondTS)).To(BeTrue())
					r := MustSucceed(i.OpenReader(ctx))
					Expect(r.Close()).To(Succeed())

					Expect(db.Delete(
						ctx,
						telem.TimeRange{Start: 23 * telem.SecondTS, End: 41 * telem.SecondTS},
						fixedOffset(3),
						fixedOffset(1),
					)).To(Succeed())
					Expect(db.GarbageCollect(ctx)).To(Succeed())

					Expect(i.SeekLE(ctx, 43*telem.SecondTS)).To(BeTrue())
					r = MustSucceed(i.OpenReader(ctx))
					b := make([]byte, 2)
					_, err := r.ReadAt(b, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(b).To(Equal([]byte{41, 43}))
					Expect(r.Close()).To(Succeed())
					Expect(i.Close()).To(Succeed())
				})

				// This regression test is used to verify that when calling gcReaders()
				// in garbage collection, the slice will be popped correctly. Previously,
				// the slice was being popped while iterated, causing out-of-bounds
				// errors.
				Specify("Reader gc", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        7 * telem.Byte,
						GCThreshold:     0,
						Instrumentation: PanicLogger(),
					}))
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).Range(19*telem.SecondTS+1), []byte{10, 11, 12, 13})).To(Succeed()) // file 1
					i := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(i.SeekLE(ctx, 43*telem.SecondTS)).To(BeTrue())
					r1 := MustSucceed(i.OpenReader(ctx))
					r2 := MustSucceed(i.OpenReader(ctx))
					Expect(r1.Close()).To(Succeed())
					Expect(r2.Close()).To(Succeed())

					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(i.Close()).To(Succeed())
					Expect(db.Close()).To(Succeed())
				})
			})

			Context("Close", func() {
				It("Should not allow GC on a closed DB", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        20 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					Expect(db.Close()).To(Succeed())
					Expect(db.GarbageCollect(ctx)).To(HaveOccurredAs(resource.NewClosedError("domain.db")))
				})
			})
		})
	}
})
