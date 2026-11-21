// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lru_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/lru"
)

var _ = Describe("Cache", func() {
	var c *lru.Cache[string, int]
	BeforeEach(func() { c = lru.New[string, int](10) })
	// value returns the value of k, which c must hold.
	value := func(k string) int {
		GinkgoHelper()
		v, ok := c.Get(k)
		Expect(ok).To(BeTrue())
		return v
	}
	// evicted takes out every value that Evict gives, in order.
	evicted := func(held func(int) bool) []int {
		GinkgoHelper()
		var out []int
		for {
			v, ok := c.Evict(held)
			if !ok {
				return out
			}
			out = append(out, v)
		}
	}

	It("Should return the value of a key, and none for a key it does not hold", func() {
		Expect(c.Put("a", 1, 4)).To(BeTrue())
		Expect(value("a")).To(Equal(1))
		_, ok := c.Get("b")
		Expect(ok).To(BeFalse())
	})

	It("Should keep the first value of a key", func() {
		Expect(c.Put("a", 1, 4)).To(BeTrue())
		Expect(c.Put("a", 2, 4)).To(BeFalse())
		Expect(value("a")).To(Equal(1))
	})

	It("Should evict nothing while it holds at most its budget", func() {
		c.Put("a", 1, 4)
		c.Put("b", 2, 6)
		Expect(evicted(nil)).To(BeEmpty())
	})

	It("Should evict the least recently used values until it holds its budget", func() {
		c.Put("a", 1, 4)
		c.Put("b", 2, 4)
		c.Put("c", 3, 4)
		Expect(value("a")).To(Equal(1))
		c.Put("d", 4, 4)
		Expect(evicted(nil)).To(Equal([]int{2, 3}))
		Expect(value("a")).To(Equal(1))
		Expect(value("d")).To(Equal(4))
		_, ok := c.Get("b")
		Expect(ok).To(BeFalse())
	})

	It("Should skip the values that are held", func() {
		c.Put("a", 1, 4)
		c.Put("b", 2, 4)
		c.Put("c", 3, 4)
		Expect(evicted(func(v int) bool { return v == 1 })).To(Equal([]int{2}))
		Expect(value("a")).To(Equal(1))
		Expect(evicted(func(int) bool { return true })).To(BeEmpty())
	})

	It(
		"Should keep the most recently used value when it alone passes the budget",
		func() {
			c.Put("a", 1, 4)
			c.Put("b", 2, 12)
			Expect(evicted(nil)).To(Equal([]int{1}))
			Expect(value("b")).To(Equal(2))
		},
	)

	It("Should hold a value again under a key that it evicted", func() {
		c.Put("a", 1, 8)
		c.Put("b", 2, 8)
		Expect(evicted(nil)).To(Equal([]int{1}))
		Expect(c.Put("a", 3, 8)).To(BeTrue())
		Expect(evicted(nil)).To(Equal([]int{2}))
		Expect(value("a")).To(Equal(3))
	})
})
