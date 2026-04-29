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
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
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
				table := MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				}))
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
				table := MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{flagIdx},
				}))
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
				table = MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				}))
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
					Where(gorp.MatchKeys[int32, indexedEntry](12)).Exec(ctx, idxDB)).To(Succeed())
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
				table = MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				}))
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

		Describe("Composition with And/Or", func() {
			var (
				table       *gorp.Table[int32, indexedEntry]
				nameIdx     *gorp.Lookup[int32, indexedEntry, string]
				categoryIdx *gorp.Lookup[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookup[int32, indexedEntry, string](
					"name", func(e *indexedEntry) string { return e.Name },
				)
				categoryIdx = gorp.NewLookup[int32, indexedEntry, string](
					"category", func(e *indexedEntry) string { return e.Category },
				)
				table = MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB: idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{
						nameIdx, categoryIdx,
					},
				}))
				seed := []indexedEntry{
					{ID: 1, Name: "a", Category: "x"},
					{ID: 2, Name: "a", Category: "y"},
					{ID: 3, Name: "b", Category: "x"},
					{ID: 4, Name: "b", Category: "y"},
					{ID: 5, Name: "c", Category: "x"},
				}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
			})
			AfterEach(func() { Expect(table.Close()).To(Succeed()) })

			It("Should intersect two indexed filters via And", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(gorp.And(nameIdx.Filter("a"), categoryIdx.Filter("x"))).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].ID).To(Equal(int32(1)))
			})

			It("Should union two indexed filters via Or", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(gorp.Or(nameIdx.Filter("c"), categoryIdx.Filter("y"))).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(2), int32(4), int32(5)))
			})

			It("Should preserve membership through nested And composition", func(ctx SpecContext) {
				// Regression: previously, And(A, B) returned a Filter with
				// Keys set but membership nil, so a subsequent And(prev, C)
				// silently dropped every key (containsKey on prev always
				// returned false) and returned an empty result.
				inner := gorp.And(nameIdx.Filter("a", "b"), categoryIdx.Filter("x", "y"))
				outer := gorp.And(inner, nameIdx.Filter("a"))
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(outer).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(1), int32(2)))
			})

			It("Should preserve membership through nested Or composition", func(ctx SpecContext) {
				// Same regression as above but for Or: an Or-result with
				// membership nil cannot be merged with another indexed
				// filter via WhereKeys intersection or further composition.
				inner := gorp.Or(nameIdx.Filter("a"), nameIdx.Filter("c"))
				outer := gorp.And(inner, categoryIdx.Filter("x"))
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(outer).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(1), int32(5)))
			})
		})

		Describe("Concurrency", func() {
			It("Should permit concurrent Filter calls while the observer processes writes", func(ctx SpecContext) {
				nameIdx := gorp.NewLookup[int32, indexedEntry, string](
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table := MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				}))
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

		// Tx delta visibility exercises the per-tx index overlay: a
		// Retrieve via idx.Filter(...) inside an open write tx must
		// see mutations staged earlier in the same tx. Without the
		// overlay, the Filter would only see committed index state
		// (the observer fires on commit) and would return stale
		// results inside the tx.
		//
		// Each case opens a tx, performs a write through a
		// table-bound query (which wires the writer through
		// wrapTableWriter so the index observer sees the stage call),
		// and then retrieves via the same tx. Rollback cases close
		// the tx without committing and assert the global index was
		// never touched.
		Describe("Tx delta visibility", func() {
			var (
				table   *gorp.Table[int32, indexedEntry]
				nameIdx *gorp.Lookup[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookup[int32, indexedEntry, string](
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table = MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
				}))
				seed := []indexedEntry{
					{ID: 1, Name: "alpha"},
					{ID: 2, Name: "beta"},
				}
				Expect(table.NewCreate().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
			})
			AfterEach(func() { Expect(table.Close()).To(Succeed()) })

			It("Should see an insert staged in the same tx", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 10, Name: "alpha"}).
					Exec(ctx, tx)).To(Succeed())

				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("alpha")).
					Entries(&res).Exec(ctx, tx)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(1), int32(10)))
			})

			It("Should reflect an update that moves a key to a different value", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 1, Name: "zeta"}).
					Exec(ctx, tx)).To(Succeed())

				var oldMatches []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("alpha")).
					Entries(&oldMatches).Exec(ctx, tx)).To(Succeed())
				Expect(oldMatches).To(BeEmpty())

				var newMatches []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("zeta")).
					Entries(&newMatches).Exec(ctx, tx)).To(Succeed())
				Expect(newMatches).To(HaveLen(1))
				Expect(newMatches[0].ID).To(Equal(int32(1)))
			})

			It("Should exclude an entry deleted in the same tx", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				Expect(table.NewDelete().
					Where(gorp.MatchKeys[int32, indexedEntry](1)).Exec(ctx, tx)).To(Succeed())

				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("alpha")).
					Entries(&res).Exec(ctx, tx)).To(Succeed())
				Expect(res).To(BeEmpty())
			})

			It("Should union staged and committed matches for multi-value filters", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 20, Name: "gamma"}).
					Exec(ctx, tx)).To(Succeed())

				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("beta", "gamma")).
					Entries(&res).Exec(ctx, tx)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(2), int32(20)))
			})

			It("Should isolate staged writes to the owning tx", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 30, Name: "alpha"}).
					Exec(ctx, tx)).To(Succeed())

				// Another Retrieve against the bare DB (a different
				// txIdentity) must not see the staged insert.
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("alpha")).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(1)))
			})

			It("Should drop the delta on rollback without touching the global index", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 40, Name: "alpha"}).
					Exec(ctx, tx)).To(Succeed())
				// Close without commit: cleanups fire via *tx.Close,
				// dropping the delta. The global index should not
				// carry the rolled-back write.
				Expect(tx.Close()).To(Succeed())

				Expect(nameIdx.Get("alpha")).To(ConsistOf(int32(1)))

				// And a fresh retrieve via the bare DB should see
				// committed-only state.
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("alpha")).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].ID).To(Equal(int32(1)))
			})

			It("Should see the staged write on committed global index after commit", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 50, Name: "alpha"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				// Committed observer should have fired, updating the
				// global index.
				Expect(nameIdx.Get("alpha")).To(ConsistOf(int32(1), int32(50)))
			})

			It("Should support set-then-delete in the same tx", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 60, Name: "alpha"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(table.NewDelete().
					Where(gorp.MatchKeys[int32, indexedEntry](60)).Exec(ctx, tx)).To(Succeed())

				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("alpha")).
					Entries(&res).Exec(ctx, tx)).To(Succeed())
				ids := make([]int32, len(res))
				for i, e := range res {
					ids[i] = e.ID
				}
				Expect(ids).To(ConsistOf(int32(1)))
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
				table := MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{scoreIdx},
				}))
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
				table = MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      idxDB,
					Indexes: []gorp.Index[int32, indexedEntry]{scoreIdx},
				}))
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

			It("Should resume pagination via After on the SortedQuery", func(ctx SpecContext) {
				var page1 []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.Asc)).
					Limit(5).
					Entries(&page1).Exec(ctx, idxDB)).To(Succeed())
				Expect(page1).To(HaveLen(5))
				lastScore := page1[len(page1)-1].Score

				var page2 []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.Asc).After(lastScore)).
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

	Describe("Not with index-backed filters", func() {
		It("Should negate an index-backed filter", func(ctx SpecContext) {
			db := gorp.Wrap(memkv.New())
			defer func() { Expect(db.Close()).To(Succeed()) }()
			nameIdx := gorp.NewLookup[int32, indexedEntry, string](
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table := MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
				DB:      db,
				Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
			}))
			defer func() { Expect(table.Close()).To(Succeed()) }()
			seed := []indexedEntry{
				{ID: 1, Name: "a"},
				{ID: 2, Name: "b"},
				{ID: 3, Name: "a"},
				{ID: 4, Name: "c"},
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&seed).Exec(ctx, db)).To(Succeed())

			var res []indexedEntry
			Expect(table.NewRetrieve().
				Where(gorp.Not(nameIdx.Filter("a"))).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			ids := make([]int32, len(res))
			for i, e := range res {
				ids[i] = e.ID
			}
			Expect(ids).To(ConsistOf(int32(2), int32(4)))
		})

		It("Should negate And(indexed, eval) without dropping the index constraint", func(ctx SpecContext) {
			db := gorp.Wrap(memkv.New())
			defer func() { Expect(db.Close()).To(Succeed()) }()
			nameIdx := gorp.NewLookup[int32, indexedEntry, string](
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table := MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
				DB:      db,
				Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
			}))
			defer func() { Expect(table.Close()).To(Succeed()) }()
			seed := []indexedEntry{
				{ID: 1, Name: "a", Score: 60},
				{ID: 2, Name: "b", Score: 60},
				{ID: 3, Name: "a", Score: 10},
				{ID: 4, Name: "b", Score: 10},
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&seed).Exec(ctx, db)).To(Succeed())

			aboveFifty := gorp.Match[int32, indexedEntry](func(_ gorp.Context, e *indexedEntry) (bool, error) {
				return e.Score > 50, nil
			})
			// Not(And(name="a", score>50)) should exclude entries that
			// are BOTH name="a" AND score>50. Only ID=1 satisfies both.
			// IDs 2, 3, 4 should be returned.
			var res []indexedEntry
			Expect(table.NewRetrieve().
				Where(gorp.Not(gorp.And(nameIdx.Filter("a"), aboveFifty))).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			ids := make([]int32, len(res))
			for i, e := range res {
				ids[i] = e.ID
			}
			Expect(ids).To(ConsistOf(int32(2), int32(3), int32(4)))
		})
	})

	Describe("Where with OrderBy", func() {
		It("Should apply an index-backed Where filter with OrderBy", func(ctx SpecContext) {
			db := gorp.Wrap(memkv.New())
			defer func() { Expect(db.Close()).To(Succeed()) }()
			nameIdx := gorp.NewLookup[int32, indexedEntry, string](
				"name", func(e *indexedEntry) string { return e.Name },
			)
			scoreIdx := gorp.NewSorted[int32, indexedEntry, int64](
				"score", func(e *indexedEntry) int64 { return e.Score },
			)
			table := MustSucceed(gorp.OpenTable[int32, indexedEntry](ctx, gorp.TableConfig[int32, indexedEntry]{
				DB:      db,
				Indexes: []gorp.Index[int32, indexedEntry]{nameIdx, scoreIdx},
			}))
			defer func() { Expect(table.Close()).To(Succeed()) }()
			seed := []indexedEntry{
				{ID: 1, Name: "a", Score: 50},
				{ID: 2, Name: "b", Score: 10},
				{ID: 3, Name: "a", Score: 30},
				{ID: 4, Name: "b", Score: 40},
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&seed).Exec(ctx, db)).To(Succeed())

			var res []indexedEntry
			Expect(table.NewRetrieve().
				Where(nameIdx.Filter("a")).
				OrderBy(scoreIdx.Ordered(gorp.Asc)).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(2))
			for _, e := range res {
				Expect(e.Name).To(Equal("a"))
			}
		})
	})

	Describe("WithIndexObservable", func() {
		// These tests prove that the index stays consistent even when the
		// observer that would otherwise feed it never fires. This is the
		// configuration used in multi-node Aspen deployments where the
		// observer is filtered to remote-only writes via
		// IgnoreHostLeaseholder, leaving local writes to be applied via
		// the per-tx delta flush (real tx) or the inline path (DB-as-tx).
		var (
			noopDB  *gorp.DB
			nameIdx *gorp.Lookup[int32, indexedEntry, string]
			table   *gorp.Table[int32, indexedEntry]
		)
		BeforeEach(func(ctx SpecContext) {
			noopDB = gorp.Wrap(
				memkv.New(),
				gorp.WithIndexObservable(observe.Noop[kv.TxReader]{}),
			)
			nameIdx = gorp.NewLookup[int32, indexedEntry, string](
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[int32, indexedEntry]{
				DB:      noopDB,
				Indexes: []gorp.Index[int32, indexedEntry]{nameIdx},
			}))
		})
		AfterEach(func() {
			Expect(table.Close()).To(Succeed())
			Expect(noopDB.Close()).To(Succeed())
		})

		It("Should update the index via per-tx delta flush on commit", func(ctx SpecContext) {
			tx := noopDB.OpenTx()
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 1, Name: "alpha"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(nameIdx.Get("alpha")).To(ConsistOf(int32(1)))
		})

		It("Should leave the index untouched when the tx is closed without commit", func(ctx SpecContext) {
			tx := noopDB.OpenTx()
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 2, Name: "beta"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(nameIdx.Get("beta")).To(BeEmpty())
		})

		It("Should update the index inline for DB-as-tx writes", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 3, Name: "gamma"}).
				Exec(ctx, noopDB)).To(Succeed())

			Expect(nameIdx.Get("gamma")).To(ConsistOf(int32(3)))
		})

		It("Should remove deleted entries from the index on commit", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 4, Name: "delta"}).
				Exec(ctx, noopDB)).To(Succeed())
			Expect(nameIdx.Get("delta")).To(ConsistOf(int32(4)))

			tx := noopDB.OpenTx()
			Expect(table.NewDelete().Where(gorp.MatchKeys[int32, indexedEntry](4)).Exec(ctx, tx)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(nameIdx.Get("delta")).To(BeEmpty())
		})

		It("Should preserve committed entries when a delete is rolled back", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 5, Name: "epsilon"}).
				Exec(ctx, noopDB)).To(Succeed())

			tx := noopDB.OpenTx()
			Expect(table.NewDelete().Where(gorp.MatchKeys[int32, indexedEntry](5)).Exec(ctx, tx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(nameIdx.Get("epsilon")).To(ConsistOf(int32(5)))
		})

		It("Should remove deleted entries inline for DB-as-tx deletes", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 6, Name: "zeta"}).
				Exec(ctx, noopDB)).To(Succeed())
			Expect(nameIdx.Get("zeta")).To(ConsistOf(int32(6)))

			Expect(table.NewDelete().Where(gorp.MatchKeys[int32, indexedEntry](6)).Exec(ctx, noopDB)).To(Succeed())
			Expect(nameIdx.Get("zeta")).To(BeEmpty())
		})
	})
})
