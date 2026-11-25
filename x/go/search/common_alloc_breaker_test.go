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
	"strings"
	"testing"

	"github.com/synnaxlabs/x/search"
)

func breakerDescription(i int) string {
	words := []string{"pressure", "at", "the", "inlet", "of", "tank", "valve", "sensor"}
	var b strings.Builder
	for j := 0; b.Len() < 1024; j++ {
		if j > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%s%d", words[(i+j)%len(words)], j)
	}
	return b.String()[:1024]
}

// BenchmarkBreakerNoSecondary builds and searches an index with no secondary text.
func BenchmarkBreakerNoSecondary(b *testing.B) {
	names := makeNames(1000)
	b.Run("build", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			idx := search.New[int]()
			for i, n := range names {
				idx.Set(i, n)
			}
		}
	})
	idx := search.New[int]()
	for i, n := range names {
		idx.Set(i, n)
	}
	for _, term := range []string{"sensor", "sens", "sensr", "gse ai"} {
		b.Run("search/"+term, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				idx.Search(search.Query[int]{Term: term, Limit: 100})
			}
		})
	}
}
