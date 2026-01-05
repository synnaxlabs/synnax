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

var _ = Describe("Integer Set", Ordered, func() {
	var (
		s    set.Integer[int]
		nums [100]int
	)
	BeforeAll(func() {
		for i := range 100 {
			nums[i] = i
		}
	})
	BeforeEach(func() {
		s = set.Integer[int]{}
	})
	Describe("Test Integer.Insert", func() {
		It("Inserting integers should work", func() {
			Expect(len(s)).To(Equal(0))
			s.Insert(0)
			Expect(s.Contains(0)).To(BeTrue())
			Expect(s.NumGreaterThan(0)).To(Equal(0))
			Expect(s.NumLessThan(0)).To(Equal(0))
			s.Insert(1)
			Expect(s.NumLessThan(2)).To(Equal(2))
			Expect(s.Size()).To(Equal(2))
			s.Insert(nums[0:3]...)
			Expect(s.NumLessThan(3)).To(Equal(3))
			Expect(s.Size()).To(Equal(3))
			s.Insert(nums[0:10]...)
			Expect(s.NumLessThan(10)).To(Equal(10))
			Expect(s.Size()).To(Equal(10))

			s.Insert(nums[15:20]...)
			s.Insert(nums[25:30]...)
			s.Insert(nums[10:40]...)
			Expect(len(s)).To(Equal(1))
			Expect(s.Size()).To(Equal(40))

			s.Remove(nums[10:100]...)
			newSlice := make([]int, 40)
			copy(newSlice[0:10], nums[0:10])
			copy(newSlice[10:30], nums[20:40])
			copy(newSlice[30:40], nums[50:60])
			s.Insert(newSlice...)
			Expect(len(s)).To(Equal(3))
			Expect(s.NumLessThan(45)).To(Equal(30))
			Expect(s.Size()).To(Equal(40))
		})
		It("Should insert a single integer", func() {
			s.Insert(88)
			Expect(s.NumLessThan(10)).To(Equal(0))
			Expect(len(s)).To(Equal(1))
		})
	})
	Describe("Check compression", func() {
		It("Should create a compressed set", func() {
			Expect(len(s)).To(Equal(0))
			s.Insert(nums[0:10]...)
			Expect(len(s)).To(Equal(1))
			s.Insert(11)
			Expect(len(s)).To(Equal(2))
			s.Insert(nums[14:20]...)
			Expect(len(s)).To(Equal(3))
			s.Insert(10)
			Expect(len(s)).To(Equal(2))
			s.Insert(nums[12:14]...)
			Expect(s.NumLessThan(nums[20])).To(Equal(20))
			Expect(len(s)).To(Equal(1))
		})
		It("should check edge cases", func() {
			s.Insert(nums[2:12]...)
			s.Insert(nums[0:2]...)
			s.Insert(nums[15:20]...)
			s.Insert(nums[20:24]...)
			s.Insert(nums[12:15]...)
			Expect(len(s)).To(Equal(1))
			Expect(s.NumLessThan(nums[24])).To(Equal(24))
		})
	})
	Describe("Check retrieval of nums", func() {
		It("Should retrieve the correct number while in the middle of an intRange", func() {
			s.Insert(nums[0:25]...)
			Expect(s.NumLessThan(nums[15])).To(Equal(15))
		})
	})
	Describe("Check removal of nums", func() {
		It("Should accurately delete and reinsert slices of nums", func() {
			s.Insert(nums[0:25]...)
			s.Remove(nums[10:20]...)
			Expect(len(s)).To(Equal(2))
			Expect(s.NumLessThan(nums[25])).To(Equal(15))
			Expect(s.NumLessThan(nums[10])).To(Equal(10))
			Expect(s.NumLessThan(nums[20])).To(Equal(10))
			s.Remove(nums[0:10]...)
			s.Remove(nums[20:25]...)
			Expect(len(s)).To(Equal(0))
			Expect(s.NumLessThan(nums[10])).To(Equal(0))
		})
		It("Should be idempotent when removing nums", func() {
			s.Remove(nums[0:100]...)
			Expect(len(s)).To(Equal(0))
			Expect(s.NumLessThan(nums[50])).To(Equal(0))
			s.Insert(nums[0:10]...)
			s.Insert(nums[25:50]...)
			s.Remove(nums[5:30]...)
			Expect(len(s)).To(Equal(2))
			Expect(s.NumLessThan(nums[99])).To(Equal(25))
		})
	})
	Describe("Testing with an empty set", func() {
		It("Should be able to get sizes from an empty set", func() {
			Expect(s.NumLessThan(nums[10])).To(Equal(0))
			Expect(len(s)).To(Equal(0))
			s.Insert(nums[0:10]...)
			s.Remove(nums[0:10]...)
			s.Remove(nums[0:10]...)
			Expect(s.Size()).To(Equal(0))
			Expect(s.Contains(0)).To(BeFalse())
			Expect(s.NumGreaterThan(0)).To(Equal(0))
			Expect(s.NumLessThan(0)).To(Equal(0))
		})
	})
	Describe("Testing Copy method", func() {
		It("should create a copy that is independent of the original", func() {
			s.Insert(1, 2, 3, 4, 5)
			copySet := s.Copy()
			Expect(copySet.Size()).To(Equal(s.Size()))
			// Modify the original.
			s.Remove(3)
			// The copy should remain unchanged.
			Expect(copySet.Size()).To(Equal(5))
			Expect(copySet.Contains(3)).To(BeTrue())
		})
		It("should return an empty set when copying an empty set", func() {
			copySet := s.Copy()
			Expect(copySet.Size()).To(Equal(0))
		})
	})
	Describe("Testing unsorted input", func() {
		It("should insert unsorted integers correctly", func() {
			// Provide unsorted input.
			s.Insert(5, 1, 4, 3, 2)
			// The resulting set should contain numbers 1 through 5.
			Expect(s.Size()).To(Equal(5))
			for i := 1; i <= 5; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
			// Verify that the set has compressed the range into a single interval.
			Expect(len(s)).To(Equal(1))
		})
		It("should handle unsorted input with duplicates", func() {
			// Insert unsorted input with duplicates.
			s.Insert(10, 5, 5, 8, 1, 3, 8, 2, 4)
			// The expected unique set is numbers: 1, 2, 3, 4, 5, 8, 10.
			Expect(s.Size()).To(Equal(7))
			expected := []int{1, 2, 3, 4, 5, 8, 10}
			for _, num := range expected {
				Expect(s.Contains(num)).To(BeTrue())
			}
		})
		It("should correctly merge intervals from unsorted input", func() {
			// Insert unsorted input that should produce multiple intervals.
			s.Insert(20, 18, 19, 5, 6, 7, 15)
			// The sorted unique values are: 5,6,7,15,18,19,20.
			// Note: 15 is isolated (not adjacent to 18), so we expect three intervals:
			// [5,8) covering 5,6,7; [15,16) covering 15; and [18,21) covering 18,19,20.
			Expect(s.Size()).To(Equal(7))
			Expect(len(s)).To(Equal(3))
			// Verify membership.
			for i := 5; i < 8; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
			Expect(s.Contains(15)).To(BeTrue())
			for i := 18; i < 21; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
		})
	})
	Describe("Comprehensive unsorted input tests", func() {
		It("should correctly handle a completely unsorted contiguous sequence", func() {
			// Unsorted contiguous sequence.
			input := []int{7, 3, 4, 1, 2, 5, 6}
			s.Insert(input...)
			// Expected: a single interval [1,8) with size 7.
			Expect(s.Size()).To(Equal(7))
			Expect(len(s)).To(Equal(1))
			for i := 1; i <= 7; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
		})
		It("should correctly handle unsorted non-contiguous inputs", func() {
			// Unsorted non-contiguous input.
			input := []int{50, 20, 40, 10, 30, 70, 60, 90, 80}
			s.Insert(input...)
			// Since these numbers are not consecutive, each should form its own interval.
			Expect(s.Size()).To(Equal(9))
			Expect(len(s)).To(Equal(9))
			for _, num := range input {
				Expect(s.Contains(num)).To(BeTrue())
			}
		})
		It("should correctly handle unsorted input with multiple contiguous groups and duplicates", func() {
			// Unsorted input with duplicates and distinct groups.
			input := []int{100, 102, 101, 200, 202, 201, 200, 150, 149, 151, 149, 152}
			s.Insert(input...)
			// Expected groups:
			// Group 1: [100,101,102] -> 3 numbers.
			// Group 2: [149,150,151,152] -> 4 numbers.
			// Group 3: [200,201,202] -> 3 numbers.
			Expect(s.Size()).To(Equal(3 + 4 + 3))
			Expect(len(s)).To(Equal(3))
			for i := 100; i <= 102; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
			for i := 149; i <= 152; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
			for i := 200; i <= 202; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
		})
		It("should correctly merge unsorted input with an existing set", func() {
			// Start with a sorted input.
			s.Insert(10, 11, 12, 13, 14)
			// Then insert unsorted values that extend the existing interval.
			s.Insert(15, 9)
			// Expected final interval is [9,16) with size 7.
			Expect(s.Size()).To(Equal(7))
			Expect(len(s)).To(Equal(1))
			for i := 9; i <= 15; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
		})
	})
	Describe("Duplicate Insertion", func() {
		It("should not increase the size when inserting duplicate contiguous values", func() {
			s.Insert(1, 2, 3)
			sizeAfterFirst := s.Size()
			s.Insert(1, 2, 3)
			// The size should remain the same and only one interval should exist.
			Expect(s.Size()).To(Equal(sizeAfterFirst))
			Expect(len(s)).To(Equal(1))
		})
	})
})
