// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/synnaxlabs/x/search"
)

var counts = []int{100, 1000, 5000, 100000}

func makeNames(n int) []string {
	names := make([]string, n)
	formats := []string{
		"gse_ai_%d", "DAQ_PT_%d", "sensor_temp_%d",
		"October %d Run", "valve_ctrl_%d",
	}
	for i := range n {
		names[i] = fmt.Sprintf(formats[i%len(formats)], i)
	}
	return names
}

func newBenchIndex(names []string) *search.Index[int] {
	idx := search.New[int]()
	for i, name := range names {
		idx.Set(i, name)
	}
	return idx
}

func BenchmarkIndex(b *testing.B) {
	for _, count := range counts {
		b.Run(fmt.Sprintf("resources=%d", count), func(b *testing.B) {
			names := makeNames(count)
			b.ReportAllocs()
			for b.Loop() {
				newBenchIndex(names)
			}
		})
	}
}

func BenchmarkReindex(b *testing.B) {
	idx := newBenchIndex([]string{"gse_ai_0"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Set(0, fmt.Sprintf("gse_ai_%d", i))
	}
}

func BenchmarkSearch(b *testing.B) {
	for _, count := range counts {
		for _, limit := range []int{0, 100} {
			for _, query := range []struct {
				name string
				term string
			}{
				{"single_word", "sensor"},
				{"multi_word", "October Run"},
				{"underscore_separated", "gse_ai"},
				{"single_char", "s"},
				{"typo", "sensr"},
			} {
				b.Run(
					fmt.Sprintf("resources=%d/limit=%d/%s", count, limit, query.name),
					func(b *testing.B) {
						idx := newBenchIndex(makeNames(count))
						b.ReportAllocs()
						for b.Loop() {
							idx.Search(query.term, limit)
						}
					},
				)
			}
		}
	}
}

func BenchmarkSearchMiss(b *testing.B) {
	for _, count := range counts {
		b.Run(fmt.Sprintf("resources=%d", count), func(b *testing.B) {
			idx := newBenchIndex(makeNames(count))
			b.ReportAllocs()
			for b.Loop() {
				idx.Search("xyznonexistent qqq", 0)
			}
		})
	}
}

func BenchmarkMemory(b *testing.B) {
	for _, count := range counts {
		b.Run(fmt.Sprintf("resources=%d", count), func(b *testing.B) {
			names := makeNames(count)
			var retained uint64
			for b.Loop() {
				var before, after runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&before)
				idx := newBenchIndex(names)
				runtime.GC()
				runtime.ReadMemStats(&after)
				runtime.KeepAlive(idx)
				retained = after.HeapAlloc - before.HeapAlloc
			}
			b.ReportMetric(float64(retained)/float64(count), "B/resource")
		})
	}
}
