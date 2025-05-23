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
})
