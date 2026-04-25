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

var _ = Describe("Integer", func() {
	var s set.Integer[int]

	BeforeEach(func() {
		s = set.Integer[int]{}
	})

	Describe("NewInteger", func() {
		It("Should create a set from sorted contiguous input", func() {
			s := set.NewInteger([]int{1, 2, 3, 4, 5})
			Expect(s.Size()).To(Equal(5))
			Expect(*s).To(HaveLen(1))
		})

		It("Should create a set from sorted non-contiguous input", func() {
			s := set.NewInteger([]int{1, 2, 5, 6})
			Expect(s.Size()).To(Equal(4))
			Expect(*s).To(HaveLen(2))
		})

		It("Should create a set from unsorted input", func() {
			s := set.NewInteger([]int{5, 1, 3, 2, 4})
			Expect(s.Size()).To(Equal(5))
			for i := 1; i <= 5; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
		})

		It("Should handle empty input", func() {
			s := set.NewInteger([]int{})
			Expect(s.Size()).To(Equal(0))
			Expect(*s).To(BeEmpty())
		})

		It("Should handle a single element", func() {
			s := set.NewInteger([]int{42})
			Expect(s.Size()).To(Equal(1))
			Expect(s.Contains(42)).To(BeTrue())
		})

		It("Should deduplicate input", func() {
			s := set.NewInteger([]int{1, 1, 2, 2, 3})
			Expect(s.Size()).To(Equal(3))
		})

		It("Should handle nil input", func() {
			s := set.NewInteger[int](nil)
			Expect(s.Size()).To(Equal(0))
		})
	})

	Describe("Insert", func() {
		It("Should insert a single element into an empty set", func() {
			s.Insert(42)
			Expect(s.Size()).To(Equal(1))
			Expect(s.Contains(42)).To(BeTrue())
		})

		It("Should insert a contiguous ascending sequence as one interval", func() {
			s.Insert(0, 1, 2, 3, 4)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(5))
		})

		It("Should insert non-contiguous elements as separate intervals", func() {
			s.Insert(1, 3, 5)
			Expect(s).To(HaveLen(3))
			Expect(s.Size()).To(Equal(3))
		})

		It("Should be idempotent for duplicate insertions", func() {
			s.Insert(1, 2, 3)
			s.Insert(1, 2, 3)
			Expect(s.Size()).To(Equal(3))
			Expect(s).To(HaveLen(1))
		})

		It("Should be a no-op with no arguments", func() {
			s.Insert(1, 2, 3)
			s.Insert()
			Expect(s.Size()).To(Equal(3))
		})

		It("Should handle unsorted contiguous input", func() {
			s.Insert(7, 3, 4, 1, 2, 5, 6)
			Expect(s.Size()).To(Equal(7))
			Expect(s).To(HaveLen(1))
		})

		It("Should handle unsorted non-contiguous input", func() {
			s.Insert(50, 20, 40, 10, 30)
			Expect(s.Size()).To(Equal(5))
			Expect(s).To(HaveLen(5))
		})

		It("Should handle unsorted input with duplicates", func() {
			s.Insert(10, 5, 5, 8, 1, 3, 8, 2, 4)
			Expect(s.Size()).To(Equal(7))
			expected := []int{1, 2, 3, 4, 5, 8, 10}
			for _, num := range expected {
				Expect(s.Contains(num)).To(BeTrue())
			}
		})

		It("Should handle negative numbers", func() {
			s.Insert(-3, -2, -1)
			Expect(s.Size()).To(Equal(3))
			Expect(s).To(HaveLen(1))
			Expect(s.Contains(-3)).To(BeTrue())
			Expect(s.Contains(-1)).To(BeTrue())
			Expect(s.Contains(0)).To(BeFalse())
		})

		It("Should handle a mix of negative and positive numbers", func() {
			s.Insert(-2, -1, 0, 1, 2)
			Expect(s.Size()).To(Equal(5))
			Expect(s).To(HaveLen(1))
			for i := -2; i <= 2; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
		})

		It("Should handle non-contiguous negative and positive ranges", func() {
			s.Insert(-5, -4, -3, 3, 4, 5)
			Expect(s.Size()).To(Equal(6))
			Expect(s).To(HaveLen(2))
		})

		It("Should compress adjacent intervals on left insertion", func() {
			s.Insert(5, 6, 7)
			s.Insert(4)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(4))
			Expect(s.Contains(4)).To(BeTrue())
		})

		It("Should compress adjacent intervals on right insertion", func() {
			s.Insert(5, 6, 7)
			s.Insert(8)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(4))
			Expect(s.Contains(8)).To(BeTrue())
		})

		It("Should bridge two intervals when the gap is filled", func() {
			s.Insert(1, 2, 3)
			s.Insert(5, 6, 7)
			Expect(s).To(HaveLen(2))
			s.Insert(4)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(7))
		})

		It("Should merge overlapping insertions", func() {
			s.Insert(1, 2, 3, 4, 5)
			s.Insert(3, 4, 5, 6, 7)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(7))
		})

		It("Should extend an existing interval to the left", func() {
			s.Insert(5, 6, 7, 8, 9)
			s.Insert(3, 4, 5)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(7))
			Expect(s.Contains(3)).To(BeTrue())
		})

		It("Should extend an existing interval to the right", func() {
			s.Insert(5, 6, 7, 8, 9)
			s.Insert(8, 9, 10, 11)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(7))
			Expect(s.Contains(11)).To(BeTrue())
		})

		It("Should handle inserting a range fully contained in an existing interval", func() {
			s.Insert(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
			s.Insert(4, 5, 6)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(10))
		})

		It("Should handle inserting a range that spans multiple existing intervals", func() {
			s.Insert(0, 1, 2)
			s.Insert(5, 6, 7)
			s.Insert(10, 11, 12)
			Expect(s).To(HaveLen(3))
			s.Insert(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(13))
		})

		It("Should handle zero", func() {
			s.Insert(0)
			Expect(s.Size()).To(Equal(1))
			Expect(s.Contains(0)).To(BeTrue())
			Expect(s.Contains(-1)).To(BeFalse())
			Expect(s.Contains(1)).To(BeFalse())
		})
	})

	Describe("Remove", func() {
		It("Should remove a single element from the middle of a range", func() {
			s.Insert(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
			s.Remove(5)
			Expect(s).To(HaveLen(2))
			Expect(s.Size()).To(Equal(9))
			Expect(s.Contains(5)).To(BeFalse())
			Expect(s.Contains(4)).To(BeTrue())
			Expect(s.Contains(6)).To(BeTrue())
		})

		It("Should remove from the start of a range", func() {
			s.Insert(0, 1, 2, 3, 4)
			s.Remove(0)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(4))
			Expect(s.Contains(0)).To(BeFalse())
			Expect(s.Contains(1)).To(BeTrue())
		})

		It("Should remove from the end of a range", func() {
			s.Insert(0, 1, 2, 3, 4)
			s.Remove(4)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(4))
			Expect(s.Contains(4)).To(BeFalse())
			Expect(s.Contains(3)).To(BeTrue())
		})

		It("Should remove an entire range", func() {
			s.Insert(0, 1, 2)
			s.Insert(5, 6, 7)
			s.Remove(0, 1, 2)
			Expect(s).To(HaveLen(1))
			Expect(s.Size()).To(Equal(3))
			Expect(s.Contains(0)).To(BeFalse())
			Expect(s.Contains(5)).To(BeTrue())
		})

		It("Should remove a contiguous sub-range from the middle", func() {
			s.Insert(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
			s.Remove(3, 4, 5, 6)
			Expect(s).To(HaveLen(2))
			Expect(s.Size()).To(Equal(6))
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeFalse())
			Expect(s.Contains(6)).To(BeFalse())
			Expect(s.Contains(7)).To(BeTrue())
		})

		It("Should remove elements spanning multiple ranges", func() {
			s.Insert(0, 1, 2, 3, 4)
			s.Insert(10, 11, 12, 13, 14)
			s.Remove(3, 4, 10, 11)
			Expect(s).To(HaveLen(2))
			Expect(s.Size()).To(Equal(6))
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeFalse())
			Expect(s.Contains(10)).To(BeFalse())
			Expect(s.Contains(12)).To(BeTrue())
		})

		It("Should be idempotent for non-existent elements", func() {
			s.Insert(1, 2, 3)
			s.Remove(10, 20, 30)
			Expect(s.Size()).To(Equal(3))
		})

		It("Should be a no-op on an empty set", func() {
			s.Remove(1, 2, 3)
			Expect(s).To(BeEmpty())
		})

		It("Should be a no-op with no arguments", func() {
			s.Insert(1, 2, 3)
			s.Remove()
			Expect(s.Size()).To(Equal(3))
		})

		It("Should remove all elements", func() {
			s.Insert(0, 1, 2, 3, 4)
			s.Remove(0, 1, 2, 3, 4)
			Expect(s).To(BeEmpty())
			Expect(s.Size()).To(Equal(0))
		})

		It("Should handle removing negative numbers", func() {
			s.Insert(-5, -4, -3, -2, -1)
			s.Remove(-3)
			Expect(s).To(HaveLen(2))
			Expect(s.Size()).To(Equal(4))
			Expect(s.Contains(-3)).To(BeFalse())
			Expect(s.Contains(-4)).To(BeTrue())
			Expect(s.Contains(-2)).To(BeTrue())
		})

		It("Should handle a mix of existing and non-existent elements", func() {
			s.Insert(1, 2, 3, 4, 5)
			s.Remove(2, 4, 10, 20)
			Expect(s.Size()).To(Equal(3))
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(4)).To(BeFalse())
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(5)).To(BeTrue())
		})
	})

	Describe("Contains", func() {
		It("Should return true for the first element of a range", func() {
			s.Insert(5, 6, 7, 8, 9)
			Expect(s.Contains(5)).To(BeTrue())
		})

		It("Should return true for the last element of a range", func() {
			s.Insert(5, 6, 7, 8, 9)
			Expect(s.Contains(9)).To(BeTrue())
		})

		It("Should return true for a middle element of a range", func() {
			s.Insert(5, 6, 7, 8, 9)
			Expect(s.Contains(7)).To(BeTrue())
		})

		It("Should return false just before the start of a range", func() {
			s.Insert(5, 6, 7, 8, 9)
			Expect(s.Contains(4)).To(BeFalse())
		})

		It("Should return false just after the end of a range", func() {
			s.Insert(5, 6, 7, 8, 9)
			Expect(s.Contains(10)).To(BeFalse())
		})

		It("Should return false for elements in gaps between ranges", func() {
			s.Insert(1, 2, 3)
			s.Insert(7, 8, 9)
			Expect(s.Contains(5)).To(BeFalse())
		})

		It("Should return false for elements before all ranges", func() {
			s.Insert(10, 11, 12)
			Expect(s.Contains(0)).To(BeFalse())
		})

		It("Should return false for elements after all ranges", func() {
			s.Insert(1, 2, 3)
			Expect(s.Contains(100)).To(BeFalse())
		})

		It("Should return false on an empty set", func() {
			Expect(s.Contains(0)).To(BeFalse())
			Expect(s.Contains(42)).To(BeFalse())
		})

		It("Should handle negative numbers", func() {
			s.Insert(-5, -4, -3)
			Expect(s.Contains(-5)).To(BeTrue())
			Expect(s.Contains(-3)).To(BeTrue())
			Expect(s.Contains(-6)).To(BeFalse())
			Expect(s.Contains(-2)).To(BeFalse())
		})

		It("Should handle a single-element range", func() {
			s.Insert(42)
			Expect(s.Contains(42)).To(BeTrue())
			Expect(s.Contains(41)).To(BeFalse())
			Expect(s.Contains(43)).To(BeFalse())
		})

		It("Should handle multiple ranges with exact boundary checks", func() {
			s.Insert(1, 2, 3)
			s.Insert(10, 11, 12)
			s.Insert(20, 21, 22)
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(4)).To(BeFalse())
			Expect(s.Contains(9)).To(BeFalse())
			Expect(s.Contains(10)).To(BeTrue())
			Expect(s.Contains(12)).To(BeTrue())
			Expect(s.Contains(13)).To(BeFalse())
			Expect(s.Contains(19)).To(BeFalse())
			Expect(s.Contains(20)).To(BeTrue())
			Expect(s.Contains(22)).To(BeTrue())
			Expect(s.Contains(23)).To(BeFalse())
		})
	})

	Describe("NumLessThan", func() {
		It("Should return 0 when value is less than all elements", func() {
			s.Insert(5, 6, 7, 8, 9)
			Expect(s.NumLessThan(5)).To(Equal(0))
			Expect(s.NumLessThan(0)).To(Equal(0))
		})

		It("Should return the full size when value exceeds all elements", func() {
			s.Insert(1, 2, 3, 4, 5)
			Expect(s.NumLessThan(6)).To(Equal(5))
			Expect(s.NumLessThan(100)).To(Equal(5))
		})

		It("Should count correctly for a value in the middle of a range", func() {
			s.Insert(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
			Expect(s.NumLessThan(5)).To(Equal(5))
		})

		It("Should count correctly for a value at the start of a range", func() {
			s.Insert(5, 6, 7, 8, 9)
			Expect(s.NumLessThan(5)).To(Equal(0))
		})

		It("Should count correctly at the exclusive end of a range", func() {
			s.Insert(0, 1, 2, 3, 4)
			Expect(s.NumLessThan(5)).To(Equal(5))
		})

		It("Should count correctly for a value in a gap between ranges", func() {
			s.Insert(0, 1, 2)
			s.Insert(8, 9, 10, 11)
			Expect(s.NumLessThan(5)).To(Equal(3))
		})

		It("Should return 0 for an empty set", func() {
			Expect(s.NumLessThan(0)).To(Equal(0))
			Expect(s.NumLessThan(100)).To(Equal(0))
		})

		It("Should handle negative numbers", func() {
			s.Insert(-5, -4, -3, -2, -1)
			Expect(s.NumLessThan(-3)).To(Equal(2))
			Expect(s.NumLessThan(0)).To(Equal(5))
			Expect(s.NumLessThan(-10)).To(Equal(0))
		})

		It("Should count across multiple ranges", func() {
			s.Insert(0, 1, 2)
			s.Insert(5, 6, 7)
			s.Insert(10, 11, 12)
			Expect(s.NumLessThan(8)).To(Equal(6))
			Expect(s.NumLessThan(11)).To(Equal(7))
		})

		It("Should handle a single-element set", func() {
			s.Insert(5)
			Expect(s.NumLessThan(5)).To(Equal(0))
			Expect(s.NumLessThan(6)).To(Equal(1))
			Expect(s.NumLessThan(4)).To(Equal(0))
		})
	})

	Describe("NumGreaterThan", func() {
		It("Should return 0 when value exceeds all elements", func() {
			s.Insert(1, 2, 3, 4, 5)
			Expect(s.NumGreaterThan(5)).To(Equal(0))
			Expect(s.NumGreaterThan(100)).To(Equal(0))
		})

		It("Should return the full size when value is less than all elements", func() {
			s.Insert(5, 6, 7, 8, 9)
			Expect(s.NumGreaterThan(4)).To(Equal(5))
			Expect(s.NumGreaterThan(0)).To(Equal(5))
		})

		It("Should count correctly for a value in the middle of a range", func() {
			s.Insert(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
			Expect(s.NumGreaterThan(5)).To(Equal(4))
		})

		It("Should count correctly for a value at the start of a range", func() {
			s.Insert(0, 1, 2, 3, 4)
			Expect(s.NumGreaterThan(0)).To(Equal(4))
		})

		It("Should count correctly for the last element of a range", func() {
			s.Insert(0, 1, 2, 3, 4)
			Expect(s.NumGreaterThan(4)).To(Equal(0))
		})

		It("Should count correctly for a value in a gap between ranges", func() {
			s.Insert(0, 1, 2)
			s.Insert(8, 9, 10, 11)
			Expect(s.NumGreaterThan(5)).To(Equal(4))
		})

		It("Should return 0 for an empty set", func() {
			Expect(s.NumGreaterThan(0)).To(Equal(0))
			Expect(s.NumGreaterThan(-100)).To(Equal(0))
		})

		It("Should handle negative numbers", func() {
			s.Insert(-5, -4, -3, -2, -1)
			Expect(s.NumGreaterThan(-3)).To(Equal(2))
			Expect(s.NumGreaterThan(-6)).To(Equal(5))
			Expect(s.NumGreaterThan(-1)).To(Equal(0))
		})

		It("Should count across multiple ranges", func() {
			s.Insert(0, 1, 2)
			s.Insert(5, 6, 7)
			s.Insert(10, 11, 12)
			Expect(s.NumGreaterThan(1)).To(Equal(7))
			Expect(s.NumGreaterThan(6)).To(Equal(4))
		})

		It("Should handle a single-element set", func() {
			s.Insert(5)
			Expect(s.NumGreaterThan(5)).To(Equal(0))
			Expect(s.NumGreaterThan(4)).To(Equal(1))
			Expect(s.NumGreaterThan(6)).To(Equal(0))
		})
	})

	Describe("Size", func() {
		It("Should return 0 for an empty set", func() {
			Expect(s.Size()).To(Equal(0))
		})

		It("Should return 1 for a single element", func() {
			s.Insert(42)
			Expect(s.Size()).To(Equal(1))
		})

		It("Should return the correct size for a single contiguous range", func() {
			s.Insert(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
			Expect(s.Size()).To(Equal(10))
		})

		It("Should return the correct size for multiple ranges", func() {
			s.Insert(0, 1, 2)
			s.Insert(5, 6, 7)
			s.Insert(10, 11, 12)
			Expect(s.Size()).To(Equal(9))
		})

		It("Should increase after insertions", func() {
			Expect(s.Size()).To(Equal(0))
			s.Insert(1, 2, 3)
			Expect(s.Size()).To(Equal(3))
			s.Insert(5, 6)
			Expect(s.Size()).To(Equal(5))
		})

		It("Should decrease after removals", func() {
			s.Insert(0, 1, 2, 3, 4)
			Expect(s.Size()).To(Equal(5))
			s.Remove(2)
			Expect(s.Size()).To(Equal(4))
			s.Remove(0, 1, 3, 4)
			Expect(s.Size()).To(Equal(0))
		})

		It("Should not change for duplicate insertions", func() {
			s.Insert(1, 2, 3)
			Expect(s.Size()).To(Equal(3))
			s.Insert(1, 2, 3)
			Expect(s.Size()).To(Equal(3))
		})

		It("Should not change for removing non-existent elements", func() {
			s.Insert(1, 2, 3)
			Expect(s.Size()).To(Equal(3))
			s.Remove(10, 20)
			Expect(s.Size()).To(Equal(3))
		})
	})

	Describe("Copy", func() {
		It("Should create an independent copy with a single interval", func() {
			s.Insert(1, 2, 3, 4, 5)
			copied := s.Copy()
			Expect(copied.Size()).To(Equal(5))
			s.Remove(3)
			Expect(copied.Size()).To(Equal(5))
			Expect(copied.Contains(3)).To(BeTrue())
		})

		It("Should create an independent copy with multiple intervals", func() {
			s.Insert(1, 2, 3)
			s.Insert(10, 11, 12)
			copied := s.Copy()
			Expect(copied.Size()).To(Equal(6))
			Expect(copied).To(HaveLen(2))
			s.Remove(2)
			Expect(copied.Size()).To(Equal(6))
		})

		It("Should handle copying an empty set", func() {
			copied := s.Copy()
			Expect(copied.Size()).To(Equal(0))
		})

		It("Should not be affected when the original is modified", func() {
			s.Insert(1, 2, 3)
			copied := s.Copy()
			s.Insert(4, 5, 6)
			s.Remove(1)
			Expect(copied.Size()).To(Equal(3))
			Expect(copied.Contains(1)).To(BeTrue())
			Expect(copied.Contains(4)).To(BeFalse())
		})

		It("Should not affect the original when the copy is modified", func() {
			s.Insert(1, 2, 3)
			copied := s.Copy()
			copied.Insert(4, 5, 6)
			copied.Remove(1)
			Expect(s.Size()).To(Equal(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(4)).To(BeFalse())
		})
	})
})
