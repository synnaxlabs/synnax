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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("File Controller", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db      *domain.DB
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeEach(func() {
				fs, cleanUp = makeFS()
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})
			Describe("Writers", func() {
				It("Should allow one writing to a file at all times", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        1 * telem.Megabyte,
						Instrumentation: PanicLogger(),
					}))
					By("Acquiring one writer on the file 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					}))
					Expect(fs.Exists("1.domain")).To(BeTrue())
					By("Acquiring a second writer, this would create a new file 2.domain")
					w2, err := db.OpenWriter(ctx, domain.WriterConfig{
						Start: 30 * telem.SecondTS,
						End:   40 * telem.SecondTS,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(fs.Exists("2.domain")).To(BeTrue())

					By("Closing the first writer")
					Expect(w1.Close()).To(Succeed())

					By("Acquiring a third writer, 1.domain should be acquired")
					w3, err := db.OpenWriter(ctx, domain.WriterConfig{
						Start: 50 * telem.SecondTS,
						End:   60 * telem.SecondTS,
					})
					Expect(err).ToNot(HaveOccurred())
					n, err := w3.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(8))
					s, err := fs.Stat("1.domain")
					Expect(err).ToNot(HaveOccurred())
					Expect(s.Size()).To(Equal(int64(8)))

					Expect(w2.Close()).To(Succeed())
					Expect(w3.Close()).To(Succeed())
				})

				It("Should obey the file size limit", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					By("Acquiring one writer on the file 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					}))
					Expect(fs.Exists("1.domain")).To(BeTrue())
					n, err := w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
					Expect(n).To(Equal(10))
					Expect(err).ToNot(HaveOccurred())
					Expect(w1.Close()).To(Succeed())
					By("Acquiring a second writer, this would create a new file 2.domain since 1.domain is full")
					w2, err := db.OpenWriter(ctx, domain.WriterConfig{
						Start: 30 * telem.SecondTS,
						End:   40 * telem.SecondTS,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(fs.Exists("2.domain")).To(BeTrue())

					Expect(w2.Close()).To(Succeed())
				})

				It("Should persist obey the file size limit", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS: fs, FileSize: 10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					By("Acquiring one writer on the file 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					}))
					Expect(fs.Exists("1.domain")).To(BeTrue())
					n, err := w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
					Expect(n).To(Equal(10))
					Expect(err).ToNot(HaveOccurred())
					Expect(w1.Close()).To(Succeed())

					By("Closing the db")
					Expect(db.Close()).To(Succeed())

					By("Reopening the db and fc")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By("Acquiring a second writer, this would create a new file 2.domain since 1.domain is full")
					w2, err := db.OpenWriter(ctx, domain.WriterConfig{
						Start: 30 * telem.SecondTS,
						End:   40 * telem.SecondTS,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(fs.Exists("2.domain")).To(BeTrue())

					Expect(w2.Close()).To(Succeed())
				})

				It("Should open a file if it is below threshold", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					By("Acquiring one writer on the file 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					}))
					Expect(fs.Exists("1.domain")).To(BeTrue())
					n, err := w1.Write([]byte{1, 2, 3, 4, 5, 6, 7})
					Expect(n).To(Equal(7))
					Expect(err).ToNot(HaveOccurred())
					Expect(w1.Close()).To(Succeed())

					By("Acquiring a second writer, this would not create a new file 2.domain since 1.domain not full")
					w2, err := db.OpenWriter(ctx, domain.WriterConfig{
						Start: 30 * telem.SecondTS,
						End:   40 * telem.SecondTS,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(fs.Exists("2.domain")).To(BeFalse())

					Expect(w2.Close()).To(Succeed())
				})

				It("Should persist and open a file if it is below threshold", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					By("Acquiring one writer on the file 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					}))
					Expect(fs.Exists("1.domain")).To(BeTrue())
					n, err := w1.Write([]byte{1, 2, 3, 4, 5, 6, 7})
					Expect(n).To(Equal(7))
					Expect(err).ToNot(HaveOccurred())
					Expect(w1.Close()).To(Succeed())

					By("Closing the db")
					Expect(db.Close()).To(Succeed())

					By("Reopening the db and fc")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By("Acquiring a second writer, this would not create a new file 2.domain since 1.domain is not full")
					w2, err := db.OpenWriter(ctx, domain.WriterConfig{
						Start: 30 * telem.SecondTS,
						End:   40 * telem.SecondTS,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(fs.Exists("2.domain")).To(BeFalse())

					Expect(w2.Close()).To(Succeed())
				})

				It("Should obey the file descriptor limit", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						MaxDescriptors:  2,
						Instrumentation: PanicLogger(),
					}))
					By("Acquiring one writer on the file 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					}))
					Expect(fs.Exists("1.domain")).To(BeTrue())

					By("Acquiring one writer on the file 2.domain")
					w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 20 * telem.SecondTS,
						End:   30 * telem.SecondTS,
					}))
					Expect(fs.Exists("2.domain")).To(BeTrue())

					By("Trying to acquire a third writer")
					acquired := make(chan struct{})

					wg := sync.WaitGroup{}
					wg.Add(1)
					go func() {
						defer wg.Done()
						w3, err := db.OpenWriter(ctx, domain.WriterConfig{
							Start: 30 * telem.SecondTS,
							End:   40 * telem.SecondTS,
						})
						Expect(err).ToNot(HaveOccurred())
						acquired <- struct{}{}
						Expect(w3.Close()).To(Succeed())
					}()

					By("Expecting the channel acquisition to fail")
					Consistently(acquired).WithTimeout(50 * telem.Millisecond.Duration()).ShouldNot(Receive())
					By("Closing the writer 1")
					Expect(w1.Close()).To(Succeed())
					By("Expecting writer 3 to successfully acquire")
					Eventually(acquired).Should(Receive())
					wg.Wait()
					Expect(w2.Close()).To(Succeed())
				})

				It("Should persist the counter file across open/close", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By("Filling up 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 1 * telem.SecondTS,
					}))
					_, err := w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
					Expect(err).ToNot(HaveOccurred())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(w1.Commit(ctx, 10*telem.SecondTS+1)).To(Succeed())
					Expect(w1.Close()).To(Succeed())

					By("Closing the db")
					Expect(db.Close()).To(Succeed())

					By("Reopening the db on the same FS")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By("Acquiring a new writer: this should go to file 2 instead of 1")
					w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 15 * telem.SecondTS,
					}))

					Expect(MustSucceed(fs.Exists("2.domain"))).To(BeTrue())

					By("Acquiring a new writer: this should go to file 3")
					w3 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 25 * telem.SecondTS,
					}))

					_, err = w2.Write([]byte{15, 16, 17})
					Expect(err).ToNot(HaveOccurred())
					Expect(w2.Commit(ctx, 17*telem.SecondTS+1)).To(Succeed())
					Expect(w2.Close()).To(Succeed())

					Expect(w3.Close()).To(Succeed())

					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(3)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(0)))
				})

				It("Should open writers on partially full files after reopening the file controller", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By("Filling up 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 1 * telem.SecondTS,
					}))
					_, err := w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
					Expect(err).ToNot(HaveOccurred())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(w1.Commit(ctx, 10*telem.SecondTS+1)).To(Succeed())
					Expect(w1.Close()).To(Succeed())

					By("Acquiring a new writer: this should give us 2.domain")
					w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 11 * telem.SecondTS,
					}))
					_, err = w2.Write([]byte{11, 12, 13})
					Expect(err).ToNot(HaveOccurred())
					Expect(w2.Commit(ctx, 13*telem.SecondTS+1)).To(Succeed())
					Expect(w2.Close()).To(Succeed())

					By("Closing the db")
					Expect(db.Close()).To(Succeed())

					By("Reopening the db on the same FS")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By("Acquiring a new writer: this should go to file 2")
					w3 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 15 * telem.SecondTS,
					}))

					_, err = w3.Write([]byte{15, 16, 17})
					Expect(err).ToNot(HaveOccurred())
					Expect(w3.Commit(ctx, 17*telem.SecondTS+1)).To(Succeed())
					Expect(w3.Close()).To(Succeed())

					By("Acquiring a new writer: this should still go to file 2")
					w4 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 20 * telem.SecondTS,
					}))

					_, err = w4.Write([]byte{20, 21, 22, 23})
					Expect(err).ToNot(HaveOccurred())
					Expect(w4.Commit(ctx, 23*telem.SecondTS+1)).To(Succeed())
					Expect(w4.Close()).To(Succeed())

					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(0)))

					By("Acquiring a new writer: this should go to file 3")
					w5 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 30 * telem.SecondTS,
					}))

					_, err = w5.Write([]byte{30, 31, 32, 33})
					Expect(err).ToNot(HaveOccurred())
					Expect(w5.Commit(ctx, 33*telem.SecondTS+1)).To(Succeed())

					By("Acquiring a new writer: this should go to file 4")
					w6 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 40 * telem.SecondTS,
					}))

					_, err = w6.Write([]byte{40, 41, 42, 43, 44, 45})
					Expect(err).ToNot(HaveOccurred())
					Expect(w6.Commit(ctx, 45*telem.SecondTS+1)).To(Succeed())

					Expect(w5.Close()).To(Succeed())
					Expect(w6.Close()).To(Succeed())

					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(4)))
					Expect(MustSucceed(fs.Stat("4.domain")).Size()).To(Equal(int64(6)))

					By("Asserting that the data is correct", func() {
						var (
							i   = db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
							buf = make([]byte, 10)
						)

						Expect(i.SeekFirst(ctx)).To(BeTrue())
						r := MustSucceed(i.OpenReader(ctx))
						_, err = r.ReadAt(buf, 0)
						Expect(err).ToNot(HaveOccurred())
						Expect(buf).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 3)
						_, err = r.ReadAt(buf, 0)
						Expect(err).ToNot(HaveOccurred())
						Expect(buf).To(Equal([]byte{11, 12, 13}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 3)
						_, err = r.ReadAt(buf, 0)
						Expect(err).ToNot(HaveOccurred())
						Expect(buf).To(Equal([]byte{15, 16, 17}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 4)
						_, err = r.ReadAt(buf, 0)
						Expect(err).ToNot(HaveOccurred())
						Expect(buf).To(Equal([]byte{20, 21, 22, 23}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 4)
						_, err = r.ReadAt(buf, 0)
						Expect(err).ToNot(HaveOccurred())
						Expect(buf).To(Equal([]byte{30, 31, 32, 33}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Next()).To(BeTrue())
						r = MustSucceed(i.OpenReader(ctx))
						buf = make([]byte, 6)
						_, err = r.ReadAt(buf, 0)
						Expect(err).ToNot(HaveOccurred())
						Expect(buf).To(Equal([]byte{40, 41, 42, 43, 44, 45}))
						Expect(r.Close()).To(Succeed())

						Expect(i.Close()).To(Succeed())
					})
				})

				It("Should work with file auto cutoff generated files", func() {
					By("Initializing a file controller")
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By("Filling up 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 1 * telem.SecondTS,
					}))
					_, err := w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
					Expect(err).ToNot(HaveOccurred())
					Expect(w1.Commit(ctx, 10*telem.SecondTS+1)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(0)))
					Expect(w1.Close()).To(Succeed())

					By("Reopening the db")
					Expect(db.Close()).To(Succeed())
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By("Acquiring a new writer")
					w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 11 * telem.SecondTS,
					}))

					_, err = w2.Write([]byte{11, 12, 13, 14})
					Expect(err).ToNot(HaveOccurred())
					Expect(w2.Commit(ctx, 14*telem.SecondTS+1)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(4)))

					_, err = w2.Write([]byte{15, 16, 17, 18, 19, 20, 21})
					Expect(err).ToNot(HaveOccurred())
					Expect(w2.Commit(ctx, 21*telem.SecondTS+1)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(11)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(0)))

					_, err = w2.Write([]byte{22, 23, 24, 25})
					Expect(err).ToNot(HaveOccurred())
					Expect(w2.Commit(ctx, 25*telem.SecondTS+1)).To(Succeed())
					Expect(MustSucceed(fs.Stat("1.domain")).Size()).To(Equal(int64(10)))
					Expect(MustSucceed(fs.Stat("2.domain")).Size()).To(Equal(int64(11)))
					Expect(MustSucceed(fs.Stat("3.domain")).Size()).To(Equal(int64(4)))

					Expect(w2.Close()).To(Succeed())
				})
			})
			Describe("Readers", func() {
				It("Should manage readers for a file", func() {
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        1 * telem.Megabyte,
						Instrumentation: PanicLogger(),
					}))
					By("Acquiring one writer on the file 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					}))
					_, err := w1.Write([]byte{1, 2, 3, 4, 5})
					Expect(err).ToNot(HaveOccurred())
					Expect(w1.Commit(ctx, 15*telem.SecondTS)).To(Succeed())
					Expect(w1.Close()).To(Succeed())

					i := db.OpenIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(i.SeekFirst(ctx)).To(BeTrue())
					r1 := MustSucceed(i.OpenReader(ctx))
					r2 := MustSucceed(i.OpenReader(ctx))
					Expect(r1.Close()).To(Succeed())
					Expect(r2.Close()).To(Succeed())
					Expect(i.Close()).To(Succeed())
				})
			})
		})
	}
})
