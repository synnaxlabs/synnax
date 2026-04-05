// Copyright 2026 Synnax Labs, Inc.
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

	Describe("New", func() {
		It("Should create a set from multiple elements", func() {
			s = set.New(1, 2, 3)
			Expect(s).To(HaveLen(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
		})

		It("Should create a set from a single element", func() {
			s = set.New(1)
			Expect(s).To(HaveLen(1))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeFalse())
		})

		It("Should deduplicate elements", func() {
			s = set.New(1, 2, 3, 2, 1)
			Expect(s).To(HaveLen(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
		})

		It("Should create an empty set with no elements", func() {
			s = set.New[int]()
			Expect(s).To(HaveLen(0))
		})
	})

	Describe("Add", func() {
		It("Should add multiple elements", func() {
			s.Add(1, 2, 3)
			Expect(s).To(HaveLen(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
		})

		It("Should handle duplicate elements across calls", func() {
			s.Add(1, 2, 3)
			s.Add(2, 3, 4)
			Expect(s).To(HaveLen(4))
			Expect(s.Contains(4)).To(BeTrue())
		})

		It("Should handle duplicate elements within a single call", func() {
			s.Add(1, 1, 2, 2, 3)
			Expect(s).To(HaveLen(3))
		})

		It("Should add a single element", func() {
			s.Add(42)
			Expect(s).To(HaveLen(1))
			Expect(s.Contains(42)).To(BeTrue())
		})

		It("Should be a no-op with no arguments", func() {
			s.Add(1)
			s.Add()
			Expect(s).To(HaveLen(1))
		})

		It("Should return the set for chaining", func() {
			result := s.Add(1, 2).Add(3, 4)
			Expect(result).To(HaveLen(4))
			Expect(s).To(HaveLen(4))
		})
	})

	Describe("Remove", func() {
		It("Should remove multiple elements", func() {
			s.Add(1, 2, 3, 4, 5)
			s.Remove(2, 4)
			Expect(s).To(HaveLen(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(5)).To(BeTrue())
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(4)).To(BeFalse())
		})

		It("Should be a no-op for non-existent elements", func() {
			s.Add(1, 2, 3)
			s.Remove(4, 5)
			Expect(s).To(HaveLen(3))
		})

		It("Should be a no-op on an empty set", func() {
			s.Remove(1, 2, 3)
			Expect(s).To(HaveLen(0))
		})

		It("Should be a no-op with no elements", func() {
			s.Add(1, 2)
			s.Remove()
			Expect(s).To(HaveLen(2))
		})

		It("Should remove all elements when all are specified", func() {
			s.Add(1, 2, 3)
			s.Remove(1, 2, 3)
			Expect(s).To(HaveLen(0))
		})

		It("Should remove a single element", func() {
			s.Add(1, 2, 3)
			s.Remove(2)
			Expect(s).To(HaveLen(2))
			Expect(s.Contains(2)).To(BeFalse())
		})

		It("Should return the set for chaining", func() {
			s.Add(1, 2, 3, 4)
			result := s.Remove(1).Remove(2)
			Expect(result).To(HaveLen(2))
			Expect(s).To(HaveLen(2))
		})
	})

	Describe("Contains", func() {
		It("Should return true for elements that exist", func() {
			s.Add(1, 3, 5)
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(5)).To(BeTrue())
		})

		It("Should return false for elements that do not exist", func() {
			s.Add(1, 3, 5)
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(4)).To(BeFalse())
		})

		It("Should return false on an empty set", func() {
			Expect(s.Contains(1)).To(BeFalse())
		})
	})

	Describe("Slice", func() {
		It("Should return all elements", func() {
			s.Add(1, 2, 3)
			Expect(s.Slice()).To(ConsistOf(1, 2, 3))
		})

		It("Should return an empty slice for an empty set", func() {
			Expect(s.Slice()).To(BeEmpty())
		})

		It("Should return a slice with a single element", func() {
			s.Add(42)
			Expect(s.Slice()).To(ConsistOf(42))
		})
	})

	Describe("Copy", func() {
		It("Should create an independent copy", func() {
			s.Add(1, 2, 3)
			copied := s.Copy()
			Expect(copied).To(HaveLen(3))
			Expect(copied.Contains(1)).To(BeTrue())
			Expect(copied.Contains(2)).To(BeTrue())
			Expect(copied.Contains(3)).To(BeTrue())
		})

		It("Should not be affected by modifications to the copy", func() {
			s.Add(1, 2, 3)
			copied := s.Copy()
			copied.Add(4, 5)
			copied.Remove(1)
			Expect(s).To(HaveLen(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(4)).To(BeFalse())
		})

		It("Should not affect the copy when modifying the original", func() {
			s.Add(1, 2, 3)
			copied := s.Copy()
			s.Add(4, 5)
			s.Remove(1)
			Expect(copied).To(HaveLen(3))
			Expect(copied.Contains(1)).To(BeTrue())
			Expect(copied.Contains(4)).To(BeFalse())
		})

		It("Should handle copying an empty set", func() {
			copied := s.Copy()
			Expect(copied).To(HaveLen(0))
			copied.Add(1, 2, 3)
			Expect(s).To(HaveLen(0))
		})
	})

	Describe("Equal", func() {
		It("Should return true for identical sets", func() {
			a := set.New(1, 2, 3)
			b := set.New(1, 2, 3)
			Expect(a.Equal(b)).To(BeTrue())
		})

		It("Should return true for empty sets", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])
			Expect(a.Equal(b)).To(BeTrue())
		})

		It("Should return false for different lengths", func() {
			a := set.New(1, 2, 3)
			b := set.New(1, 2)
			Expect(a.Equal(b)).To(BeFalse())
		})

		It("Should return false for same length but different elements", func() {
			a := set.New(1, 2, 3)
			b := set.New(1, 2, 4)
			Expect(a.Equal(b)).To(BeFalse())
		})

		It("Should be order-independent", func() {
			a := set.New(3, 1, 2)
			b := set.New(1, 2, 3)
			Expect(a.Equal(b)).To(BeTrue())
		})

		It("Should be symmetric", func() {
			a := set.New(1, 2, 3)
			b := set.New(4, 5, 6)
			Expect(a.Equal(b)).To(Equal(b.Equal(a)))
		})

		It("Should work with string sets", func() {
			a := set.New("apple", "banana", "cherry")
			b := set.New("cherry", "apple", "banana")
			Expect(a.Equal(b)).To(BeTrue())
		})

		It("Should return false when one set is a superset of the other", func() {
			a := set.New(1, 2, 3)
			b := set.New(1, 2, 3, 4, 5)
			Expect(a.Equal(b)).To(BeFalse())
		})
	})

	Describe("IsSubsetOf", func() {
		It("Should return true for a proper subset", func() {
			a := set.New(1, 2, 3)
			b := set.New(1, 2, 3, 4, 5)
			Expect(a.IsSubsetOf(b)).To(BeTrue())
		})

		It("Should return true for equal sets", func() {
			a := set.New(1, 2, 3)
			b := set.New(1, 2, 3)
			Expect(a.IsSubsetOf(b)).To(BeTrue())
		})

		It("Should return true when the set is empty", func() {
			a := make(set.Set[int])
			b := set.New(1, 2, 3)
			Expect(a.IsSubsetOf(b)).To(BeTrue())
		})

		It("Should return true when both sets are empty", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])
			Expect(a.IsSubsetOf(b)).To(BeTrue())
		})

		It("Should return false when the set has extra elements", func() {
			a := set.New(1, 2, 3, 4)
			b := set.New(1, 2, 3)
			Expect(a.IsSubsetOf(b)).To(BeFalse())
		})

		It("Should return false when sets have no overlap", func() {
			a := set.New(1, 2, 3)
			b := set.New(4, 5, 6)
			Expect(a.IsSubsetOf(b)).To(BeFalse())
		})

		It("Should return false when sets have partial overlap", func() {
			a := set.New(1, 2, 3, 4)
			b := set.New(2, 3, 5, 6)
			Expect(a.IsSubsetOf(b)).To(BeFalse())
		})

		It("Should return false when other is empty but set is not", func() {
			a := set.New(1, 2, 3)
			b := make(set.Set[int])
			Expect(a.IsSubsetOf(b)).To(BeFalse())
		})

		It("Should work with string sets", func() {
			a := set.New("apple", "banana")
			b := set.New("apple", "banana", "cherry", "date")
			Expect(a.IsSubsetOf(b)).To(BeTrue())
		})

		It("Should work with a single element subset", func() {
			a := set.New(42)
			b := set.New(42, 43, 44)
			Expect(a.IsSubsetOf(b)).To(BeTrue())
		})

		It("Should return false for a single element not in other", func() {
			a := set.New(1)
			b := set.New(2, 3, 4)
			Expect(a.IsSubsetOf(b)).To(BeFalse())
		})
	})

	Describe("Difference", func() {
		It("Should return elements in a but not in b", func() {
			a := set.New(1, 2, 3, 4, 5)
			b := set.New(3, 4, 5, 6, 7)
			diff := a.Difference(b)
			Expect(diff).To(HaveLen(2))
			Expect(diff.Contains(1)).To(BeTrue())
			Expect(diff.Contains(2)).To(BeTrue())
			Expect(diff.Contains(3)).To(BeFalse())
		})

		It("Should return empty when a is a subset of b", func() {
			a := set.New(1, 2, 3)
			b := set.New(1, 2, 3, 4, 5)
			Expect(a.Difference(b)).To(HaveLen(0))
		})

		It("Should return all of a when b is empty", func() {
			a := set.New(1, 2, 3)
			b := make(set.Set[int])
			diff := a.Difference(b)
			Expect(diff).To(HaveLen(3))
			Expect(diff.Contains(1)).To(BeTrue())
			Expect(diff.Contains(2)).To(BeTrue())
			Expect(diff.Contains(3)).To(BeTrue())
		})

		It("Should return empty when a is empty", func() {
			a := make(set.Set[int])
			b := set.New(1, 2, 3)
			Expect(a.Difference(b)).To(HaveLen(0))
		})

		It("Should return empty when both are empty", func() {
			a := make(set.Set[int])
			b := make(set.Set[int])
			Expect(a.Difference(b)).To(HaveLen(0))
		})

		It("Should return all of a when there is no overlap", func() {
			a := set.New(1, 2, 3)
			b := set.New(4, 5, 6)
			diff := a.Difference(b)
			Expect(diff).To(HaveLen(3))
			Expect(diff.Contains(1)).To(BeTrue())
			Expect(diff.Contains(2)).To(BeTrue())
			Expect(diff.Contains(3)).To(BeTrue())
		})

		It("Should return empty when sets are identical", func() {
			a := set.New(1, 2, 3)
			b := set.New(1, 2, 3)
			Expect(a.Difference(b)).To(HaveLen(0))
		})

		It("Should not modify the original sets", func() {
			a := set.New(1, 2, 3, 4)
			b := set.New(3, 4, 5, 6)
			_ = a.Difference(b)
			Expect(a).To(HaveLen(4))
			Expect(b).To(HaveLen(4))
			Expect(a.Contains(3)).To(BeTrue())
			Expect(b.Contains(5)).To(BeTrue())
		})

		It("Should be asymmetric", func() {
			a := set.New(1, 2, 3)
			b := set.New(2, 3, 4)
			ab := a.Difference(b)
			ba := b.Difference(a)
			Expect(ab).To(HaveLen(1))
			Expect(ab.Contains(1)).To(BeTrue())
			Expect(ba).To(HaveLen(1))
			Expect(ba.Contains(4)).To(BeTrue())
		})
	})
})
