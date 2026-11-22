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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Multi", func() {
	var (
		m   *index.Multi[*scope, int, string]
		txn *scope
	)
	BeforeEach(func() {
		m = index.NewMulti[*scope, int, string](true)
		insert, finish := m.Populate()
		insert(1, []string{"a", "b"})
		insert(2, []string{"b"})
		finish(nil)
		txn = &scope{}
	})

	It("Should map each value to every key that holds it", func() {
		Expect(m.Get(nil, "a")).To(ConsistOf(1))
		Expect(m.Get(nil, "b")).To(ConsistOf(1, 2))
		Expect(m.Get(nil, "a", "b")).To(ConsistOf(1, 2))
		Expect(m.Get(nil)).To(BeEmpty())
	})

	It("Should replace all values of a key on Set", func() {
		m.Set(1, []string{"c"})
		Expect(m.Get(nil, "a")).To(BeEmpty())
		Expect(m.Get(nil, "b")).To(ConsistOf(2))
		Expect(m.Get(nil, "c")).To(ConsistOf(1))
	})

	It("Should ignore a write of the current values", func() {
		m.Set(1, []string{"a", "b"})
		Expect(m.Get(nil, "a")).To(ConsistOf(1))
		Expect(m.Get(nil, "b")).To(ConsistOf(1, 2))
	})

	It("Should show staged changes only to their scope", func() {
		m.Stage(txn, 1, []string{"b"})
		m.Stage(txn, 3, []string{"a"})
		m.StageDelete(txn, 2)
		Expect(m.Get(txn, "a")).To(ConsistOf(3))
		Expect(m.Get(txn, "b")).To(ConsistOf(1))
		Expect(m.Get(nil, "a")).To(ConsistOf(1))
	})

	It("Should apply staged changes when the scope commits", func() {
		m.Stage(txn, 1, nil)
		txn.end(true)
		Expect(m.Get(nil, "a")).To(BeEmpty())
		Expect(m.Get(nil, "b")).To(ConsistOf(2))
	})

	It("Should track many holders of one value", func() {
		for k := 10; k < 110; k++ {
			m.Set(k, []string{"c"})
		}
		Expect(m.Get(nil, "c")).To(HaveLen(100))
		for k := 10; k < 109; k++ {
			m.Delete(k)
		}
		Expect(m.Get(nil, "c")).To(ConsistOf(109))
	})

	It("Should drop a staged delete when the scope does not commit", func() {
		m.StageDelete(txn, 1)
		Expect(m.Get(txn, "a")).To(BeEmpty())
		txn.end(false)
		Expect(m.Get(nil, "a")).To(ConsistOf(1))
		Expect(m.Get(txn, "a")).To(ConsistOf(1))
	})

	Describe("GetEach", func() {
		It("Should return the keys that hold each value at its position", func() {
			Expect(m.GetEach(nil, []string{"a", "b", "z"})).To(Equal([][]int{
				{1},
				{1, 2},
				nil,
			}))
		})

		It("Should return nil when no key holds any value", func() {
			Expect(m.GetEach(nil, []string{"y", "z"})).To(BeNil())
		})

		It("Should show staged changes only to their scope", func() {
			m.Stage(txn, 3, []string{"a"})
			m.StageDelete(txn, 1)
			each := MustSucceed(m.GetEach(txn, []string{"a", "b"}))
			Expect(each[0]).To(ConsistOf(3))
			Expect(each[1]).To(ConsistOf(2))
			Expect(m.GetEach(nil, []string{"a"})).To(Equal([][]int{{1}}))
		})

		It("Should return the load error", func() {
			failed := index.NewMulti[*scope, int, string](true)
			_, finish := failed.Populate()
			finish(errLoad)
			Expect(failed.GetEach(nil, []string{"a"})).Error().To(MatchError(errLoad))
		})
	})

	Describe("CommittingEach", func() {
		It(
			"Should return the keys that a scope in its commit maps to each value",
			func() {
				m.Stage(txn, 3, []string{"a"})
				Expect(m.CommittingEach([]string{"a", "b"})).To(BeNil())
				txn.committing = true
				Expect(
					m.CommittingEach([]string{"a", "b"}),
				).To(Equal([][]int{{3}, nil}))
			},
		)
	})

	Describe("Holding", func() {
		BeforeEach(func() { m.Set(3, nil) })

		It("Should return the keys that hold values or none", func() {
			Expect(m.Holding(nil, true)).To(ConsistOf(1, 2))
			Expect(m.Holding(nil, false)).To(ConsistOf(3))
		})

		It("Should move a key between the two sets on a committed write", func() {
			m.Set(1, nil)
			m.Set(3, []string{"c"})
			Expect(m.Holding(nil, true)).To(ConsistOf(2, 3))
			Expect(m.Holding(nil, false)).To(ConsistOf(1))
		})

		It("Should drop a deleted key from both sets", func() {
			m.Delete(3)
			m.Delete(2)
			Expect(m.Holding(nil, true)).To(ConsistOf(1))
			Expect(m.Holding(nil, false)).To(BeEmpty())
		})

		It("Should show staged changes only to their scope", func() {
			m.Stage(txn, 1, nil)
			m.Stage(txn, 4, []string{"d"})
			m.StageDelete(txn, 3)
			Expect(m.Holding(txn, true)).To(ConsistOf(2, 4))
			Expect(m.Holding(txn, false)).To(ConsistOf(1))
			Expect(m.Holding(nil, false)).To(ConsistOf(3))
		})

		It("Should return the keys of a scope in its commit", func() {
			m.Stage(txn, 1, nil)
			m.Stage(txn, 4, []string{"d"})
			Expect(m.CommittingHolding(false)).To(BeEmpty())
			txn.committing = true
			Expect(m.CommittingHolding(false)).To(ConsistOf(1))
			Expect(m.CommittingHolding(true)).To(ConsistOf(4))
		})

		It("Should return the load error", func() {
			failed := index.NewMulti[*scope, int, string](true)
			_, finish := failed.Populate()
			finish(errLoad)
			Expect(failed.Holding(nil, false)).Error().To(MatchError(errLoad))
		})
	})

	Describe("A key that a scope stages with no value and that held nothing", func() {
		It("Should hold none only in its scope until the scope commits", func() {
			m.Stage(txn, 5, nil)
			Expect(m.Holding(txn, false)).To(ConsistOf(5))
			Expect(m.Holding(nil, false)).To(BeEmpty())
			Expect(m.Current(txn, 5, nil)).To(BeTrue())
			txn.end(true)
			Expect(m.Holding(nil, false)).To(ConsistOf(5))
		})

		It("Should be dropped when the scope does not commit", func() {
			m.Stage(txn, 5, nil)
			txn.end(false)
			Expect(m.Holding(nil, false)).To(BeEmpty())
			Expect(m.Holding(nil, true)).To(ConsistOf(1, 2))
		})

		It("Should hold the values that the scope stages for it later", func() {
			m.Stage(txn, 5, nil)
			m.Stage(txn, 5, []string{"e"})
			Expect(m.Holding(txn, false)).To(BeEmpty())
			txn.committing = true
			Expect(m.CommittingHolding(false)).To(BeEmpty())
			txn.end(true)
			Expect(m.Holding(nil, false)).To(BeEmpty())
			Expect(m.Get(nil, "e")).To(ConsistOf(5))
		})

		It("Should be gone when the scope deletes it later", func() {
			m.Stage(txn, 5, nil)
			m.StageDelete(txn, 5)
			Expect(m.Holding(txn, false)).To(BeEmpty())
			txn.committing = true
			Expect(m.CommittingHolding(false)).To(BeEmpty())
			txn.end(true)
			Expect(m.Holding(nil, false)).To(BeEmpty())
			Expect(m.Holding(nil, true)).To(ConsistOf(1, 2))
		})

		It("Should hold none when the scope staged values for it first", func() {
			m.Stage(txn, 5, []string{"e"})
			m.Stage(txn, 5, nil)
			txn.end(true)
			Expect(m.Holding(nil, false)).To(ConsistOf(5))
			Expect(m.Get(nil, "e")).To(BeEmpty())
		})

		It("Should be among the keys of a scope in its commit", func() {
			m.Stage(txn, 5, nil)
			Expect(m.CommittingHolding(false)).To(BeEmpty())
			txn.committing = true
			Expect(m.CommittingHolding(false)).To(ConsistOf(5))
		})

		It("Should be staged with no allocation", func() {
			txn.cleanups = make([]func(bool), 0, 1)
			key := 100
			stage := func() {
				key++
				m.Stage(txn, key, nil)
				for _, fn := range txn.cleanups {
					fn(false)
				}
				txn.cleanups = txn.cleanups[:0]
			}
			stage()
			Expect(testing.AllocsPerRun(100, stage)).To(BeZero())
		})
	})

	Describe("HeldValues", func() {
		all := func(int) bool { return true }

		It("Should return the values that some key holds", func() {
			Expect(m.HeldValues(nil, []string{"a", "b", "c"}, all)).
				To(Equal(set.New("a", "b")))
		})

		It("Should show staged changes only to their scope", func() {
			m.Stage(txn, 1, []string{"c"})
			Expect(m.HeldValues(txn, []string{"a", "b", "c"}, all)).
				To(Equal(set.New("b", "c")))
			m.StageDelete(txn, 2)
			Expect(m.HeldValues(txn, []string{"a", "b", "c"}, all)).
				To(Equal(set.New("c")))
			Expect(m.HeldValues(nil, []string{"a", "b", "c"}, all)).
				To(Equal(set.New("a", "b")))
		})

		It("Should count only the keys that in accepts", func() {
			two := func(k int) bool { return k == 2 }
			Expect(m.HeldValues(nil, []string{"a", "b"}, two)).
				To(Equal(set.New("b")))
			m.Stage(txn, 2, []string{"c"})
			m.Stage(txn, 3, []string{"a"})
			Expect(m.HeldValues(txn, []string{"a", "b", "c"}, two)).
				To(Equal(set.New("c")))
		})

		It("Should return the load error", func() {
			failed := index.NewMulti[*scope, int, string](true)
			_, finish := failed.Populate()
			finish(errLoad)
			Expect(
				failed.HeldValues(nil, []string{"a"}, all),
			).Error().
				To(MatchError(errLoad))
		})
	})

	Describe("Held", func() {
		held := func(s *scope, key int) []string {
			GinkgoHelper()
			values, known := m.Held(s, key)
			Expect(known).To(BeTrue())
			return values
		}

		It("Should return the values a key holds", func() {
			Expect(held(nil, 1)).To(Equal([]string{"a", "b"}))
			Expect(held(nil, 9)).To(BeEmpty())
		})

		It("Should return the values that a scope staged", func() {
			m.Stage(txn, 1, []string{"c"})
			Expect(held(txn, 1)).To(Equal([]string{"c"}))
			Expect(held(nil, 1)).To(Equal([]string{"a", "b"}))
		})

		It("Should not know the values before the load finishes", func() {
			loading := index.NewMulti[*scope, int, string](true)
			_, finish := loading.Populate()
			_, known := loading.Held(nil, 1)
			Expect(known).To(BeFalse())
			finish(errLoad)
			_, known = loading.Held(nil, 1)
			Expect(known).To(BeFalse())
		})
	})

	Describe("Current", func() {
		It("Should report the values a key holds as current", func() {
			Expect(m.Current(nil, 1, []string{"a", "b"})).To(BeTrue())
			Expect(m.Current(nil, 1, []string{"a"})).To(BeFalse())
			Expect(m.Current(nil, 1, []string{"b", "a"})).To(BeFalse())
		})

		It("Should tell a key that holds none from a key it does not hold", func() {
			m.Set(3, nil)
			Expect(m.Current(nil, 3, nil)).To(BeTrue())
			Expect(m.Current(nil, 9, nil)).To(BeFalse())
		})

		It("Should apply the changes that a scope staged", func() {
			m.Stage(txn, 1, []string{"c"})
			m.StageDelete(txn, 2)
			Expect(m.Current(txn, 1, []string{"c"})).To(BeTrue())
			Expect(m.Current(nil, 1, []string{"a", "b"})).To(BeTrue())
			Expect(m.Current(txn, 2, nil)).To(BeFalse())
		})

		It("Should report staged values over committed ones", func() {
			m.Stage(txn, 1, []string{"c"})
			Expect(m.Current(txn, 1, []string{"a", "b"})).To(BeFalse())
		})

		It("Should report nothing as current before the load finishes", func() {
			loading := index.NewMulti[*scope, int, string](true)
			_, finish := loading.Populate()
			Expect(loading.Current(nil, 1, nil)).To(BeFalse())
			finish(errLoad)
			Expect(loading.Current(nil, 1, nil)).To(BeFalse())
		})
	})
})

var _ = Describe("Multi bulk load", func() {
	// heldOf returns the values that key holds in m, failing when m cannot tell.
	heldOf := func(m *index.Multi[*scope, int, string], key int) []string {
		GinkgoHelper()
		held, ok := m.Held(nil, key)
		Expect(ok).To(BeTrue())
		return held
	}
	// fill bulk loads keys 1 to n, each holding the values that values returns.
	fill := func(
		tracked bool,
		n int,
		values func(k int) []string,
	) *index.Multi[*scope, int, string] {
		GinkgoHelper()
		m := index.NewMulti[*scope, int, string](tracked)
		insert, finish := m.Populate()
		buf := make([]string, 0, 8)
		for k := 1; k <= n; k++ {
			buf = append(buf[:0], values(k)...)
			insert(k, buf)
		}
		finish(nil)
		return m
	}

	It("Should answer as an index filled by Set", func() {
		m := fill(true, 3, func(k int) []string {
			return [][]string{{"a", "b"}, {"b"}, {}}[k-1]
		})
		bySet := index.NewMulti[*scope, int, string](true)
		_, finish := bySet.Populate()
		finish(nil)
		bySet.Set(1, []string{"a", "b"})
		bySet.Set(2, []string{"b"})
		bySet.Set(3, nil)
		all := func(int) bool { return true }
		for _, idx := range []*index.Multi[*scope, int, string]{m, bySet} {
			Expect(idx.Get(nil, "a")).To(ConsistOf(1))
			Expect(idx.Get(nil, "b")).To(ConsistOf(1, 2))
			Expect(idx.Holding(nil, true)).To(ConsistOf(1, 2))
			Expect(idx.Holding(nil, false)).To(ConsistOf(3))
			Expect(idx.HeldValues(nil, []string{"a", "c"}, all)).To(Equal(set.New("a")))
			Expect(heldOf(idx, 1)).To(Equal([]string{"a", "b"}))
		}
	})

	It("Should copy each key's values out of the caller's buffer", func() {
		m := fill(false, 2, func(k int) []string {
			return []string{[]string{"a", "b"}[k-1]}
		})
		Expect(heldOf(m, 1)).To(Equal([]string{"a"}))
		Expect(heldOf(m, 2)).To(Equal([]string{"b"}))
	})

	It("Should keep an append to one key's values from a neighbor's", func() {
		m := fill(false, 2, func(k int) []string { return []string{"a"} })
		_ = append(heldOf(m, 1), "z")
		Expect(heldOf(m, 2)).To(Equal([]string{"a"}))
	})

	It("Should copy on a write after the load", func() {
		m := fill(false, 2, func(k int) []string { return []string{"a"} })
		m.Set(1, []string{"a", "b"})
		Expect(heldOf(m, 1)).To(Equal([]string{"a", "b"}))
		Expect(heldOf(m, 2)).To(Equal([]string{"a"}))
		Expect(m.Get(nil, "b")).To(ConsistOf(1))
		Expect(m.Get(nil, "a")).To(ConsistOf(1, 2))
	})

	It("Should hold a value past the size of a holder list", func() {
		m := fill(false, 100, func(int) []string { return []string{"shared"} })
		Expect(m.Get(nil, "shared")).To(HaveLen(100))
		m.Delete(50)
		Expect(m.Get(nil, "shared")).To(HaveLen(99))
	})

	It("Should count a value that one key holds twice once", func() {
		m := fill(false, 1, func(int) []string { return []string{"a", "a"} })
		Expect(m.Get(nil, "a")).To(Equal([]int{1}))
		m.Delete(1)
		Expect(m.Get(nil, "a")).To(BeEmpty())
	})
})

var _ = Describe("Multi that keeps no record of empty keys", func() {
	It("Should stage a key with no value and nothing before with no change", func() {
		m := index.NewMulti[*scope, int, string](false)
		insert, finish := m.Populate()
		insert(1, []string{"a"})
		finish(nil)
		txn := &scope{}
		m.Stage(txn, 2, nil)
		Expect(m.Current(txn, 2, nil)).To(BeTrue())
		Expect(m.Holding(txn, true)).To(ConsistOf(1))
		m.Stage(txn, 1, nil)
		Expect(m.Holding(txn, true)).To(BeEmpty())
		txn.end(true)
		Expect(m.Holding(nil, true)).To(BeEmpty())
	})
})
