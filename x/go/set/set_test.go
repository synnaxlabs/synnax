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

	Describe("Equals", func() {
		It("should return true for identical sets", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3)
			b.Add(1, 2, 3)

			Expect(a.Equals(b)).To(BeTrue())
			Expect(b.Equals(a)).To(BeTrue())
		})

		It("should return true for empty sets", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			Expect(a.Equals(b)).To(BeTrue())
		})

		It("should return false when sets have different lengths", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3)
			b.Add(1, 2)

			Expect(a.Equals(b)).To(BeFalse())
		})

		It("should return false when sets have same length but different elements", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3)
			b.Add(1, 2, 4)

			Expect(a.Equals(b)).To(BeFalse())
		})

		It("should return true regardless of insertion order", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(3, 1, 2)
			b.Add(1, 2, 3)

			Expect(a.Equals(b)).To(BeTrue())
		})

		It("should work with string sets", func() {
			a := make(set.Set[string])
			b := make(set.Set[string])

			a.Add("apple", "banana", "cherry")
			b.Add("cherry", "apple", "banana")

			Expect(a.Equals(b)).To(BeTrue())
		})

		It("should return false when one set has extra elements", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3)
			b.Add(1, 2, 3, 4, 5)

			Expect(a.Equals(b)).To(BeFalse())
		})

		It("should work with Mapped types", func() {
			a := make(set.Mapped[string, int])
			b := make(set.Mapped[string, int])

			a["one"] = 1
			a["two"] = 2
			a["three"] = 3

			b["one"] = 1
			b["two"] = 2
			b["three"] = 3

			Expect(a.Equals(b)).To(BeTrue())
		})

		It("should compare keys only for Mapped types, not values", func() {
			a := make(set.Mapped[string, int])
			b := make(set.Mapped[string, int])

			a["one"] = 1
			a["two"] = 2

			b["one"] = 100
			b["two"] = 200

			Expect(a.Equals(b)).To(BeTrue())
		})
	})

	Describe("Subset", func() {
		It("should return true when s is a subset of other", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			s.Add(1, 2, 3)
			other.Add(1, 2, 3, 4, 5)

			Expect(s.IsSubsetOf(other)).To(BeTrue())
		})

		It("should return true when sets are equal", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			s.Add(1, 2, 3)
			other.Add(1, 2, 3)

			Expect(s.IsSubsetOf(other)).To(BeTrue())
		})

		It("should return true when s is empty", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			other.Add(1, 2, 3)

			Expect(s.IsSubsetOf(other)).To(BeTrue())
		})

		It("should return true when both sets are empty", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			Expect(s.IsSubsetOf(other)).To(BeTrue())
		})

		It("should return false when s has elements not in other", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			s.Add(1, 2, 3, 4)
			other.Add(1, 2, 3)

			Expect(s.IsSubsetOf(other)).To(BeFalse())
		})

		It("should return false when sets have no overlap", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			s.Add(1, 2, 3)
			other.Add(4, 5, 6)

			Expect(s.IsSubsetOf(other)).To(BeFalse())
		})

		It("should return false when s has some overlapping and some non-overlapping elements", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			s.Add(1, 2, 3, 4)
			other.Add(2, 3, 5, 6)

			Expect(s.IsSubsetOf(other)).To(BeFalse())
		})

		It("should return false when other is empty but s is not", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			s.Add(1, 2, 3)

			Expect(s.IsSubsetOf(other)).To(BeFalse())
		})

		It("should work with string sets", func() {
			s := make(set.Set[string])
			other := make(set.Set[string])

			s.Add("apple", "banana")
			other.Add("apple", "banana", "cherry", "date")

			Expect(s.IsSubsetOf(other)).To(BeTrue())
		})

		It("should work with single element sets", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			s.Add(42)
			other.Add(42, 43, 44)

			Expect(s.IsSubsetOf(other)).To(BeTrue())
		})

		It("should return false for single element not in other", func() {
			s := make(set.Set[int])
			other := make(set.Set[int])

			s.Add(1)
			other.Add(2, 3, 4)

			Expect(s.IsSubsetOf(other)).To(BeFalse())
		})

		It("should work with Mapped types", func() {
			s := make(set.Mapped[string, int])
			other := make(set.Mapped[string, int])

			s["one"] = 1
			s["two"] = 2

			other["one"] = 100
			other["two"] = 200
			other["three"] = 300

			Expect(s.IsSubsetOf(other)).To(BeTrue())
		})
	})

	Describe("Difference", func() {
		It("should return elements in a but not in b", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3, 4, 5)
			b.Add(3, 4, 5, 6, 7)

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(2))
			Expect(diff.Contains(1)).To(BeTrue())
			Expect(diff.Contains(2)).To(BeTrue())
			Expect(diff.Contains(3)).To(BeFalse())
			Expect(diff.Contains(4)).To(BeFalse())
			Expect(diff.Contains(5)).To(BeFalse())
		})

		It("should return empty set when a is a subset of b", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3)
			b.Add(1, 2, 3, 4, 5)

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(0))
		})

		It("should return all elements of a when b is empty", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3)

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(3))
			Expect(diff.Contains(1)).To(BeTrue())
			Expect(diff.Contains(2)).To(BeTrue())
			Expect(diff.Contains(3)).To(BeTrue())
		})

		It("should return empty set when a is empty", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			b.Add(1, 2, 3)

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(0))
		})

		It("should return empty set when both sets are empty", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(0))
		})

		It("should return all elements when sets have no overlap", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3)
			b.Add(4, 5, 6)

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(3))
			Expect(diff.Contains(1)).To(BeTrue())
			Expect(diff.Contains(2)).To(BeTrue())
			Expect(diff.Contains(3)).To(BeTrue())
		})

		It("should return empty set when sets are identical", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3)
			b.Add(1, 2, 3)

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(0))
		})

		It("should work with Mapped types with actual values", func() {
			a := make(set.Mapped[string, int])
			b := make(set.Mapped[string, int])

			a["one"] = 1
			a["two"] = 2
			a["three"] = 3
			a["four"] = 4

			b["two"] = 20
			b["three"] = 30
			b["five"] = 50

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(2))
			Expect(diff["one"]).To(Equal(1))
			Expect(diff["four"]).To(Equal(4))
			_, hasTwoKey := diff["two"]
			Expect(hasTwoKey).To(BeFalse())
			_, hasThreeKey := diff["three"]
			Expect(hasThreeKey).To(BeFalse())
		})

		It("should preserve values from set a in the result", func() {
			a := make(set.Mapped[string, int])
			b := make(set.Mapped[string, int])

			a["apple"] = 10
			a["banana"] = 20
			a["cherry"] = 30

			b["banana"] = 999 // Different value in b

			diff := set.Difference(a, b)

			Expect(len(diff)).To(Equal(2))
			Expect(diff["apple"]).To(Equal(10))
			Expect(diff["cherry"]).To(Equal(30))
			_, hasBanana := diff["banana"]
			Expect(hasBanana).To(BeFalse())
		})

		It("should not modify the original sets", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])

			a.Add(1, 2, 3, 4)
			b.Add(3, 4, 5, 6)

			originalALen := len(a)
			originalBLen := len(b)

			_ = set.Difference(a, b)

			Expect(len(a)).To(Equal(originalALen))
			Expect(len(b)).To(Equal(originalBLen))
			Expect(a.Contains(1)).To(BeTrue())
			Expect(a.Contains(3)).To(BeTrue())
			Expect(b.Contains(3)).To(BeTrue())
			Expect(b.Contains(5)).To(BeTrue())
		})
	})
})
