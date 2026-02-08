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
	"context"
	"fmt"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/internal/resource"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/internal/search"
)

func makeResources(n int) []resource.Resource {
	resources := make([]resource.Resource, n)
	names := []string{
		"gse_ai_%d", "DAQ_PT_%d", "sensor_temp_%d",
		"October %d Run", "valve_ctrl_%d",
	}
	for i := range n {
		name := fmt.Sprintf(names[i%len(names)], i)
		resources[i] = resource.Resource{
			ID:   resource.ID{Type: "bench", Key: fmt.Sprintf("%d", i)},
			Name: name,
		}
	}
	return resources
}

func newBenchIndex(b *testing.B) *search.Index {
	b.Helper()
	idx, err := search.New()
	if err != nil {
		b.Fatal(err)
	}
	idx.Register(context.Background(), "bench")
	return idx
}

func BenchmarkIndex(b *testing.B) {
	for _, count := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("resources=%d", count), func(b *testing.B) {
			resources := makeResources(count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := newBenchIndex(b)
				if err := idx.Index(resources); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkReindex(b *testing.B) {
	idx := newBenchIndex(b)
	r := resource.Resource{
		ID:   resource.ID{Type: "bench", Key: "0"},
		Name: "gse_ai_0",
	}
	if err := idx.Index([]resource.Resource{r}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Name = fmt.Sprintf("gse_ai_%d", i)
		if err := idx.Index([]resource.Resource{r}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	for _, count := range []int{100, 1000, 5000} {
		for _, queryType := range []struct {
			name string
			term string
		}{
			{"single_word", "sensor"},
			{"multi_word", "October Run"},
			{"underscore_separated", "gse_ai"},
		} {
			b.Run(fmt.Sprintf("resources=%d/%s", count, queryType.name), func(b *testing.B) {
				idx := newBenchIndex(b)
				if err := idx.Index(makeResources(count)); err != nil {
					b.Fatal(err)
				}
				ctx := context.Background()
				req := search.Request{Term: queryType.term}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := idx.Search(ctx, req); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkSearchMiss(b *testing.B) {
	for _, count := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("resources=%d", count), func(b *testing.B) {
			idx := newBenchIndex(b)
			if err := idx.Index(makeResources(count)); err != nil {
				b.Fatal(err)
			}
			ctx := context.Background()
			req := search.Request{Term: "xyznonexistent qqq"}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := idx.Search(ctx, req); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
