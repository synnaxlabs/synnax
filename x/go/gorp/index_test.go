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
	"context"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
)

// indexedEntry is the test entry type used by the index suite. It carries two
// indexable fields (Name, Category) and one orderable field (Score) so the
// suite can exercise LookupIndex, SortedIndex, and OrderBy against realistic data shapes.
type indexedEntry struct {
	ID       int32
	Name     string
	Category string
	Score    int64
	Flag     bool
}

func (e indexedEntry) GorpKey() int32    { return e.ID }
func (e indexedEntry) SetOptions() []any { return nil }

// relationEntry is a composite-string-keyed test entry. Its gorp key is
// the "<from>-><to>" string, mimicking the shape of an ontology
// relationship key.
type relationEntry struct {
	From  string
	To    string
	Label string
}

func (e relationEntry) GorpKey() string   { return e.From + "->" + e.To }
func (e relationEntry) SetOptions() []any { return nil }

func idsOf(res []indexedEntry) []int32 {
	out := make([]int32, len(res))
	for i, e := range res {
		out[i] = e.ID
	}
	return out
}

func scoresOf(res []indexedEntry) []int64 {
	out := make([]int64, len(res))
	for i, e := range res {
		out[i] = e.Score
	}
	return out
}

func fromsOf(res []relationEntry) []string {
	out := make([]string, len(res))
	for i, e := range res {
		out[i] = e.From
	}
	return out
}

func openIndexedTable[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	db *gorp.DB,
	indexes ...gorp.Index[K, E],
) *gorp.Table[K, E] {
	return MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[K, E]{
		DB:      db,
		Indexes: indexes,
	}))
}

// failOnceDB wraps a kv.DB and forces the first OpenIterator call routed
// directly to the DB to fail, allowing subsequent calls through to the
// underlying DB. Calls routed through OpenTx().OpenIterator(...) are not
// intercepted because tx OpenIterator dispatches off the underlying tx
// rather than the wrapper. This is exactly the failure shape needed:
// migrations open iterators via tx (so they succeed), populate opens
// iterators via DB-as-Tx (so this trips them), and the post-failure
// sequential-scan fallback in Retrieve also goes via DB-as-Tx and lands
// on the second-call branch (success).
type failOnceDB struct {
	kv.DB
	calls atomic.Int32
	err   error
}

func (f *failOnceDB) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	if f.calls.Add(1) == 1 {
		return nil, f.err
	}
	return f.DB.OpenIterator(opts)
}

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

				nameIdx := gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table := openIndexedTable(ctx, idxDB, nameIdx)
				defer func() { Expect(table.Close()).To(Succeed()) }()

				keys := MustSucceed(nameIdx.Get(nil, "alpha"))
				Expect(keys).To(ConsistOf(int32(1), int32(3)))
				Expect(nameIdx.Get(nil, "beta")).To(ConsistOf(int32(2)))
				Expect(nameIdx.Get(nil, "missing")).To(BeEmpty())
			})

			It("Should populate a bool-typed Lookup", func(ctx SpecContext) {
				seed := []indexedEntry{
					{ID: 1, Flag: true},
					{ID: 2, Flag: false},
					{ID: 3, Flag: true},
					{ID: 4, Flag: false},
				}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

				flagIdx := gorp.NewLookupIndex(
					"flag", func(e *indexedEntry) bool { return e.Flag },
				)
				table := openIndexedTable(ctx, idxDB, flagIdx)
				defer func() { Expect(table.Close()).To(Succeed()) }()

				Expect(flagIdx.Get(nil, true)).To(ConsistOf(int32(1), int32(3)))
				Expect(flagIdx.Get(nil, false)).To(ConsistOf(int32(2), int32(4)))
			})
		})

		Describe("Observer maintenance", func() {
			var (
				table   *gorp.Table[int32, indexedEntry]
				nameIdx *gorp.LookupIndex[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table = openIndexedTable(ctx, idxDB, nameIdx)
			})
			AfterEach(func() { Expect(table.Close()).To(Succeed()) })

			It("Should index newly created entries", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 10, Name: "gamma"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(nil, "gamma")).To(ConsistOf(int32(10)))
			})

			It("Should reindex an entry when its indexed field changes", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 11, Name: "delta"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(nil, "delta")).To(ConsistOf(int32(11)))

				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 11, Name: "epsilon"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(nil, "delta")).To(BeEmpty())
				Expect(nameIdx.Get(nil, "epsilon")).To(ConsistOf(int32(11)))
			})

			It("Should drop deleted entries from the index", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 12, Name: "zeta"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(nil, "zeta")).To(ConsistOf(int32(12)))

				Expect(gorp.NewDelete[int32, indexedEntry]().
					Where(gorp.MatchKeys[int32, indexedEntry](12)).Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(nil, "zeta")).To(BeEmpty())
			})

			It("Should leave the index unchanged when a set does not modify the indexed field", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 13, Name: "eta", Score: 5}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(nil, "eta")).To(ConsistOf(int32(13)))

				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 13, Name: "eta", Score: 10}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(nil, "eta")).To(ConsistOf(int32(13)))
			})
		})

		Describe("Filter integration with Retrieve", func() {
			var (
				table   *gorp.Table[int32, indexedEntry]
				nameIdx *gorp.LookupIndex[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table = openIndexedTable(ctx, idxDB, nameIdx)
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
				Expect(idsOf(res)).To(ConsistOf(int32(1), int32(3)))
			})

			It("Should return matching entries for multiple values", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("a", "b")).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(idsOf(res)).To(ConsistOf(int32(1), int32(2), int32(3), int32(5)))
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
				nameIdx     *gorp.LookupIndex[int32, indexedEntry, string]
				categoryIdx *gorp.LookupIndex[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				categoryIdx = gorp.NewLookupIndex(
					"category", func(e *indexedEntry) string { return e.Category },
				)
				table = openIndexedTable(ctx, idxDB, nameIdx, categoryIdx)
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
				Expect(idsOf(res)).To(ConsistOf(int32(1)))
			})

			It("Should union two indexed filters via Or", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(gorp.Or(nameIdx.Filter("c"), categoryIdx.Filter("y"))).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(idsOf(res)).To(ConsistOf(int32(2), int32(4), int32(5)))
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
				Expect(idsOf(res)).To(ConsistOf(int32(1), int32(2)))
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
				Expect(idsOf(res)).To(ConsistOf(int32(1), int32(5)))
			})
		})

		Describe("Concurrency", func() {
			It("Should permit concurrent Filter calls while the observer processes writes", func(ctx SpecContext) {
				nameIdx := gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table := openIndexedTable(ctx, idxDB, nameIdx)
				defer func() { Expect(table.Close()).To(Succeed()) }()

				var wg sync.WaitGroup
				wg.Go(func() {
					for i := range int32(100) {
						_ = gorp.NewCreate[int32, indexedEntry]().
							Entry(&indexedEntry{ID: i, Name: "shared"}).
							Exec(ctx, idxDB)
					}
				})
				wg.Go(func() {
					for range 200 {
						_, _ = nameIdx.Get(nil, "shared")
					}
				})
				wg.Wait()
				Eventually(func() int {
					keys, _ := nameIdx.Get(nil, "shared")
					return len(keys)
				}).Should(Equal(100))
			})

			It("Should be safe to Exec a shared Or filter from multiple goroutines", func(ctx SpecContext) {
				// Or(indexed, indexed) builds a resolver that previously
				// mutated a shared filters slice in place; concurrent
				// Execs raced on the per-child keys / membership fields.
				// The fix materializes children per Exec, so the race
				// detector should stay quiet here.
				nameIdx := gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				categoryIdx := gorp.NewLookupIndex(
					"category", func(e *indexedEntry) string { return e.Category },
				)
				table := openIndexedTable(ctx, idxDB, nameIdx, categoryIdx)
				defer func() { Expect(table.Close()).To(Succeed()) }()

				Expect(gorp.NewCreate[int32, indexedEntry]().Entries(&[]indexedEntry{
					{ID: 1, Name: "a", Category: "x"},
					{ID: 2, Name: "b", Category: "y"},
					{ID: 3, Name: "a", Category: "z"},
				}).Exec(ctx, idxDB)).To(Succeed())

				shared := gorp.Or(nameIdx.Filter("a"), categoryIdx.Filter("y"))
				var wg sync.WaitGroup
				for range 16 {
					wg.Go(func() {
						for range 32 {
							var res []indexedEntry
							Expect(table.NewRetrieve().
								Where(shared).
								Entries(&res).Exec(ctx, idxDB)).To(Succeed())
							Expect(idsOf(res)).To(ConsistOf(int32(1), int32(2), int32(3)))
						}
					})
				}
				wg.Wait()
			})

			It("Should be safe to Exec a shared Not filter from multiple goroutines", func(ctx SpecContext) {
				// Not(indexed) previously wrote back into the captured
				// child filter at resolve time; concurrent Execs raced
				// on those writes. The fix uses a local materialized
				// copy per resolve call.
				nameIdx := gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table := openIndexedTable(ctx, idxDB, nameIdx)
				defer func() { Expect(table.Close()).To(Succeed()) }()

				Expect(gorp.NewCreate[int32, indexedEntry]().Entries(&[]indexedEntry{
					{ID: 1, Name: "a"},
					{ID: 2, Name: "b"},
					{ID: 3, Name: "a"},
				}).Exec(ctx, idxDB)).To(Succeed())

				shared := gorp.Not(nameIdx.Filter("a"))
				var wg sync.WaitGroup
				for range 16 {
					wg.Go(func() {
						for range 32 {
							var res []indexedEntry
							Expect(table.NewRetrieve().
								Where(shared).
								Entries(&res).Exec(ctx, idxDB)).To(Succeed())
							Expect(idsOf(res)).To(ConsistOf(int32(2)))
						}
					})
				}
				wg.Wait()
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
				nameIdx *gorp.LookupIndex[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table = openIndexedTable(ctx, idxDB, nameIdx)
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
				Expect(idsOf(res)).To(ConsistOf(int32(1), int32(10)))
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
				Expect(idsOf(res)).To(ConsistOf(int32(2), int32(20)))
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
				Expect(idsOf(res)).To(ConsistOf(int32(1)))
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

				Expect(nameIdx.Get(nil, "alpha")).To(ConsistOf(int32(1)))

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
				Expect(nameIdx.Get(nil, "alpha")).To(ConsistOf(int32(1), int32(50)))
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
				Expect(idsOf(res)).To(ConsistOf(int32(1)))
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
				scoreIdx := gorp.NewSortedIndex(
					"score", func(e *indexedEntry) int64 { return e.Score },
				)
				table := openIndexedTable(ctx, idxDB, scoreIdx)
				defer func() { Expect(table.Close()).To(Succeed()) }()

				Expect(scoreIdx.Get(nil, int64(20))).To(ConsistOf(int32(3), int32(4)))
				Expect(scoreIdx.Get(nil, int64(10))).To(ConsistOf(int32(2)))
				Expect(scoreIdx.Get(nil, int64(99))).To(BeEmpty())
			})
		})

		Describe("OrderBy pagination", func() {
			var (
				table    *gorp.Table[int32, indexedEntry]
				scoreIdx *gorp.SortedIndex[int32, indexedEntry, int64]
			)
			BeforeEach(func(ctx SpecContext) {
				scoreIdx = gorp.NewSortedIndex(
					"score", func(e *indexedEntry) int64 { return e.Score },
				)
				table = openIndexedTable(ctx, idxDB, scoreIdx)
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
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
					Limit(5).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(scoresOf(res)).To(Equal([]int64{0, 10, 20, 30, 40}))
			})

			It("Should walk descending order with a limit", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionDesc)).
					Limit(3).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(scoresOf(res)).To(Equal([]int64{190, 180, 170}))
			})

			It("Should resume pagination via After on the SortedQuery", func(ctx SpecContext) {
				var page1 []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
					Limit(5).
					Entries(&page1).Exec(ctx, idxDB)).To(Succeed())
				Expect(page1).To(HaveLen(5))
				lastScore := page1[len(page1)-1].Score

				var page2 []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc).After(lastScore)).
					Limit(5).
					Entries(&page2).Exec(ctx, idxDB)).To(Succeed())
				Expect(scoresOf(page2)).To(Equal([]int64{50, 60, 70, 80, 90}))
			})

			It("Should compose with a Where post-filter", func(ctx SpecContext) {
				var res []indexedEntry
				aboveFifty := gorp.Match(
					func(_ gorp.Context, e *indexedEntry) (bool, error) {
						return e.Score > 50, nil
					},
				)
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
					Limit(8).
					Where(aboveFifty).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				// OrderBy walks the first 8 entries (scores 0..70); the
				// post-filter keeps only those with Score > 50.
				Expect(scoresOf(res)).To(Equal([]int64{60, 70}))
			})

			It("Should run validators on the ordered result set", func(ctx SpecContext) {
				var (
					seen []indexedEntry
					res  []indexedEntry
				)
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
					Limit(3).
					Validate(func(_ gorp.Context, entries []indexedEntry) error {
						seen = append([]indexedEntry(nil), entries...)
						return nil
					}).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(scoresOf(res)).To(Equal([]int64{0, 10, 20}))
				Expect(seen).To(Equal(res))
			})

			It("Should short-circuit Exec with a validator error on the ordered path", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
					Limit(3).
					Validate(func(_ gorp.Context, _ []indexedEntry) error {
						return errors.New("ordered validator rejected query")
					}).
					Entries(&res).Exec(ctx, idxDB)).
					To(MatchError(ContainSubstring("ordered validator rejected query")))
			})
		})
	})

	Describe("Not with index-backed filters", func() {
		It("Should negate an index-backed filter", func(ctx SpecContext) {
			nameIdx := gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table := openIndexedTable(ctx, idxDB, nameIdx)
			defer func() { Expect(table.Close()).To(Succeed()) }()
			seed := []indexedEntry{
				{ID: 1, Name: "a"},
				{ID: 2, Name: "b"},
				{ID: 3, Name: "a"},
				{ID: 4, Name: "c"},
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

			var res []indexedEntry
			Expect(table.NewRetrieve().
				Where(gorp.Not(nameIdx.Filter("a"))).
				Entries(&res).Exec(ctx, idxDB)).To(Succeed())
			Expect(idsOf(res)).To(ConsistOf(int32(2), int32(4)))
		})

		It("Should negate And(indexed, eval) without dropping the index constraint", func(ctx SpecContext) {
			nameIdx := gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table := openIndexedTable(ctx, idxDB, nameIdx)
			defer func() { Expect(table.Close()).To(Succeed()) }()
			seed := []indexedEntry{
				{ID: 1, Name: "a", Score: 60},
				{ID: 2, Name: "b", Score: 60},
				{ID: 3, Name: "a", Score: 10},
				{ID: 4, Name: "b", Score: 10},
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

			aboveFifty := gorp.Match(func(_ gorp.Context, e *indexedEntry) (bool, error) {
				return e.Score > 50, nil
			})
			// Not(And(name="a", score>50)) should exclude entries that
			// are BOTH name="a" AND score>50. Only ID=1 satisfies both.
			// IDs 2, 3, 4 should be returned.
			var res []indexedEntry
			Expect(table.NewRetrieve().
				Where(gorp.Not(gorp.And(nameIdx.Filter("a"), aboveFifty))).
				Entries(&res).Exec(ctx, idxDB)).To(Succeed())
			Expect(idsOf(res)).To(ConsistOf(int32(2), int32(3), int32(4)))
		})
	})

	Describe("Where with OrderBy", func() {
		It("Should apply an index-backed Where filter with OrderBy", func(ctx SpecContext) {
			nameIdx := gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			scoreIdx := gorp.NewSortedIndex(
				"score", func(e *indexedEntry) int64 { return e.Score },
			)
			table := openIndexedTable(ctx, idxDB, nameIdx, scoreIdx)
			defer func() { Expect(table.Close()).To(Succeed()) }()
			seed := []indexedEntry{
				{ID: 1, Name: "a", Score: 50},
				{ID: 2, Name: "b", Score: 10},
				{ID: 3, Name: "a", Score: 30},
				{ID: 4, Name: "b", Score: 40},
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

			var res []indexedEntry
			Expect(table.NewRetrieve().
				Where(nameIdx.Filter("a")).
				OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
				Entries(&res).Exec(ctx, idxDB)).To(Succeed())
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
			nameIdx *gorp.LookupIndex[int32, indexedEntry, string]
			table   *gorp.Table[int32, indexedEntry]
		)
		BeforeEach(func(ctx SpecContext) {
			noopDB = gorp.Wrap(
				memkv.New(),
				gorp.WithIndexObservable(observe.Noop[kv.TxReader]{}),
			)
			nameIdx = gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table = openIndexedTable(ctx, noopDB, nameIdx)
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

			Expect(nameIdx.Get(nil, "alpha")).To(ConsistOf(int32(1)))
		})

		It("Should leave the index untouched when the tx is closed without commit", func(ctx SpecContext) {
			tx := noopDB.OpenTx()
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 2, Name: "beta"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(nameIdx.Get(nil, "beta")).To(BeEmpty())
		})

		It("Should update the index inline for DB-as-tx writes", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 3, Name: "gamma"}).
				Exec(ctx, noopDB)).To(Succeed())

			Expect(nameIdx.Get(nil, "gamma")).To(ConsistOf(int32(3)))
		})

		It("Should remove deleted entries from the index on commit", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 4, Name: "delta"}).
				Exec(ctx, noopDB)).To(Succeed())
			Expect(nameIdx.Get(nil, "delta")).To(ConsistOf(int32(4)))

			tx := noopDB.OpenTx()
			Expect(table.NewDelete().Where(gorp.MatchKeys[int32, indexedEntry](4)).Exec(ctx, tx)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(nameIdx.Get(nil, "delta")).To(BeEmpty())
		})

		It("Should preserve committed entries when a delete is rolled back", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 5, Name: "epsilon"}).
				Exec(ctx, noopDB)).To(Succeed())

			tx := noopDB.OpenTx()
			Expect(table.NewDelete().Where(gorp.MatchKeys[int32, indexedEntry](5)).Exec(ctx, tx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(nameIdx.Get(nil, "epsilon")).To(ConsistOf(int32(5)))
		})

		It("Should remove deleted entries inline for DB-as-tx deletes", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 6, Name: "zeta"}).
				Exec(ctx, noopDB)).To(Succeed())
			Expect(nameIdx.Get(nil, "zeta")).To(ConsistOf(int32(6)))

			Expect(table.NewDelete().Where(gorp.MatchKeys[int32, indexedEntry](6)).Exec(ctx, noopDB)).To(Succeed())
			Expect(nameIdx.Get(nil, "zeta")).To(BeEmpty())
		})
	})
})

var _ = Describe("Composite-string-keyed Lookup", func() {
	var idxDB *gorp.DB
	BeforeEach(func() {
		idxDB = gorp.Wrap(memkv.New())
	})
	AfterEach(func() {
		Expect(idxDB.Close()).To(Succeed())
	})

	Describe("Population at OpenTable", func() {
		It("Should populate the index from existing entries", func(ctx SpecContext) {
			seed := []relationEntry{
				{From: "a", To: "x", Label: "alpha"},
				{From: "b", To: "y", Label: "beta"},
				{From: "c", To: "x", Label: "gamma"},
			}
			Expect(gorp.NewCreate[string, relationEntry]().
				Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

			toIdx := gorp.NewLookupIndex(
				"to", func(e *relationEntry) string { return e.To },
			)
			table := openIndexedTable(ctx, idxDB, toIdx)
			defer func() { Expect(table.Close()).To(Succeed()) }()

			keys := MustSucceed(toIdx.Get(nil, "x"))
			asStrings := []string{string(keys[0]), string(keys[1])}
			Expect(asStrings).To(ConsistOf("a->x", "c->x"))
			Expect(toIdx.Get(nil, "y")).To(HaveLen(1))
			Expect(toIdx.Get(nil, "missing")).To(BeEmpty())
		})
	})

	Describe("Observer maintenance", func() {
		var (
			table *gorp.Table[string, relationEntry]
			toIdx *gorp.LookupIndex[string, relationEntry, string]
		)
		BeforeEach(func(ctx SpecContext) {
			toIdx = gorp.NewLookupIndex(
				"to", func(e *relationEntry) string { return e.To },
			)
			table = openIndexedTable(ctx, idxDB, toIdx)
		})
		AfterEach(func() { Expect(table.Close()).To(Succeed()) })

		It("Should index newly created entries", func(ctx SpecContext) {
			Expect(gorp.NewCreate[string, relationEntry]().
				Entry(&relationEntry{From: "a", To: "x"}).
				Exec(ctx, idxDB)).To(Succeed())
			keys := MustSucceed(toIdx.Get(nil, "x"))
			Expect(keys).To(HaveLen(1))
			Expect(string(keys[0])).To(Equal("a->x"))
		})

		It("Should remove entries from the index on delete", func(ctx SpecContext) {
			e := relationEntry{From: "a", To: "x"}
			Expect(gorp.NewCreate[string, relationEntry]().Entry(&e).Exec(ctx, idxDB)).To(Succeed())
			Expect(toIdx.Get(nil, "x")).To(HaveLen(1))

			Expect(gorp.NewDelete[string, relationEntry]().
				Where(gorp.MatchKeys[string, relationEntry](e.GorpKey())).
				Exec(ctx, idxDB)).To(Succeed())
			Expect(toIdx.Get(nil, "x")).To(BeEmpty())
		})
	})

	Describe("Filter routing", func() {
		It("Should route a Filter call through execKeys instead of execFilter", func(ctx SpecContext) {
			seed := []relationEntry{
				{From: "a", To: "x"},
				{From: "b", To: "y"},
				{From: "c", To: "x"},
				{From: "d", To: "z"},
			}
			Expect(gorp.NewCreate[string, relationEntry]().
				Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

			toIdx := gorp.NewLookupIndex(
				"to", func(e *relationEntry) string { return e.To },
			)
			table := openIndexedTable(ctx, idxDB, toIdx)
			defer func() { Expect(table.Close()).To(Succeed()) }()

			var res []relationEntry
			Expect(gorp.NewRetrieve[string, relationEntry]().
				Where(toIdx.Filter("x")).
				Entries(&res).
				Exec(ctx, idxDB)).To(Succeed())
			Expect(fromsOf(res)).To(ConsistOf("a", "c"))
		})

		It("Should return an empty result when no entries match", func(ctx SpecContext) {
			seed := []relationEntry{{From: "a", To: "x"}}
			Expect(gorp.NewCreate[string, relationEntry]().
				Entries(&seed).Exec(ctx, idxDB)).To(Succeed())

			toIdx := gorp.NewLookupIndex(
				"to", func(e *relationEntry) string { return e.To },
			)
			table := openIndexedTable(ctx, idxDB, toIdx)
			defer func() { Expect(table.Close()).To(Succeed()) }()

			var res []relationEntry
			Expect(gorp.NewRetrieve[string, relationEntry]().
				Where(toIdx.Filter("missing")).
				Entries(&res).
				Exec(ctx, idxDB)).To(Succeed())
			Expect(res).To(BeEmpty())
		})
	})

	// Tx delta visibility mirrors the Lookup test block: inserts,
	// updates, deletes, and rollback exercised against a LookupIndex
	// over a composite-string-keyed entry. This is the analogue that
	// covers the ontology relationship shape directly — the motivating
	// broken case (dagWriter.DefineRelationship cycle check over a byTo
	// index) is reproduced here.
	Describe("Tx delta visibility", func() {
		var (
			table *gorp.Table[string, relationEntry]
			toIdx *gorp.LookupIndex[string, relationEntry, string]
		)
		BeforeEach(func(ctx SpecContext) {
			toIdx = gorp.NewLookupIndex(
				"to", func(e *relationEntry) string { return e.To },
			)
			table = openIndexedTable(ctx, idxDB, toIdx)
			seed := []relationEntry{
				{From: "a", To: "x"},
				{From: "b", To: "y"},
			}
			Expect(table.NewCreate().
				Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
		})
		AfterEach(func() { Expect(table.Close()).To(Succeed()) })

		It("Should see an insert staged in the same tx", func(ctx SpecContext) {
			tx := idxDB.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()

			Expect(table.NewCreate().
				Entry(&relationEntry{From: "c", To: "x"}).
				Exec(ctx, tx)).To(Succeed())

			var res []relationEntry
			Expect(table.NewRetrieve().
				Where(toIdx.Filter("x")).
				Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(fromsOf(res)).To(ConsistOf("a", "c"))
		})

		It("Should exclude an entry deleted in the same tx", func(ctx SpecContext) {
			tx := idxDB.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()

			Expect(table.NewDelete().
				Where(gorp.MatchKeys[string, relationEntry]("a->x")).Exec(ctx, tx)).To(Succeed())

			var res []relationEntry
			Expect(table.NewRetrieve().
				Where(toIdx.Filter("x")).
				Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(BeEmpty())
		})

		It("Should drop the delta on rollback without touching the global index", func(ctx SpecContext) {
			tx := idxDB.OpenTx()
			Expect(table.NewCreate().
				Entry(&relationEntry{From: "z", To: "x"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			keys := MustSucceed(toIdx.Get(nil, "x"))
			Expect(keys).To(HaveLen(1))
			Expect(string(keys[0])).To(Equal("a->x"))
		})

		It("Should persist the staged insert on commit", func(ctx SpecContext) {
			tx := idxDB.OpenTx()
			Expect(table.NewCreate().
				Entry(&relationEntry{From: "q", To: "x"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			keys := MustSucceed(toIdx.Get(nil, "x"))
			asStrings := make([]string, len(keys))
			for i, k := range keys {
				asStrings[i] = string(k)
			}
			Expect(asStrings).To(ConsistOf("a->x", "q->x"))
		})

		// Regression for the ontology DAG writer cycle check. The
		// dagWriter.DefineRelationship flow writes a relationship
		// and then traverses descendants via byTo.Filter to detect
		// cycles. Before the overlay, the filter missed
		// in-tx writes and cycle detection silently passed invalid
		// graphs. This test reproduces the shape: insert A->B and
		// then retrieve via byTo.Filter("B") in the same tx.
		It("Should surface a newly-written relationship via byTo.Filter in the same tx", func(ctx SpecContext) {
			tx := idxDB.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()

			rel := relationEntry{From: "A", To: "B"}
			Expect(table.NewCreate().Entry(&rel).Exec(ctx, tx)).To(Succeed())

			var descendants []relationEntry
			Expect(table.NewRetrieve().
				Where(toIdx.Filter("B")).
				Entries(&descendants).Exec(ctx, tx)).To(Succeed())
			Expect(fromsOf(descendants)).To(ConsistOf("A"))
		})
	})
})

var _ = Describe("Async populate", func() {
	Describe("WaitForIndexes", func() {
		var idxDB *gorp.DB
		BeforeEach(func() {
			mem := DeferClose(memkv.New())
			idxDB = gorp.Wrap(mem)
		})

		It("Should return nil for a table with no indexes", func(ctx SpecContext) {
			table := openIndexedTable[int32, indexedEntry](ctx, idxDB)
			DeferClose(table)
			Expect(table.WaitForIndexes(ctx)).To(Succeed())
		})

		It("Should return nil after a successful populate", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entry(&indexedEntry{ID: 1, Name: "alpha"}).
				Exec(ctx, idxDB)).To(Succeed())
			nameIdx := gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table := openIndexedTable(ctx, idxDB, nameIdx)
			DeferClose(table)
			Expect(table.WaitForIndexes(ctx)).To(Succeed())
		})

		It("Should return ctx.Err when ctx cancels before populate completes",
			func(ctx SpecContext) {
				nameIdx := gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table := openIndexedTable(ctx, idxDB, nameIdx)
				DeferClose(table)
				canceledCtx, cancel := context.WithCancel(ctx)
				cancel()
				err := table.WaitForIndexes(canceledCtx)
				if err != nil {
					Expect(err).To(MatchError(context.Canceled))
				}
				// Tolerate the race where populate finishes before our
				// cancel takes effect — both outcomes are correct.
			})
	})

	Describe("Populate failure", func() {
		var (
			db      *gorp.DB
			nameIdx *gorp.LookupIndex[int32, indexedEntry, string]
			table   *gorp.Table[int32, indexedEntry]
		)
		BeforeEach(func(ctx SpecContext) {
			mem := DeferClose(memkv.New())
			db = gorp.Wrap(&failOnceDB{DB: mem, err: errors.New("populate boom")})
			seed := []indexedEntry{
				{ID: 1, Name: "alpha"},
				{ID: 2, Name: "beta"},
				{ID: 3, Name: "alpha"},
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&seed).Exec(ctx, db)).To(Succeed())
			nameIdx = gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table = openIndexedTable(ctx, db, nameIdx)
			DeferClose(table)
		})

		It("Should surface the populate error from WaitForIndexes",
			func(ctx SpecContext) {
				Expect(table.WaitForIndexes(ctx)).
					To(MatchError(ContainSubstring("populate boom")))
			})

		It("Should return ErrIndexInvalid from direct Get", func(ctx SpecContext) {
			Expect(table.WaitForIndexes(ctx)).To(MatchError(ContainSubstring("populate boom")))
			Expect(nameIdx.Get(nil, "alpha")).Error().
				To(MatchError(gorp.ErrIndexInvalid))
		})

		It("Should fall back to a sequential scan from Retrieve via Filter",
			func(ctx SpecContext) {
				Expect(table.WaitForIndexes(ctx)).To(MatchError(ContainSubstring("populate boom")))
				var got []indexedEntry
				Expect(gorp.NewRetrieve[int32, indexedEntry]().
					Where(nameIdx.Filter("alpha")).
					Entries(&got).Exec(ctx, db)).To(Succeed())
				Expect(idsOf(got)).To(ConsistOf(int32(1), int32(3)))
			})

		It("Should preserve a MatchKeys sibling when the indexed child fails to populate",
			func(ctx SpecContext) {
				Expect(table.WaitForIndexes(ctx)).To(MatchError(ContainSubstring("populate boom")))
				var got []indexedEntry
				Expect(gorp.NewRetrieve[int32, indexedEntry]().
					Where(gorp.And(
						nameIdx.Filter("alpha"),
						gorp.MatchKeys[int32, indexedEntry](1),
					)).
					Entries(&got).Exec(ctx, db)).To(Succeed())
				Expect(idsOf(got)).To(ConsistOf(int32(1)))
			})

		It("Should fall back to a sequential scan for Or when populate fails",
			func(ctx SpecContext) {
				// Regression: without a construction-time eval on Or's
				// resolver branch, resolveFilter would swallow
				// ErrIndexInvalid and leave r.filter.eval nil, so
				// match() would return true for every entry. The
				// "gamma" branch matches nothing in the seed, so the
				// correct result excludes the "beta" entry; the buggy
				// behavior would include it.
				Expect(table.WaitForIndexes(ctx)).To(MatchError(ContainSubstring("populate boom")))
				var got []indexedEntry
				Expect(gorp.NewRetrieve[int32, indexedEntry]().
					Where(gorp.Or(
						nameIdx.Filter("alpha"),
						nameIdx.Filter("gamma"),
					)).
					Entries(&got).Exec(ctx, db)).To(Succeed())
				Expect(idsOf(got)).To(ConsistOf(int32(1), int32(3)))
			})

		It("Should fall back to a sequential scan for Not when populate fails",
			func(ctx SpecContext) {
				// Same regression shape as Or: Not's resolver branch
				// previously omitted the construction-time eval, so a
				// failed populate would let every entry through (Not of
				// a vacuously-true match).
				Expect(table.WaitForIndexes(ctx)).To(MatchError(ContainSubstring("populate boom")))
				var got []indexedEntry
				Expect(gorp.NewRetrieve[int32, indexedEntry]().
					Where(gorp.Not(nameIdx.Filter("alpha"))).
					Entries(&got).Exec(ctx, db)).To(Succeed())
				Expect(idsOf(got)).To(ConsistOf(int32(2)))
			})
	})
})
