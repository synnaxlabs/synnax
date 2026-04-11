// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

// indexedEntry is the test entry type used by the index suite. It carries two
// indexable fields (Name, Category) and one orderable field (Score) so the
// suite can exercise Lookup, Sorted, and OrderBy against realistic data shapes.
type indexedEntry struct {
	ID       int32
	Name     string
	Category string
	Score    int64
	Flag     bool
}

func (e indexedEntry) GorpKey() int32    { return e.ID }
func (e indexedEntry) SetOptions() []any { return nil }

var _ = Describe("Index", func() {
	var (
		idxDB *gorp.DB
	)
	BeforeEach(func() {
		idxDB = gorp.Wrap(memkv.New())
	})
	AfterEach(func() {
		Expect(idxDB.Close()).To(Succeed())
	})

	Describe("Lookup", func() {
		Describe("Population at OpenTable", func() {
			It("Should populate the index from existing entries", func(ctx SpecContext) {
				seed := []indexedEntry{
					{ID: 1, Name: "alpha", Category: "x"},
					{ID: 2, Name: "beta", Category: "y"},
					{ID: 3, Name: "alpha", Category: "z"},
				}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

				nameIdx := gorp.NewLookup[int32, indexedEntry, string](
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table, err := gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				})
				Expect(err).ToNot(HaveOccurred())
				defer func() { Expect(table.Close()).To(Succeed()) }()

				keys := nameIdx.Get("alpha")
				Expect(keys).To(ConsistOf(int32(1), int32(3)))
				Expect(nameIdx.Get("beta")).To(ConsistOf(int32(2)))
				Expect(nameIdx.Get("missing")).To(BeEmpty())
			})

			It("Should populate a bool-typed Lookup via the bool-bucket backing", func(ctx SpecContext) {
				seed := []indexedEntry{
					{ID: 1, Flag: true},
					{ID: 2, Flag: false},
					{ID: 3, Flag: true},
					{ID: 4, Flag: false},
				}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

				flagIdx := gorp.NewLookup[int32, indexedEntry, bool](
					"flag", func(e *indexedEntry) bool { return e.Flag },
				)
				table, err := gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{flagIdx},
				})
				Expect(err).ToNot(HaveOccurred())
				defer func() { Expect(table.Close()).To(Succeed()) }()

				Expect(flagIdx.Get(true)).To(ConsistOf(int32(1), int32(3)))
				Expect(flagIdx.Get(false)).To(ConsistOf(int32(2), int32(4)))
			})
		})

		Describe("Observer maintenance", func() {
			var (
				table   *gorp.Table[int32, indexedEntry]
				nameIdx *gorp.Lookup[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookup[int32, indexedEntry, string](
					"name", func(e *indexedEntry) string { return e.Name },
				)
				var err error
				table, err = gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				})
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() { Expect(table.Close()).To(Succeed()) })

			It("Should index newly created entries", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 10, Name: "gamma"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get("gamma")).To(ConsistOf(int32(10)))
			})

			It("Should reindex an entry when its indexed field changes", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 11, Name: "delta"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get("delta")).To(ConsistOf(int32(11)))

				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 11, Name: "epsilon"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get("delta")).To(BeEmpty())
				Expect(nameIdx.Get("epsilon")).To(ConsistOf(int32(11)))
			})

			It("Should drop deleted entries from the index", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 12, Name: "zeta"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get("zeta")).To(ConsistOf(int32(12)))

				Expect(gorp.NewDelete[int32, indexedEntry]().
					WhereKeys(12).Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get("zeta")).To(BeEmpty())
			})

			It("Should leave the index unchanged when a set does not modify the indexed field", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 13, Name: "eta", Score: 5}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get("eta")).To(ConsistOf(int32(13)))

				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 13, Name: "eta", Score: 10}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get("eta")).To(ConsistOf(int32(13)))
			})
		})

		Describe("Filter integration with Retrieve", func() {
			var (
				table   *gorp.Table[int32, indexedEntry]
				nameIdx *gorp.Lookup[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookup[int32, indexedEntry, string](
					"name", func(e *indexedEntry) string { return e.Name },
				)
				var err error
				table, err = gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				})
				Expect(err).ToNot(HaveOccurred())
				seed := []indexedEntry{
					{ID: 1, Name: "a"},
					{ID: 2, Name: "b"},
					{ID: 3, Name: "a"},
					{ID: 4, Name: "c"},
					{ID: 5, Name: "b"},
				}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
			})
			AfterEach(func() { Expect(table.Close()).To(Succeed()) })

			It("Should return matching entries via Where", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("a")).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(1), int32(3)))
			})

			It("Should return matching entries for multiple values", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("a", "b")).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(1), int32(2), int32(3), int32(5)))
			})

			It("Should return an empty result when no values match", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("missing")).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(res).To(BeEmpty())
			})
		})

		Describe("Concurrency", func() {
			It("Should permit concurrent Filter calls while the observer processes writes", func(ctx SpecContext) {
				nameIdx := gorp.NewLookup[int32, indexedEntry, string](
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table, err := gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				})
				Expect(err).ToNot(HaveOccurred())
				defer func() { Expect(table.Close()).To(Succeed()) }()

				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					for i := int32(0); i < 100; i++ {
						_ = gorp.NewCreate[int32, indexedEntry]().
							Entry(&indexedEntry{ID: i, Name: "shared"}).
							Exec(ctx, idxDB)
					}
				}()
				go func() {
					defer wg.Done()
					for i := 0; i < 200; i++ {
						_ = nameIdx.Get("shared")
					}
				}()
				wg.Wait()
				Eventually(func() int { return len(nameIdx.Get("shared")) }).
					Should(Equal(100))
			})
		})
	})

	Describe("Sorted", func() {
		Describe("Population and exact-match Filter", func() {
			It("Should populate a Sorted index and serve exact-match lookups", func(ctx SpecContext) {
				seed := []indexedEntry{
					{ID: 1, Score: 30},
					{ID: 2, Score: 10},
					{ID: 3, Score: 20},
					{ID: 4, Score: 20},
				}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
				scoreIdx := gorp.NewSorted[int32, indexedEntry, int64](
					"score", func(e *indexedEntry) int64 { return e.Score },
				)
				table, err := gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{scoreIdx},
				})
				Expect(err).ToNot(HaveOccurred())
				defer func() { Expect(table.Close()).To(Succeed()) }()

				Expect(scoreIdx.Get(int64(20))).To(ConsistOf(int32(3), int32(4)))
				Expect(scoreIdx.Get(int64(10))).To(ConsistOf(int32(2)))
				Expect(scoreIdx.Get(int64(99))).To(BeEmpty())
			})
		})

		Describe("OrderBy pagination", func() {
			var (
				table    *gorp.Table[int32, indexedEntry]
				scoreIdx *gorp.Sorted[int32, indexedEntry, int64]
			)
			BeforeEach(func(ctx SpecContext) {
				scoreIdx = gorp.NewSorted[int32, indexedEntry, int64](
					"score", func(e *indexedEntry) int64 { return e.Score },
				)
				var err error
				table, err = gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{scoreIdx},
				})
				Expect(err).ToNot(HaveOccurred())
				seed := make([]indexedEntry, 20)
				for i := range 20 {
					seed[i] = indexedEntry{ID: int32(i), Score: int64(i * 10)}
				}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
			})
			AfterEach(func() { Expect(table.Close()).To(Succeed()) })

			It("Should walk ascending order with a limit", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.Asc)).
					Limit(5).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(res).To(HaveLen(5))
				scores := make([]int64, len(res))
				for i, e := range res {
					scores[i] = e.Score
				}
				Expect(scores).To(Equal([]int64{0, 10, 20, 30, 40}))
			})

			It("Should walk descending order with a limit", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.Desc)).
					Limit(3).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(res).To(HaveLen(3))
				scores := make([]int64, len(res))
				for i, e := range res {
					scores[i] = e.Score
				}
				Expect(scores).To(Equal([]int64{190, 180, 170}))
			})

			It("Should resume pagination via After", func(ctx SpecContext) {
				var page1 []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.Asc)).
					Limit(5).
					Entries(&page1).Exec(ctx, idxDB)).To(Succeed())
				Expect(page1).To(HaveLen(5))
				lastScore := page1[len(page1)-1].Score

				var page2 []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.Asc)).
					After(lastScore).
					Limit(5).
					Entries(&page2).Exec(ctx, idxDB)).To(Succeed())
				Expect(page2).To(HaveLen(5))
				scores := make([]int64, len(page2))
				for i, e := range page2 {
					scores[i] = e.Score
				}
				Expect(scores).To(Equal([]int64{50, 60, 70, 80, 90}))
			})

			It("Should compose with a Where post-filter", func(ctx SpecContext) {
				var res []indexedEntry
				aboveFifty := gorp.Match[int32, indexedEntry](
					func(_ gorp.Context, e *indexedEntry) (bool, error) {
						return e.Score > 50, nil
					},
				)
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.Asc)).
					Limit(8).
					Where(aboveFifty).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				// OrderBy walks the first 8 entries (scores 0..70); the
				// post-filter keeps only those with Score > 50.
				scores := make([]int64, len(res))
				for i, e := range res {
					scores[i] = e.Score
				}
				Expect(scores).To(Equal([]int64{60, 70}))
			})
		})
	})
})
