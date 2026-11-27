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
	"fmt"
	"testing"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// staleBatch is the number of iterators that BenchmarkIteratorStale holds on a domain
// before each set of cuts.
const staleBatch = 1 << 14

// BenchmarkIteratorPass steps an iterator over every domain of a DB, forward and
// backward. One op is one step.
func BenchmarkIteratorPass(b *testing.B) {
	for _, n := range []int{1_000, 100_000} {
		b.Run(fmt.Sprintf("domains=%d", n), func(b *testing.B) {
			db := openBenchDB(b)
			defer closeBenchDB(b, db)
			for i := range n {
				writeBenchDomain(b, db, telem.TimeStamp(i+1)*telem.SecondTS)
			}
			benchmarkPass(b, db, n)
		})
	}
}

// BenchmarkIteratorCutPass steps an iterator across one domain that k deletes cut into
// k+1 pieces, with a write in each cut. One op is one step.
func BenchmarkIteratorCutPass(b *testing.B) {
	for _, k := range []int{10, 100, 1_000} {
		b.Run(fmt.Sprintf("cuts=%d", k), func(b *testing.B) {
			db := openBenchDB(b)
			defer closeBenchDB(b, db)
			writeCutDomain(b, db, k)
			cutDomain(b, db, k)
			benchmarkPass(b, db, 2*k+3)
		})
	}
}

// BenchmarkIteratorStale steps an iterator from a domain that k deletes cut into k+1
// pieces, with a write in each cut, after the iterator reached the domain. One op is
// one
// step.
func BenchmarkIteratorStale(b *testing.B) {
	for _, k := range []int{10, 100, 1_000} {
		for _, dir := range []string{"next", "prev"} {
			b.Run(fmt.Sprintf("cuts=%d/%s", k, dir), func(b *testing.B) {
				benchmarkStale(b, k, dir == "next")
			})
		}
	}
}

func benchmarkPass(b *testing.B, db *domain.DB, count int) {
	b.Helper()
	ctx := b.Context()
	i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
	defer func() {
		if err := i.Close(); err != nil {
			b.Error(err)
		}
	}()
	steps := 0
	for i.SeekFirst(ctx); i.Valid(); i.Next() {
		steps++
	}
	if steps != count {
		b.Fatalf("visited %d domains, want %d", steps, count)
	}
	b.Run("next", func(b *testing.B) {
		i.SeekFirst(ctx)
		b.ReportAllocs()
		for b.Loop() {
			if !i.Next() {
				i.SeekFirst(ctx)
			}
		}
	})
	b.Run("prev", func(b *testing.B) {
		i.SeekLast(ctx)
		b.ReportAllocs()
		for b.Loop() {
			if !i.Prev() {
				i.SeekLast(ctx)
			}
		}
	})
}

func benchmarkStale(b *testing.B, k int, next bool) {
	b.Helper()
	ctx := b.Context()
	iters := make([]*domain.Iterator, 0, staleBatch)
	b.ReportAllocs()
	b.StopTimer()
	for done := 0; done < b.N; done += len(iters) {
		db := openBenchDB(b)
		writeCutDomain(b, db, k)
		iters = iters[:0]
		for range min(staleBatch, b.N-done) {
			i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
			if !i.SeekGE(ctx, cutStart) {
				b.Fatal("iterator did not reach the domain")
			}
			iters = append(iters, i)
		}
		cutDomain(b, db, k)
		b.StartTimer()
		for _, i := range iters {
			if next {
				i.Next()
			} else {
				i.Prev()
			}
		}
		b.StopTimer()
		for _, i := range iters {
			if err := i.Close(); err != nil {
				b.Fatal(err)
			}
		}
		closeBenchDB(b, db)
	}
}

// cutStart is the start of the domain that writeCutDomain writes.
const cutStart = 10 * telem.SecondTS

// writeCutDomain writes one domain of 3k+1 one-byte samples, one each second from
// cutStart, between one domain before it and one after it.
func writeCutDomain(b *testing.B, db *domain.DB, k int) {
	b.Helper()
	writeBenchDomain(b, db, telem.SecondTS)
	samples := 3*k + 1
	tr := cutStart.SpanRange(telem.TimeSpan(samples) * telem.Second)
	if err := testutil.WriteDomain(
		b.Context(),
		db,
		tr,
		make([]byte, samples),
	); err != nil {
		b.Fatal(err)
	}
	writeBenchDomain(b, db, tr.End+telem.SecondTS)
}

// cutDomain deletes the sample at 3j+1 of the domain that writeCutDomain wrote, for
// each j below k, and writes a domain in each cut.
func cutDomain(b *testing.B, db *domain.DB, k int) {
	b.Helper()
	ctx := b.Context()
	// From the last cut back, each cut is in the first piece, which starts at cutStart.
	for j := k - 1; j >= 0; j-- {
		tr := (cutStart + telem.TimeStamp(3*j+1)*telem.SecondTS).SpanRange(telem.Second)
		if err := db.Delete(
			ctx,
			tr,
			fixedOffsetAndTimeStamp(telem.Size(3*j+1), tr.Start),
			fixedOffsetAndTimeStamp(telem.Size(3*j+2), tr.End),
		); err != nil {
			b.Fatal(err)
		}
		writeBenchDomain(b, db, tr.Start)
	}
}

func writeBenchDomain(b *testing.B, db *domain.DB, start telem.TimeStamp) {
	b.Helper()
	tr := start.SpanRange(500 * telem.Millisecond)
	if err := testutil.WriteDomain(b.Context(), db, tr, []byte{1}); err != nil {
		b.Fatal(err)
	}
}

func openBenchDB(b *testing.B) *domain.DB {
	b.Helper()
	db, err := domain.Open(domain.Config{FS: xfs.NewMem()})
	if err != nil {
		b.Fatal(err)
	}
	return db
}

func closeBenchDB(b *testing.B, db *domain.DB) {
	b.Helper()
	if err := db.Close(); err != nil {
		b.Error(err)
	}
}
