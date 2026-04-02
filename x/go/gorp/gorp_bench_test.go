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
	"strings"
	"testing"

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

// benchEntry uses a numeric (int32) key, exercising big-endian byte reversal
// encoding on every operation.
type benchEntry struct {
	ID   int32
	Data string
}

func (e benchEntry) GorpKey() int32    { return e.ID }
func (e benchEntry) SetOptions() []any { return nil }

// benchStringEntry uses a string key, exercising direct concatenation encoding.
type benchStringEntry struct {
	ID   string
	Data string
}

func (e benchStringEntry) GorpKey() string   { return e.ID }
func (e benchStringEntry) SetOptions() []any { return nil }

// benchLargeEntry has a realistically-sized value body (~1KB) to make
// serialization cost visible relative to key encoding overhead.
type benchLargeEntry struct {
	ID          int32
	Name        string
	Description string
	Tags        [10]string
	Value       float64
}

func (e benchLargeEntry) GorpKey() int32    { return e.ID }
func (e benchLargeEntry) SetOptions() []any { return nil }

func setupBenchDB(b *testing.B) *gorp.DB {
	b.Helper()
	db := gorp.Wrap(memkv.New())
	b.Cleanup(func() {
		if err := db.Close(); err != nil {
			b.Fatal(err)
		}
	})
	return db
}

func makeBenchEntries(n int) []benchEntry {
	entries := make([]benchEntry, n)
	for i := range entries {
		entries[i] = benchEntry{ID: int32(i), Data: "data-" + strconv.Itoa(i)}
	}
	return entries
}

func makeBenchStringEntries(n int) []benchStringEntry {
	entries := make([]benchStringEntry, n)
	for i := range entries {
		entries[i] = benchStringEntry{
			ID:   fmt.Sprintf("key-%06d", i),
			Data: "data-" + strconv.Itoa(i),
		}
	}
	return entries
}

func makeBenchLargeEntries(n int) []benchLargeEntry {
	entries := make([]benchLargeEntry, n)
	for i := range entries {
		var tags [10]string
		for j := range tags {
			tags[j] = fmt.Sprintf("tag-%d-%d", i, j)
		}
		entries[i] = benchLargeEntry{
			ID:          int32(i),
			Name:        fmt.Sprintf("entry-%d", i),
			Description: strings.Repeat("x", 512),
			Tags:        tags,
			Value:       float64(i) * 1.5,
		}
	}
	return entries
}

// populateEntries is a helper that creates entries in the DB and returns the
// keys. It calls b.Fatal on error so callers don't need error handling.
func populateEntries(b *testing.B, db *gorp.DB, n int) []int32 {
	b.Helper()
	entries := makeBenchEntries(n)
	if err := gorp.NewCreate[int32, benchEntry]().Entries(&entries).Exec(
		context.Background(), db,
	); err != nil {
		b.Fatal(err)
	}
	keys := make([]int32, n)
	for i := range keys {
		keys[i] = int32(i)
	}
	return keys
}

func populateStringEntries(b *testing.B, db *gorp.DB, n int) []string {
	b.Helper()
	entries := makeBenchStringEntries(n)
	if err := gorp.NewCreate[string, benchStringEntry]().Entries(&entries).Exec(
		context.Background(), db,
	); err != nil {
		b.Fatal(err)
	}
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%06d", i)
	}
	return keys
}

// --- Create Benchmarks ---

var crudSizes = []int{1, 10, 100, 1000}

func BenchmarkCreate(b *testing.B) {
	b.Run("int32_key", func(b *testing.B) {
		for _, size := range crudSizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				db := setupBenchDB(b)
				ctx := context.Background()
				entries := makeBenchEntries(size)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Use offset keys so each iteration inserts fresh entries
					// rather than overwriting the same keys.
					e := make([]benchEntry, len(entries))
					for j := range e {
						e[j] = benchEntry{
							ID:   int32(i*size+j) + 1,
							Data: entries[j].Data,
						}
					}
					if err := gorp.NewCreate[int32, benchEntry]().Entries(&e).Exec(ctx, db); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
	b.Run("string_key", func(b *testing.B) {
		for _, size := range crudSizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				db := setupBenchDB(b)
				ctx := context.Background()
				base := makeBenchStringEntries(size)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					e := make([]benchStringEntry, len(base))
					for j := range e {
						e[j] = benchStringEntry{
							ID:   fmt.Sprintf("key-%d-%06d", i, j),
							Data: base[j].Data,
						}
					}
					if err := gorp.NewCreate[string, benchStringEntry]().Entries(&e).Exec(ctx, db); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
	b.Run("large_value", func(b *testing.B) {
		for _, size := range []int{1, 10, 100} {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				db := setupBenchDB(b)
				ctx := context.Background()
				base := makeBenchLargeEntries(size)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					e := make([]benchLargeEntry, len(base))
					for j := range e {
						e[j] = base[j]
						e[j].ID = int32(i*size+j) + 1
					}
					if err := gorp.NewCreate[int32, benchLargeEntry]().Entries(&e).Exec(ctx, db); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
	b.Run("merge_existing", func(b *testing.B) {
		for _, size := range []int{1, 10, 100} {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				db := setupBenchDB(b)
				ctx := context.Background()
				populateEntries(b, db, size)
				entries := makeBenchEntries(size)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					e := make([]benchEntry, len(entries))
					copy(e, entries)
					if err := gorp.NewCreate[int32, benchEntry]().
						MergeExisting(func(_ gorp.Context, creating, existing benchEntry) (benchEntry, error) {
							creating.Data = existing.Data + "-merged"
							return creating, nil
						}).
						Entries(&e).
						Exec(ctx, db); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
}

// --- Retrieve Benchmarks ---

func BenchmarkRetrieveByKeys(b *testing.B) {
	b.Run("int32_key", func(b *testing.B) {
		for _, size := range crudSizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				db := setupBenchDB(b)
				ctx := context.Background()
				keys := populateEntries(b, db, size)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var results []benchEntry
					if err := gorp.NewRetrieve[int32, benchEntry]().
						WhereKeys(keys...).
						Entries(&results).
						Exec(ctx, db); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
	b.Run("string_key", func(b *testing.B) {
		for _, size := range crudSizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				db := setupBenchDB(b)
				ctx := context.Background()
				keys := populateStringEntries(b, db, size)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var results []benchStringEntry
					if err := gorp.NewRetrieve[string, benchStringEntry]().
						WhereKeys(keys...).
						Entries(&results).
						Exec(ctx, db); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
}

func BenchmarkRetrieveByFilter(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("dbSize=%d", size), func(b *testing.B) {
			db := setupBenchDB(b)
			ctx := context.Background()
			populateEntries(b, db, size)
			half := int32(size / 2)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var results []benchEntry
				if err := gorp.NewRetrieve[int32, benchEntry]().
					Where(func(_ gorp.Context, e *benchEntry) (bool, error) {
						return e.ID < half, nil
					}).
					Entries(&results).
					Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkRetrieveExists(b *testing.B) {
	b.Run("present", func(b *testing.B) {
		db := setupBenchDB(b)
		ctx := context.Background()
		populateEntries(b, db, 1000)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := gorp.NewRetrieve[int32, benchEntry]().
				WhereKeys(500).
				Exists(ctx, db); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("absent", func(b *testing.B) {
		db := setupBenchDB(b)
		ctx := context.Background()
		populateEntries(b, db, 1000)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := gorp.NewRetrieve[int32, benchEntry]().
				WhereKeys(9999).
				Exists(ctx, db); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// --- Update Benchmarks ---

func BenchmarkUpdateByKeys(b *testing.B) {
	for _, size := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := setupBenchDB(b)
			ctx := context.Background()
			keys := populateEntries(b, db, size)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				suffix := strconv.Itoa(i)
				if err := gorp.NewUpdate[int32, benchEntry]().
					WhereKeys(keys...).
					Change(func(_ gorp.Context, entry benchEntry) benchEntry {
						entry.Data = "updated-" + suffix
						return entry
					}).
					Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// --- Delete Benchmarks ---

func BenchmarkDeleteByKeys(b *testing.B) {
	for _, size := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			db := setupBenchDB(b)
			ctx := context.Background()
			base := makeBenchEntries(size)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// Each iteration creates fresh entries with unique keys,
				// then deletes them. This avoids StopTimer/StartTimer which
				// distorts benchmark calibration.
				e := make([]benchEntry, len(base))
				for j := range e {
					e[j] = benchEntry{
						ID:   int32(i*size + j),
						Data: base[j].Data,
					}
				}
				if err := gorp.NewCreate[int32, benchEntry]().Entries(&e).Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
				keys := make([]int32, size)
				for j := range keys {
					keys[j] = int32(i*size + j)
				}
				if err := gorp.NewDelete[int32, benchEntry]().
					WhereKeys(keys...).
					Exec(ctx, db); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// --- Transaction Overhead Benchmarks ---

func BenchmarkWithTx(b *testing.B) {
	b.Run("with_tx", func(b *testing.B) {
		db := setupBenchDB(b)
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := int32(i % 10000)
			if err := db.WithTx(ctx, func(tx gorp.Tx) error {
				entry := benchEntry{ID: key, Data: "data"}
				if err := gorp.NewCreate[int32, benchEntry]().Entry(&entry).Exec(ctx, tx); err != nil {
					return err
				}
				var result benchEntry
				return gorp.NewRetrieve[int32, benchEntry]().
					WhereKeys(key).
					Entry(&result).
					Exec(ctx, tx)
			}); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("without_tx", func(b *testing.B) {
		db := setupBenchDB(b)
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := int32(i % 10000)
			entry := benchEntry{ID: key, Data: "data"}
			if err := gorp.NewCreate[int32, benchEntry]().Entry(&entry).Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
			var result benchEntry
			if err := gorp.NewRetrieve[int32, benchEntry]().
				WhereKeys(key).
				Entry(&result).
				Exec(ctx, db); err != nil {
				b.Fatal(err)
			}
		}
	})
}
