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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Multi", func() {
	var (
		m   *index.Multi[*scope, int, string]
		txn *scope
	)
	BeforeEach(func() {
		m = index.NewMulti[*scope, int, string]()
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
			failed := index.NewMulti[*scope, int, string]()
			_, finish := failed.Populate()
			finish(errLoad)
			Expect(failed.Holding(nil, false)).Error().To(MatchError(errLoad))
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
			failed := index.NewMulti[*scope, int, string]()
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
			loading := index.NewMulti[*scope, int, string]()
			_, finish := loading.Populate()
			_, known := loading.Held(nil, 1)
			Expect(known).To(BeFalse())
			finish(errLoad)
			_, known = loading.Held(nil, 1)
			Expect(known).To(BeFalse())
		})
	})
})
