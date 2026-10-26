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
	"strings"
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

// BenchmarkSmallIndex builds and searches an index of a few entries, as a caller that
// builds an index for each search does.
func BenchmarkSmallIndex(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		idx := search.New[int]()
		idx.Set(1, "Node 1")
		idx.Set(2, "Node 2")
		idx.Set(3, "Node 3")
		idx.Search("node 1", 100)
	}
}

func BenchmarkSearchAfterGC(b *testing.B) {
	idx := newBenchIndex(makeNames(100000))
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		runtime.GC()
		runtime.GC()
		b.StartTimer()
		idx.Search("sensor", 100)
	}
}

func BenchmarkGC(b *testing.B) {
	for _, count := range counts {
		b.Run(fmt.Sprintf("resources=%d", count), func(b *testing.B) {
			idx := newBenchIndex(makeNames(count))
			for b.Loop() {
				runtime.GC()
			}
			runtime.KeepAlive(idx)
		})
	}
}

func BenchmarkSetManyWords(b *testing.B) {
	for _, count := range []int{1000, 10000, 40000} {
		words := make([]string, count)
		for i := range words {
			words[i] = fmt.Sprintf("w%d", i)
		}
		text := strings.Join(words, " ")
		b.Run(fmt.Sprintf("words=%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				search.New[int]().Set(0, text)
			}
		})
	}
}

func BenchmarkSearchLongTerm(b *testing.B) {
	idx := newBenchIndex(makeNames(100000))
	term := strings.Repeat("sensor 12 ", 100000)
	b.ReportAllocs()
	for b.Loop() {
		idx.Search(term, 100)
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
