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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("WriterBehavior", func() {
	var db *domain.DB
	BeforeEach(func() {
		db = MustSucceed(domain.Open(domain.Config{FS: fs.NewMem()}))
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
