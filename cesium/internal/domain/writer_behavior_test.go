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
	var db *domain.DB
	BeforeEach(func() {
		db = MustSucceed(domain.Open(domain.Config{FS: mfs}))
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
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
	Describe("Auto-commit", func() {
		var extractLength = func() uint32 {
			var buf = make([]byte, 4)
			r := MustSucceed(mfs.Open("index.domain", os.O_RDONLY))
			_, err := r.ReadAt(buf, 22)
			Expect(err).ToNot(HaveOccurred())
			return telem.ByteOrder.Uint32(buf)
		}
		var extractEnd = func() telem.TimeStamp {
			var buf = make([]byte, 8)
			r := MustSucceed(mfs.Open("index.domain", os.O_RDONLY))
			_, err := r.ReadAt(buf, 8)
			Expect(err).ToNot(HaveOccurred())
			return telem.TimeStamp(telem.ByteOrder.Uint64(buf))
		}
		Describe("Writer has no end", func() {
			It("Should not commit until the non-zero threshold is reached", func() {
				w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS, AutoPersistThreshold: 5}))

				By("Writer should commit but index should not persist")
				n := MustSucceed(w.WriteAndAutoCommit(ctx, []byte{1, 2, 3}, 13*telem.SecondTS, 3))
				Expect(n).To(Equal(3))
				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{4}, 14*telem.SecondTS, 1))
				Expect(n).To(Equal(1))

				// Writer is committed.
				i := db.NewIterator(domain.IterRange(telem.TimeRangeMax))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(14 * telem.SecondTS)))
				Expect(i.Close()).To(Succeed())

				// But index is not persisted.
				s := MustSucceed(mfs.Stat("index.domain"))
				Expect(s.Size()).To(Equal(int64(0)))

				By("Writer should commit and index should persist")
				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{5}, 15*telem.SecondTS, 1))
				Expect(n).To(Equal(1))

				Eventually(extractLength()).Should(Equal(uint32(5)))
				Eventually(extractEnd()).Should(Equal(15 * telem.SecondTS))

				i = db.NewIterator(domain.IterRange(telem.TimeRangeMax))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				Expect(i.Len()).To(Equal(int64(5)))
				Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(15 * telem.SecondTS)))
				Expect(i.Close()).To(Succeed())

				By("Writer should commit but index should not persist")
				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{6, 7, 8, 9}, 19*telem.SecondTS, 4))
				Expect(n).To(Equal(4))

				// Not persisted
				Consistently(extractLength()).Should(Equal(uint32(5)))
				Consistently(extractEnd()).Should(Equal(15 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{10, 11, 12, 13, 14}, 24*telem.SecondTS, 5))
				Expect(n).To(Equal(5))

				// Writer should commit and index should persist
				Eventually(extractLength()).Should(Equal(uint32(14)))
				Eventually(extractEnd()).Should(Equal(24 * telem.SecondTS))

				Expect(w.Close()).To(Succeed())

				i = db.NewIterator(domain.IterRange(telem.TimeRangeMax))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				r := MustSucceed(i.NewReader(ctx))

				var buf = make([]byte, 14)
				n = MustSucceed(r.ReadAt(buf, 0))
				Expect(n).To(Equal(14))

				Expect(buf).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}))
				Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(24 * telem.SecondTS)))

				Expect(i.Close()).To(Succeed())
			})

			It("Should always commit when the threshold is set to 1", func() {
				w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS, AutoPersistThreshold: 1}))
				n := MustSucceed(w.WriteAndAutoCommit(ctx, []byte{1, 2, 3}, 13*telem.SecondTS, 3))
				Expect(n).To(Equal(3))

				Eventually(extractLength()).Should(Equal(uint32(3)))
				Eventually(extractEnd()).Should(Equal(13 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{4}, 14*telem.SecondTS, 1))
				Expect(n).To(Equal(1))

				Eventually(extractLength()).Should(Equal(uint32(4)))
				Eventually(extractEnd()).Should(Equal(14 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{5, 6}, 16*telem.SecondTS, 2))
				Expect(n).To(Equal(2))

				Eventually(extractLength()).Should(Equal(uint32(6)))
				Eventually(extractEnd()).Should(Equal(16 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{7, 8, 9, 10}, 20*telem.SecondTS, 4))
				Expect(n).To(Equal(4))

				Eventually(extractLength()).Should(Equal(uint32(10)))
				Eventually(extractEnd()).Should(Equal(20 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{11, 12, 13, 14, 15}, 25*telem.SecondTS, 5))
				Expect(n).To(Equal(5))

				Eventually(extractLength()).Should(Equal(uint32(15)))
				Eventually(extractEnd()).Should(Equal(25 * telem.SecondTS))

				// Should not commit when nothing is written
				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{}, 25*telem.SecondTS, 0))
				Expect(n).To(Equal(0))

				Consistently(extractLength()).Should(Equal(uint32(15)))
				Consistently(extractEnd()).Should(Equal(25 * telem.SecondTS))

				Expect(w.Close()).To(Succeed())

				i := db.NewIterator(domain.IterRange(telem.TimeRangeMax))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				r := MustSucceed(i.NewReader(ctx))

				var buf = make([]byte, 14)
				n = MustSucceed(r.ReadAt(buf, 0))
				Expect(n).To(Equal(14))

				Expect(buf).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}))
				Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(25 * telem.SecondTS)))

				Expect(i.Close()).To(Succeed())
			})
		})
		Describe("Writer has end", func() {
			It("Should not change the end index in the persisted index", func() {
				w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, AutoPersistThreshold: 1}))
				n := MustSucceed(w.WriteAndAutoCommit(ctx, []byte{1, 2, 3}, 13*telem.SecondTS, 3))
				Expect(n).To(Equal(3))

				Eventually(extractLength()).Should(Equal(uint32(3)))
				// End should not change.
				Eventually(extractEnd()).Should(Equal(100 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{4}, 14*telem.SecondTS, 1))
				Expect(n).To(Equal(1))

				Eventually(extractLength()).Should(Equal(uint32(4)))
				Eventually(extractEnd()).Should(Equal(100 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{5, 6}, 16*telem.SecondTS, 2))
				Expect(n).To(Equal(2))

				Eventually(extractLength()).Should(Equal(uint32(6)))
				Eventually(extractEnd()).Should(Equal(100 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{7, 8, 9, 10}, 20*telem.SecondTS, 4))
				Expect(n).To(Equal(4))

				Eventually(extractLength()).Should(Equal(uint32(10)))
				Eventually(extractEnd()).Should(Equal(100 * telem.SecondTS))

				n = MustSucceed(w.WriteAndAutoCommit(ctx, []byte{11, 12, 13, 14, 15}, 25*telem.SecondTS, 5))
				Expect(n).To(Equal(5))

				Eventually(extractLength()).Should(Equal(uint32(15)))
				Eventually(extractEnd()).Should(Equal(100 * telem.SecondTS))

				Expect(w.Close()).To(Succeed())

				i := db.NewIterator(domain.IterRange(telem.TimeRangeMax))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				r := MustSucceed(i.NewReader(ctx))

				var buf = make([]byte, 14)
				n = MustSucceed(r.ReadAt(buf, 0))
				Expect(n).To(Equal(14))

				Expect(buf).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}))
				Expect(i.TimeRange()).To(Equal((10 * telem.SecondTS).Range(100 * telem.SecondTS)))

				Expect(i.Close()).To(Succeed())
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
