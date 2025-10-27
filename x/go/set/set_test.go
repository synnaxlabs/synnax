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
})
