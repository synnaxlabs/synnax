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
	Describe("New", func() {
		It("should create a set from a variadic list of elements", func() {
			s := set.New(1, 2, 3)
			Expect(len(s)).To(Equal(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
		})
		It("should remove duplicates", func() {
			s := set.New(1, 2, 3, 2, 1, 1, 1, 1, 1)
			Expect(len(s)).To(Equal(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
		})
		It("should work with an empty list", func() {
			s := set.New[int]()
			Expect(len(s)).To(Equal(0))
		})
	})
	Describe("Add", func() {
		It("should add elements to the set", func() {
			s := set.New[int]()
			s.Add(1)
			Expect(len(s)).To(Equal(1))
			Expect(s.Contains(1)).To(BeTrue())
		})
		It("should handle duplicate elements", func() {
			s := set.New[int]()
			s.Add(1, 2, 3, 1, 1, 1)
			s.Add(2, 3, 4, 3, 4, 3, 4, 4, 4)
			Expect(len(s)).To(Equal(4))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(4)).To(BeTrue())
		})
		It("should work with 0 elements", func() {
			s := set.New[int]()
			s.Add()
			s.Add()
			Expect(len(s)).To(Equal(0))
		})
	})
	Describe("Remove", func() {
		It("should remove elements from the set", func() {
			s := set.New(1, 2, 3, 4, 5)
			s.Remove(2, 4)
			Expect(len(s)).To(Equal(3))
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(5)).To(BeTrue())
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(4)).To(BeFalse())
		})
		It("should handle removing non-existent elements", func() {
			s := set.New(1, 2, 3)
			s.Remove(4, 5)
			Expect(len(s)).To(Equal(3))
		})
		It("should work with 0 elements", func() {
			s := set.New[int]()
			s.Remove()
			Expect(len(s)).To(Equal(0))
		})
	})
	Describe("Contains", func() {
		It("should correctly check if elements exist", func() {
			s := set.New(1, 3, 3, 3, 3, 5)
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(3)).To(BeTrue())
			Expect(s.Contains(4)).To(BeFalse())
			Expect(s.Contains(5)).To(BeTrue())
		})
		It("should return false for empty sets", func() {
			s := set.New[int]()
			Expect(s.Contains(1)).To(BeFalse())
		})
	})
	Describe("Elements", func() {
		It("should return all elements in the set", func() {
			s := set.New(1, 2, 3, 2, 3)
			s.Add(2)
			s.Add(3)
			keys := s.Elements()
			Expect(len(keys)).To(Equal(3))
			Expect(keys).To(ContainElements(1, 2, 3))
		})
		It("should return an empty slice for empty sets", func() {
			s := set.New[int]()
			keys := s.Elements()
			Expect(len(keys)).To(Equal(0))
		})
	})
})
