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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Group", func() {
	var (
		db      *domain.DB
		changes []telem.TimeRange
	)

	BeforeEach(func() {
		changes = nil
		db = MustOpen(domain.Open(domain.Config{
			FS:              FileSystems["memFS"](),
			Instrumentation: PanicLogger(),
			OnHistoryChange: func(_ context.Context, tr telem.TimeRange) {
				changes = append(changes, tr)
			},
		}))
	})

	seconds := func(start, end int) telem.TimeRange {
		return (telem.TimeStamp(start) * telem.SecondTS).
			Range(telem.TimeStamp(end) * telem.SecondTS)
	}

	// openWriter opens a writer on a domain that starts at start seconds.
	openWriter := func(ctx context.Context, start int) *domain.Writer {
		GinkgoHelper()
		return DeferClose(MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
			Start: telem.TimeStamp(start) * telem.SecondTS,
		})))
	}

	// commitIn writes one byte with w and commits it in g, through end seconds.
	commitIn := func(ctx context.Context, w *domain.Writer, end int, g *domain.Group) {
		GinkgoHelper()
		MustSucceed(w.Write([]byte{1}))
		Expect(w.Commit(ctx, telem.TimeStamp(end)*telem.SecondTS, g)).To(Succeed())
	}

	// visible returns the domains that a new iterator sees, in order.
	visible := func(ctx context.Context) []telem.TimeRange {
		GinkgoHelper()
		var domains []telem.TimeRange
		i := DeferClose(db.OpenIterator(domain.IterRange(telem.TimeRangeMax)))
		for ok := i.SeekFirst(ctx); ok; ok = i.Next() {
			domains = append(domains, i.TimeRange())
		}
		return domains
	}

	// read returns the bytes that a reader sees.
	read := func(ctx context.Context) []byte {
		GinkgoHelper()
		return MustSucceed(testutil.ReadDomain(ctx, db, telem.TimeRangeMax))
	}

	Describe("Publish", func() {
		It("Should hide a new domain until the group publishes", func(ctx SpecContext) {
			var g domain.Group
			commitIn(ctx, openWriter(ctx, 10), 20, &g)
			Expect(visible(ctx)).To(BeEmpty())
			Expect(MustSucceed(db.HasDataFor(ctx, seconds(10, 20)))).To(BeFalse())
			g.Publish()
			Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 20)}))
		})

		It("Should show a grown domain as it was until the group publishes",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 20, nil)
				commitIn(ctx, w, 30, &g)
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 20)}))
				Expect(read(ctx)).To(Equal([]byte{1}))
				g.Publish()
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 30)}))
				Expect(read(ctx)).To(Equal([]byte{1, 1}))
			})

		It("Should show the domain from before the first of two unpublished commits",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 20, nil)
				commitIn(ctx, w, 30, &g)
				commitIn(ctx, w, 40, &g)
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 20)}))
				g.Publish()
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 40)}))
			})

		It("Should keep a new domain hidden through a second unpublished commit",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 20, &g)
				commitIn(ctx, w, 30, &g)
				Expect(visible(ctx)).To(BeEmpty())
				g.Publish()
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 30)}))
			})

		It("Should hide the commits after a publish until the next one",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 20, &g)
				g.Publish()
				commitIn(ctx, w, 30, &g)
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 20)}))
				g.Publish()
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 30)}))
			})

		It("Should publish only the commits of its own group", func(ctx SpecContext) {
			var g1, g2 domain.Group
			commitIn(ctx, openWriter(ctx, 10), 20, &g1)
			commitIn(ctx, openWriter(ctx, 30), 40, &g2)
			g2.Publish()
			Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(30, 40)}))
			g1.Publish()
			Expect(visible(ctx)).To(Equal([]telem.TimeRange{
				seconds(10, 20),
				seconds(30, 40),
			}))
		})

		It("Should never show a commit of one DB without the commit of the other",
			func(ctx SpecContext) {
				const commits = 2000
				other := MustOpen(domain.Open(domain.Config{
					FS:              FileSystems["memFS"](),
					Instrumentation: PanicLogger(),
				}))
				// end returns the end of the last domain that a reader of d sees.
				end := func(d *domain.DB) telem.TimeStamp {
					i := d.OpenIterator(domain.IterRange(telem.TimeRangeMax))
					defer func() { Expect(i.Close()).To(Succeed()) }()
					if !i.SeekLast(ctx) {
						return 0
					}
					return i.TimeRange().End
				}
				var (
					g  domain.Group
					w1 = openWriter(ctx, 10)
					w2 = DeferClose(MustSucceed(other.OpenWriter(
						ctx,
						domain.WriterConfig{Start: 10 * telem.SecondTS},
					)))
					wg   sync.WaitGroup
					done = make(chan struct{})
				)
				wg.Go(func() {
					defer GinkgoRecover()
					for {
						select {
						case <-done:
							return
						default:
						}
						// db commits first, so a reader that sees its commit must then
						// see the commit of other.
						first := end(db)
						Expect(end(other)).To(BeNumerically(">=", first))
					}
				})
				for i := range commits {
					ts := 10*telem.SecondTS + telem.TimeStamp(i) + 1
					MustSucceed(w1.Write([]byte{1}))
					Expect(w1.Commit(ctx, ts, &g)).To(Succeed())
					MustSucceed(w2.Write([]byte{1}))
					Expect(w2.Commit(ctx, ts, &g)).To(Succeed())
					g.Publish()
				}
				close(done)
				wg.Wait()
			})

		It("Should show a commit outside a group at once", func(ctx SpecContext) {
			var g domain.Group
			w := openWriter(ctx, 10)
			commitIn(ctx, w, 20, &g)
			commitIn(ctx, w, 30, nil)
			Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 30)}))
		})
	})

	Describe("Writers", func() {
		It("Should check a new writer against an unpublished domain",
			func(ctx SpecContext) {
				var g domain.Group
				commitIn(ctx, openWriter(ctx, 10), 20, &g)
				Expect(db.OpenWriter(ctx, domain.WriterConfig{
					Start: 15 * telem.SecondTS,
				})).Error().To(MatchError(domain.ErrWriteConflict))
			})

		It("Should show an unpublished commit once its writer closes",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 20, &g)
				Expect(w.Close()).To(Succeed())
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 20)}))
			})

		It(
			"Should show an unpublished commit once its writer commits to another domain",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 20, &g)
				Expect(w.Finish(ctx)).To(Succeed())
				Expect(w.SetStart(30 * telem.SecondTS)).To(Succeed())
				commitIn(ctx, w, 40, &g)
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(10, 20)}))
			},
		)
	})

	Describe("Iterators", func() {
		It("Should find a domain that was published after the iterator opened",
			func(ctx SpecContext) {
				var g domain.Group
				commitIn(ctx, openWriter(ctx, 10), 20, nil)
				commitIn(ctx, openWriter(ctx, 30), 40, &g)
				i := DeferClose(db.OpenIterator(domain.IterRange(telem.TimeRangeMax)))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				g.Publish()
				Expect(i.Next()).To(BeTrue())
				Expect(i.TimeRange()).To(Equal(seconds(30, 40)))
			})

		It("Should keep the domain it holds through a publish", func(ctx SpecContext) {
			var g domain.Group
			w := openWriter(ctx, 10)
			commitIn(ctx, w, 20, nil)
			commitIn(ctx, w, 30, &g)
			commitIn(ctx, openWriter(ctx, 50), 60, nil)
			i := DeferClose(db.OpenIterator(domain.IterRange(telem.TimeRangeMax)))
			Expect(i.SeekFirst(ctx)).To(BeTrue())
			g.Publish()
			Expect(i.TimeRange()).To(Equal(seconds(10, 20)))
			Expect(i.Next()).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(seconds(50, 60)))
			Expect(i.SeekFirst(ctx)).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(seconds(10, 30)))
		})

		It("Should seek past the unpublished part of a domain", func(ctx SpecContext) {
			var g domain.Group
			w := openWriter(ctx, 10)
			commitIn(ctx, w, 20, nil)
			commitIn(ctx, w, 30, &g)
			commitIn(ctx, openWriter(ctx, 50), 60, nil)
			i := DeferClose(db.OpenIterator(domain.IterRange(telem.TimeRangeMax)))
			Expect(i.SeekGE(ctx, 25*telem.SecondTS)).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(seconds(50, 60)))
			Expect(i.SeekLE(ctx, 25*telem.SecondTS)).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(seconds(10, 20)))
			Expect(i.Prev()).To(BeFalse())
		})

		It("Should skip unpublished domains in both directions", func(ctx SpecContext) {
			var g domain.Group
			commitIn(ctx, openWriter(ctx, 10), 20, nil)
			commitIn(ctx, openWriter(ctx, 30), 40, &g)
			commitIn(ctx, openWriter(ctx, 50), 60, nil)
			i := DeferClose(db.OpenIterator(domain.IterRange(telem.TimeRangeMax)))
			Expect(i.SeekGE(ctx, 35*telem.SecondTS)).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(seconds(50, 60)))
			Expect(i.Prev()).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(seconds(10, 20)))
			Expect(i.SeekLE(ctx, 35*telem.SecondTS)).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(seconds(10, 20)))
			Expect(i.SeekLast(ctx)).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(seconds(50, 60)))
		})
	})

	Describe("Deletes", func() {
		It("Should show what a delete leaves of an unpublished commit at once",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				MustSucceed(w.Write([]byte{1, 2}))
				Expect(w.Commit(ctx, 20*telem.SecondTS, nil)).To(Succeed())
				MustSucceed(w.Write([]byte{3, 4}))
				Expect(w.Commit(ctx, 30*telem.SecondTS, &g)).To(Succeed())
				Expect(db.Delete(
					ctx,
					seconds(10, 25),
					fixedOffset(0),
					fixedOffset(3),
				)).To(Succeed())
				Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(25, 30)}))
				Expect(read(ctx)).To(Equal([]byte{4}))
			})

		It("Should hide a domain that a delete does not reach", func(ctx SpecContext) {
			var g domain.Group
			commitIn(ctx, openWriter(ctx, 10), 20, nil)
			commitIn(ctx, openWriter(ctx, 30), 40, &g)
			Expect(db.Delete(
				ctx,
				seconds(10, 20),
				fixedOffset(0),
				fixedOffset(1),
			)).To(Succeed())
			Expect(visible(ctx)).To(BeEmpty())
			g.Publish()
			Expect(visible(ctx)).To(Equal([]telem.TimeRange{seconds(30, 40)}))
		})
	})

	Describe("History changes", func() {
		It("Should report a domain added before the last once the group publishes",
			func(ctx SpecContext) {
				var g domain.Group
				commitIn(ctx, openWriter(ctx, 30), 40, nil)
				commitIn(ctx, openWriter(ctx, 10), 20, &g)
				Expect(changes).To(BeEmpty())
				g.Publish()
				Expect(changes).To(Equal([]telem.TimeRange{seconds(10, 20)}))
				g.Publish()
				Expect(changes).To(HaveLen(1))
			})

		It(
			"Should report a commit to a domain before the last once the group publishes",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 15, nil)
				commitIn(ctx, openWriter(ctx, 30), 40, nil)
				commitIn(ctx, w, 20, &g)
				Expect(changes).To(BeEmpty())
				g.Publish()
				Expect(changes).To(Equal([]telem.TimeRange{seconds(15, 20)}))
			},
		)
		It("Should report the change of an unpublished commit once its writer closes",
			func(ctx SpecContext) {
				var g domain.Group
				commitIn(ctx, openWriter(ctx, 30), 40, nil)
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 20, &g)
				Expect(w.Close()).To(Succeed())
				Expect(changes).To(Equal([]telem.TimeRange{seconds(10, 20)}))
				g.Publish()
				Expect(changes).To(HaveLen(1))
			})

		It(
			"Should report an unpublished change once its writer commits to another domain",
			func(ctx SpecContext) {
				var g domain.Group
				commitIn(ctx, openWriter(ctx, 50), 60, nil)
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 20, &g)
				Expect(w.Finish(ctx)).To(Succeed())
				Expect(w.SetStart(30 * telem.SecondTS)).To(Succeed())
				commitIn(ctx, w, 40, &g)
				Expect(changes).To(Equal([]telem.TimeRange{seconds(10, 20)}))
				g.Publish()
				Expect(changes).To(Equal([]telem.TimeRange{
					seconds(10, 20),
					seconds(30, 40),
				}))
			},
		)

		It(
			"Should report an unpublished change with a later commit outside the group",
			func(ctx SpecContext) {
				var g domain.Group
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 15, nil)
				commitIn(ctx, openWriter(ctx, 30), 40, nil)
				commitIn(ctx, w, 20, &g)
				commitIn(ctx, w, 25, nil)
				Expect(changes).To(Equal([]telem.TimeRange{seconds(15, 25)}))
				g.Publish()
				Expect(changes).To(HaveLen(1))
			},
		)

		It("Should report the changes of two unpublished commits as one",
			func(ctx SpecContext) {
				var g domain.Group
				commitIn(ctx, openWriter(ctx, 30), 40, nil)
				w := openWriter(ctx, 10)
				commitIn(ctx, w, 15, &g)
				commitIn(ctx, w, 20, &g)
				g.Publish()
				Expect(changes).To(Equal([]telem.TimeRange{seconds(10, 20)}))
			})

		It("Should report the change of an unpublished commit that a delete reaches",
			func(ctx SpecContext) {
				var g domain.Group
				commitIn(ctx, openWriter(ctx, 30), 40, nil)
				commitIn(ctx, openWriter(ctx, 10), 20, &g)
				Expect(db.Delete(
					ctx,
					seconds(10, 20),
					fixedOffset(0),
					fixedOffset(1),
				)).To(Succeed())
				Expect(changes).To(Equal([]telem.TimeRange{
					seconds(10, 20),
					seconds(10, 20),
				}))
				g.Publish()
				Expect(changes).To(HaveLen(2))
			})
	})
})
