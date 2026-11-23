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
	"runtime"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
)

// heapAlloc returns the live heap after two collections.
func heapAlloc() int64 {
	runtime.GC()
	runtime.GC()
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	return int64(s.HeapAlloc)
}

const (
	bigTx = 200_000
	// pinLimit is the most heap that one big transaction may leave behind.
	pinLimit = 1 << 20
)

var _ = Describe("Checker r2: memory a big transaction leaves behind", func() {
	It(
		"Should not pin the gone keys of a scope that deletes and restages keys in none",
		func() {
			m := index.NewMulti[*scope, int, string]()
			insert, finish := m.Populate()
			for k := range bigTx {
				insert(k, nil)
			}
			finish(nil)
			before := heapAlloc()
			txn := &scope{}
			for k := range bigTx {
				m.StageDelete(txn, k)
			}
			for k := range bigTx {
				m.Stage(txn, k, nil)
			}
			txn.end(true)
			small := &scope{}
			m.StageDelete(small, 0)
			m.Stage(small, 0, nil)
			small.end(true)
			after := heapAlloc()
			Expect(m.Holding(nil, false)).To(HaveLen(bigTx))
			Expect(after - before).To(BeNumerically("<", pinLimit))
			runtime.KeepAlive(m)
		},
	)

	It("Should keep one entry and no memory for a key staged many times", func() {
		l := index.NewLookup[*scope, int, string]()
		_, lfinish := l.Populate()
		lfinish(nil)
		m := index.NewMulti[*scope, int, string]()
		_, mfinish := m.Populate()
		mfinish(nil)
		before := heapAlloc()
		txn := &scope{}
		for i := range bigTx {
			v := "v" + strconv.Itoa(i)
			l.Stage(txn, 1, v)
			switch i % 3 {
			case 0:
				m.Stage(txn, 1, []string{v})
			case 1:
				m.Stage(txn, 1, nil)
			default:
				m.StageDelete(txn, 1)
			}
		}
		txn.end(true)
		after := heapAlloc()
		last := "v" + strconv.Itoa(bigTx-1)
		Expect(l.Get(nil, last)).To(ConsistOf(1))
		Expect(l.Get(nil, "v0")).To(BeEmpty())
		// The last stage, at bigTx-1, stages no value.
		Expect((bigTx - 1) % 3).To(Equal(1))
		Expect(m.Holding(nil, false)).To(ConsistOf(1))
		Expect(m.Holding(nil, true)).To(BeEmpty())
		Expect(after - before).To(BeNumerically("<", pinLimit))
		runtime.KeepAlive(l)
		runtime.KeepAlive(m)
	})

	It("Should keep every key of a big scope and a small one after it", func() {
		m := index.NewMulti[*scope, int, string]()
		insert, finish := m.Populate()
		for k := range 3000 {
			insert(k, nil)
		}
		finish(nil)
		txn := &scope{}
		for k := range 3000 {
			m.StageDelete(txn, k)
		}
		for k := 3000; k < 6000; k++ {
			m.Stage(txn, k, nil)
		}
		for k := 6000; k < 8000; k++ {
			m.Stage(txn, k, []string{"a"})
		}
		txn.end(true)
		small := &scope{}
		m.Stage(small, 1, nil)
		m.StageDelete(small, 6000)
		Expect(m.Holding(small, false)).To(HaveLen(3001))
		small.end(true)
		Expect(m.Holding(nil, false)).To(HaveLen(3001))
		Expect(m.Get(nil, "a")).To(HaveLen(1999))
	})

	It(
		"Should not pin the fresh keys of a scope that creates and deletes new keys",
		func() {
			m := index.NewMulti[*scope, int, string]()
			_, finish := m.Populate()
			finish(nil)
			before := heapAlloc()
			txn := &scope{}
			for k := range bigTx {
				m.Stage(txn, k, nil)
			}
			for k := range bigTx {
				m.StageDelete(txn, k)
			}
			txn.end(true)
			after := heapAlloc()
			Expect(m.Holding(nil, false)).To(BeEmpty())
			Expect(after - before).To(BeNumerically("<", pinLimit))
			runtime.KeepAlive(m)
		},
	)

	It(
		"Should not pin the forward map of a Sorted scope that creates and deletes",
		func() {
			s := index.NewSorted[*scope, int, string](
				func(a, b int) int { return a - b },
			)
			_, finish := s.Populate()
			finish(nil)
			names := make([]string, bigTx)
			for k := range bigTx {
				names[k] = "name-" + strconv.Itoa(k)
			}
			before := heapAlloc()
			txn := &scope{}
			for k := range bigTx {
				s.Stage(txn, k, names[k])
			}
			for k := range bigTx {
				s.StageDelete(txn, k)
			}
			txn.end(true)
			after := heapAlloc()
			Expect(s.Len()).To(BeZero())
			Expect(after - before).To(BeNumerically("<", pinLimit))
			runtime.KeepAlive(s)
			runtime.KeepAlive(names)
		},
	)
})
