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
	"slices"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// hookedFS runs writeAt on each write to the index file.
type hookedFS struct {
	xfs.FS
	writeAt func(p []byte) error
}

func (fs *hookedFS) Open(name string, flag int) (xfs.File, error) {
	f, err := fs.FS.Open(name, flag)
	if err != nil || name != "index.domain" {
		return f, err
	}
	return &hookedFile{File: f, fs: fs}, nil
}

type hookedFile struct {
	xfs.File
	fs *hookedFS
}

func (f *hookedFile) WriteAt(p []byte, off int64) (int, error) {
	if f.fs.writeAt != nil {
		if err := f.fs.writeAt(p); err != nil {
			return 0, err
		}
	}
	return f.File.WriteAt(p, off)
}

var _ = Describe("Index Persist", Ordered, func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var (
				db *domain.DB
				fs xfs.FS
			)
			BeforeEach(func() {
				fs = openFS()
				db = MustOpen(domain.Open(domain.Config{
					FS:              fs,
					FileSize:        5 * telem.Byte,
					GCThreshold:     0,
					Instrumentation: PanicLogger(),
				}))
			})

			Describe("Concurrent writers", func() {
				It("Should persist every domain", func(ctx SpecContext) {
					const (
						workers = 8
						commits = 128
					)
					var wg sync.WaitGroup
					for w := range workers {
						wg.Go(func() {
							defer GinkgoRecover()
							for c := range commits {
								start := telem.TimeStamp(c*workers+w+1) * telem.SecondTS
								w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
									Start:            start,
									End:              start + 1,
									EnableAutoCommit: new(false),
								}))
								MustSucceed(w.Write([]byte{1}))
								Expect(w.Commit(ctx, start+1)).To(Succeed())
								Expect(w.Close()).To(Succeed())
							}
						})
					}
					wg.Wait()
					Expect(db.Close()).To(Succeed())
					db = MustOpen(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        5 * telem.Byte,
						GCThreshold:     0,
						Instrumentation: PanicLogger(),
					}))
					Expect(MustSucceed(
						testutil.ReadDomain(ctx, db, telem.TimeRangeMax),
					)).To(HaveLen(workers * commits))
				})

				It(
					"Should run writes in the order they prepare",
					func(ctx SpecContext) {
						entered := make(chan struct{}, 1)
						released, release := context.WithTimeout(ctx, 10*time.Second)
						DeferCleanup(release)
						cfg := domain.Config{
							FS:              openFS(),
							FileSize:        5 * telem.Byte,
							GCThreshold:     0,
							Instrumentation: PanicLogger(),
						}
						cfg.OnHistoryChange = func(context.Context, telem.TimeRange) {
							defer GinkgoRecover()
							entered <- struct{}{}
							Eventually(
								released.Done(),
								10*time.Second,
							).Should(BeClosed())
						}
						ordered := MustOpen(domain.Open(cfg))
						write := func(start telem.TimeStamp) error {
							w, err := ordered.OpenWriter(ctx, domain.WriterConfig{
								Start:            start,
								End:              start + 1,
								EnableAutoCommit: new(false),
							})
							if err != nil {
								return err
							}
							if _, err = w.Write([]byte{1}); err != nil {
								return err
							}
							if err = w.Commit(ctx, start+1); err != nil {
								return err
							}
							return w.Close()
						}
						Expect(write(20 * telem.SecondTS)).To(Succeed())

						older, newer := make(chan error, 1), make(chan error, 1)
						go func() { older <- write(10 * telem.SecondTS) }()
						Eventually(entered).Should(Receive())
						go func() { newer <- write(30 * telem.SecondTS) }()
						Consistently(newer, 50*time.Millisecond).ShouldNot(Receive())
						release()
						Eventually(older).Should(Receive(BeNil()))
						Eventually(newer).Should(Receive(BeNil()))

						Expect(ordered.Close()).To(Succeed())
						cfg.OnHistoryChange = nil
						ordered = MustOpen(domain.Open(cfg))
						Expect(MustSucceed(
							testutil.ReadDomain(ctx, ordered, telem.TimeRangeMax),
						)).To(HaveLen(3))
					},
				)
			})

			Describe("Happy path", func() {
				It("Should persist the index", func(ctx SpecContext) {
					By("Writing some data")
					Expect(
						testutil.WriteDomain(
							ctx,
							db,
							(10 * telem.SecondTS).Range(15*telem.SecondTS+1),
							[]byte{10, 11, 12, 13, 14, 15},
						),
					).To(Succeed())
					Expect(
						testutil.WriteDomain(
							ctx,
							db,
							(20 * telem.SecondTS).Range(23*telem.SecondTS+1),
							[]byte{20, 21, 22, 23},
						),
					).To(Succeed())
					Expect(
						testutil.WriteDomain(
							ctx,
							db,
							(26 * telem.SecondTS).Range(30*telem.SecondTS+1),
							[]byte{26, 27, 30},
						),
					).To(Succeed())
					Expect(
						testutil.WriteDomain(
							ctx,
							db,
							(40 * telem.SecondTS).Range(42*telem.SecondTS+1),
							[]byte{40, 41, 42},
						),
					).To(Succeed())

					By("Deleting some data")
					Expect(db.Delete(
						ctx,
						(12 * telem.SecondTS).Range(15*telem.SecondTS),
						fixedOffset(2),
						fixedOffset(5),
					)).To(Succeed())
					Expect(db.Delete(
						ctx,
						(27 * telem.SecondTS).Range(30*telem.SecondTS),
						fixedOffset(1),
						fixedOffset(2),
					)).To(Succeed())

					By("Re-opening the database")
					Expect(db.Close()).To(Succeed())
					db = MustOpen(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        5 * telem.Byte,
						GCThreshold:     0,
						Instrumentation: PanicLogger(),
					}))

					By("Asserting that the data is still there")
					i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
					Expect(i.SeekFirst(ctx)).To(BeTrue())
					r := MustSucceed(i.OpenReader(ctx))
					buf := make([]byte, 2)

					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{10, 11}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					r = MustSucceed(i.OpenReader(ctx))
					buf = make([]byte, 1)
					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{15}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					r = MustSucceed(i.OpenReader(ctx))
					buf = make([]byte, 4)
					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{20, 21, 22, 23}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					r = MustSucceed(i.OpenReader(ctx))
					buf = make([]byte, 1)
					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{26}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					r = MustSucceed(i.OpenReader(ctx))
					buf = make([]byte, 1)
					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{30}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeTrue())
					r = MustSucceed(i.OpenReader(ctx))
					buf = make([]byte, 3)
					MustSucceed(r.ReadAt(buf, 0))
					MustSucceed(r.ReadAt(buf, 0))
					Expect(buf).To(Equal([]byte{40, 41, 42}))
					Expect(r.Close()).To(Succeed())

					Expect(i.Next()).To(BeFalse())
					Expect(i.Close()).To(Succeed())
				})

				It("Should persist an empty index", func(ctx SpecContext) {
					Expect(db.Close()).To(Succeed())
					db = MustOpen(domain.Open(domain.Config{
						FS:              fs,
						Instrumentation: PanicLogger(),
					}))
					i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
					Expect(i.SeekFirst(ctx)).To(BeFalse())
					Expect(i.Close()).To(Succeed())
				})
			})

			Describe("Buffer reuse", func() {
				var hooked *hookedFS
				reopen := func(fs xfs.FS) {
					GinkgoHelper()
					Expect(db.Close()).To(Succeed())
					db = MustOpen(domain.Open(domain.Config{
						FS:              fs,
						FileSize:        telem.Megabyte,
						GCThreshold:     0,
						Instrumentation: PanicLogger(),
					}))
				}
				openWriter := func(
					ctx SpecContext,
					start telem.TimeStamp,
					data []byte,
				) *domain.Writer {
					GinkgoHelper()
					w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						End:              start + 3*telem.SecondTS,
						EnableAutoCommit: new(false),
					}))
					Expect(w.SetStart(start)).To(Succeed())
					MustSucceed(w.Write(data))
					return w
				}
				expectDomains := func(
					ctx SpecContext,
					data []byte,
					ranges ...telem.TimeRange,
				) {
					GinkgoHelper()
					reopen(fs)
					Expect(
						testutil.ReadDomain(ctx, db, telem.TimeRangeMax),
					).To(Equal(data))
					i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
					persisted := make([]telem.TimeRange, 0, len(ranges))
					for ok := i.SeekFirst(ctx); ok; ok = i.Next() {
						persisted = append(persisted, i.TimeRange())
					}
					Expect(i.Close()).To(Succeed())
					Expect(persisted).To(Equal(ranges))
				}
				BeforeEach(func() {
					hooked = &hookedFS{FS: fs}
					reopen(hooked)
				})

				It(
					"Should keep a blocked write intact while another write prepares",
					func(ctx SpecContext) {
						w1 := openWriter(ctx, 10*telem.SecondTS, []byte{1, 2})
						Expect(w1.Commit(ctx, 12*telem.SecondTS)).To(Succeed())
						w2 := openWriter(ctx, 20*telem.SecondTS, []byte{3, 4})
						Expect(w2.Commit(ctx, 22*telem.SecondTS)).To(Succeed())
						MustSucceed(w1.Write([]byte{5}))
						MustSucceed(w2.Write([]byte{6}))

						entered := make(chan struct{}, 2)
						released, release := context.WithTimeout(ctx, 10*time.Second)
						DeferCleanup(release)
						hooked.writeAt = func(p []byte) error {
							encoded := slices.Clone(p)
							entered <- struct{}{}
							<-released.Done()
							if !slices.Equal(p, encoded) {
								return errors.New("buffer changed before its write")
							}
							return nil
						}
						done := make(chan error, 2)
						go func() { done <- w1.Commit(ctx, 13*telem.SecondTS) }()
						Eventually(entered).Should(Receive())
						go func() { done <- w2.Commit(ctx, 23*telem.SecondTS) }()
						Consistently(done, 50*time.Millisecond).ShouldNot(Receive())
						release()
						Eventually(done).Should(Receive(BeNil()))
						Eventually(done).Should(Receive(BeNil()))
						hooked.writeAt = nil

						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
						expectDomains(
							ctx,
							[]byte{1, 2, 5, 3, 4, 6},
							(10 * telem.SecondTS).Range(13*telem.SecondTS),
							(20 * telem.SecondTS).Range(23*telem.SecondTS),
						)
					},
				)

				It(
					"Should persist every pointer after a failed write",
					func(ctx SpecContext) {
						w := openWriter(ctx, 10*telem.SecondTS, []byte{1, 2})
						Expect(w.Commit(ctx, 12*telem.SecondTS)).To(Succeed())
						MustSucceed(w.Write([]byte{3}))

						hooked.writeAt = func([]byte) error { return ErrFault }
						Expect(
							w.Commit(ctx, 13*telem.SecondTS),
						).To(MatchError(ErrFault))
						hooked.writeAt = nil

						Expect(w.Commit(ctx, 13*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						expectDomains(
							ctx,
							[]byte{1, 2, 3},
							(10 * telem.SecondTS).Range(13*telem.SecondTS),
						)
					},
				)

				It(
					"Should succeed when a first commit retries after a failed write",
					func(ctx SpecContext) {
						w := openWriter(ctx, 10*telem.SecondTS, []byte{1, 2})
						hooked.writeAt = func([]byte) error { return ErrFault }
						Expect(
							w.Commit(ctx, 12*telem.SecondTS),
						).To(MatchError(ErrFault))
						hooked.writeAt = nil

						Expect(w.Commit(ctx, 12*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						expectDomains(
							ctx,
							[]byte{1, 2},
							(10 * telem.SecondTS).Range(13*telem.SecondTS),
						)
					},
				)

				writeSecond := func(ctx SpecContext, second int) error {
					start := telem.TimeStamp(second) * telem.SecondTS
					return testutil.WriteDomain(
						ctx,
						db,
						start.Range(start+1),
						[]byte{byte(second)},
					)
				}
				secondRanges := func(seconds ...int) []telem.TimeRange {
					ranges := make([]telem.TimeRange, len(seconds))
					for i, second := range seconds {
						start := telem.TimeStamp(second) * telem.SecondTS
						ranges[i] = start.Range(start + 1)
					}
					return ranges
				}
				// failFirstWrite blocks the next index write until release, then fails
				// it. entered receives once that write starts.
				failFirstWrite := func(
					ctx SpecContext,
				) (<-chan struct{}, context.CancelFunc) {
					entered := make(chan struct{}, 1)
					released, release := context.WithTimeout(ctx, 10*time.Second)
					DeferCleanup(release)
					var blocked atomic.Bool
					hooked.writeAt = func([]byte) error {
						if blocked.Swap(true) {
							return nil
						}
						defer GinkgoRecover()
						entered <- struct{}{}
						Eventually(released.Done(), 10*time.Second).Should(BeClosed())
						return ErrFault
					}
					return entered, release
				}

				It(
					"Should write only the pointers after the last persist",
					func(ctx SpecContext) {
						var written []int
						hooked.writeAt = func(p []byte) error {
							written = append(written, len(p))
							return nil
						}
						for second := 1; second <= 4; second++ {
							Expect(writeSecond(ctx, second)).To(Succeed())
						}
						hooked.writeAt = nil
						Expect(written).To(HaveLen(4))
						Expect(written).To(HaveEach(written[0]))
						expectDomains(
							ctx,
							[]byte{1, 2, 3, 4},
							secondRanges(1, 2, 3, 4)...,
						)
					},
				)

				It(
					"Should rewrite the pointers a failed write left out",
					func(ctx SpecContext) {
						Expect(writeSecond(ctx, 1)).To(Succeed())
						hooked.writeAt = func([]byte) error { return ErrFault }
						Expect(writeSecond(ctx, 2)).To(MatchError(ErrFault))
						hooked.writeAt = nil
						Expect(writeSecond(ctx, 3)).To(Succeed())
						expectDomains(ctx, []byte{1, 2, 3}, secondRanges(1, 2, 3)...)
					},
				)

				It(
					"Should keep a queued write after an earlier write fails",
					func(ctx SpecContext) {
						prepared := make(chan struct{}, 1)
						Expect(db.Close()).To(Succeed())
						// A domain inserted before the last one notifies the history
						// change after its write prepares and before it runs.
						db = MustOpen(domain.Open(domain.Config{
							FS:              hooked,
							FileSize:        telem.Megabyte,
							Instrumentation: PanicLogger(),
							OnHistoryChange: func(context.Context, telem.TimeRange) {
								prepared <- struct{}{}
							},
						}))
						Expect(writeSecond(ctx, 1)).To(Succeed())
						entered, release := failFirstWrite(ctx)
						earlier, queued := make(chan error, 1), make(chan error, 1)
						go func() { earlier <- writeSecond(ctx, 3) }()
						Eventually(entered).Should(Receive())
						go func() { queued <- writeSecond(ctx, 2) }()
						Eventually(prepared).Should(Receive())
						release()
						Eventually(earlier).Should(Receive(MatchError(ErrFault)))
						Eventually(queued).Should(Receive(BeNil()))
						hooked.writeAt = nil
						expectDomains(
							ctx,
							[]byte{1, 2, 3},
							secondRanges(1, 2, 3)...,
						)
					},
				)

				It(
					"Should keep a queued write that follows a full rewrite",
					func(ctx SpecContext) {
						prepared := make(chan struct{}, 1)
						Expect(db.Close()).To(Succeed())
						db = MustOpen(domain.Open(domain.Config{
							FS:              hooked,
							FileSize:        telem.Megabyte,
							Instrumentation: PanicLogger(),
							OnHistoryChange: func(context.Context, telem.TimeRange) {
								prepared <- struct{}{}
							},
						}))
						Expect(writeSecond(ctx, 2)).To(Succeed())
						entered, release := failFirstWrite(ctx)
						failed := make(chan error, 1)
						go func() { failed <- writeSecond(ctx, 4) }()
						Eventually(entered).Should(Receive())
						// The first domain rewrites every pointer, then the third
						// queues a partial write behind it.
						rewrite, partial := make(chan error, 1), make(chan error, 1)
						go func() { rewrite <- writeSecond(ctx, 1) }()
						Eventually(prepared).Should(Receive())
						go func() { partial <- writeSecond(ctx, 3) }()
						Eventually(prepared).Should(Receive())
						release()
						Eventually(failed).Should(Receive(MatchError(ErrFault)))
						Eventually(rewrite).Should(Receive(BeNil()))
						Eventually(partial).Should(Receive(BeNil()))
						hooked.writeAt = nil
						expectDomains(
							ctx,
							[]byte{1, 2, 3, 4},
							secondRanges(1, 2, 3, 4)...,
						)
					},
				)

				It(
					"Should keep a delete queued behind a failed write",
					func(ctx SpecContext) {
						prepared := make(chan struct{}, 1)
						Expect(db.Close()).To(Succeed())
						db = MustOpen(domain.Open(domain.Config{
							FS:              hooked,
							FileSize:        telem.Megabyte,
							Instrumentation: PanicLogger(),
							OnHistoryChange: func(context.Context, telem.TimeRange) {
								prepared <- struct{}{}
							},
						}))
						Expect(writeSecond(ctx, 1)).To(Succeed())
						Expect(writeSecond(ctx, 2)).To(Succeed())
						entered, release := failFirstWrite(ctx)
						failed := make(chan error, 1)
						go func() { failed <- writeSecond(ctx, 4) }()
						Eventually(entered).Should(Receive())
						deleted := make(chan error, 1)
						go func() {
							deleted <- db.Delete(
								ctx,
								(2 * telem.SecondTS).Range(3*telem.SecondTS),
								fixedOffset(0),
								fixedOffset(0),
							)
						}()
						Eventually(prepared).Should(Receive())
						release()
						Eventually(failed).Should(Receive(MatchError(ErrFault)))
						Eventually(deleted).Should(Receive(BeNil()))
						hooked.writeAt = nil
						expectDomains(ctx, []byte{1, 4}, secondRanges(1, 4)...)
					},
				)

				It(
					"Should keep a close queued behind a failed write",
					func(ctx SpecContext) {
						Expect(writeSecond(ctx, 1)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start:                    10 * telem.SecondTS,
							EnableAutoCommit:         new(true),
							AutoIndexPersistInterval: telem.Hour,
						}))
						MustSucceed(w.Write([]byte{10, 11}))
						Expect(w.Commit(ctx, 11*telem.SecondTS+1)).To(Succeed())
						MustSucceed(w.Write([]byte{12}))
						Expect(w.Commit(ctx, 12*telem.SecondTS+1)).To(Succeed())
						entered, release := failFirstWrite(ctx)
						failed := make(chan error, 1)
						go func() { failed <- writeSecond(ctx, 20) }()
						Eventually(entered).Should(Receive())
						closed := make(chan error, 1)
						go func() { closed <- w.Close() }()
						Consistently(closed, 50*time.Millisecond).ShouldNot(Receive())
						release()
						Eventually(failed).Should(Receive(MatchError(ErrFault)))
						Eventually(closed).Should(Receive(BeNil()))
						hooked.writeAt = nil
						expectDomains(
							ctx,
							[]byte{1, 10, 11, 12, 20},
							secondRanges(1)[0],
							(10 * telem.SecondTS).Range(12*telem.SecondTS+1),
							secondRanges(20)[0],
						)
					},
				)

				It(
					"Should persist an earlier unpersisted commit on a delete",
					func(ctx SpecContext) {
						Expect(testutil.WriteDomain(
							ctx,
							db,
							(10 * telem.SecondTS).Range(12*telem.SecondTS+1),
							[]byte{10, 11, 12},
						)).To(Succeed())
						Expect(testutil.WriteDomain(
							ctx,
							db,
							(30 * telem.SecondTS).Range(35*telem.SecondTS+1),
							[]byte{30, 31, 32, 33, 34, 35},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start:                    20 * telem.SecondTS,
							EnableAutoCommit:         new(true),
							AutoIndexPersistInterval: telem.Hour,
						}))
						MustSucceed(w.Write([]byte{20, 21}))
						Expect(w.Commit(ctx, 21*telem.SecondTS+1)).To(Succeed())
						// The writer persisted its first commit, so it keeps the
						// second.
						MustSucceed(w.Write([]byte{22, 23}))
						Expect(w.Commit(ctx, 23*telem.SecondTS+1)).To(Succeed())
						Expect(db.Delete(
							ctx,
							telem.TimeRange{
								Start: 32*telem.SecondTS + 1,
								End:   34*telem.SecondTS + 1,
							},
							fixedOffset(3),
							fixedOffset(5),
						)).To(Succeed())
						// Failing the persist of the writer's close leaves the index
						// that the delete wrote.
						hooked.writeAt = func([]byte) error { return ErrFault }
						Expect(w.Close()).To(MatchError(ErrFault))
						hooked.writeAt = nil
						reopen(fs)
						Expect(
							testutil.ReadDomain(ctx, db, telem.TimeRangeMax),
						).To(Equal([]byte{10, 11, 12, 20, 21, 22, 23, 30, 31, 32, 35}))
					},
				)

				It(
					"Should persist a shorter index than the last write",
					func(ctx SpecContext) {
						for _, start := range []telem.TimeStamp{10, 20, 30, 40} {
							Expect(testutil.WriteDomain(
								ctx,
								db,
								(start * telem.SecondTS).Range(
									(start+2)*telem.SecondTS,
								),
								[]byte{byte(start), byte(start + 1)},
							)).To(Succeed())
						}

						Expect(db.Delete(
							ctx,
							(20 * telem.SecondTS).Range(32*telem.SecondTS),
							fixedOffset(0),
							fixedOffset(2),
						)).To(Succeed())

						expectDomains(
							ctx,
							[]byte{10, 11, 40, 41},
							(10 * telem.SecondTS).Range(12*telem.SecondTS),
							(40 * telem.SecondTS).Range(42*telem.SecondTS),
						)
					},
				)
			})
		})
	}
})
