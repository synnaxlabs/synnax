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

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

// indexBenchEntry is a small entry with two indexable fields and one
// orderable field. The shape mirrors what production tables look like:
// a numeric primary key, a string lookup field, and an int64 sort field.
type indexBenchEntry struct {
	ID    int32
	Name  string
	Score int64
}

func (e indexBenchEntry) GorpKey() int32    { return e.ID }
func (e indexBenchEntry) SetOptions() []any { return nil }

func makeIndexBenchEntries(n int) []indexBenchEntry {
	entries := make([]indexBenchEntry, n)
	for i := range entries {
		entries[i] = indexBenchEntry{
			ID:    int32(i),
			Name:  "name-" + strconv.Itoa(i),
			Score: int64(i),
		}
	}
	return entries
}

// indexSizes spans the range from "small enough to fit in cache" to "large
// enough that scan vs index has a meaningful asymptotic gap". 100k is the
// upper bound called out in the gorp index RFC.
var indexSizes = []int{100, 1_000, 10_000, 100_000}

// --- Lookup index ---

// BenchmarkLookupSetup measures the cost of opening a Table with a Lookup
// index registered, populating it from N pre-existing entries, and tearing
// it down. This is the cold-start path for every Service that owns a lookup
// index.
func BenchmarkLookupSetup(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := gorp.Wrap(memkv.New())
			defer func() { _ = db.Close() }()
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				nameIdx := gorp.NewLookup[int32, indexBenchEntry, string](
					"name", func(e *indexBenchEntry) string { return e.Name },
				)
				table, err := gorp.OpenTable[int32, indexBenchEntry](
					ctx,
					gorp.TableConfig[int32, indexBenchEntry]{
						DB:      db,
						Indexes: []gorp.Index[int32, indexBenchEntry]{nameIdx},
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				_ = table.Close()
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
			db := gorp.Wrap(memkv.New())
			defer func() { _ = db.Close() }()
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			nameIdx := gorp.NewLookup[int32, indexBenchEntry, string](
				"name", func(e *indexBenchEntry) string { return e.Name },
			)
			table, err := gorp.OpenTable[int32, indexBenchEntry](
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{
					DB:      db,
					Indexes: []gorp.Index[int32, indexBenchEntry]{nameIdx},
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = table.Close() }()
			target := "name-" + strconv.Itoa(size/2)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
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
			db := gorp.Wrap(memkv.New())
			defer func() { _ = db.Close() }()
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			table, err := gorp.OpenTable[int32, indexBenchEntry](
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{DB: db},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = table.Close() }()
			target := "name-" + strconv.Itoa(size/2)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
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

// BenchmarkLookupObserverUpdate measures the per-update cost paid by the
// observer when an entry is created in a table whose Lookup index is
// already populated with N rows. This is the hot path for every channel
// rename / user update / etc.
func BenchmarkLookupObserverUpdate(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := gorp.Wrap(memkv.New())
			defer func() { _ = db.Close() }()
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			nameIdx := gorp.NewLookup[int32, indexBenchEntry, string](
				"name", func(e *indexBenchEntry) string { return e.Name },
			)
			table, err := gorp.OpenTable[int32, indexBenchEntry](
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{
					DB:      db,
					Indexes: []gorp.Index[int32, indexBenchEntry]{nameIdx},
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = table.Close() }()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
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

// BenchmarkSortedSetup is the cold-start path for a Sorted index. Asserts
// that bulk-load + sort-once is O(N log N), not O(N²) per insert. Without
// the bulk-load optimization, n=100000 would take many seconds.
func BenchmarkSortedSetup(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := gorp.Wrap(memkv.New())
			defer func() { _ = db.Close() }()
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				scoreIdx := gorp.NewSorted[int32, indexBenchEntry, int64](
					"score", func(e *indexBenchEntry) int64 { return e.Score },
				)
				table, err := gorp.OpenTable[int32, indexBenchEntry](
					ctx,
					gorp.TableConfig[int32, indexBenchEntry]{
						DB:      db,
						Indexes: []gorp.Index[int32, indexBenchEntry]{scoreIdx},
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				_ = table.Close()
			}
		})
	}
}

// BenchmarkSortedObserverUpdate measures the per-update cost paid by the
// observer when an entry is created in a table whose Sorted index is
// already populated with N rows. The new entries land in the middle of the
// score range to force the worst-case slice-shift cost in sortedStorage.put,
// which is the operation a B-tree replacement would speed up.
func BenchmarkSortedObserverUpdate(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := gorp.Wrap(memkv.New())
			defer func() { _ = db.Close() }()
			ctx := context.Background()
			entries := makeIndexBenchEntries(size)
			if err := gorp.NewCreate[int32, indexBenchEntry]().
				Entries(&entries).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			scoreIdx := gorp.NewSorted[int32, indexBenchEntry, int64](
				"score", func(e *indexBenchEntry) int64 { return e.Score },
			)
			table, err := gorp.OpenTable[int32, indexBenchEntry](
				ctx,
				gorp.TableConfig[int32, indexBenchEntry]{
					DB:      db,
					Indexes: []gorp.Index[int32, indexBenchEntry]{scoreIdx},
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = table.Close() }()
			midScore := int64(size / 2)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
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
