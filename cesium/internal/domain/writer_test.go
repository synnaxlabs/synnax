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
	"encoding/binary"
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func extractPointer(f fs.File) (p struct {
	telem.TimeRange
	fileKey uint16
	offset  uint32
	length  uint32
}) {
	b := make([]byte, 26)
	_, err := f.Read(b)
	Expect(err).ToNot(HaveOccurred())
	p.Start = telem.TimeStamp(binary.LittleEndian.Uint64(b[0:8]))
	p.End = telem.TimeStamp(binary.LittleEndian.Uint64(b[8:16]))
	p.fileKey = binary.LittleEndian.Uint16(b[16:18])
	p.offset = binary.LittleEndian.Uint32(b[18:22])
	p.length = binary.LittleEndian.Uint32(b[22:26])

	return
}

func fileSizes(info []os.FileInfo) (sizes []telem.Size) {
	return lo.Map(info, func(info os.FileInfo, _ int) (sz telem.Size) { return telem.Size(info.Size()) })
}

func filterDataFiles(info []os.FileInfo) []os.FileInfo {
	return lo.Filter(info, func(item os.FileInfo, _ int) bool {
		return item.Name() != "counter.domain" && item.Name() != "index.domain" && item.Name() != "tombstone.domain"
	})
}

var _ = Describe("Writer Behavior", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, func() {
			var (
				db      *domain.DB
				fs      fs.FS
				cleanUp func() error
			)
			BeforeEach(func() {
				fs, cleanUp = makeFS()
				db = MustSucceed(domain.Open(domain.Config{
					FS:              fs,
					Instrumentation: PanicLogger(),
				}))
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})
			Describe("Happiest of paths", func() {
				It("Should work with a preset end", func() {
					w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS}))
					Expect(w.Write([]byte{10, 11, 12, 13, 14})).To(Equal(5))
					Expect(w.Len()).To(Equal(int64(5)))
					Expect(w.Commit(ctx, 14*telem.SecondTS+1)).To(Succeed())
					Expect(w.Write([]byte{15, 16, 17, 18, 19})).To(Equal(5))
					Expect(w.Len()).To(Equal(int64(10)))
					Expect(w.Commit(ctx, 19*telem.SecondTS+1)).To(Succeed())
					Expect(w.Write([]byte{100, 101, 102, 103, 104})).To(Equal(5))
					Expect(w.Len()).To(Equal(int64(15)))
					Expect(w.Commit(ctx, 104*telem.SecondTS+1)).To(MatchError(ContainSubstring("cannot be greater than preset end timestamp")))
					Expect(w.Close()).To(Succeed())
				})
			})
			Describe("Closed database", func() {
				It("Should not allow opening a writer on a closed database", func() {
					Expect(db.Close()).To(Succeed())
					_, err := db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS})
					Expect(err).To(HaveOccurredAs(resource.NewErrClosed("domain.db")))
				})
			})
			Describe("Start Validation", func() {
				Context("No domain overlap", func() {
					It("Should successfully open the writer", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("TimeRange overlap", func() {
					It("Should fail to open the writer", func() {
						w := MustSucceed(db.OpenWriter(
							ctx,
							domain.WriterConfig{
								Start: 10 * telem.SecondTS,
							}))
						Expect(w.Write([]byte{1, 2, 3, 4, 5, 6})).To(Equal(6))
						Expect(w.Commit(ctx, 15*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						_, err := db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						})
						Expect(err).To(HaveOccurredAs(domain.ErrWriteConflict))
					})
				})
			})
			Describe("CheckFileSizeAndMaybeSwitchFile", func() {
				Context("No preset end", func() {
					It("Should start writing to a new file when one file is full", func() {
						fs2, cleanUp2 := makeFS()
						db2 := MustSucceed(domain.Open(domain.Config{
							FS:              fs2,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By("Writing some telemetry")
						w := MustSucceed(db2.OpenWriter(ctx, domain.WriterConfig{Start: 1 * telem.SecondTS}))
						Expect(w.Write([]byte{1, 2, 3, 4, 5})).To(Equal(5))
						l := MustSucceed(fs2.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain" && item.Name() != "tombstone.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(telem.Byte * 5)))
						Expect(w.Commit(ctx, 5*telem.SecondTS+1)).To(Succeed())

						By("Asserting that it should not switch file when the file is not oversize")
						Expect(w.Write([]byte{6, 7, 8, 9, 10, 11})).To(Equal(6))
						l = MustSucceed(fs2.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain" && item.Name() != "tombstone.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(telem.Byte * 11)))
						Expect(w.Commit(ctx, 11*telem.SecondTS+1)).To(Succeed())

						By("Asserting that it should switch files when the file is oversize")
						Expect(w.Write([]byte{21, 22, 23})).To(Equal(3))
						Expect(w.Commit(ctx, 23*telem.SecondTS+1)).To(Succeed())
						l = MustSucceed(fs2.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain" && item.Name() != "tombstone.domain"
						})
						Expect(l).To(HaveLen(2))
						sizes := lo.Map(l, func(info os.FileInfo, _ int) (sz int64) { return info.Size() })
						Expect(sizes).To(ConsistOf(int64(telem.Byte*11), int64(telem.Byte*3)))

						By("Asserting the data is stored as expected")
						i := db2.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((1 * telem.SecondTS).Range(11*telem.SecondTS + 1)))
						Expect(i.Size()).To(Equal(telem.Byte * 11))

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((11*telem.SecondTS + 1).Range(23*telem.SecondTS + 1)))
						Expect(i.Size()).To(Equal(telem.Byte * 3))

						By("Closing resources")
						Expect(i.Close()).To(Succeed())
						Expect(w.Close()).To(Succeed())
						Expect(db2.Close()).To(Succeed())
						Expect(cleanUp2()).To(Succeed())
					})

					It("Should work when the file size exceeds the limit by just 1", func() {
						fs2, cleanUp2 := makeFS()
						db2 := MustSucceed(domain.Open(domain.Config{
							FS:              fs2,
							FileSize:        5 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By("Writing some telemetry")
						w := MustSucceed(db2.OpenWriter(ctx, domain.WriterConfig{Start: 1 * telem.SecondTS}))
						Expect(w.Write([]byte{1, 2, 3, 4, 5})).To(Equal(5))
						l := MustSucceed(fs2.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain" && item.Name() != "tombstone.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(telem.Byte * 5)))
						Expect(w.Commit(ctx, 5*telem.SecondTS+1)).To(Succeed())

						By("Asserting that it should switch files when the file is oversize")
						Expect(w.Write([]byte{21, 22, 23})).To(Equal(3))
						Expect(w.Commit(ctx, 23*telem.SecondTS+1)).To(Succeed())
						l = MustSucceed(fs2.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain" && item.Name() != "tombstone.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect(lo.Map(l, func(info os.FileInfo, _ int) (sz int64) { return info.Size() })).To(ConsistOf(int64(5), int64(3)))

						By("Asserting the data is stored as expected")
						i := db2.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((1 * telem.SecondTS).Range(5*telem.SecondTS + 1)))
						Expect(i.Size()).To(Equal(telem.Byte * 5))

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((5*telem.SecondTS + 1).Range(23*telem.SecondTS + 1)))
						Expect(i.Size()).To(Equal(telem.Byte * 3))

						By("Closing resources")
						Expect(i.Close()).To(Succeed())
						Expect(w.Close()).To(Succeed())
						Expect(db2.Close()).To(Succeed())
						Expect(cleanUp2()).To(Succeed())
					})
				})

				Context("With preset end", func() {
					It("Should start writing to a new file when one file is full", func() {
						fs2, cleanUp2 := makeFS()
						db2 := MustSucceed(domain.Open(domain.Config{
							FS:              fs2,
							FileSize:        10 * telem.Byte,
							Instrumentation: PanicLogger(),
						}))

						By("Writing some telemetry")
						w := MustSucceed(db2.OpenWriter(ctx, domain.WriterConfig{Start: 1 * telem.SecondTS, End: 100 * telem.SecondTS}))
						Expect(w.Write([]byte{1, 2, 3, 4, 5})).To(Equal(5))
						Expect(w.Commit(ctx, 5*telem.SecondTS+1)).To(Succeed())
						l := filterDataFiles(MustSucceed(fs2.List("")))
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(telem.Byte * 5)))

						By("Asserting that it should not switch file when the file is not oversize")
						Expect(w.Write([]byte{6, 7, 8, 9, 10, 11})).To(Equal(6))
						l = filterDataFiles(MustSucceed(fs2.List("")))
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(telem.Byte * 11)))
						Expect(w.Commit(ctx, 11*telem.SecondTS+1)).To(Succeed())

						By("Asserting that it should switch files when the file is oversize")
						Expect(w.Write([]byte{21, 22, 23})).To(Equal(3))
						Expect(w.Commit(ctx, 23*telem.SecondTS+1)).To(Succeed())
						l = filterDataFiles(MustSucceed(fs2.List("")))
						Expect(l).To(HaveLen(2))
						Expect(fileSizes(l)).To(ConsistOf(telem.Byte*11, telem.Byte*3))

						By("Asserting the data is stored as expected")
						i := db2.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((1 * telem.SecondTS).Range(11*telem.SecondTS + 1)))
						Expect(i.Size()).To(Equal(telem.Byte * 11))

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((11*telem.SecondTS + 1).Range(100 * telem.SecondTS)))
						Expect(i.Size()).To(Equal(telem.Byte * 3))

						By("Closing resources")
						Expect(i.Close()).To(Succeed())
						Expect(w.Close()).To(Succeed())
						Expect(db2.Close()).To(Succeed())
						Expect(cleanUp2()).To(Succeed())
					})
				})
			})
			Describe("End Validation", func() {
				Context("No domain overlap", func() {
					It("Should successfully commit", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("TimeRange overlap", func() {
					It("Should fail to commit", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						w = MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 4 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 15*telem.SecondTS)).To(HaveOccurredAs(domain.ErrWriteConflict))
						Expect(w.Close()).To(Succeed())
					})
					It("Should fail to commit an update to a writer", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						w = MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 4 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4}))
						Expect(w.Commit(ctx, 8*telem.SecondTS)).To(Succeed())
						Expect(w.Commit(ctx, 15*telem.SecondTS)).To(HaveOccurredAs(domain.ErrWriteConflict))
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("Writing past preset end", func() {
					It("Should fail to commit", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
							End:   20 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 30*telem.SecondTS)).To(MatchError(ContainSubstring("commit timestamp %s cannot be greater than preset end timestamp %s", 30*telem.SecondTS, 20*telem.SecondTS)))
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("Commit at start", func() {
					It("Should fail to commit if data was written", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 10*telem.SecondTS)).To(HaveOccurredAs(validate.Error))
						Expect(w.Close()).To(Succeed())
					})
					It("Should not fail to commit if no data was written", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						Expect(w.Commit(ctx, 10*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("Commit before start", func() {
					It("Should fail to commit", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 5*telem.SecondTS)).To(HaveOccurredAs(validate.Error))
						Expect(w.Close()).To(Succeed())
					})
				})
				Describe("End of one domain is the start of another", func() {
					It("Should successfully commit", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						w = MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 20 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 30*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("Multi Commit", func() {
					It("Should correctly commit a writer multiple times", func() {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 30*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
					})
					Context("Commit before previous commit", func() {
						It("Should fail to commit", func() {
							w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
								Start: 10 * telem.SecondTS,
							}))
							MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
							Expect(w.Commit(ctx, 15*telem.SecondTS)).To(Succeed())
							Expect(w.Commit(ctx, 14*telem.SecondTS)).To(HaveOccurredAs(validate.Error))
							Expect(w.Close()).To(Succeed())
						})
					})
				})
				Context("Concurrent Writes", func() {
					It("Should fail to commit all but one of the writes", func() {
						writerCount := 20
						errors := make([]error, writerCount)
						writers := make([]*domain.Writer, writerCount)
						var wg sync.WaitGroup
						wg.Add(writerCount)
						for i := range writerCount {
							writers[i] = MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
								Start: 10 * telem.SecondTS,
							}))
						}
						for i, w := range writers {
							i, w := i, w
							go func(i int, w *domain.Writer) {
								defer wg.Done()
								MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
								errors[i] = w.Commit(ctx, 15*telem.SecondTS)
								Expect(w.Close()).To(Succeed())
							}(i, w)
						}
						wg.Wait()

						occurred := lo.Filter(errors, func(err error, i int) bool {
							return err != nil
						})
						Expect(occurred).To(HaveLen(writerCount - 1))
						for _, err := range occurred {
							Expect(err).To(HaveOccurredAs(domain.ErrWriteConflict))
						}
					})
				})
			})
			Describe("AutoPersist", func() {
				It("Should persist to disk every subsequent call after the set time interval", func() {
					By("Opening a writer")
					w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS, AutoIndexPersistInterval: 50 * telem.Millisecond}))

					modTime := MustSucceed(fs.Stat("index.domain")).ModTime()

					By("Writing some data and committing it right after")
					_, err := w.Write([]byte{6, 7, 8, 9, 10})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 20*telem.SecondTS+1)).To(Succeed())

					_, err = w.Write([]byte{11, 12, 13, 14, 15})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 25*telem.SecondTS+1)).To(Succeed())

					By("Asserting that the previous commits have not been persisted")
					s := MustSucceed(fs.Stat("index.domain"))
					Expect(s.Size()).To(Equal(int64(0)))

					By("Sleeping for some time")
					time.Sleep(time.Duration(50 * telem.Millisecond))
					_, err = w.Write([]byte{16, 17, 18, 19, 20})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 30*telem.SecondTS+1)).To(Succeed())

					By("Asserting that the commits will be persisted the next time we use the method after the set time interval")
					Eventually(func() time.Time {
						return MustSucceed(fs.Stat("index.domain")).ModTime()
					}).ShouldNot(Equal(modTime))

					f := MustSucceed(fs.Open("index.domain", os.O_RDONLY))
					p := extractPointer(f)
					Expect(p.End).To(Equal(30*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(15)))

					Expect(f.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})

				It("Should persist to disk every time when the interval is set to always persist", func() {
					By("Opening a writer")
					w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS, AutoIndexPersistInterval: domain.AlwaysIndexPersistOnAutoCommit}))

					By("Writing some data and committing it")
					_, err := w.Write([]byte{1, 2, 3, 4, 5})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 15*telem.SecondTS+1)).To(Succeed())

					By("Asserting that the previous commit has been persisted")
					f := MustSucceed(fs.Open("index.domain", os.O_RDONLY))
					p := extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.End).To(Equal(15*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(5)))

					By("Writing some data and committing it with auto persist right after")
					_, err = w.Write([]byte{6, 7, 8, 9, 10})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 20*telem.SecondTS+1)).To(Succeed())

					By("Asserting that the previous commit has been persisted")
					f = MustSucceed(fs.Open("index.domain", os.O_RDONLY))
					p = extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.End).To(Equal(20*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(10)))

					By("Writing some data and committing it with auto persist right after")
					_, err = w.Write([]byte{11, 12, 13, 14, 15})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 25*telem.SecondTS+1)).To(Succeed())

					By("Asserting that the previous commits have not been persisted")
					f = MustSucceed(fs.Open("index.domain", os.O_RDONLY))
					p = extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.End).To(Equal(25*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(15)))

					Expect(w.Close()).To(Succeed())
				})

				It("Should persist any unpersisted, but committed (stranded) data on close", func() {
					By("Opening a writer")
					w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS, AutoIndexPersistInterval: 10 * telem.Second}))

					By("Writing some data and committing it")
					_, err := w.Write([]byte{1, 2, 3, 4, 5})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 15*telem.SecondTS+1)).To(Succeed())

					By("Writing some data and committing it")
					_, err = w.Write([]byte{6, 7, 8, 9, 10})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 20*telem.SecondTS+1)).To(Succeed())

					By("Closing the writer")
					Expect(w.Close()).To(Succeed())

					By("Asserting that the commit has been persisted")
					f := MustSucceed(fs.Open("index.domain", os.O_RDONLY))
					p := extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.End).To(Equal(20*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(10)))

					Expect(w.Close()).To(Succeed())
				})

				It("Should always persist if auto commit is not enabled, no matter the interval", func() {
					By("Opening a writer")
					w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
						Start:                    10 * telem.SecondTS,
						AutoIndexPersistInterval: 1 * telem.Hour,
						EnableAutoCommit:         config.False(),
					}))

					By("Writing some data and committing it")
					_, err := w.Write([]byte{1, 2, 3, 4, 5})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 15*telem.SecondTS+1)).To(Succeed())

					By("Writing some data and committing it")
					_, err = w.Write([]byte{6, 7, 8, 9, 10})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Commit(ctx, 20*telem.SecondTS+1)).To(Succeed())

					By("Asserting that the commit has been persisted")
					f := MustSucceed(fs.Open("index.domain", os.O_RDONLY))
					p := extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.End).To(Equal(20*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(10)))

					By("Closing the writer")
					Expect(w.Close()).To(Succeed())
				})
			})
			Describe("Close", func() {
				It("Should not allow operations on a closed writer", func() {
					var (
						w = MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS}))
						e = resource.NewErrClosed("domain.writer")
					)
					Expect(w.Close()).To(Succeed())
					err := w.Commit(ctx, telem.TimeStampMax)
					Expect(err).To(HaveOccurredAs(e))
					_, err = w.Write([]byte{1, 2, 3})
					Expect(err).To(HaveOccurredAs(e))
					Expect(w.Close()).To(Succeed())
				})

				It("Should not open a writer on a closed database", func() {
					Expect(db.Close()).To(Succeed())
					_, err := db.OpenWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS})
					Expect(err).To(HaveOccurredAs(resource.NewErrClosed("domain.db")))
				})

				It("Should not write on a closed database", func() {
					Expect(db.Close()).To(Succeed())
					Expect(domain.Write(ctx, db, telem.TimeStamp(0).Range(telem.TimeStamp(1)), []byte{1, 2, 3})).To(HaveOccurredAs(resource.NewErrClosed("domain.db")))
				})
			})
		})
	}
})
