// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package set_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Set", func() {
	var s set.Set[int]

	BeforeEach(func() {
		s = make(set.Set[int])
	})

	Describe("Add", func() {
		It("should add elements to the set", func() {
			s.Add(1, 2, 3)
			Expect(len(s)).To(Equal(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
		})

		It("should handle duplicate elements", func() {
			s.Add(1, 2, 3)
			s.Add(2, 3, 4)
			Expect(len(s)).To(Equal(4))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(4)).To(BeTrue())
		})
	})

	Describe("Remove", func() {
		It("should remove elements from the set", func() {
			s.Add(1, 2, 3, 4, 5)
			s.Remove(2, 4)
			Expect(len(s)).To(Equal(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(5)).To(BeTrue())
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(4)).To(BeFalse())
		})

		It("should handle removing non-existent elements", func() {
			s.Add(1, 2, 3)
			s.Remove(4, 5)
			Expect(len(s)).To(Equal(3))
		})
	})

	Describe("Contains", func() {
		It("should correctly check if elements exist", func() {
			s.Add(1, 3, 5)
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(5)).To(BeTrue())
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(4)).To(BeFalse())
		})

		It("should return false for empty sets", func() {
			Expect(s.Contains(1)).To(BeFalse())
		})
	})

	Describe("Keys", func() {
		It("should return all keys in the set", func() {
			s.Add(1, 2, 3)
			keys := s.Keys()
			Expect(len(keys)).To(Equal(3))
			Expect(keys).To(ContainElements(1, 2, 3))
		})

		It("should return an empty slice for empty sets", func() {
			keys := s.Keys()
			Expect(len(keys)).To(Equal(0))
		})
	})

	Describe("FromSlice", func() {
		It("should create a set from a slice", func() {
			slice := []int{1, 2, 3, 2, 1}
			s = set.FromSlice(slice)
			Expect(len(s)).To(Equal(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
		})
	})

	Describe("Values", func() {
		It("should return all values in a mapped set", func() {
			m := make(set.Mapped[string, int])
			m["one"] = 1
			m["two"] = 2
			m["three"] = 3

			values := m.Values()
			Expect(len(values)).To(Equal(3))
			Expect(values).To(ContainElements(1, 2, 3))
		})
	})

	Describe("Reset", func() {
		It("should clear all elements from the set", func() {
			s.Add(1, 2, 3, 4, 5)
			Expect(len(s)).To(Equal(5))

			s.Reset()
			Expect(len(s)).To(Equal(0))
			Expect(s.Contains(1)).To(BeFalse())
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(3)).To(BeFalse())
		})

		It("should be idempotent when called on an empty set", func() {
			s.Reset()
			Expect(len(s)).To(Equal(0))

			s.Reset()
			Expect(len(s)).To(Equal(0))
		})

		It("should allow adding elements after reset", func() {
			s.Add(1, 2, 3)
			s.Reset()
			s.Add(4, 5, 6)

			Expect(len(s)).To(Equal(3))
			Expect(s.Contains(1)).To(BeFalse())
			Expect(s.Contains(4)).To(BeTrue())
			Expect(s.Contains(5)).To(BeTrue())
			Expect(s.Contains(6)).To(BeTrue())
		})
	})

	Describe("Copy", func() {
		It("should create an independent copy of the set", func() {
			s.Add(1, 2, 3)
			copied := s.Copy()

			Expect(len(copied)).To(Equal(3))
			Expect(copied.Contains(1)).To(BeTrue())
			Expect(copied.Contains(2)).To(BeTrue())
			Expect(copied.Contains(3)).To(BeTrue())
		})

		It("should not affect the original when modifying the copy", func() {
			s.Add(1, 2, 3)
			copied := s.Copy()

			copied.Add(4, 5)
			copied.Remove(1)

			Expect(len(s)).To(Equal(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(4)).To(BeFalse())

			Expect(len(copied)).To(Equal(4))
			Expect(copied.Contains(1)).To(BeFalse())
			Expect(copied.Contains(4)).To(BeTrue())
		})

		It("should not affect the copy when modifying the original", func() {
			s.Add(1, 2, 3)
			copied := s.Copy()

			s.Add(4, 5)
			s.Remove(1)

			Expect(len(s)).To(Equal(4))
			Expect(s.Contains(1)).To(BeFalse())
			Expect(s.Contains(4)).To(BeTrue())

			Expect(len(copied)).To(Equal(3))
			Expect(copied.Contains(1)).To(BeTrue())
			Expect(copied.Contains(4)).To(BeFalse())
		})

		It("should handle copying empty sets", func() {
			copied := s.Copy()

			Expect(len(copied)).To(Equal(0))
			copied.Add(1, 2, 3)
			Expect(len(s)).To(Equal(0))
			Expect(len(copied)).To(Equal(3))
		})

		It("should work with Mapped types with non-empty values", func() {
			m := make(set.Mapped[string, int])
			m["one"] = 1
			m["two"] = 2
			m["three"] = 3

			copied := m.Copy()

			Expect(len(copied)).To(Equal(3))
			Expect(copied["one"]).To(Equal(1))
			Expect(copied["two"]).To(Equal(2))
			Expect(copied["three"]).To(Equal(3))

			// Modify copy
			copied["four"] = 4
			delete(copied, "one")

			// Original should be unchanged
			Expect(len(m)).To(Equal(3))
			Expect(m["one"]).To(Equal(1))
			_, ok := m["four"]
			Expect(ok).To(BeFalse())
		})
	})
})
