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
	"github.com/synnaxlabs/cesium/internal/core"
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("WriterBehavior", func() {
	for fsName, fs := range fileSystems {
		fs := fs()
		Context("FS: "+fsName, func() {
			var db *domain.DB
			BeforeEach(func() {
				db = MustSucceed(domain.Open(domain.Config{FS: MustSucceed(fs.Sub(rootPath))}))
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
				Expect(fs.Remove(rootPath)).To(Succeed())
			})
			Describe("Happiest of paths", func() {
				It("Should work with a preset end", func() {
					w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS}))
					Expect(w.Write([]byte{10, 11, 12, 13, 14})).To(Equal(5))
					Expect(w.Commit(ctx, 14*telem.SecondTS+1)).To(Succeed())
					Expect(w.Write([]byte{15, 16, 17, 18, 19})).To(Equal(5))
					Expect(w.Commit(ctx, 19*telem.SecondTS+1)).To(Succeed())
					Expect(w.Write([]byte{100, 101, 102, 103, 104})).To(Equal(5))
					Expect(w.Commit(ctx, 104*telem.SecondTS+1)).To(Succeed())
				})
			})
			Describe("Start Validation", func() {
				Context("No domain overlap", func() {
					It("Should successfully open the writer", func() {
						w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("TimeRange overlap", func() {
					It("Should fail to open the writer", func() {
						w := MustSucceed(db.NewWriter(
							ctx,
							domain.WriterConfig{
								Start: 10 * telem.SecondTS,
							}))
						Expect(w.Write([]byte{1, 2, 3, 4, 5, 6})).To(Equal(6))
						Expect(w.Commit(ctx, 15*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						_, err := db.NewWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						})
						Expect(err).To(HaveOccurredAs(domain.ErrDomainOverlap))
					})
				})
			})
			Describe("CheckFileSizeAndMaybeSwitchFile", func() {
				Context("No preset end", func() {
					It("Should start writing to a new file when one file is full", func() {
						var db2 = MustSucceed(domain.Open(domain.Config{FS: MustSucceed(fs.Sub(rootPath)), FileSizeCap: 10 * telem.ByteSize}))

						By("Writing some telemetry")
						w := MustSucceed(db2.NewWriter(ctx, domain.WriterConfig{Start: 1 * telem.SecondTS}))
						Expect(w.Write([]byte{1, 2, 3, 4, 5})).To(Equal(5))
						l := MustSucceed(db2.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(5)))
						Expect(w.Commit(ctx, 5*telem.SecondTS+1)).To(Succeed())

						By("Asserting that it should not switch file when the file is not oversize")
						Expect(w.Write([]byte{6, 7, 8, 9, 10, 11})).To(Equal(6))
						l = MustSucceed(db2.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(11)))
						Expect(w.Commit(ctx, 11*telem.SecondTS+1)).To(Succeed())

						By("Asserting that it should switch files when the file is oversize")
						Expect(w.Write([]byte{21, 22, 23})).To(Equal(3))
						Expect(w.Commit(ctx, 23*telem.SecondTS+1)).To(Succeed())
						l = MustSucceed(db2.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect(lo.Map(l, func(info os.FileInfo, _ int) (sz int64) { return info.Size() })).To(ConsistOf(int64(11), int64(3)))

						By("Asserting the data is stored as expected")
						i := db2.NewIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((1 * telem.SecondTS).Range(11*telem.SecondTS + 1)))
						Expect(i.Len()).To(Equal(int64(11)))

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((11*telem.SecondTS + 1).Range(23*telem.SecondTS + 1)))
						Expect(i.Len()).To(Equal(int64(3)))

						By("Closing resources")
						Expect(i.Close()).To(Succeed())
						Expect(w.Close()).To(Succeed())
						Expect(db2.Close()).To(Succeed())
					})
				})

				Context("With preset end", func() {
					It("Should start writing to a new file when one file is full", func() {
						var db2 = MustSucceed(domain.Open(domain.Config{FS: MustSucceed(fs.Sub(rootPath)), FileSizeCap: 10 * telem.ByteSize}))

						By("Writing some telemetry")
						w := MustSucceed(db2.NewWriter(ctx, domain.WriterConfig{Start: 1 * telem.SecondTS, End: 100 * telem.SecondTS}))
						Expect(w.Write([]byte{1, 2, 3, 4, 5})).To(Equal(5))
						Expect(w.Commit(ctx, 5*telem.SecondTS+1)).To(Succeed())
						l := MustSucceed(db2.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(5)))

						By("Asserting that it should not switch file when the file is not oversize")
						Expect(w.Write([]byte{6, 7, 8, 9, 10, 11})).To(Equal(6))
						l = MustSucceed(db2.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(11)))
						Expect(w.Commit(ctx, 11*telem.SecondTS+1)).To(Succeed())

						By("Asserting that it should switch files when the file is oversize")
						Expect(w.Write([]byte{21, 22, 23})).To(Equal(3))
						Expect(w.Commit(ctx, 23*telem.SecondTS+1)).To(Succeed())
						l = MustSucceed(db2.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect(lo.Map(l, func(info os.FileInfo, _ int) (sz int64) { return info.Size() })).To(ConsistOf(int64(11), int64(3)))

						By("Asserting the data is stored as expected")
						i := db2.NewIterator(domain.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((1 * telem.SecondTS).Range(11*telem.SecondTS + 1)))
						Expect(i.Len()).To(Equal(int64(11)))

						Expect(i.Next()).To(BeTrue())
						Expect(i.TimeRange()).To(Equal((11*telem.SecondTS + 1).Range(100 * telem.SecondTS)))
						Expect(i.Len()).To(Equal(int64(3)))

						By("Closing resources")
						Expect(i.Close()).To(Succeed())
						Expect(w.Close()).To(Succeed())
						Expect(db2.Close()).To(Succeed())
					})
				})
			})
			Describe("End Validation", func() {
				Context("No domain overlap", func() {
					It("Should successfully commit", func() {
						w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("TimeRange overlap", func() {
					It("Should fail to commit", func() {
						w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						w = MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
							Start: 4 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 15*telem.SecondTS)).To(HaveOccurredAs(domain.ErrDomainOverlap))
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("Commit before start", func() {
					It("Should fail to commit", func() {
						w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 5*telem.SecondTS)).To(HaveOccurredAs(validate.Error))
						Expect(w.Close()).To(Succeed())
					})
				})
				Describe("End of one domain is the start of another", func() {
					It("Should successfully commit", func() {
						w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
							Start: 10 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						w = MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
							Start: 20 * telem.SecondTS,
						}))
						MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
						Expect(w.Commit(ctx, 30*telem.SecondTS)).To(Succeed())
						Expect(w.Close()).To(Succeed())
					})
				})
				Context("Multi Commit", func() {
					It("Should correctly commit a writer multiple times", func() {
						w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
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
							w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
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
					It("Should fail to commit one of the writes", func() {
						writerCount := 20
						errors := make([]error, writerCount)
						writers := make([]*domain.Writer, writerCount)
						var wg sync.WaitGroup
						wg.Add(writerCount)
						for i := 0; i < writerCount; i++ {
							writers[i] = MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
								Start: 10 * telem.SecondTS,
							}))
						}
						for i, w := range writers {
							go func(i int, w *domain.Writer) {
								defer wg.Done()
								MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6}))
								errors[i] = w.Commit(ctx, 15*telem.SecondTS)
							}(i, w)
						}
						wg.Wait()

						occurred := lo.Filter(errors, func(err error, i int) bool {
							return err != nil
						})
						Expect(occurred).To(HaveLen(writerCount - 1))
						for _, err := range occurred {
							Expect(err).To(HaveOccurredAs(domain.ErrDomainOverlap))
						}
					})
				})
			})
			Describe("Close", func() {
				It("Should not allow operations on a closed writer", func() {
					var (
						w = MustSucceed(db.NewWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS}))
						e = core.EntityClosed("domain.writer")
					)
					Expect(w.Close()).To(Succeed())
					Expect(w.Commit(ctx, telem.TimeStampMax)).To(MatchError(e))
					_, err := w.Write([]byte{1, 2, 3})
					Expect(err).To(MatchError(e))
					Expect(w.Close()).To(Succeed())
				})
			})
		})
	}
})
