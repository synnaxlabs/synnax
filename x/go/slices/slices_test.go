// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slices_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/slices"
)

var _ = Describe("Slices", func() {
	Describe("Truncate", func() {
		It("should return the full slice if its length is less than maxDisplayValues", func() {
			slice := []int{1, 2, 3}
			first, last := slices.Truncate(slice, 5)
			Expect(first).To(Equal([]int{1, 2, 3}))
			Expect(last).To(BeNil())
		})

		It("should return the full slice if its length is equal to maxDisplayValues", func() {
			slice := []int{1, 2, 3, 4, 5}
			first, last := slices.Truncate(slice, 5)
			Expect(first).To(Equal([]int{1, 2, 3, 4, 5}))
			Expect(last).To(BeNil())
		})

		It("should truncate and split evenly between first and last for long slices (even split)", func() {
			slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			first, last := slices.Truncate(slice, 6)
			// startCount = 3, endCount = 3
			Expect(first).To(Equal([]int{1, 2, 3}))
			Expect(last).To(Equal([]int{8, 9, 10}))
		})

		It("should truncate and split unevenly between first and last for long slices (odd split)", func() {
			slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			first, last := slices.Truncate(slice, 5)
			// startCount = 2, endCount = 3
			Expect(first).To(Equal([]int{1, 2}))
			Expect(last).To(Equal([]int{8, 9, 10}))
		})

		It("should handle maxDisplayValues of 1", func() {
			slice := []int{1, 2, 3, 4, 5}
			first, last := slices.Truncate(slice, 1)
			// startCount = 0, endCount = 1
			Expect(first).To(Equal([]int{}))
			Expect(last).To(Equal([]int{5}))
		})

		It("should handle maxDisplayValues of 0", func() {
			slice := []int{1, 2, 3, 4, 5}
			first, last := slices.Truncate(slice, 0)
			Expect(first).To(Equal([]int{1, 2, 3, 4, 5}))
			Expect(last).To(BeNil())
		})

		It("should handle empty slices", func() {
			var slice []int
			first, last := slices.Truncate(slice, 5)
			Expect(first).To(BeEmpty())
			Expect(last).To(BeNil())
		})

		It("should handle slices of strings", func() {
			slice := []string{"a", "b", "c", "d", "e", "f", "g"}
			first, last := slices.Truncate(slice, 4)
			// startCount = 2, endCount = 2
			Expect(first).To(Equal([]string{"a", "b"}))
			Expect(last).To(Equal([]string{"f", "g"}))
		})

		It("should handle maxDisplayValues greater than slice length", func() {
			slice := []int{1, 2, 3}
			first, last := slices.Truncate(slice, 10)
			Expect(first).To(Equal([]int{1, 2, 3}))
			Expect(last).To(BeNil())
		})
	})

	Describe("ConvertNegativeIndex", func() {
		It("Should correctly convert a negative to a positive index", func() {
			Expect(slices.ConvertNegativeIndex(-1, 10)).To(Equal(9))
		})

		It("Should pass through a positive index", func() {
			Expect(slices.ConvertNegativeIndex(5, 10)).To(Equal(5))
		})

		It("Should panic if the negative index is out of bounds", func() {
			Expect(func() {
				slices.ConvertNegativeIndex(-11, 10)
			}).To(PanicWith("index out of range [-11] with length 10"))
		})
	})

	Describe("IterEndlessly", func() {
		It("Should yield elements in order", func() {
			values := []int{1, 2, 3}
			var results []int
			for v := range slices.IterEndlessly(values) {
				results = append(results, v)
				if len(results) == 3 {
					break
				}
			}
			Expect(results).To(Equal([]int{1, 2, 3}))
		})

		It("Should wrap around after reaching the end", func() {
			values := []int{1, 2, 3}
			var results []int
			for v := range slices.IterEndlessly(values) {
				results = append(results, v)
				if len(results) == 7 {
					break
				}
			}
			Expect(results).To(Equal([]int{1, 2, 3, 1, 2, 3, 1}))
		})

		It("Should work with a single element", func() {
			values := []string{"only"}
			var results []string
			for v := range slices.IterEndlessly(values) {
				results = append(results, v)
				if len(results) == 5 {
					break
				}
			}
			Expect(results).To(Equal([]string{"only", "only", "only", "only", "only"}))
		})

		It("Should stop when the caller breaks", func() {
			values := []int{1, 2, 3, 4, 5}
			count := 0
			for range slices.IterEndlessly(values) {
				count++
				if count == 2 {
					break
				}
			}
			Expect(count).To(Equal(2))
		})

		It("Should panic with an empty slice", func() {
			var values []int
			Expect(func() {
				for range slices.IterEndlessly(values) {
					break
				}
			}).To(Panic())
		})
	})
})
