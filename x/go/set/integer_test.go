// Copyright 2024 Synnax Labs, Inc.
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
		for i := 0; i < 100; i++ {
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
})
