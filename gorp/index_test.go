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
	"math"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// indexedEntry is the test entry type used by the index suite. It carries two indexable
// fields (Name, Category) and one orderable field (Score) so the suite can exercise
// LookupIndex, SortedIndex, and OrderBy against realistic data shapes.
type indexedEntry struct {
	ID       int32
	Name     string
	Category string
	Score    int64
	Flag     bool
}

func (e indexedEntry) GorpKey() int32 { return e.ID }

// relationEntry is a composite-string-keyed test entry. Its gorp key is the
// "<from>-><to>" string.
type relationEntry struct {
	From  string
	To    string
	Label string
}

func (e relationEntry) GorpKey() string { return e.From + "->" + e.To }

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
	GinkgoHelper()
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

// blockingDB is a kv.DB whose iterators open only once release is closed.
type blockingDB struct {
	kv.DB
	release chan struct{}
}

func (b *blockingDB) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	<-b.release
	return b.DB.OpenIterator(opts)
}

// failingCommitDB is a kv.DB whose transactions call during, when set, before they
// commit. Once fail is set, each commit fails with it instead.
type failingCommitDB struct {
	kv.DB
	fail   error
	during func()
}

type failingCommitTx struct {
	kv.Tx
	db *failingCommitDB
}

func (f *failingCommitDB) OpenTx() kv.Tx {
	return failingCommitTx{Tx: f.DB.OpenTx(), db: f}
}

func (t failingCommitTx) Commit(ctx context.Context, opts ...any) error {
	if t.db.during != nil {
		t.db.during()
	}
	if t.db.fail == nil {
		return t.Tx.Commit(ctx, opts...)
	}
	return t.db.fail
}

func wrapMsgpack(kvDB kv.DB) *gorp.DB {
	return gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec))
}

var _ = Describe("Index", func() {
	var idxDB *gorp.DB
	BeforeEach(func() {
		idxDB = DeferClose(OpenGorpMsgpackDB())
	})

	Describe("Filter from an observer of a commit", func() {
		var (
			db       *gorp.DB
			table    *gorp.Table[int32, indexedEntry]
			nameIdx  *gorp.LookupIndex[int32, indexedEntry, string]
			scoreIdx *gorp.SortedIndex[int32, indexedEntry, int64]
			byName   []int32
			byScore  []int32
		)
		BeforeEach(func(ctx SpecContext) {
			db = DeferClose(wrapMsgpack(memkv.New()))
			nameIdx = gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			scoreIdx = gorp.NewSortedIndex(
				"score", func(e *indexedEntry) int64 { return e.Score },
			)
			table = DeferClose(openIndexedTable(ctx, db, nameIdx, scoreIdx))
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 1, Name: "alpha", Score: 10}).
				Exec(ctx, db)).To(Succeed())
			byName, byScore = nil, nil
			disconnect := table.Observe().OnChange(func(
				ctx context.Context,
				_ gorp.TxReader[int32, indexedEntry],
			) {
				retrieve := func(f gorp.Filter[int32, indexedEntry]) []int32 {
					GinkgoHelper()
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(f).Entries(&res).Exec(ctx, db)).To(Succeed())
					return idsOf(res)
				}
				byName = retrieve(nameIdx.Filter("zeta"))
				byScore = retrieve(scoreIdx.Filter(int64(99)))
			})
			DeferCleanup(disconnect)
		})

		It("Should find an entry written through a tx", func(ctx SpecContext) {
			tx := db.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 1, Name: "zeta", Score: 99}).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(byName).To(ConsistOf(int32(1)))
			Expect(byScore).To(ConsistOf(int32(1)))
		})

		It("Should find an entry written through the DB", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 2, Name: "zeta", Score: 99}).
				Exec(ctx, db)).To(Succeed())
			Expect(byName).To(ConsistOf(int32(2)))
			Expect(byScore).To(ConsistOf(int32(2)))
		})

		It(
			"Should find a key once when its value does not change",
			func(ctx SpecContext) {
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 2, Name: "zeta", Score: 99}).
					Exec(ctx, db)).To(Succeed())
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 2, Name: "zeta", Score: 99}).
					Exec(ctx, db)).To(Succeed())
				Expect(byName).To(Equal([]int32{2}))
				Expect(byScore).To(Equal([]int32{2}))
			},
		)
	})

	It("Should not find an entry through a commit that fails", func(ctx SpecContext) {
		failing := &failingCommitDB{DB: memkv.New()}
		db := wrapMsgpack(failing)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		nameIdx := gorp.NewLookupIndex(
			"name", func(e *indexedEntry) string { return e.Name },
		)
		table := openIndexedTable(ctx, db, nameIdx)
		DeferCleanup(func() { Expect(table.Close()).To(Succeed()) })
		Expect(table.NewCreate().
			Entry(&indexedEntry{ID: 1, Name: "alpha"}).Exec(ctx, db)).To(Succeed())
		tx := db.OpenTx()
		DeferCleanup(func() { Expect(tx.Close()).To(Succeed()) })
		Expect(table.NewCreate().
			Entry(&indexedEntry{ID: 1, Name: "zeta"}).Exec(ctx, tx)).To(Succeed())
		var (
			during   []indexedEntry
			inFlight []int32
		)
		failing.fail = errors.New("commit failed")
		failing.during = func() {
			Expect(table.NewRetrieve().
				Where(nameIdx.Filter("zeta")).
				Entries(&during).Exec(ctx, db)).To(Succeed())
			inFlight = MustSucceed(nameIdx.Get(ctx, nil, "zeta"))
		}
		Expect(tx.Commit(ctx)).To(MatchError("commit failed"))
		Expect(during).To(BeEmpty())
		Expect(inFlight).To(ConsistOf(int32(1)))
		Expect(nameIdx.Get(ctx, nil, "zeta")).To(BeEmpty())
		Expect(nameIdx.Get(ctx, nil, "alpha")).To(ConsistOf(int32(1)))
	})

	It("Should finish a write through the DB of a NaN value", func(ctx SpecContext) {
		db := DeferClose(wrapMsgpack(memkv.New()))
		ratioIdx := gorp.NewSortedIndex(
			"ratio", func(*indexedEntry) float64 { return math.NaN() },
		)
		table := DeferClose(openIndexedTable(ctx, db, ratioIdx))
		Expect(table.NewCreate().
			Entry(&indexedEntry{ID: 1, Name: "alpha"}).
			Exec(ctx, db)).To(Succeed())
	})

	It(
		"Should find an entry only by its stored value during a commit that updates it",
		func(ctx SpecContext) {
			store := &failingCommitDB{DB: memkv.New()}
			db := DeferClose(wrapMsgpack(store))
			nameIdx := gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table := DeferClose(openIndexedTable(ctx, db, nameIdx))
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 1, Name: "alpha"}).
				Exec(ctx, db)).To(Succeed())
			retrieve := func(f gorp.Filter[int32, indexedEntry]) []int32 {
				GinkgoHelper()
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(f).Entries(&res).Exec(ctx, db)).To(Succeed())
				return idsOf(res)
			}
			var byOld, byNew []int32
			store.during = func() {
				byOld = retrieve(nameIdx.Filter("alpha"))
				byNew = retrieve(nameIdx.Filter("zeta"))
			}
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 1, Name: "zeta"}).
				Exec(ctx, db)).To(Succeed())
			Expect(byOld).To(ConsistOf(int32(1)))
			Expect(byNew).To(BeEmpty())
			store.during = nil
			Expect(retrieve(nameIdx.Filter("alpha"))).To(BeEmpty())
			Expect(retrieve(nameIdx.Filter("zeta"))).To(ConsistOf(int32(1)))
		},
	)

	Describe("Lookup", func() {
		Describe("Population at OpenTable", func() {
			It(
				"Should populate the index from existing entries",
				func(ctx SpecContext) {
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

					keys := MustSucceed(nameIdx.Get(ctx, nil, "alpha"))
					Expect(keys).To(ConsistOf(int32(1), int32(3)))
					Expect(nameIdx.Get(ctx, nil, "beta")).To(ConsistOf(int32(2)))
					Expect(nameIdx.Get(ctx, nil, "missing")).To(BeEmpty())
				},
			)

			It("Should fail population when an entry does not decode",
				func(ctx SpecContext) {
					createIndexed(ctx, idxDB,
						indexedEntry{ID: 1, Name: "alpha"},
						indexedEntry{ID: 2, Name: "beta"},
					)
					corrupt[int32, indexedEntry](ctx, idxDB, 2)
					nameIdx := gorp.NewLookupIndex(
						"name", func(e *indexedEntry) string { return e.Name },
					)
					DeferClose(openIndexedTable(ctx, idxDB, nameIdx))
					Expect(nameIdx.Get(ctx, nil, "alpha")).Error().
						To(MatchError(gorp.ErrIndexInvalid))
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

				Expect(flagIdx.Get(ctx, nil, true)).To(ConsistOf(int32(1), int32(3)))
				Expect(flagIdx.Get(ctx, nil, false)).To(ConsistOf(int32(2), int32(4)))
			})
		})

		Describe("Maintenance", func() {
			var nameIdx *gorp.LookupIndex[int32, indexedEntry, string]
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				DeferClose(openIndexedTable(ctx, idxDB, nameIdx))
			})

			It(
				"Should index writes through another open table of the type",
				func(ctx SpecContext) {
					other := DeferClose(openIndexedTable(
						ctx,
						idxDB,
						gorp.NewLookupIndex(
							"other", func(e *indexedEntry) string { return e.Name },
						),
					))
					Expect(other.NewCreate().
						Entry(&indexedEntry{ID: 20, Name: "delta"}).
						Exec(ctx, idxDB)).To(Succeed())
					Expect(nameIdx.Get(ctx, nil, "delta")).To(Equal([]int32{20}))
				},
			)

			It("Should index newly created entries", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 10, Name: "gamma"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(ctx, nil, "gamma")).To(ConsistOf(int32(10)))
			})

			It(
				"Should reindex an entry when its indexed field changes",
				func(ctx SpecContext) {
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&indexedEntry{ID: 11, Name: "delta"}).
						Exec(ctx, idxDB)).To(Succeed())
					Expect(nameIdx.Get(ctx, nil, "delta")).To(ConsistOf(int32(11)))

					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&indexedEntry{ID: 11, Name: "epsilon"}).
						Exec(ctx, idxDB)).To(Succeed())
					Expect(nameIdx.Get(ctx, nil, "delta")).To(BeEmpty())
					Expect(nameIdx.Get(ctx, nil, "epsilon")).To(ConsistOf(int32(11)))
				},
			)

			It("Should drop deleted entries from the index", func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entry(&indexedEntry{ID: 12, Name: "zeta"}).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(ctx, nil, "zeta")).To(ConsistOf(int32(12)))

				Expect(gorp.NewDelete[int32, indexedEntry]().
					Where(gorp.MatchKeys[int32, indexedEntry](12)).
					Exec(ctx, idxDB)).To(Succeed())
				Expect(nameIdx.Get(ctx, nil, "zeta")).To(BeEmpty())
			})

			It(
				"Should leave the index unchanged when a set does not modify the indexed field",
				func(ctx SpecContext) {
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&indexedEntry{ID: 13, Name: "eta", Score: 5}).
						Exec(ctx, idxDB)).To(Succeed())
					Expect(nameIdx.Get(ctx, nil, "eta")).To(ConsistOf(int32(13)))

					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&indexedEntry{ID: 13, Name: "eta", Score: 10}).
						Exec(ctx, idxDB)).To(Succeed())
					Expect(nameIdx.Get(ctx, nil, "eta")).To(ConsistOf(int32(13)))
				},
			)
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
				table = DeferClose(openIndexedTable(ctx, idxDB, nameIdx))
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

			It("Should return matching entries via Where", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("a")).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(idsOf(res)).To(ConsistOf(int32(1), int32(3)))
			})

			It(
				"Should return matching entries for multiple values",
				func(ctx SpecContext) {
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(nameIdx.Filter("a", "b")).
						Entries(&res).Exec(ctx, idxDB)).To(Succeed())
					Expect(
						idsOf(res),
					).To(ConsistOf(int32(1), int32(2), int32(3), int32(5)))
				},
			)

			It(
				"Should return an entry once for a repeated value",
				func(ctx SpecContext) {
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(nameIdx.Filter("a", "a")).
						Entries(&res).Exec(ctx, idxDB)).To(Succeed())
					Expect(idsOf(res)).To(ConsistOf(int32(1), int32(3)))
				},
			)

			It(
				"Should return an empty result when no values match",
				func(ctx SpecContext) {
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(nameIdx.Filter("missing")).
						Entries(&res).Exec(ctx, idxDB)).To(Succeed())
					Expect(res).To(BeEmpty())
				},
			)
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
				table = DeferClose(openIndexedTable(ctx, idxDB, nameIdx, categoryIdx))
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

			It(
				"Should preserve membership through nested And composition",
				func(ctx SpecContext) {
					// Regression: previously, And(A, B) returned a Filter with
					// Keys set but membership nil, so a subsequent And(prev, C)
					// silently dropped every key (containsKey on prev always
					// returned false) and returned an empty result.
					inner := gorp.And(
						nameIdx.Filter("a", "b"),
						categoryIdx.Filter("x", "y"),
					)
					outer := gorp.And(inner, nameIdx.Filter("a"))
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(outer).
						Entries(&res).Exec(ctx, idxDB)).To(Succeed())
					Expect(idsOf(res)).To(ConsistOf(int32(1), int32(2)))
				},
			)

			It(
				"Should preserve membership through nested Or composition",
				func(ctx SpecContext) {
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
				},
			)
		})

		Describe("Concurrency", func() {
			It(
				"Should permit concurrent Filter calls while writes commit",
				func(ctx SpecContext) {
					nameIdx := gorp.NewLookupIndex(
						"name", func(e *indexedEntry) string { return e.Name },
					)
					table := openIndexedTable(ctx, idxDB, nameIdx)
					defer func() { Expect(table.Close()).To(Succeed()) }()

					var wg sync.WaitGroup
					wg.Go(func() {
						defer GinkgoRecover()
						for i := range int32(100) {
							Expect(table.NewCreate().
								Entry(&indexedEntry{ID: i, Name: "shared"}).
								Exec(ctx, idxDB)).To(Succeed())
						}
					})
					wg.Go(func() {
						defer GinkgoRecover()
						for range 200 {
							MustSucceed(nameIdx.Get(ctx, nil, "shared"))
						}
					})
					wg.Wait()
					Eventually(func() ([]int32, error) {
						return nameIdx.Get(ctx, nil, "shared")
					}).Should(HaveLen(100))
				},
			)

			It(
				"Should be safe to Exec a shared Or filter from multiple goroutines",
				func(ctx SpecContext) {
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

					Expect(
						gorp.NewCreate[int32, indexedEntry]().Entries(&[]indexedEntry{
							{ID: 1, Name: "a", Category: "x"},
							{ID: 2, Name: "b", Category: "y"},
							{ID: 3, Name: "a", Category: "z"},
						}).Exec(ctx, idxDB),
					).To(Succeed())

					shared := gorp.Or(nameIdx.Filter("a"), categoryIdx.Filter("y"))
					var wg sync.WaitGroup
					for range 16 {
						wg.Go(func() {
							for range 32 {
								var res []indexedEntry
								Expect(table.NewRetrieve().
									Where(shared).
									Entries(&res).Exec(ctx, idxDB)).To(Succeed())
								Expect(
									idsOf(res),
								).To(ConsistOf(int32(1), int32(2), int32(3)))
							}
						})
					}
					wg.Wait()
				},
			)

			It(
				"Should be safe to Exec a shared Not filter from multiple goroutines",
				func(ctx SpecContext) {
					// Not(indexed) previously wrote back into the captured
					// child filter at resolve time; concurrent Execs raced
					// on those writes. The fix uses a local materialized
					// copy per resolve call.
					nameIdx := gorp.NewLookupIndex(
						"name", func(e *indexedEntry) string { return e.Name },
					)
					table := openIndexedTable(ctx, idxDB, nameIdx)
					defer func() { Expect(table.Close()).To(Succeed()) }()

					Expect(
						gorp.NewCreate[int32, indexedEntry]().Entries(&[]indexedEntry{
							{ID: 1, Name: "a"},
							{ID: 2, Name: "b"},
							{ID: 3, Name: "a"},
						}).Exec(ctx, idxDB),
					).To(Succeed())

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
				},
			)
		})

		// Each case writes through a table-bound query in an open transaction, then
		// filters through the index in the same transaction, which must see the staged
		// write. Rollback cases check that the committed index never changed.
		Describe("Tx delta visibility", func() {
			var (
				table   *gorp.Table[int32, indexedEntry]
				nameIdx *gorp.LookupIndex[int32, indexedEntry, string]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				table = DeferClose(openIndexedTable(ctx, idxDB, nameIdx))
				seed := []indexedEntry{
					{ID: 1, Name: "alpha"},
					{ID: 2, Name: "beta"},
				}
				Expect(table.NewCreate().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
			})

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

			It(
				"Should reflect an update that moves a key to a different value",
				func(ctx SpecContext) {
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
				},
			)

			It("Should exclude an entry deleted in the same tx", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				Expect(table.NewDelete().
					Where(gorp.MatchKeys[int32, indexedEntry](1)).
					Exec(ctx, tx)).To(Succeed())

				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(nameIdx.Filter("alpha")).
					Entries(&res).Exec(ctx, tx)).To(Succeed())
				Expect(res).To(BeEmpty())
			})

			It(
				"Should union staged and committed matches for multi-value filters",
				func(ctx SpecContext) {
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
				},
			)

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

			It(
				"Should drop the delta on rollback without touching the global index",
				func(ctx SpecContext) {
					tx := idxDB.OpenTx()
					Expect(table.NewCreate().
						Entry(&indexedEntry{ID: 40, Name: "alpha"}).
						Exec(ctx, tx)).To(Succeed())
					// Close without commit: cleanups fire via *tx.Close,
					// dropping the delta. The global index should not
					// carry the rolled-back write.
					Expect(tx.Close()).To(Succeed())

					Expect(nameIdx.Get(ctx, nil, "alpha")).To(ConsistOf(int32(1)))

					// And a fresh retrieve via the bare DB should see
					// committed-only state.
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(nameIdx.Filter("alpha")).
						Entries(&res).Exec(ctx, idxDB)).To(Succeed())
					Expect(res).To(HaveLen(1))
					Expect(res[0].ID).To(Equal(int32(1)))
				},
			)

			It(
				"Should see the staged write on committed global index after commit",
				func(ctx SpecContext) {
					tx := idxDB.OpenTx()
					Expect(table.NewCreate().
						Entry(&indexedEntry{ID: 50, Name: "alpha"}).
						Exec(ctx, tx)).To(Succeed())
					Expect(tx.Commit(ctx)).To(Succeed())
					Expect(tx.Close()).To(Succeed())

					Expect(
						nameIdx.Get(ctx, nil, "alpha"),
					).To(ConsistOf(int32(1), int32(50)))
				},
			)

			It("Should support set-then-delete in the same tx", func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 60, Name: "alpha"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(table.NewDelete().
					Where(gorp.MatchKeys[int32, indexedEntry](60)).
					Exec(ctx, tx)).To(Succeed())

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
			It(
				"Should populate a Sorted index and serve exact-match lookups",
				func(ctx SpecContext) {
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

					Expect(
						scoreIdx.Get(ctx, nil, int64(20)),
					).To(ConsistOf(int32(3), int32(4)))
					Expect(scoreIdx.Get(ctx, nil, int64(10))).To(ConsistOf(int32(2)))
					Expect(scoreIdx.Get(ctx, nil, int64(99))).To(BeEmpty())
				},
			)
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
				table = DeferClose(openIndexedTable(ctx, idxDB, scoreIdx))
				seed := make([]indexedEntry, 20)
				for i := range 20 {
					seed[i] = indexedEntry{ID: int32(i), Score: int64(i * 10)}
				}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
			})

			It("Should panic when a key cursor combines with an order", func() {
				ordered := table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc))
				Expect(func() { ordered.After(3) }).
					To(PanicWith("gorp: After cannot combine with OrderBy"))
				after := table.NewRetrieve().After(3)
				Expect(func() { after.OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)) }).
					To(PanicWith("gorp: After cannot combine with OrderBy"))
			})

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

			It(
				"Should resume pagination via After on the SortedQuery",
				func(ctx SpecContext) {
					var page1 []indexedEntry
					Expect(table.NewRetrieve().
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
						Limit(5).
						Entries(&page1).Exec(ctx, idxDB)).To(Succeed())
					Expect(page1).To(HaveLen(5))

					last := page1[len(page1)-1]

					var page2 []indexedEntry
					Expect(table.NewRetrieve().
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc).After(last.Score, last.ID)).
						Limit(5).
						Entries(&page2).Exec(ctx, idxDB)).To(Succeed())
					Expect(scoresOf(page2)).To(Equal([]int64{50, 60, 70, 80, 90}))
				},
			)

			It("Should keep entries that tie at a page edge", func(ctx SpecContext) {
				tied := []indexedEntry{{ID: 40, Score: 40}, {ID: 41, Score: 40}}
				Expect(gorp.NewCreate[int32, indexedEntry]().
					Entries(&tied).Exec(ctx, idxDB)).To(Succeed())
				var page1 []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
					Limit(5).
					Entries(&page1).Exec(ctx, idxDB)).To(Succeed())
				Expect(idsOf(page1)).To(Equal([]int32{0, 1, 2, 3, 4}))
				var page2 []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc).After(40, 4)).
					Limit(3).
					Entries(&page2).Exec(ctx, idxDB)).To(Succeed())
				Expect(idsOf(page2)).To(Equal([]int32{40, 41, 5}))
			})

			It("Should count the limit in matching entries", func(ctx SpecContext) {
				var res []indexedEntry
				aboveFifty := gorp.Match(
					func(_ gorp.Context, e *indexedEntry) (bool, error) {
						return e.Score > 50, nil
					},
				)
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
					Limit(3).
					Where(aboveFifty).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(scoresOf(res)).To(Equal([]int64{60, 70, 80}))
			})

			It(
				"Should fill the limit past the first walk batch",
				func(ctx SpecContext) {
					many := make([]indexedEntry, 300)
					for i := range many {
						many[i] = indexedEntry{
							ID:    int32(100 + i),
							Score: int64(1000 + i),
						}
					}
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entries(&many).Exec(ctx, idxDB)).To(Succeed())
					var res []indexedEntry
					odd := gorp.Match(
						func(_ gorp.Context, e *indexedEntry) (bool, error) {
							return e.ID >= 100 && e.ID%2 == 1, nil
						},
					)
					Expect(table.NewRetrieve().
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
						Limit(140).
						Where(odd).
						Entries(&res).Exec(ctx, idxDB)).To(Succeed())
					Expect(res).To(HaveLen(140))
					Expect(res[0].ID).To(Equal(int32(101)))
					Expect(res[139].ID).To(Equal(int32(379)))
				},
			)

			It("Should walk only the keys of a key filter", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(gorp.MatchKeys[int32, indexedEntry](9, 2, 15, 99)).
					OrderBy(scoreIdx.Ordered(gorp.DirectionDesc).After(150, 15)).
					Limit(5).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(idsOf(res)).To(Equal([]int32{9, 2}))
			})

			It(
				"Should stop a walk of a key filter at the limit",
				func(ctx SpecContext) {
					keys := make([]int32, 20)
					for i := range keys {
						keys[i] = int32(i)
					}
					matched := 0
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(gorp.And(
							gorp.MatchKeys[int32, indexedEntry](keys...),
							gorp.Match(func(gorp.Context, *indexedEntry) (bool, error) {
								matched++
								return true, nil
							}),
						)).
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
						Limit(3).
						Entries(&res).Exec(ctx, idxDB)).To(Succeed())
					Expect(scoresOf(res)).To(Equal([]int64{0, 10, 20}))
					Expect(matched).To(Equal(3))
				},
			)

			It("Should return every entry without a limit", func(ctx SpecContext) {
				var res []indexedEntry
				Expect(table.NewRetrieve().
					OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(res).To(HaveLen(20))
			})

			It(
				"Should skip an entry that moved after the walk found it",
				func(ctx SpecContext) {
					tx := idxDB.OpenTx()
					defer func() { Expect(tx.Close()).To(Succeed()) }()
					moved := indexedEntry{ID: 1, Score: 500}
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&moved).Exec(ctx, tx)).To(Succeed())
					var res []indexedEntry
					Expect(table.NewRetrieve().
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
						Limit(3).
						Entries(&res).Exec(ctx, tx)).To(Succeed())
					Expect(scoresOf(res)).To(Equal([]int64{0, 20, 30}))
				},
			)

			It(
				"Should sort an entry that moved ahead inside the page into place",
				func(ctx SpecContext) {
					tx := idxDB.OpenTx()
					defer func() { Expect(tx.Close()).To(Succeed()) }()
					moved := indexedEntry{ID: 1, Score: 25}
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&moved).Exec(ctx, tx)).To(Succeed())
					var res []indexedEntry
					Expect(table.NewRetrieve().
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
						Limit(3).
						Entries(&res).Exec(ctx, tx)).To(Succeed())
					Expect(idsOf(res)).To(Equal([]int32{0, 2, 1}))
				},
			)

			It(
				"Should sort an entry that moved back inside the page into place",
				func(ctx SpecContext) {
					tx := idxDB.OpenTx()
					defer func() { Expect(tx.Close()).To(Succeed()) }()
					moved := indexedEntry{ID: 2, Score: 5}
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&moved).Exec(ctx, tx)).To(Succeed())
					var res []indexedEntry
					Expect(table.NewRetrieve().
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
						Limit(3).
						Entries(&res).Exec(ctx, tx)).To(Succeed())
					Expect(idsOf(res)).To(Equal([]int32{0, 2, 1}))
				},
			)

			It(
				"Should sort an entry that moved after the walk found it into a full result",
				func(ctx SpecContext) {
					tx := idxDB.OpenTx()
					defer func() { Expect(tx.Close()).To(Succeed()) }()
					moved := indexedEntry{ID: 1, Score: 500}
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&moved).Exec(ctx, tx)).To(Succeed())
					var res []indexedEntry
					Expect(table.NewRetrieve().
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
						Entries(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(HaveLen(20))
					Expect(scoresOf(res)[:2]).To(Equal([]int64{0, 20}))
					Expect(res[19]).To(Equal(moved))
				},
			)

			It(
				"Should sort an entry that moved after the walk found it into a key filter",
				func(ctx SpecContext) {
					tx := idxDB.OpenTx()
					defer func() { Expect(tx.Close()).To(Succeed()) }()
					moved := indexedEntry{ID: 1, Score: 500}
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&moved).Exec(ctx, tx)).To(Succeed())
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(gorp.MatchKeys[int32, indexedEntry](2, 1, 0)).
						OrderBy(scoreIdx.Ordered(gorp.DirectionDesc)).
						Limit(2).
						Entries(&res).Exec(ctx, tx)).To(Succeed())
					Expect(idsOf(res)).To(Equal([]int32{1, 2}))
				},
			)

			It(
				"Should drop an entry that moved behind the cursor after the walk found it",
				func(ctx SpecContext) {
					tx := idxDB.OpenTx()
					defer func() { Expect(tx.Close()).To(Succeed()) }()
					moved := indexedEntry{ID: 15, Score: 5}
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entry(&moved).Exec(ctx, tx)).To(Succeed())
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(gorp.MatchKeys[int32, indexedEntry](2, 15, 9)).
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc).After(50, 5)).
						Entries(&res).Exec(ctx, tx)).To(Succeed())
					Expect(idsOf(res)).To(Equal([]int32{9}))
				},
			)

			It(
				"Should run validators on the ordered result set",
				func(ctx SpecContext) {
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
				},
			)

			It(
				"Should short-circuit Exec with a validator error on the ordered path",
				func(ctx SpecContext) {
					var res []indexedEntry
					Expect(table.NewRetrieve().
						OrderBy(scoreIdx.Ordered(gorp.DirectionAsc)).
						Limit(3).
						Validate(func(_ gorp.Context, _ []indexedEntry) error {
							return errors.New("ordered validator rejected query")
						}).
						Entries(&res).Exec(ctx, idxDB)).
						To(MatchError(ContainSubstring("ordered validator rejected query")))
				},
			)
		})
	})

	Describe("Sorted range filter", func() {
		var (
			table    *gorp.Table[int32, indexedEntry]
			scoreIdx *gorp.SortedIndex[int32, indexedEntry, int64]
		)
		BeforeEach(func(ctx SpecContext) {
			scoreIdx = gorp.NewSortedIndex(
				"score", func(e *indexedEntry) int64 { return e.Score },
			)
			table = DeferClose(openIndexedTable(ctx, idxDB, scoreIdx))
			entries := make([]indexedEntry, 10)
			for i := range entries {
				entries[i] = indexedEntry{ID: int32(i), Score: int64(i * 10)}
			}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&entries).Exec(ctx, idxDB)).To(Succeed())
		})
		inRange := func(
			ctx context.Context,
			snap gorp.Snapshot,
			lower, upper *gorp.Bound[int64],
		) []int32 {
			GinkgoHelper()
			var res []indexedEntry
			Expect(table.NewRetrieve().
				Where(scoreIdx.FilterRange(lower, upper)).
				Entries(&res).Exec(ctx, snap)).To(Succeed())
			return idsOf(res)
		}

		DescribeTable(
			"Should match the entries between the bounds",
			func(ctx SpecContext, lower, upper *gorp.Bound[int64], ids []int32) {
				Expect(inRange(ctx, idxDB, lower, upper)).To(ConsistOf(ids))
			},
			Entry("an inclusive range", &gorp.Bound[int64]{Value: 20},
				&gorp.Bound[int64]{Value: 40}, []int32{2, 3, 4}),
			Entry("an exclusive range", &gorp.Bound[int64]{Value: 20, Exclusive: true},
				&gorp.Bound[int64]{Value: 40, Exclusive: true}, []int32{3}),
			Entry(
				"an open lower end",
				nil,
				&gorp.Bound[int64]{Value: 15},
				[]int32{0, 1},
			),
			Entry("an open upper end", &gorp.Bound[int64]{Value: 85}, nil, []int32{9}),
			Entry("bounds between values", &gorp.Bound[int64]{Value: 11},
				&gorp.Bound[int64]{Value: 29}, []int32{2}),
			Entry("an empty range", &gorp.Bound[int64]{Value: 40},
				&gorp.Bound[int64]{Value: 20}, []int32{}),
			Entry("no bounds", nil, nil, []int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}),
		)

		It("Should see the writes staged in the same tx", func(ctx SpecContext) {
			tx := DeferClose(idxDB.OpenTx())
			moved := indexedEntry{ID: 1, Score: 35}
			added := indexedEntry{ID: 20, Score: 30}
			Expect(gorp.NewCreate[int32, indexedEntry]().
				Entries(&[]indexedEntry{moved, added}).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int32, indexedEntry]().
				Where(gorp.MatchKeys[int32, indexedEntry](3)).Exec(ctx, tx)).
				To(Succeed())
			bounds := []*gorp.Bound[int64]{{Value: 10}, {Value: 40}}
			Expect(inRange(ctx, tx, bounds[0], bounds[1])).To(ConsistOf(
				int32(1), int32(2), int32(4), int32(20),
			))
			Expect(inRange(ctx, idxDB, bounds[0], bounds[1])).To(ConsistOf(
				int32(1), int32(2), int32(3), int32(4),
			))
		})

		Describe("And of ranges", func() {
			var atLeast10, atMost30 gorp.Filter[int32, indexedEntry]
			BeforeEach(func() {
				atLeast10 = scoreIdx.FilterRange(&gorp.Bound[int64]{Value: 10}, nil)
				atMost30 = scoreIdx.FilterRange(nil, &gorp.Bound[int64]{Value: 30})
			})
			andIDs := func(
				ctx context.Context,
				snap gorp.Snapshot,
				filters ...gorp.Filter[int32, indexedEntry],
			) []int32 {
				GinkgoHelper()
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(gorp.And(filters...)).
					Entries(&res).Exec(ctx, snap)).To(Succeed())
				return idsOf(res)
			}

			It("Should match the entries in every range", func(ctx SpecContext) {
				Expect(andIDs(ctx, idxDB, atLeast10, atMost30)).
					To(ConsistOf(int32(1), int32(2), int32(3)))
				Expect(andIDs(ctx, idxDB, atMost30, atLeast10)).
					To(ConsistOf(int32(1), int32(2), int32(3)))
			})

			It(
				"Should check the wider range against the writes staged in the tx",
				func(ctx SpecContext) {
					tx := DeferClose(idxDB.OpenTx())
					staged := []indexedEntry{
						{ID: 2, Score: 95},
						{ID: 8, Score: 25},
						{ID: 20, Score: 5},
					}
					Expect(gorp.NewCreate[int32, indexedEntry]().
						Entries(&staged).Exec(ctx, tx)).To(Succeed())
					Expect(andIDs(ctx, tx, atLeast10, atMost30)).
						To(ConsistOf(int32(1), int32(3), int32(8)))
				},
			)
		})

		It("Should page an order through a wide range", func(ctx SpecContext) {
			var res []indexedEntry
			Expect(table.NewRetrieve().
				Where(scoreIdx.FilterRange(&gorp.Bound[int64]{Value: 10}, nil)).
				OrderBy(scoreIdx.Ordered(gorp.DirectionDesc).After(70, 7)).
				Limit(3).
				Entries(&res).Exec(ctx, idxDB)).To(Succeed())
			Expect(idsOf(res)).To(Equal([]int32{6, 5, 4}))
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

		It(
			"Should negate And(indexed, eval) without dropping the index constraint",
			func(ctx SpecContext) {
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

				aboveFifty := gorp.Match(
					func(_ gorp.Context, e *indexedEntry) (bool, error) {
						return e.Score > 50, nil
					},
				)
				// Not(And(name="a", score>50)) should exclude entries that
				// are BOTH name="a" AND score>50. Only ID=1 satisfies both.
				// IDs 2, 3, 4 should be returned.
				var res []indexedEntry
				Expect(table.NewRetrieve().
					Where(gorp.Not(gorp.And(nameIdx.Filter("a"), aboveFifty))).
					Entries(&res).Exec(ctx, idxDB)).To(Succeed())
				Expect(idsOf(res)).To(ConsistOf(int32(2), int32(3), int32(4)))
			},
		)
	})

	Describe("Where with OrderBy", func() {
		It(
			"Should apply an index-backed Where filter with OrderBy",
			func(ctx SpecContext) {
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
			},
		)
	})

	Describe("Commit maintenance", func() {
		var (
			commitDB *gorp.DB
			nameIdx  *gorp.LookupIndex[int32, indexedEntry, string]
			table    *gorp.Table[int32, indexedEntry]
		)
		BeforeEach(func(ctx SpecContext) {
			commitDB = DeferClose(wrapMsgpack(memkv.New()))
			nameIdx = gorp.NewLookupIndex(
				"name", func(e *indexedEntry) string { return e.Name },
			)
			table = DeferClose(openIndexedTable(ctx, commitDB, nameIdx))
		})

		It(
			"Should update the index via per-tx delta flush on commit",
			func(ctx SpecContext) {
				tx := commitDB.OpenTx()
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 1, Name: "alpha"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				Expect(nameIdx.Get(ctx, nil, "alpha")).To(ConsistOf(int32(1)))
			},
		)

		It(
			"Should leave the index untouched when the tx is closed without commit",
			func(ctx SpecContext) {
				tx := commitDB.OpenTx()
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 2, Name: "beta"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				Expect(nameIdx.Get(ctx, nil, "beta")).To(BeEmpty())
			},
		)

		It("Should update the index inline for DB-as-tx writes", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 3, Name: "gamma"}).
				Exec(ctx, commitDB)).To(Succeed())

			Expect(nameIdx.Get(ctx, nil, "gamma")).To(ConsistOf(int32(3)))
		})

		It(
			"Should remove deleted entries from the index on commit",
			func(ctx SpecContext) {
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 4, Name: "delta"}).
					Exec(ctx, commitDB)).To(Succeed())
				Expect(nameIdx.Get(ctx, nil, "delta")).To(ConsistOf(int32(4)))

				tx := commitDB.OpenTx()
				Expect(
					table.NewDelete().
						Where(gorp.MatchKeys[int32, indexedEntry](4)).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				Expect(nameIdx.Get(ctx, nil, "delta")).To(BeEmpty())
			},
		)

		It(
			"Should preserve committed entries when a delete is rolled back",
			func(ctx SpecContext) {
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 5, Name: "epsilon"}).
					Exec(ctx, commitDB)).To(Succeed())

				tx := commitDB.OpenTx()
				Expect(
					table.NewDelete().
						Where(gorp.MatchKeys[int32, indexedEntry](5)).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				Expect(nameIdx.Get(ctx, nil, "epsilon")).To(ConsistOf(int32(5)))
			},
		)

		It(
			"Should remove deleted entries inline for DB-as-tx deletes",
			func(ctx SpecContext) {
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 6, Name: "zeta"}).
					Exec(ctx, commitDB)).To(Succeed())
				Expect(nameIdx.Get(ctx, nil, "zeta")).To(ConsistOf(int32(6)))

				Expect(
					table.NewDelete().
						Where(gorp.MatchKeys[int32, indexedEntry](6)).
						Exec(ctx, commitDB),
				).To(Succeed())
				Expect(nameIdx.Get(ctx, nil, "zeta")).To(BeEmpty())
			},
		)

		It("Should index the sets of a raw writer on commit", func(ctx SpecContext) {
			tx := commitDB.OpenTx()
			Expect(gorp.WrapWriter[int32, indexedEntry](tx).
				Set(ctx, indexedEntry{ID: 7, Name: "eta"})).To(Succeed())
			Expect(nameIdx.Get(ctx, nil, "eta")).To(BeEmpty())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			Expect(nameIdx.Get(ctx, nil, "eta")).To(ConsistOf(int32(7)))
		})

		It("Should drop the deletes of a raw writer on commit", func(ctx SpecContext) {
			Expect(table.NewCreate().
				Entry(&indexedEntry{ID: 8, Name: "theta"}).
				Exec(ctx, commitDB)).To(Succeed())
			tx := commitDB.OpenTx()
			Expect(gorp.WrapWriter[int32, indexedEntry](tx).Delete(ctx, 8)).
				To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			Expect(nameIdx.Get(ctx, nil, "theta")).To(BeEmpty())
		})

		It(
			"Should drop the deletes of a query that no table binds",
			func(ctx SpecContext) {
				Expect(table.NewCreate().
					Entry(&indexedEntry{ID: 9, Name: "iota"}).
					Exec(ctx, commitDB)).To(Succeed())
				Expect(gorp.NewDelete[int32, indexedEntry]().
					Where(gorp.MatchKeys[int32, indexedEntry](9)).
					Exec(ctx, commitDB)).To(Succeed())
				Expect(nameIdx.Get(ctx, nil, "iota")).To(BeEmpty())
			},
		)

		It(
			"Should stop indexing raw writes once the table closes",
			func(ctx SpecContext) {
				Expect(table.Close()).To(Succeed())
				Expect(gorp.WrapWriter[int32, indexedEntry](commitDB).
					Set(ctx, indexedEntry{ID: 10, Name: "kappa"})).To(Succeed())
				Expect(nameIdx.Get(ctx, nil, "kappa")).To(BeEmpty())
			},
		)
	})
})

var _ = Describe("Composite-string-keyed Lookup", func() {
	var idxDB *gorp.DB
	BeforeEach(func() {
		idxDB = DeferClose(OpenGorpMsgpackDB())
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

			keys := MustSucceed(toIdx.Get(ctx, nil, "x"))
			asStrings := []string{string(keys[0]), string(keys[1])}
			Expect(asStrings).To(ConsistOf("a->x", "c->x"))
			Expect(toIdx.Get(ctx, nil, "y")).To(HaveLen(1))
			Expect(toIdx.Get(ctx, nil, "missing")).To(BeEmpty())
		})
	})

	Describe("Maintenance", func() {
		var toIdx *gorp.LookupIndex[string, relationEntry, string]
		BeforeEach(func(ctx SpecContext) {
			toIdx = gorp.NewLookupIndex(
				"to", func(e *relationEntry) string { return e.To },
			)
			DeferClose(openIndexedTable(ctx, idxDB, toIdx))
		})

		It("Should index newly created entries", func(ctx SpecContext) {
			Expect(gorp.NewCreate[string, relationEntry]().
				Entry(&relationEntry{From: "a", To: "x"}).
				Exec(ctx, idxDB)).To(Succeed())
			keys := MustSucceed(toIdx.Get(ctx, nil, "x"))
			Expect(keys).To(HaveLen(1))
			Expect(string(keys[0])).To(Equal("a->x"))
		})

		It("Should remove entries from the index on delete", func(ctx SpecContext) {
			e := relationEntry{From: "a", To: "x"}
			Expect(
				gorp.NewCreate[string, relationEntry]().Entry(&e).Exec(ctx, idxDB),
			).To(Succeed())
			Expect(toIdx.Get(ctx, nil, "x")).To(HaveLen(1))

			Expect(gorp.NewDelete[string, relationEntry]().
				Where(gorp.MatchKeys[string, relationEntry](e.GorpKey())).
				Exec(ctx, idxDB)).To(Succeed())
			Expect(toIdx.Get(ctx, nil, "x")).To(BeEmpty())
		})
	})

	Describe("Filter routing", func() {
		It(
			"Should route a Filter call through execKeys instead of execFilter",
			func(ctx SpecContext) {
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
			},
		)

		It(
			"Should return an empty result when no entries match",
			func(ctx SpecContext) {
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
			},
		)
	})

	// Tx delta visibility mirrors the Lookup test block: inserts, updates, deletes, and
	// rollback exercised against a LookupIndex over a composite-string-keyed entry.
	Describe("Tx delta visibility", func() {
		var (
			table *gorp.Table[string, relationEntry]
			toIdx *gorp.LookupIndex[string, relationEntry, string]
		)
		BeforeEach(func(ctx SpecContext) {
			toIdx = gorp.NewLookupIndex(
				"to", func(e *relationEntry) string { return e.To },
			)
			table = DeferClose(openIndexedTable(ctx, idxDB, toIdx))
			seed := []relationEntry{
				{From: "a", To: "x"},
				{From: "b", To: "y"},
			}
			Expect(table.NewCreate().
				Entries(&seed).Exec(ctx, idxDB)).To(Succeed())
		})

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
				Where(gorp.MatchKeys[string, relationEntry]("a->x")).
				Exec(ctx, tx)).To(Succeed())

			var res []relationEntry
			Expect(table.NewRetrieve().
				Where(toIdx.Filter("x")).
				Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(BeEmpty())
		})

		It(
			"Should drop the delta on rollback without touching the global index",
			func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				Expect(table.NewCreate().
					Entry(&relationEntry{From: "z", To: "x"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				keys := MustSucceed(toIdx.Get(ctx, nil, "x"))
				Expect(keys).To(HaveLen(1))
				Expect(string(keys[0])).To(Equal("a->x"))
			},
		)

		It("Should persist the staged insert on commit", func(ctx SpecContext) {
			tx := idxDB.OpenTx()
			Expect(table.NewCreate().
				Entry(&relationEntry{From: "q", To: "x"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			keys := MustSucceed(toIdx.Get(ctx, nil, "x"))
			asStrings := make([]string, len(keys))
			for i, k := range keys {
				asStrings[i] = string(k)
			}
			Expect(asStrings).To(ConsistOf("a->x", "q->x"))
		})

		// A graph walk that writes an edge and then filters on it in the same tx, such
		// as a cycle check, must see the write.
		It(
			"Should surface a newly-written relationship via byTo.Filter in the same tx",
			func(ctx SpecContext) {
				tx := idxDB.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()

				rel := relationEntry{From: "A", To: "B"}
				Expect(table.NewCreate().Entry(&rel).Exec(ctx, tx)).To(Succeed())

				var descendants []relationEntry
				Expect(table.NewRetrieve().
					Where(toIdx.Filter("B")).
					Entries(&descendants).Exec(ctx, tx)).To(Succeed())
				Expect(fromsOf(descendants)).To(ConsistOf("A"))
			},
		)
	})
})

var _ = Describe("Populate failure", func() {
	var (
		db       *gorp.DB
		nameIdx  *gorp.LookupIndex[int32, indexedEntry, string]
		scoreIdx *gorp.SortedIndex[int32, indexedEntry, int64]
		logs     *observer.ObservedLogs
	)
	BeforeEach(func(ctx SpecContext) {
		mem := DeferClose(memkv.New())
		db = gorp.Wrap(
			&failOnceDB{DB: mem, err: errors.New("populate boom")},
			gorp.WithCodec(msgpack.Codec),
		)
		seed := []indexedEntry{
			{ID: 1, Name: "alpha", Score: 10},
			{ID: 2, Name: "beta", Score: 20},
			{ID: 3, Name: "alpha", Score: 30},
		}
		Expect(gorp.NewCreate[int32, indexedEntry]().
			Entries(&seed).Exec(ctx, db)).To(Succeed())
		nameIdx = gorp.NewLookupIndex(
			"name", func(e *indexedEntry) string { return e.Name },
		)
		scoreIdx = gorp.NewSortedIndex(
			"score", func(e *indexedEntry) int64 { return e.Score },
		)
		var ins alamos.Instrumentation
		ins, logs = ObservedInstrumentation(zapcore.ErrorLevel)
		DeferClose(
			MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[int32, indexedEntry]{
				DB:              db,
				Indexes:         []gorp.Index[int32, indexedEntry]{nameIdx, scoreIdx},
				Instrumentation: ins,
			})),
		)
		Expect(nameIdx.Get(ctx, nil, "alpha")).Error().
			To(MatchError(gorp.ErrIndexInvalid))
	})

	It("Should log the populate error", func() {
		failed := logs.FilterMessageSnippet("populate failed").All()
		Expect(failed).To(HaveLen(1))
		Expect(
			failed[0].ContextMap()["error"],
		).To(ContainSubstring("populate boom"))
	})

	It("Should fall back to a sequential scan from Retrieve via Filter",
		func(ctx SpecContext) {
			var got []indexedEntry
			Expect(gorp.NewRetrieve[int32, indexedEntry]().
				Where(nameIdx.Filter("alpha")).
				Entries(&got).Exec(ctx, db)).To(Succeed())
			Expect(idsOf(got)).To(ConsistOf(int32(1), int32(3)))
		})

	It(
		"Should preserve a MatchKeys sibling when the indexed child fails to populate",
		func(ctx SpecContext) {
			var got []indexedEntry
			Expect(gorp.NewRetrieve[int32, indexedEntry]().
				Where(gorp.And(
					nameIdx.Filter("alpha"),
					gorp.MatchKeys[int32, indexedEntry](1),
				)).
				Entries(&got).Exec(ctx, db)).To(Succeed())
			Expect(idsOf(got)).To(ConsistOf(int32(1)))
		},
	)

	It("Should fall back to a sequential scan for Or when populate fails",
		func(ctx SpecContext) {
			// The "gamma" branch matches nothing, so a fallback that matched every
			// entry would include the "beta" entry.
			var got []indexedEntry
			Expect(gorp.NewRetrieve[int32, indexedEntry]().
				Where(gorp.Or(
					nameIdx.Filter("alpha"),
					nameIdx.Filter("gamma"),
				)).
				Entries(&got).Exec(ctx, db)).To(Succeed())
			Expect(idsOf(got)).To(ConsistOf(int32(1), int32(3)))
		})

	It(
		"Should fall back to a sequential scan for a sorted range",
		func(ctx SpecContext) {
			var got []indexedEntry
			Expect(gorp.NewRetrieve[int32, indexedEntry]().
				Where(scoreIdx.FilterRange(&gorp.Bound[int64]{Value: 15}, nil)).
				Entries(&got).Exec(ctx, db)).To(Succeed())
			Expect(idsOf(got)).To(ConsistOf(int32(2), int32(3)))
		},
	)

	It(
		"Should narrow an And of ranges by the range whose index populated",
		func(ctx SpecContext) {
			healthy := gorp.NewSortedIndex(
				"score", func(e *indexedEntry) int64 { return e.Score },
			)
			DeferClose(
				MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[int32, indexedEntry]{
					DB:      db,
					Name:    "healthy",
					Indexes: []gorp.Index[int32, indexedEntry]{healthy},
				})),
			)
			Expect(healthy.Get(ctx, nil, 20)).To(ConsistOf(int32(2)))
			read := 0
			var got []indexedEntry
			Expect(gorp.NewRetrieve[int32, indexedEntry]().
				Where(gorp.And(
					gorp.Match(func(gorp.Context, *indexedEntry) (bool, error) {
						read++
						return true, nil
					}),
					scoreIdx.FilterRange(&gorp.Bound[int64]{Value: 15}, nil),
					healthy.FilterRange(nil, &gorp.Bound[int64]{Value: 25}),
				)).
				Entries(&got).Exec(ctx, db)).To(Succeed())
			Expect(idsOf(got)).To(ConsistOf(int32(2)))
			Expect(read).To(Equal(2))
		},
	)

	It("Should fall back to a sequential scan for Not when populate fails",
		func(ctx SpecContext) {
			// A fallback that matched every entry would make Not match none.
			var got []indexedEntry
			Expect(gorp.NewRetrieve[int32, indexedEntry]().
				Where(gorp.Not(nameIdx.Filter("alpha"))).
				Entries(&got).Exec(ctx, db)).To(Succeed())
			Expect(idsOf(got)).To(ConsistOf(int32(2)))
		})
})
