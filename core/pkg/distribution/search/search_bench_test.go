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
	"io"
	"iter"
	"strconv"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
)

type benchService struct {
	observe.Noop[iter.Seq[ontology.Change]]
}

func (s *benchService) Type() ontology.ResourceType { return "bench" }

func (s *benchService) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	return func(func(ontology.Resource) bool) {}, xio.NopCloser, nil
}

func makeResources(n int) []ontology.Resource {
	resources := make([]ontology.Resource, n)
	names := []string{
		"gse_ai_%d", "DAQ_PT_%d", "sensor_temp_%d",
		"October %d Run", "valve_ctrl_%d",
	}
	for i := range n {
		name := fmt.Sprintf(names[i%len(names)], i)
		resources[i] = ontology.Resource{
			ID:   ontology.ID{Type: "bench", Key: strconv.Itoa(i)},
			Name: name,
		}
	}
	return resources
}

func newBenchIndex(b *testing.B) *search.Index {
	b.Helper()
	idx, err := search.Open()
	if err != nil {
		b.Fatal(err)
	}
	svc := &benchService{}
	idx.RegisterService(svc)
	if err := idx.Initialize(context.Background()); err != nil {
		b.Fatal(err)
	}
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
				if err := idx.IndexResources(resources); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkReindex(b *testing.B) {
	idx := newBenchIndex(b)
	r := ontology.Resource{
		ID:   ontology.ID{Type: "bench", Key: "0"},
		Name: "gse_ai_0",
	}
	if err := idx.IndexResources([]ontology.Resource{r}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Name = fmt.Sprintf("gse_ai_%d", i)
		if err := idx.IndexResources([]ontology.Resource{r}); err != nil {
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
				if err := idx.IndexResources(makeResources(count)); err != nil {
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
			if err := idx.IndexResources(makeResources(count)); err != nil {
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
