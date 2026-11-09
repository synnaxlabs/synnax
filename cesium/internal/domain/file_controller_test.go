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
	"encoding/binary"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// panicFS panics on domain file opens while armed.
type panicFS struct {
	xfs.FS
	armed *atomic.Bool
}

func (f panicFS) Open(name string, flag int) (xfs.File, error) {
	if f.armed.Load() && name != "counter.domain" && name != "index.domain" {
		panic("open failed")
	}
	return f.FS.Open(name, flag)
}

var errHandleFault = errors.New("handle fault")

// handleFaultFS fails operations on open handles while armed. FaultyFS reaches neither.
type handleFaultFS struct {
	xfs.FS
	failClose *atomic.Bool
	failStat  *atomic.Bool
}

func (f handleFaultFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return &handleFaultFile{File: file, fs: f}, nil
}

type handleFaultFile struct {
	xfs.File
	fs    handleFaultFS
	stats int
}

// Stat fails on the second armed call, which is the size lookup that follows the one
// xio.NewTrackedWriteCloser makes.
func (f *handleFaultFile) Stat() (xfs.FileInfo, error) {
	if f.fs.failStat.Load() {
		if f.stats++; f.stats == 2 {
			return nil, errHandleFault
		}
	}
	return f.File.Stat()
}

// Close closes the file in both cases, as close(2) does when it reports an error.
func (f *handleFaultFile) Close() error {
	err := f.File.Close()
	if f.fs.failClose.CompareAndSwap(true, false) {
		return errHandleFault
	}
	return err
}

// createCounter stores value as the file counter of the domain DB in fs.
func createCounter(fs xfs.FS, value int32) {
	GinkgoHelper()
	f := MustSucceed(fs.Open("counter.domain", os.O_CREATE|os.O_RDWR))
	Expect(f.Write(binary.LittleEndian.AppendUint32(nil, uint32(value)))).To(Equal(4))
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("File Controller", Ordered, func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db *domain.DB
				fs xfs.FS
			)
			BeforeEach(func() {
				fs = openFS()
			})
			Describe("Writers", func() {
				It(
					"Should allow one writing to a file at all times",
					func(ctx SpecContext) {
						By("Initializing a file controller")
						db = MustOpen(domain.Open(domain.Config{
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
						By(
							"Acquiring a second writer, this would create a new file 2.domain",
						)
						w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 30 * telem.SecondTS,
							End:   40 * telem.SecondTS,
						}))
						Expect(fs.Exists("2.domain")).To(BeTrue())

						By("Closing the first writer")
						Expect(w1.Close()).To(Succeed())

						By("Acquiring a third writer, 1.domain should be acquired")
						w3 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 50 * telem.SecondTS,
							End:   60 * telem.SecondTS,
						}))
						n := MustSucceed(w3.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
						Expect(n).To(Equal(8))
						s := MustSucceed(fs.Stat("1.domain"))
						Expect(s.Size()).To(Equal(int64(8)))

						Expect(w2.Close()).To(Succeed())
						Expect(w3.Close()).To(Succeed())
					},
				)

				It("Should obey the file size limit", func(ctx SpecContext) {
					By("Initializing a file controller")
					db = MustOpen(domain.Open(domain.Config{
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
					n := MustSucceed(w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
					Expect(n).To(Equal(10))
					Expect(w1.Close()).To(Succeed())
					By(
						"Acquiring a second writer, this would create a new file 2.domain since 1.domain is full",
					)
					w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 30 * telem.SecondTS,
						End:   40 * telem.SecondTS,
					}))
					Expect(fs.Exists("2.domain")).To(BeTrue())

					Expect(w2.Close()).To(Succeed())
				})

				It("Should persist obey the file size limit", func(ctx SpecContext) {
					By("Initializing a file controller")
					db = MustOpen(domain.Open(domain.Config{
						FS: fs, FileSize: 10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					By("Acquiring one writer on the file 1.domain")
					w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					}))
					Expect(fs.Exists("1.domain")).To(BeTrue())
					n := MustSucceed(w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
					Expect(n).To(Equal(10))
					Expect(w1.Close()).To(Succeed())

					By("Closing the db")
					Expect(db.Close()).To(Succeed())

					By("Reopening the db and fc")
					db = MustOpen(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        10 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))

					By(
						"Acquiring a second writer, this would create a new file 2.domain since 1.domain is full",
					)
					w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: 30 * telem.SecondTS,
						End:   40 * telem.SecondTS,
					}))
					Expect(fs.Exists("2.domain")).To(BeTrue())

					Expect(w2.Close()).To(Succeed())
				})

				It(
					"Should open a file if it is below threshold",
					func(ctx SpecContext) {
						By("Initializing a file controller")
						db = MustOpen(domain.Open(domain.Config{
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
						n := MustSucceed(w1.Write([]byte{1, 2, 3, 4, 5, 6, 7}))
						Expect(n).To(Equal(7))
						Expect(w1.Close()).To(Succeed())

						By(
							"Acquiring a second writer, this would not create a new file 2.domain since 1.domain not full",
						)
						w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 30 * telem.SecondTS,
							End:   40 * telem.SecondTS,
						}))
						Expect(fs.Exists("2.domain")).To(BeFalse())

						Expect(w2.Close()).To(Succeed())
					},
				)

				It(
					"Should persist and open a file if it is below threshold",
					func(ctx SpecContext) {
						By("Initializing a file controller")
						db = MustOpen(domain.Open(domain.Config{
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
						n := MustSucceed(w1.Write([]byte{1, 2, 3, 4, 5, 6, 7}))
						Expect(n).To(Equal(7))
						Expect(w1.Close()).To(Succeed())

						By("Closing the db")
						Expect(db.Close()).To(Succeed())

						By("Reopening the db and fc")
						db = MustOpen(domain.Open(domain.Config{
							FS:              fs,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By(
							"Acquiring a second writer, this would not create a new file 2.domain since 1.domain is not full",
						)
						w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 30 * telem.SecondTS,
							End:   40 * telem.SecondTS,
						}))
						Expect(fs.Exists("2.domain")).To(BeFalse())

						Expect(w2.Close()).To(Succeed())
					},
				)

				It(
					"Should obey the file descriptor limit under concurrent opens",
					func(ctx SpecContext) {
						const (
							limit = 4
							opens = 64
						)
						db = MustOpen(domain.Open(domain.Config{
							FS:              fs,
							MaxDescriptors:  limit,
							Instrumentation: PanicLogger(),
						}))
						opened := make(chan *domain.Writer, opens)
						for i := range opens {
							go func() {
								defer GinkgoRecover()
								start := telem.TimeStamp(i+1) * telem.SecondTS
								opened <- MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
									Start: start,
									End:   start + telem.SecondTS,
								}))
							}()
						}
						writers := make([]*domain.Writer, limit)
						for i := range writers {
							Eventually(opened).Should(Receive(&writers[i]))
						}
						Consistently(opened, 100*time.Millisecond).ShouldNot(Receive())
						for _, w := range writers {
							Expect(w.Close()).To(Succeed())
						}
						for range opens - limit {
							var w *domain.Writer
							Eventually(opened).Should(Receive(&w))
							Expect(w.Close()).To(Succeed())
						}
						Expect(fs.Exists("5.domain")).To(BeFalse())
					},
				)

				It("Should obey the file descriptor limit", func(ctx SpecContext) {
					By("Initializing a file controller")
					db = MustOpen(domain.Open(domain.Config{
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
					wg.Go(func() {
						w3 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 30 * telem.SecondTS,
							End:   40 * telem.SecondTS,
						}))
						acquired <- struct{}{}
						Expect(w3.Close()).To(Succeed())
					})

					By("Expecting the channel acquisition to fail")
					Consistently(
						acquired,
					).WithTimeout(50 * telem.Millisecond.Duration()).
						ShouldNot(Receive())
					By("Closing the writer 1")
					Expect(w1.Close()).To(Succeed())
					By("Expecting writer 3 to successfully acquire")
					Eventually(acquired).Should(Receive())
					wg.Wait()
					Expect(w2.Close()).To(Succeed())
				})

				It(
					"Should persist the counter file across open/close",
					func(ctx SpecContext) {
						By("Initializing a file controller")
						db = MustOpen(domain.Open(domain.Config{
							FS:              fs,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By("Filling up 1.domain")
						w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 1 * telem.SecondTS,
						}))
						MustSucceed(w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(w1.Commit(ctx, 10*telem.SecondTS+1)).To(Succeed())
						Expect(w1.Close()).To(Succeed())

						By("Closing the db")
						Expect(db.Close()).To(Succeed())

						By("Reopening the db on the same FS")
						db = MustOpen(domain.Open(domain.Config{
							FS:              fs,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By(
							"Acquiring a new writer: this should go to file 2 instead of 1",
						)
						w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 15 * telem.SecondTS,
						}))

						Expect(MustSucceed(fs.Exists("2.domain"))).To(BeTrue())

						By("Acquiring a new writer: this should go to file 3")
						w3 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 25 * telem.SecondTS,
						}))

						MustSucceed(w2.Write([]byte{15, 16, 17}))
						Expect(w2.Commit(ctx, 17*telem.SecondTS+1)).To(Succeed())
						Expect(w2.Close()).To(Succeed())

						Expect(w3.Close()).To(Succeed())

						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("2.domain")).Size(),
						).To(Equal(int64(3)))
						Expect(
							MustSucceed(fs.Stat("3.domain")).Size(),
						).To(Equal(int64(0)))
					},
				)

				It(
					"Should open writers on partially full files after reopening the file controller",
					func(ctx SpecContext) {
						By("Initializing a file controller")
						db = MustOpen(domain.Open(domain.Config{
							FS:              fs,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By("Filling up 1.domain")
						w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 1 * telem.SecondTS,
						}))
						MustSucceed(w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(w1.Commit(ctx, 10*telem.SecondTS+1)).To(Succeed())
						Expect(w1.Close()).To(Succeed())

						By("Acquiring a new writer: this should give us 2.domain")
						w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 11 * telem.SecondTS,
						}))
						MustSucceed(w2.Write([]byte{11, 12, 13}))
						Expect(w2.Commit(ctx, 13*telem.SecondTS+1)).To(Succeed())
						Expect(w2.Close()).To(Succeed())

						By("Closing the db")
						Expect(db.Close()).To(Succeed())

						By("Reopening the db on the same FS")
						db = MustOpen(domain.Open(domain.Config{
							FS:              fs,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By("Acquiring a new writer: this should go to file 2")
						w3 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 15 * telem.SecondTS,
						}))

						MustSucceed(w3.Write([]byte{15, 16, 17}))
						Expect(w3.Commit(ctx, 17*telem.SecondTS+1)).To(Succeed())
						Expect(w3.Close()).To(Succeed())

						By("Acquiring a new writer: this should still go to file 2")
						w4 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 20 * telem.SecondTS,
						}))

						MustSucceed(w4.Write([]byte{20, 21, 22, 23}))
						Expect(w4.Commit(ctx, 23*telem.SecondTS+1)).To(Succeed())
						Expect(w4.Close()).To(Succeed())

						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("2.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("3.domain")).Size(),
						).To(Equal(int64(0)))

						By("Acquiring a new writer: this should go to file 3")
						w5 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 30 * telem.SecondTS,
						}))

						MustSucceed(w5.Write([]byte{30, 31, 32, 33}))
						Expect(w5.Commit(ctx, 33*telem.SecondTS+1)).To(Succeed())

						By("Acquiring a new writer: this should go to file 4")
						w6 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 40 * telem.SecondTS,
						}))

						MustSucceed(w6.Write([]byte{40, 41, 42, 43, 44, 45}))
						Expect(w6.Commit(ctx, 45*telem.SecondTS+1)).To(Succeed())

						Expect(w5.Close()).To(Succeed())
						Expect(w6.Close()).To(Succeed())

						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("2.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("3.domain")).Size(),
						).To(Equal(int64(4)))
						Expect(
							MustSucceed(fs.Stat("4.domain")).Size(),
						).To(Equal(int64(6)))

						By("Asserting that the data is correct", func() {
							var (
								i = db.OpenIterator(
									domain.IteratorConfig{Bounds: telem.TimeRangeMax},
								)
								buf = make([]byte, 10)
							)

							Expect(i.SeekFirst(ctx)).To(BeTrue())
							r := MustSucceed(i.OpenReader(ctx))
							MustSucceed(r.ReadAt(buf, 0))
							Expect(buf).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
							Expect(r.Close()).To(Succeed())

							Expect(i.Next()).To(BeTrue())
							r = MustSucceed(i.OpenReader(ctx))
							buf = make([]byte, 3)
							MustSucceed(r.ReadAt(buf, 0))
							Expect(buf).To(Equal([]byte{11, 12, 13}))
							Expect(r.Close()).To(Succeed())

							Expect(i.Next()).To(BeTrue())
							r = MustSucceed(i.OpenReader(ctx))
							buf = make([]byte, 3)
							MustSucceed(r.ReadAt(buf, 0))
							Expect(buf).To(Equal([]byte{15, 16, 17}))
							Expect(r.Close()).To(Succeed())

							Expect(i.Next()).To(BeTrue())
							r = MustSucceed(i.OpenReader(ctx))
							buf = make([]byte, 4)
							MustSucceed(r.ReadAt(buf, 0))
							Expect(buf).To(Equal([]byte{20, 21, 22, 23}))
							Expect(r.Close()).To(Succeed())

							Expect(i.Next()).To(BeTrue())
							r = MustSucceed(i.OpenReader(ctx))
							buf = make([]byte, 4)
							MustSucceed(r.ReadAt(buf, 0))
							Expect(buf).To(Equal([]byte{30, 31, 32, 33}))
							Expect(r.Close()).To(Succeed())

							Expect(i.Next()).To(BeTrue())
							r = MustSucceed(i.OpenReader(ctx))
							buf = make([]byte, 6)
							MustSucceed(r.ReadAt(buf, 0))
							Expect(buf).To(Equal([]byte{40, 41, 42, 43, 44, 45}))
							Expect(r.Close()).To(Succeed())

							Expect(i.Close()).To(Succeed())
						})
					},
				)

				It(
					"Should work with file auto cutoff generated files",
					func(ctx SpecContext) {
						By("Initializing a file controller")
						db = MustOpen(domain.Open(domain.Config{
							FS:              fs,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By("Filling up 1.domain")
						w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 1 * telem.SecondTS,
						}))
						MustSucceed(w1.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
						Expect(w1.Commit(ctx, 10*telem.SecondTS+1)).To(Succeed())
						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("2.domain")).Size(),
						).To(Equal(int64(0)))
						Expect(w1.Close()).To(Succeed())

						By("Reopening the db")
						Expect(db.Close()).To(Succeed())
						db = MustOpen(domain.Open(domain.Config{
							FS:              fs,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By("Acquiring a new writer")
						w2 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 11 * telem.SecondTS,
						}))

						MustSucceed(w2.Write([]byte{11, 12, 13, 14}))
						Expect(w2.Commit(ctx, 14*telem.SecondTS+1)).To(Succeed())
						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("2.domain")).Size(),
						).To(Equal(int64(4)))

						MustSucceed(w2.Write([]byte{15, 16, 17, 18, 19, 20, 21}))
						Expect(w2.Commit(ctx, 21*telem.SecondTS+1)).To(Succeed())
						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("2.domain")).Size(),
						).To(Equal(int64(11)))
						Expect(
							MustSucceed(fs.Stat("3.domain")).Size(),
						).To(Equal(int64(0)))

						MustSucceed(w2.Write([]byte{22, 23, 24, 25}))
						Expect(w2.Commit(ctx, 25*telem.SecondTS+1)).To(Succeed())
						Expect(
							MustSucceed(fs.Stat("1.domain")).Size(),
						).To(Equal(int64(10)))
						Expect(
							MustSucceed(fs.Stat("2.domain")).Size(),
						).To(Equal(int64(11)))
						Expect(
							MustSucceed(fs.Stat("3.domain")).Size(),
						).To(Equal(int64(4)))

						Expect(w2.Close()).To(Succeed())
					},
				)
			})
			Describe("Waiting at the file descriptor limit", func() {
				const watchdog = 5 * time.Second
				openDB := func(fs xfs.FS, limit int) {
					GinkgoHelper()
					db = MustOpen(domain.Open(domain.Config{
						FS:              fs,
						MaxDescriptors:  limit,
						Instrumentation: PanicLogger(),
					}))
				}
				openAt := func(ctx context.Context, second int) *domain.Writer {
					GinkgoHelper()
					start := telem.TimeStamp(second) * telem.SecondTS
					return MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start: start,
						End:   start + telem.SecondTS,
					}))
				}
				commitAt := func(ctx context.Context, second int) *domain.Writer {
					GinkgoHelper()
					w := openAt(ctx, second)
					MustSucceed(w.Write([]byte{1, 2, 3}))
					end := (telem.TimeStamp(second) * telem.SecondTS) + 1
					Expect(w.Commit(ctx, end)).To(Succeed())
					return w
				}
				// readAsync closes the returned channel once a read on the first domain
				// finishes.
				readAsync := func(ctx context.Context) <-chan struct{} {
					done := make(chan struct{})
					go func() {
						defer GinkgoRecover()
						defer close(done)
						i := db.OpenIterator(
							domain.IteratorConfig{Bounds: telem.TimeRangeMax},
						)
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(MustSucceed(i.OpenReader(ctx)).Close()).To(Succeed())
						Expect(i.Close()).To(Succeed())
					}()
					return done
				}

				It(
					"Should let a reader reclaim idle writer handles",
					func(ctx SpecContext) {
						openDB(fs, 2)
						w1, w2 := commitAt(ctx, 10), commitAt(ctx, 20)
						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
						Eventually(readAsync(ctx), watchdog).Should(BeClosed())
						By("Reopening the reclaimed file for writing")
						Expect(openAt(ctx, 30).Close()).To(Succeed())
						Expect(fs.Exists("3.domain")).To(BeFalse())
					},
				)

				It(
					"Should keep idle writer handles when an idle reader frees a place",
					func(ctx SpecContext) {
						rec := NewRecorder(fs)
						openDB(rec, 2)
						w1, w2 := commitAt(ctx, 10), commitAt(ctx, 20)
						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
						Eventually(readAsync(ctx), watchdog).Should(BeClosed())
						i := db.OpenIterator(
							domain.IteratorConfig{Bounds: telem.TimeRangeMax},
						)
						Expect(i.SeekLast(ctx)).To(BeTrue())
						Expect(MustSucceed(i.OpenReader(ctx)).Close()).To(Succeed())
						Expect(i.Close()).To(Succeed())
						rec.Reset()
						Expect(openAt(ctx, 30).Close()).To(Succeed())
						Expect(rec.Count(MatchOp(OpOpen))).To(BeZero())
					},
				)

				It(
					"Should wake a reader and a writer on one release",
					func(ctx SpecContext) {
						openDB(fs, 2)
						w1, w2 := commitAt(ctx, 10), commitAt(ctx, 20)
						read := readAsync(ctx)
						Consistently(read, 100*time.Millisecond).ShouldNot(BeClosed())
						opened := make(chan *domain.Writer, 1)
						go func() {
							defer GinkgoRecover()
							opened <- openAt(ctx, 30)
						}()
						Consistently(opened, 100*time.Millisecond).ShouldNot(Receive())
						Expect(w1.Close()).To(Succeed())
						var w3 *domain.Writer
						Eventually(opened, watchdog).Should(Receive(&w3))
						Expect(w3.Close()).To(Succeed())
						Eventually(read, watchdog).Should(BeClosed())
						Expect(w2.Close()).To(Succeed())
					},
				)

				Describe("Handle faults", func() {
					var (
						failClose, failStat *atomic.Bool
						faulty              *FaultyFS
					)
					BeforeEach(func() {
						failClose, failStat = &atomic.Bool{}, &atomic.Bool{}
						faulty = WrapFaultyFS(handleFaultFS{
							FS:        fs,
							failClose: failClose,
							failStat:  failStat,
						})
					})

					It(
						"Should close a file whose size lookup fails",
						func(ctx SpecContext) {
							openDB(faulty, 2)
							Expect(commitAt(ctx, 10).Close()).To(Succeed())
							Expect(db.Close()).To(Succeed())
							openDB(faulty, 2)
							handles := faulty.OpenFiles()
							failStat.Store(true)
							Expect(db.OpenWriter(ctx, domain.WriterConfig{
								Start: 20 * telem.SecondTS,
								End:   21 * telem.SecondTS,
							})).Error().To(MatchError(errHandleFault))
							failStat.Store(false)
							Expect(faulty.OpenFiles()).To(Equal(handles))
							Expect(openAt(ctx, 30).Close()).To(Succeed())
							Expect(fs.Exists("2.domain")).To(BeFalse())
						},
					)

					It(
						"Should retire a reader whose close fails",
						func(ctx SpecContext) {
							openDB(faulty, 1)
							Expect(commitAt(ctx, 10).Close()).To(Succeed())
							Eventually(readAsync(ctx), watchdog).Should(BeClosed())
							failClose.Store(true)
							Expect(db.OpenWriter(ctx, domain.WriterConfig{
								Start: 20 * telem.SecondTS,
								End:   21 * telem.SecondTS,
							})).Error().To(MatchError(errHandleFault))
							held := openAt(ctx, 30)
							opened := make(chan *domain.Writer, 1)
							go func() {
								defer GinkgoRecover()
								opened <- openAt(context.Background(), 40)
							}()
							Consistently(opened, 100*time.Millisecond).
								ShouldNot(Receive())
							Expect(held.Close()).To(Succeed())
							var w *domain.Writer
							Eventually(opened, watchdog).Should(Receive(&w))
							Expect(w.Close()).To(Succeed())
						},
					)

					It(
						"Should retire a writer whose close fails",
						func(ctx SpecContext) {
							openDB(faulty, 1)
							Expect(commitAt(ctx, 10).Close()).To(Succeed())
							i := db.OpenIterator(
								domain.IteratorConfig{Bounds: telem.TimeRangeMax},
							)
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							failClose.Store(true)
							Expect(i.OpenReader(ctx)).Error().
								To(MatchError(errHandleFault))
							Expect(i.Close()).To(Succeed())
							Expect(commitAt(ctx, 20).Close()).To(Succeed())
						},
					)
				})

				It(
					"Should keep no place for an open that panics",
					func(ctx SpecContext) {
						armed := &atomic.Bool{}
						openDB(panicFS{FS: fs, armed: armed}, 2)
						armed.Store(true)
						for i := range 2 {
							Expect(
								func() { openAt(ctx, i+1) },
							).To(PanicWith("open failed"))
						}
						armed.Store(false)
						opened := make(chan *domain.Writer, 1)
						go func() {
							defer GinkgoRecover()
							opened <- openAt(ctx, 10)
						}()
						var w *domain.Writer
						Eventually(opened, watchdog).Should(Receive(&w))
						Expect(w.Close()).To(Succeed())
					},
				)

				It(
					"Should stop a writer's wait when its context ends",
					func(ctx SpecContext) {
						openDB(fs, 1)
						held := openAt(ctx, 10)
						waitCtx, cancel := context.WithCancel(ctx)
						defer cancel()
						done := make(chan struct{})
						go func() {
							defer GinkgoRecover()
							defer close(done)
							Expect(db.OpenWriter(waitCtx, domain.WriterConfig{
								Start: 20 * telem.SecondTS,
								End:   21 * telem.SecondTS,
							})).Error().To(MatchError(context.Canceled))
						}()
						Consistently(done, 100*time.Millisecond).ShouldNot(BeClosed())
						cancel()
						Eventually(done, watchdog).Should(BeClosed())
						Expect(held.Close()).To(Succeed())
					},
				)

				It(
					"Should stop a reader's wait when its context ends",
					func(ctx SpecContext) {
						openDB(fs, 1)
						held := commitAt(ctx, 10)
						i := db.OpenIterator(
							domain.IteratorConfig{Bounds: telem.TimeRangeMax},
						)
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						waitCtx, cancel := context.WithCancel(ctx)
						defer cancel()
						done := make(chan struct{})
						go func() {
							defer GinkgoRecover()
							defer close(done)
							Expect(i.OpenReader(waitCtx)).Error().To(
								MatchError(context.Canceled),
							)
						}()
						Consistently(done, 100*time.Millisecond).ShouldNot(BeClosed())
						cancel()
						Eventually(done, watchdog).Should(BeClosed())
						Expect(i.Close()).To(Succeed())
						Expect(held.Close()).To(Succeed())
					},
				)
			})

			Describe("Readers", func() {
				It(
					"Should open a new file handle for each simultaneously held reader",
					func(ctx SpecContext) {
						rec := NewRecorder(fs)
						db = MustOpen(domain.Open(domain.Config{
							FS:              rec,
							FileSize:        1 * telem.Megabyte,
							Instrumentation: PanicLogger(),
						}))
						By("Acquiring one writer on the file 1.domain")
						w1 := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
							End:   20 * telem.SecondTS,
						}))
						MustSucceed(w1.Write([]byte{1, 2, 3, 4, 5}))
						Expect(w1.Commit(ctx, 15*telem.SecondTS)).To(Succeed())
						Expect(w1.Close()).To(Succeed())
						rec.Reset()

						i := db.OpenIterator(
							domain.IteratorConfig{Bounds: telem.TimeRangeMax},
						)
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						r1 := MustSucceed(i.OpenReader(ctx))
						r2 := MustSucceed(i.OpenReader(ctx))
						Expect(r1.Close()).To(Succeed())
						Expect(r2.Close()).To(Succeed())
						Expect(i.Close()).To(Succeed())

						// Two simultaneously held readers cannot share a handle, so
						// each acquire that finds every existing handle busy must
						// open a fresh one.
						Expect(
							rec.Count(
								MatchOp(OpOpen),
								MatchName("1.domain"),
							),
						).
							To(Equal(2))
					},
				)

				It(
					"Should reuse reader file handles across sequential acquires of the same file",
					func(ctx SpecContext) {
						rec := NewRecorder(fs)
						db = MustOpen(domain.Open(domain.Config{
							FS:              rec,
							FileSize:        1 * telem.Megabyte,
							Instrumentation: PanicLogger(),
						}))

						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
							End:   20 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5}))
						Expect(w.Commit(ctx, 15*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())

						// Reset after writes to scope the assertion to reader I/O.
						rec.Reset()

						// Acquire-and-release sequentially: each iteration releases
						// its reader before the next acquire, so the file
						// controller's pool should hand back the same underlying
						// handle every time.
						const acquires = 8
						for range acquires {
							i := db.OpenIterator(
								domain.IteratorConfig{Bounds: telem.TimeRangeMax},
							)
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							r := MustSucceed(i.OpenReader(ctx))
							Expect(r.Close()).To(Succeed())
							Expect(i.Close()).To(Succeed())
						}

						Expect(
							rec.Count(
								MatchOp(OpOpen),
								MatchName("1.domain"),
							),
						).
							To(Equal(1))
					},
				)
			})
		})
	}
	Describe("File key limit", func() {
		It("Should not create a file past the maximum file key", func(ctx SpecContext) {
			fs := xfs.NewMem()
			createCounter(fs, math.MaxUint16)
			db := MustOpen(domain.Open(domain.Config{
				FS:              fs,
				Instrumentation: PanicLogger(),
			}))
			Expect(db.OpenWriter(ctx, domain.WriterConfig{
				Start: 10 * telem.SecondTS,
			})).Error().To(MatchError(domain.ErrFileLimit))
		})
	})
})
