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
	"fmt"
	"strconv"
	"testing"

	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

// indexBenchEntry is a small entry with two indexable fields and one
// orderable field. The shape mirrors what production tables look like:
// a numeric primary key, a string lookup field, and an int64 sort field.
// Category is a low-cardinality field used by the composition benchmarks
// so that intersection / union have meaningful selectivity to measure.
type indexBenchEntry struct {
	ID       int32
	Name     string
	Category string
	Score    int64
}

func (e indexBenchEntry) GorpKey() int32 { return e.ID }

// indexBenchCategoryCount caps Category cardinality so each value matches
// roughly N/16 entries — wide enough that intersect/union do real work but
// narrow enough that the result fits comfortably in cache.
const indexBenchCategoryCount = 16

func makeIndexBenchEntries(n int) []indexBenchEntry {
	entries := make([]indexBenchEntry, n)
	for i := range entries {
		entries[i] = indexBenchEntry{
			ID:       int32(i),
			Name:     "name-" + strconv.Itoa(i),
			Category: "cat-" + strconv.Itoa(i%indexBenchCategoryCount),
			Score:    int64(i),
		}
	}
	return entries
}

// indexSizes spans the range from "small enough to fit in cache" to "large
// enough that scan vs index has a meaningful asymptotic gap". 100k is the
// upper bound called out in the gorp index RFC.
var indexSizes = []int{100, 1_000, 10_000, 100_000}

// --- Lookup index ---

// BenchmarkLookupSetup measures the cost of opening a Table with a LookupIndex
// index registered, populating it from N pre-existing entries, and tearing
// it down. This is the cold-start path for every Service that owns a lookup
// index.
func BenchmarkLookupSetup(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := openBenchDB(b)
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				nameIdx := gorp.NewLookupIndex(
					"name", func(e *indexBenchEntry) string { return e.Name },
				)
				table, err := gorp.OpenTable(
					ctx,
					gorp.TableConfig[int32, indexBenchEntry]{
						DB:      db,
						Indexes: []gorp.Index[int32, indexBenchEntry]{nameIdx},
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				if err := table.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkLookupQueryViaIndex compares an index-backed exact-match query
// against a full-table scan via gorp.Match. Indexed should be O(1)-ish
// regardless of N; the scan grows linearly.
func BenchmarkLookupQueryViaIndex(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := openBenchDB(b)
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			nameIdx := gorp.NewLookupIndex(
				"name", func(e *indexBenchEntry) string { return e.Name },
			)
			table, err := gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{
					DB:      db,
					Indexes: []gorp.Index[int32, indexBenchEntry]{nameIdx},
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			target := "name-" + strconv.Itoa(size/2)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var out []indexBenchEntry
				if err := table.NewRetrieve().
					Where(nameIdx.Filter(target)).
					Entries(&out).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
				if len(out) != 1 {
					b.Fatalf("expected 1 result, got %d", len(out))
				}
			}
		})
	}
}

// BenchmarkLookupQueryViaScan is the baseline: same query as
// BenchmarkLookupQueryViaIndex but routed through gorp.Match instead of an
// index. Used to quantify how much the index actually buys us.
func BenchmarkLookupQueryViaScan(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := openBenchDB(b)
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			table, err := gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{DB: db},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			target := "name-" + strconv.Itoa(size/2)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var out []indexBenchEntry
				if err := table.NewRetrieve().
					Where(gorp.Match(func(_ gorp.Context, e *indexBenchEntry) (bool, error) {
						return e.Name == target, nil
					})).
					Entries(&out).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
				if len(out) != 1 {
					b.Fatalf("expected 1 result, got %d", len(out))
				}
			}
		})
	}
}

// BenchmarkLookupUpdate measures the cost of one create in a table whose LookupIndex
// holds N entries. Every channel rename and user update takes this path.
func BenchmarkLookupUpdate(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := openBenchDB(b)
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			nameIdx := gorp.NewLookupIndex(
				"name", func(e *indexBenchEntry) string { return e.Name },
			)
			table, err := gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{
					DB:      db,
					Indexes: []gorp.Index[int32, indexBenchEntry]{nameIdx},
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				e := indexBenchEntry{
					ID:    int32(size + i),
					Name:  "new-" + strconv.Itoa(i),
					Score: int64(size + i),
				}
				if err := gorp.NewCreate[int32, indexBenchEntry]().
					Entry(&e).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// --- Sorted index ---

// BenchmarkSortedSetup is the cold-start path for a SortedIndex index. Asserts
// that bulk-load + sort-once is O(N log N), not O(N²) per insert. Without
// the bulk-load optimization, n=100000 would take many seconds.
func BenchmarkSortedSetup(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := openBenchDB(b)
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				scoreIdx := gorp.NewSortedIndex(
					"score", func(e *indexBenchEntry) int64 { return e.Score },
				)
				table, err := gorp.OpenTable(
					ctx,
					gorp.TableConfig[int32, indexBenchEntry]{
						DB:      db,
						Indexes: []gorp.Index[int32, indexBenchEntry]{scoreIdx},
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				if err := table.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSortedUpdate measures the cost of one create in a table whose SortedIndex
// holds N entries. The new entries land in the middle of the score range, which forces
// the worst-case slice shift in sortedStorage.put.
func BenchmarkSortedUpdate(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := openBenchDB(b)
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			scoreIdx := gorp.NewSortedIndex(
				"score", func(e *indexBenchEntry) int64 { return e.Score },
			)
			table, err := gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{
					DB:      db,
					Indexes: []gorp.Index[int32, indexBenchEntry]{scoreIdx},
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			midScore := int64(size / 2)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				e := indexBenchEntry{
					ID:    int32(size + i),
					Name:  "new-" + strconv.Itoa(i),
					Score: midScore,
				}
				if err := gorp.NewCreate[int32, indexBenchEntry]().
					Entry(&e).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// --- Composition (And / Or) ---

// openBenchDB opens a DB and runs it until the benchmark ends.
func openBenchDB(b *testing.B) *gorp.DB {
	b.Helper()
	db := OpenGorpMsgpackDB()
	stop := testutil.GoRun(b.Context(), db.Run)
	b.Cleanup(func() {
		if err := errors.Join(stop(), db.Close()); err != nil {
			b.Error(err)
		}
	})
	return db
}

// compositionFixture spins up a Table with two LookupIndex indexes (name +
// category) populated from N entries and returns the table plus the indexes
// for use by composition benchmarks. Caller must defer table.Close().
func compositionFixture(b *testing.B, size int) (
	*gorp.DB,
	*gorp.Table[int32, indexBenchEntry],
	*gorp.LookupIndex[int32, indexBenchEntry, string],
	*gorp.LookupIndex[int32, indexBenchEntry, string],
) {
	b.Helper()
	db := openBenchDB(b)
	ctx := context.Background()
	entries := makeIndexBenchEntries(size)
	if err := gorp.NewCreate[int32, indexBenchEntry]().
		Entries(&entries).Exec(ctx, db); err != nil {
		b.Fatal(err)
	}
	nameIdx := gorp.NewLookupIndex(
		"name", func(e *indexBenchEntry) string { return e.Name },
	)
	categoryIdx := gorp.NewLookupIndex(
		"category", func(e *indexBenchEntry) string { return e.Category },
	)
	table, err := gorp.OpenTable(
		ctx,
		gorp.TableConfig[int32, indexBenchEntry]{
			DB: db,
			Indexes: []gorp.Index[int32, indexBenchEntry]{
				nameIdx, categoryIdx,
			},
		},
	)
	if err != nil {
		b.Fatal(err)
	}
	return db, table, nameIdx, categoryIdx
}

// BenchmarkComposeAndIndexed measures Where(And(IndexA, IndexB)) where one
// child is a narrow name lookup (1 entry) and the other is a wider category
// lookup (~N/16 entries). The intersection should walk the smaller side via
// the slices.SortFunc-by-Keys-length step in intersectKeys, so the per-call
// cost stays bounded by the narrow side regardless of N.
func BenchmarkComposeAndIndexed(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db, table, nameIdx, categoryIdx := compositionFixture(b, size)
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			ctx := context.Background()
			targetName := "name-" + strconv.Itoa(size/2)
			targetCat := "cat-" + strconv.Itoa((size/2)%indexBenchCategoryCount)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var out []indexBenchEntry
				if err := table.NewRetrieve().
					Where(gorp.And(
						nameIdx.Filter(targetName),
						categoryIdx.Filter(targetCat),
					)).
					Entries(&out).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
				if len(out) != 1 {
					b.Fatalf("expected 1 result, got %d", len(out))
				}
			}
		})
	}
}

// BenchmarkComposeAndScan is the baseline for BenchmarkComposeAndIndexed:
// same logical query but routed through gorp.Match closures so it falls
// through to a full table scan. The gap to the indexed version measures the
// real-world payoff of intersectKeys' membership-probe path.
func BenchmarkComposeAndScan(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := openBenchDB(b)
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			table, err := gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{DB: db},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			targetName := "name-" + strconv.Itoa(size/2)
			targetCat := "cat-" + strconv.Itoa((size/2)%indexBenchCategoryCount)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var out []indexBenchEntry
				if err := table.NewRetrieve().
					Where(gorp.And(
						gorp.Match(
							func(_ gorp.Context, e *indexBenchEntry) (bool, error) {
								return e.Name == targetName, nil
							},
						),
						gorp.Match(
							func(_ gorp.Context, e *indexBenchEntry) (bool, error) {
								return e.Category == targetCat, nil
							},
						),
					)).
					Entries(&out).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
				if len(out) != 1 {
					b.Fatalf("expected 1 result, got %d", len(out))
				}
			}
		})
	}
}

// BenchmarkComposeOrIndexed measures Where(Or(IndexA, IndexB)) over two
// narrow name lookups. unionKeys must dedupe across children via the typed
// membership probes; if dedupe fell back to a linear scan or any-boxing,
// per-call cost would scale with N.
func BenchmarkComposeOrIndexed(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db, table, nameIdx, _ := compositionFixture(b, size)
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			ctx := context.Background()
			targetA := "name-" + strconv.Itoa(size/4)
			targetB := "name-" + strconv.Itoa((3*size)/4)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var out []indexBenchEntry
				if err := table.NewRetrieve().
					Where(gorp.Or(
						nameIdx.Filter(targetA),
						nameIdx.Filter(targetB),
					)).
					Entries(&out).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
				if len(out) != 2 {
					b.Fatalf("expected 2 results, got %d", len(out))
				}
			}
		})
	}
}

// BenchmarkComposeOrIndexedShared mirrors BenchmarkComposeOrIndexed but
// constructs the Or filter once outside the loop. That isolates the per-
// Exec cost the resolver path pays now that Or materializes children
// fresh on every call (instead of mutating shared captured state).
func BenchmarkComposeOrIndexedShared(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db, table, nameIdx, _ := compositionFixture(b, size)
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			ctx := context.Background()
			targetA := "name-" + strconv.Itoa(size/4)
			targetB := "name-" + strconv.Itoa((3*size)/4)
			shared := gorp.Or(
				nameIdx.Filter(targetA),
				nameIdx.Filter(targetB),
			)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var out []indexBenchEntry
				if err := table.NewRetrieve().
					Where(shared).
					Entries(&out).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
				if len(out) != 2 {
					b.Fatalf("expected 2 results, got %d", len(out))
				}
			}
		})
	}
}

// BenchmarkComposeNestedAnd exercises the membership-propagation path that
// previously dropped every key: an inner And produces a Filter whose Keys
// must be re-wrapped in a typed membership predicate so the outer And can
// probe it. Without rebuildMembership, this benchmark would either return
// no results or fall through to execFilter and scale linearly with N.
func BenchmarkComposeNestedAnd(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db, table, nameIdx, categoryIdx := compositionFixture(b, size)
			defer func() {
				if err := table.Close(); err != nil {
					b.Error(err)
				}
			}()
			ctx := context.Background()
			targetName := "name-" + strconv.Itoa(size/2)
			targetCat := "cat-" + strconv.Itoa((size/2)%indexBenchCategoryCount)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				inner := gorp.And(
					nameIdx.Filter(targetName),
					categoryIdx.Filter(targetCat),
				)
				outer := gorp.And(inner, nameIdx.Filter(targetName))
				var out []indexBenchEntry
				if err := table.NewRetrieve().
					Where(outer).
					Entries(&out).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
				if len(out) != 1 {
					b.Fatalf("expected 1 result, got %d", len(out))
				}
			}
		})
	}
}

// --- Dependents ---

type dependentsBenchHolder struct {
	Key     int32
	Targets []int32
}

func (h dependentsBenchHolder) GorpKey() int32 { return h.Key }

// dependentsBenchRefs is the number of references that point at the target table, the
// count a channel has across its holder tables.
const dependentsBenchRefs = 30

// newDependentsBenchRefs returns dependentsBenchRefs weak references to the target
// table. Holder h holds its targets through reference h % dependentsBenchRefs.
func newDependentsBenchRefs() []gorp.Reference[int32, dependentsBenchHolder] {
	refs := make([]gorp.Reference[int32, dependentsBenchHolder], dependentsBenchRefs)
	for i := range refs {
		ref := int32(i)
		refs[i] = gorp.NewKeyReference(
			gorp.KeyReferenceConfig[int32, dependentsBenchHolder, int32]{
				Name:     "targets_" + strconv.Itoa(i),
				Table:    "target",
				OnDelete: gorp.Weak,
				Kind:     gorp.Use,
				Targets: func(h *dependentsBenchHolder) []int32 {
					if h.Key%dependentsBenchRefs != ref {
						return nil
					}
					return h.Targets
				},
			},
		)
	}
	return refs
}

// BenchmarkDependents measures Dependents for a page of 100 targets that
// dependentsBenchRefs weak references point at. Each target has 3 holders, and each
// holder holds through one reference. Compare it against
// BenchmarkRetrieveByKeys/int32_key/n=100.
func BenchmarkDependents(b *testing.B) {
	const (
		pageSize = 100
		holders  = 3 * pageSize
	)
	db := openBenchDB(b)
	ctx := b.Context()
	targets, err := gorp.OpenTable(ctx, gorp.TableConfig[int32, benchEntry]{
		DB:     db,
		Name:   "target",
		Issued: func(int32) bool { return true },
	})
	if err != nil {
		b.Fatal(err)
	}
	holderTable, err := gorp.OpenTable(
		ctx,
		gorp.TableConfig[int32, dependentsBenchHolder]{
			DB:         db,
			Name:       "holder",
			References: newDependentsBenchRefs(),
		},
	)
	if err != nil {
		b.Fatal(err)
	}
	entries := make([]dependentsBenchHolder, holders)
	for i := range entries {
		entries[i] = dependentsBenchHolder{
			Key:     int32(i),
			Targets: []int32{int32(i % pageSize)},
		}
	}
	if err = holderTable.NewCreate().Entries(&entries).Exec(ctx, db); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for k := range int32(pageSize) {
			if _, err = targets.Dependents(ctx, db, k); err != nil {
				b.Fatal(err)
			}
		}
	}
}
