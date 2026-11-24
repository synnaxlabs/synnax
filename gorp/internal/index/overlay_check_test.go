// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
)

// heapInUse returns the bytes of live heap objects after a collection.
func heapInUse() int64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return int64(stats.HeapAlloc)
}

// value returns a 64-byte value for n, so that a staged value the index keeps is seen
// in the heap.
func value(n int) string { return fmt.Sprintf("%064d", n) }

var _ = Describe("Overlay memory", func() {
	const bound = 2 << 20

	It("Should not grow a lookup's memory across transactions of one size", func() {
		l := index.NewLookup[*scope, int, string]()
		_, finish := l.Populate()
		finish(nil)
		run := func(tx int) {
			s := &scope{}
			for k := range 100 {
				l.Stage(s, k, value(tx*100+k))
			}
			s.end(tx%2 == 0)
		}
		for tx := range 100 {
			run(tx)
		}
		before := heapInUse()
		for tx := range 2000 {
			run(100 + tx)
		}
		Expect(heapInUse() - before).To(BeNumerically("<", bound))
		runtime.KeepAlive(l)
	})

	It("Should give back the memory of a large transaction once it ends", func() {
		l := index.NewLookup[*scope, int, string]()
		_, finish := l.Populate()
		finish(nil)
		small := &scope{}
		l.Stage(small, 0, value(0))
		small.end(false)
		before := heapInUse()
		large := &scope{}
		for k := range 400_000 {
			l.Stage(large, k, value(k))
		}
		large.end(false)
		Expect(heapInUse() - before).To(BeNumerically("<", bound))
		runtime.KeepAlive(l)
	})
})

var _ = Describe("Multi Empty around a commit", func() {
	It("Should see a holder that a scope in its commit staged, before and after the "+
		"commit", func() {
		m := index.NewMulti[*scope, int, string]()
		_, finish := m.Populate()
		finish(nil)
		s := &scope{}
		m.Stage(s, 1, []string{"a"})
		s.committing = true
		Expect(m.Empty(nil)).To(BeFalse())
		s.end(true)
		Expect(m.Empty(nil)).To(BeFalse())
		Expect(m.Get(nil, "a")).To(ConsistOf(1))
	})
})
