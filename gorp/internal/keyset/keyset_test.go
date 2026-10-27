// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package keyset_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/keyset"
)

var _ = Describe("Set", func() {
	Describe("Zero value", func() {
		It("Should be unbounded and contain every key", func() {
			var s keyset.Set[int]
			Expect(s.Bounded()).To(BeFalse())
			Expect(s.Keys()).To(BeNil())
			Expect(s.Contains(42)).To(BeTrue())
		})
	})

	DescribeTable("Constructors",
		func(build func([]int) keyset.Set[int]) {
			s := build([]int{1, 2, 3})
			Expect(s.Bounded()).To(BeTrue())
			Expect(s.Keys()).To(Equal([]int{1, 2, 3}))
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(4)).To(BeFalse())
			empty := build(nil)
			Expect(empty.Bounded()).To(BeTrue())
			Expect(empty.Keys()).To(BeEmpty())
			Expect(empty.Keys()).ToNot(BeNil())
			Expect(empty.Contains(1)).To(BeFalse())
		},
		Entry("Of", keyset.Of[int]),
		Entry("Hashed", keyset.Hashed[int]),
		Entry("Ranked", keyset.Ranked[int]),
	)

	It("Should answer concurrent probes of a hashed set", func() {
		s := keyset.Hashed([]int{1, 2, 3})
		var wg sync.WaitGroup
		results := make([]bool, 8)
		for i := range results {
			wg.Go(func() { results[i] = s.Contains(2) })
		}
		wg.Wait()
		Expect(results).To(HaveEach(BeTrue()))
	})

	Describe("Intersect", func() {
		It("Should be unbounded when no set is bounded", func() {
			Expect(keyset.Intersect[int]().Bounded()).To(BeFalse())
			Expect(keyset.Intersect(keyset.Set[int]{}, keyset.Set[int]{}).Bounded()).
				To(BeFalse())
		})

		It("Should ignore unbounded sets", func() {
			s := keyset.Intersect(keyset.Set[int]{}, keyset.Of([]int{1, 2}))
			Expect(s.Keys()).To(Equal([]int{1, 2}))
		})

		It("Should keep only the keys every bounded set contains", func() {
			s := keyset.Intersect(
				keyset.Of([]int{1, 2, 3, 4}),
				keyset.Hashed([]int{2, 3, 5}),
				keyset.Of([]int{3, 2}),
			)
			Expect(s.Keys()).To(ConsistOf(2, 3))
		})

		It("Should keep the order of a ranked set that is not the largest", func() {
			s := keyset.Intersect(
				keyset.Of([]int{5, 4, 3, 2, 1}),
				keyset.Ranked([]int{2, 9, 4}),
			)
			Expect(s.Keys()).To(Equal([]int{2, 4}))
		})

		It("Should keep the order of a ranked set that is the largest", func() {
			s := keyset.Intersect(keyset.Ranked([]int{3, 1, 2}), keyset.Of([]int{2, 3}))
			Expect(s.Keys()).To(Equal([]int{3, 2}))
		})

		It("Should keep the order of the shortest ranked set", func() {
			s := keyset.Intersect(
				keyset.Ranked([]int{1, 2, 3, 4}),
				keyset.Ranked([]int{3, 1}),
			)
			Expect(s.Keys()).To(Equal([]int{3, 1}))
		})

		It("Should rank the result, so an outer intersection keeps the order", func() {
			inner := keyset.Intersect(
				keyset.Ranked([]int{2, 4}),
				keyset.Of([]int{4, 2, 1}),
			)
			s := keyset.Intersect(keyset.Of([]int{4, 3, 2, 1}), inner)
			Expect(s.Keys()).To(Equal([]int{2, 4}))
		})

		It("Should be bounded and empty when the sets do not overlap", func() {
			s := keyset.Intersect(keyset.Of([]int{1}), keyset.Of([]int{2}))
			Expect(s.Bounded()).To(BeTrue())
			Expect(s.Keys()).To(BeEmpty())
			Expect(s.Contains(1)).To(BeFalse())
		})
	})

	Describe("Union", func() {
		It("Should be unbounded when no sets are given", func() {
			Expect(keyset.Union[int]().Bounded()).To(BeFalse())
		})

		It("Should be unbounded when any set is unbounded", func() {
			s := keyset.Union(keyset.Of([]int{1}), keyset.Set[int]{})
			Expect(s.Bounded()).To(BeFalse())
		})

		It("Should contain each key of every set once", func() {
			s := keyset.Union(
				keyset.Of([]int{1, 2, 3}),
				keyset.Hashed([]int{3, 4}),
				keyset.Of([]int{4, 5, 1}),
			)
			Expect(s.Keys()).To(ConsistOf(1, 2, 3, 4, 5))
		})

		It("Should be bounded and empty for empty sets", func() {
			s := keyset.Union(keyset.Of[int](nil), keyset.Hashed[int](nil))
			Expect(s.Bounded()).To(BeTrue())
			Expect(s.Keys()).To(BeEmpty())
		})
	})
})
