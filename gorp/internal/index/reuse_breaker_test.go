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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
)

// gatedScope is a scope whose first Committing call waits for proceed to close.
type gatedScope struct {
	scope
	entered, proceed chan struct{}
	once             sync.Once
}

func (g *gatedScope) Committing() bool {
	g.once.Do(func() {
		close(g.entered)
		<-g.proceed
	})
	return g.committing
}

func heapInUse() uint64 {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapAlloc
}

var _ = Describe("Reuse breaker", func() {
	Describe("Lookup", func() {
		var l *index.Lookup[*scope, int, string]
		BeforeEach(func() {
			l = index.NewLookup[*scope, int, string]()
			insert, finish := l.Populate()
			insert(1, "a")
			insert(2, "b")
			finish(nil)
		})

		It(
			"Should show a reusing scope no value, holder, or delete of an aborted one",
			func() {
				a := &scope{}
				l.Stage(a, 1, "x")
				l.Stage(a, 3, "x")
				l.StageDelete(a, 2)
				a.committing = true
				a.end(false)
				b := &scope{}
				l.Stage(b, 9, "z")
				Expect(l.Get(b, "x")).To(BeEmpty())
				Expect(l.Get(b, "a")).To(ConsistOf(1))
				Expect(l.Get(b, "b")).To(ConsistOf(2))
				v, ok := MustSucceed2(l.Value(b, 2))
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("b"))
				_, found := MustSucceed2(l.Holder(b, "x", nil))
				Expect(found).To(BeFalse())
				k, found := MustSucceed2(l.Holder(b, "b", nil))
				Expect(found).To(BeTrue())
				Expect(k).To(Equal(2))
				Expect(l.Committing("x", "z")).To(BeEmpty())
				b.committing = true
				Expect(l.Committing("x", "z")).To(ConsistOf(9))
				b.end(true)
				Expect(l.Get(nil, "x")).To(BeEmpty())
				Expect(l.Get(nil, "z")).To(ConsistOf(9))
			},
		)

		It(
			"Should keep a committed value when the next scope rewrites the arena",
			func() {
				a := &scope{}
				for k := 10; k < 20; k++ {
					l.Stage(a, k, "first")
				}
				a.end(true)
				b := &scope{}
				for k := 10; k < 20; k++ {
					l.Stage(b, k, "second")
				}
				Expect(l.Get(nil, "first")).To(HaveLen(10))
				Expect(l.Get(nil, "second")).To(BeEmpty())
				for k := 10; k < 20; k++ {
					v, ok := MustSucceed2(l.Value(nil, k))
					Expect(ok).To(BeTrue())
					Expect(v).To(Equal("first"))
				}
				b.end(false)
				Expect(l.Get(nil, "first")).To(HaveLen(10))
			},
		)

		It("Should keep two live scopes apart when a third reuses", func() {
			a, b := &scope{}, &scope{}
			l.Stage(a, 5, "x")
			l.Stage(b, 6, "y")
			a.end(false)
			c := &scope{}
			l.Stage(c, 7, "w")
			Expect(l.Get(b, "x", "y", "w")).To(ConsistOf(6))
			Expect(l.Get(c, "x", "y", "w")).To(ConsistOf(7))
			b.end(true)
			d := &scope{}
			l.Stage(d, 8, "v")
			Expect(l.Get(c, "x", "y", "w", "v")).To(ConsistOf(6, 7))
			Expect(l.Get(d, "x", "y", "w", "v")).To(ConsistOf(6, 8))
			c.end(true)
			d.end(true)
			Expect(l.Get(nil, "x", "y", "w", "v")).To(ConsistOf(6, 7, 8))
		})

		It(
			"Should apply a scope that grows past the spare size and the next one",
			func() {
				a := &scope{}
				for k := 100; k < 200; k++ {
					l.Stage(a, k, "g")
				}
				for k := 100; k < 199; k++ {
					l.StageDelete(a, k)
				}
				a.end(false)
				b := &scope{}
				l.Stage(b, 300, "g")
				Expect(l.Get(b, "g")).To(ConsistOf(300))
				b.end(true)
				Expect(l.Get(nil, "g")).To(ConsistOf(300))
			},
		)

		It("Should keep the holders of a value through every remove order", func() {
			for k := range 5 {
				l.Set(10+k, "h")
			}
			l.Delete(10)
			Expect(l.Get(nil, "h")).To(ConsistOf(11, 12, 13, 14))
			l.Delete(12)
			Expect(l.Get(nil, "h")).To(ConsistOf(11, 13, 14))
			l.Delete(14)
			l.Set(15, "h")
			Expect(l.Get(nil, "h")).To(ConsistOf(11, 13, 15))
			l.Delete(11)
			l.Delete(13)
			l.Delete(15)
			Expect(l.Get(nil, "h")).To(BeEmpty())
			l.Set(16, "h")
			Expect(l.Get(nil, "h")).To(ConsistOf(16))
			l.Set(0, "h")
			l.Delete(16)
			k, found := MustSucceed2(l.Holder(nil, "h", nil))
			Expect(found).To(BeTrue())
			Expect(k).To(Equal(0))
			_, found = MustSucceed2(l.Holder(nil, "h", new(0)))
			Expect(found).To(BeFalse())
			l.Delete(99)
			Expect(l.Get(nil, "h")).To(ConsistOf(0))
		})

		It("Should skip a committed holder that a reusing scope moves", func() {
			l.Set(20, "m")
			l.Set(21, "m")
			a := &scope{}
			l.Stage(a, 20, "n")
			a.end(false)
			b := &scope{}
			l.Stage(b, 20, "n")
			l.Stage(b, 21, "n")
			_, found := MustSucceed2(l.Holder(b, "m", nil))
			Expect(found).To(BeFalse())
			k, found := MustSucceed2(l.Holder(b, "n", new(20)))
			Expect(found).To(BeTrue())
			Expect(k).To(Equal(21))
			b.end(true)
			Expect(l.Get(nil, "m")).To(BeEmpty())
			Expect(l.Get(nil, "n")).To(ConsistOf(20, 21))
		})
	})

	Describe("Population", func() {
		It("Should reuse the changes of a scope that aborted during the load", func() {
			l := index.NewLookup[*scope, int, string]()
			insert, finish := l.Populate()
			a := &scope{}
			l.Stage(a, 1, "x")
			l.StageDelete(a, 2)
			a.end(false)
			b := &scope{}
			l.Stage(b, 3, "y")
			insert(2, "x")
			finish(nil)
			Expect(l.Get(b, "x", "y")).To(ConsistOf(2, 3))
			b.end(true)
			Expect(l.Get(nil, "x", "y")).To(ConsistOf(2, 3))
		})
	})

	Describe("Multi fresh keys", func() {
		It(
			"Should keep fresh keys of a reusing scope apart from an aborted one",
			func() {
				m := index.NewMulti[*scope, int, string]()
				_, finish := m.Populate()
				finish(nil)
				a := &scope{}
				m.Stage(a, 1, []string{"a"})
				m.Stage(a, 2, nil)
				a.end(false)
				b := &scope{}
				m.Stage(b, 3, nil)
				m.Stage(b, 1, nil)
				Expect(m.Holding(b, false)).To(ConsistOf(1, 3))
				Expect(m.Holding(b, true)).To(BeEmpty())
				Expect(m.Current(b, 1, nil)).To(BeTrue())
				b.end(true)
				Expect(m.Holding(nil, false)).To(ConsistOf(1, 3))
			},
		)
	})

	Describe("Multi", func() {
		var m *index.Multi[*scope, int, string]
		BeforeEach(func() {
			m = index.NewMulti[*scope, int, string]()
			insert, finish := m.Populate()
			insert(1, []string{"a", "b"})
			finish(nil)
		})

		It("Should show a reusing scope none of the values of an aborted one", func() {
			a := &scope{}
			m.Stage(a, 2, []string{"a", "x"})
			m.StageDelete(a, 1)
			m.Stage(a, 3, nil)
			a.end(false)
			b := &scope{}
			m.Stage(b, 4, []string{"y"})
			Expect(m.Get(b, "a", "x")).To(ConsistOf(1))
			Expect(
				m.GetEach(b, []string{"a", "x", "y"}),
			).To(Equal([][]int{{1}, {}, {4}}))
			Expect(m.Holding(b, false)).To(BeEmpty())
			Expect(m.Holding(b, true)).To(ConsistOf(1, 4))
			held := MustSucceed(m.HeldValues(b, []string{"a", "x"}, func(int) bool {
				return true
			}))
			Expect(held.Contains("x")).To(BeFalse())
			vs, ok := m.Held(b, 1)
			Expect(ok).To(BeTrue())
			Expect(vs).To(Equal([]string{"a", "b"}))
			b.committing = true
			Expect(m.CommittingEach([]string{"a", "x", "y"})).
				To(Equal([][]int{nil, nil, {4}}))
			b.end(true)
			Expect(m.Get(nil, "a", "x", "y")).To(ConsistOf(1, 4))
		})

		It("Should keep committed values when the next scope stages over them", func() {
			a := &scope{}
			m.Stage(a, 5, []string{"p", "q"})
			a.end(true)
			b := &scope{}
			m.Stage(b, 5, []string{"r"})
			m.Stage(b, 6, []string{"p"})
			vs, ok := m.Held(nil, 5)
			Expect(ok).To(BeTrue())
			Expect(vs).To(Equal([]string{"p", "q"}))
			Expect(m.Get(nil, "p")).To(ConsistOf(5))
			b.end(false)
			Expect(m.Get(nil, "p", "q", "r")).To(ConsistOf(5))
		})
	})

	Describe("Sorted", func() {
		var s *index.Sorted[*scope, int, int]
		BeforeEach(func() {
			s = index.NewSorted[*scope, int, int](func(a, b int) int { return a - b })
			insert, finish := s.Populate()
			insert(1, 10)
			insert(2, 20)
			finish(nil)
		})

		It("Should walk and range committed state after a reuse", func() {
			a := &scope{}
			s.Stage(a, 3, 30)
			s.StageDelete(a, 1)
			a.end(false)
			b := &scope{}
			s.Stage(b, 4, 15)
			Expect(s.Range(b, nil, nil)).To(ConsistOf(1, 2, 4))
			Expect(s.Keep(b, []int{1, 3, 4}, nil, nil)).To(ConsistOf(1, 4))
			b.end(true)
			c := &scope{}
			s.Stage(c, 4, 40)
			Expect(s.Walk(false, nil, 0, nil)).To(Equal([]index.Entry[int, int]{
				{Key: 1, Value: 10}, {Key: 4, Value: 15}, {Key: 2, Value: 20},
			}))
			Expect(s.Range(c, &index.Bound[int]{Value: 35}, nil)).To(ConsistOf(4))
			c.end(false)
			Expect(s.Get(nil, 15)).To(ConsistOf(4))
			Expect(s.Get(nil, 40)).To(BeEmpty())
		})
	})

	Describe("Search", func() {
		It("Should show a reusing scope none of the text of an aborted one", func() {
			sr := index.NewSearch[*scope, int]()
			insert, finish := sr.Populate()
			insert(1, []string{"alpha"})
			finish(nil)
			a := &scope{}
			sr.Stage(a, 2, []string{"bravo"})
			sr.StageDelete(a, 1)
			a.end(false)
			b := &scope{}
			sr.Stage(b, 3, []string{"charlie"})
			res := MustSucceed(sr.Get(b, search.Query[int]{Term: "bravo"}))
			Expect(res.Keys).To(BeEmpty())
			res = MustSucceed(sr.Get(b, search.Query[int]{Term: "alpha"}))
			Expect(res.Keys).To(ConsistOf(1))
		})
	})

	Describe("Committing during a commit", func() {
		It("Should not change a committing delta while a reader holds it", func() {
			l := index.NewLookup[*gatedScope, int, string]()
			_, finish := l.Populate()
			finish(nil)
			t := &gatedScope{entered: make(chan struct{}), proceed: make(chan struct{})}
			l.Stage(t, 1, "a")
			t.committing = true
			read := make(chan []int)
			go func() { read <- l.Committing("a") }()
			<-t.entered
			close(t.proceed)
			t.end(true)
			<-read
			Expect(l.Get(nil, "a")).To(ConsistOf(1))
		})
	})

	Describe("Committing during a failed commit", func() {
		It("Should not crash a reader of a committing delta", func() {
			m := index.NewMulti[*gatedScope, int, int]()
			_, finish := m.Populate()
			finish(nil)
			values := make([]int, 1<<18)
			for i := range values {
				values[i] = i
			}
			t := &gatedScope{entered: make(chan struct{}), proceed: make(chan struct{})}
			m.Stage(t, 1, values)
			t.committing = true
			read := make(chan [][]int)
			go func() { read <- m.CommittingEach(values) }()
			<-t.entered
			close(t.proceed)
			t.end(false)
			<-read
			Expect(m.Get(nil, 0)).To(BeEmpty())
		})
	})

	Describe("Retained storage", func() {
		It("Should not keep the grown holders of a few keys with many values", func() {
			m := index.NewMulti[*scope, int, int]()
			_, finish := m.Populate()
			finish(nil)
			values := make([]int, 1<<17)
			for i := range values {
				values[i] = i
			}
			before := heapInUse()
			a := &scope{}
			m.Stage(a, 1, values)
			a.end(false)
			retained := int64(heapInUse()) - int64(before)
			Expect(retained).To(BeNumerically("<", 1<<20))
			runtime.KeepAlive(m)
		})

		It("Should not keep the arenas of one key staged many times", func() {
			l := index.NewLookup[*scope, int, int]()
			_, finish := l.Populate()
			finish(nil)
			before := heapInUse()
			a := &scope{}
			for v := range 1 << 18 {
				l.Stage(a, 1, v)
			}
			a.end(false)
			retained := int64(heapInUse()) - int64(before)
			Expect(retained).To(BeNumerically("<", 1<<20))
			runtime.KeepAlive(l)
		})

		It("Should not keep a large scope that took the spare", func() {
			l := index.NewLookup[*scope, int, int]()
			_, finish := l.Populate()
			finish(nil)
			small := &scope{}
			l.Stage(small, 1, 1)
			small.end(false)
			before := heapInUse()
			large := &scope{}
			for k := range 1 << 16 {
				l.Stage(large, k, k)
			}
			large.end(false)
			retained := int64(heapInUse()) - int64(before)
			Expect(retained).To(BeNumerically("<", 1<<20))
			runtime.KeepAlive(l)
		})
	})
})
