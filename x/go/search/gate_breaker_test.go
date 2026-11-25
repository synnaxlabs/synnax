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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/search"
)

var _ = Describe("Gate breaker", func() {
	It("Should build and search a small index with no secondary text in the bytes "+
		"of base", func() {
		res := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				idx := search.New[int]()
				idx.Set(1, "Node 1")
				idx.Set(2, "Node 2")
				idx.Set(3, "Node 3")
				idx.Search(search.Query[int]{Term: "node 1", Limit: 100})
			}
		})
		// BenchmarkSmallIndex at 5831318722: 2,192 bytes and 44 allocs per op.
		Expect(res.AllocsPerOp()).To(BeNumerically("<=", 44))
		Expect(res.AllocedBytesPerOp()).To(BeNumerically("<=", 2192))
	})
})
