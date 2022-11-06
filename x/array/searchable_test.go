package array_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/array"
	"github.com/synnaxlabs/x/compare"
)

var _ = Describe("Searchable", func() {
	Describe("Search", func() {
		Context("Value in array", func() {
			It("Should return the index of the insert position of the key", func() {
				arr := array.Searchable[int]{Array: array.NewRolling[int](10)}
				arr.Append(1, 2, 3, 4)
				v, i := arr.Search(compare.NumericUnary(2))
				Expect(v).To(Equal(2))
				Expect(i).To(Equal(1))
			})
		})
		Context("Value not in array, but is within range", func() {
			It("Should return the index of the insert position of the key", func() {
				arr := array.Searchable[int]{Array: array.NewRolling[int](10)}
				arr.Append(1, 3, 5, 7)
				v, i := arr.Search(compare.NumericUnary(2))
				Expect(v).To(Equal(1))
				Expect(i).To(Equal(0))
			})
		})
		Context("Value before start of array", func() {
			It("Should return the first value in the array and index -1", func() {
				arr := array.Searchable[int]{Array: array.NewRolling[int](10)}
				arr.Append(1, 3, 5, 7)
				v, i := arr.Search(compare.NumericUnary(-10))
				Expect(v).To(Equal(1))
				Expect(i).To(Equal(-1))
			})
		})
		Context("Value after end of array", func() {
			It("Should return the last value in the array and an index equal to array len", func() {
				arr := array.Searchable[int]{Array: array.NewRolling[int](10)}
				arr.Append(1, 3, 5, 7)
				v, i := arr.Search(compare.NumericUnary(10))
				Expect(v).To(Equal(7))
				Expect(i).To(Equal(4))
			})
		})
		Context("Single value array", func() {
			Context("Value before start of array", func() {
				It("Should return the value of array an index -1", func() {
					arr := array.Searchable[int]{Array: array.Wrap([]int{1})}
					v, i := arr.Search(compare.NumericUnary(-10))
					Expect(v).To(Equal(1))
					Expect(i).To(Equal(-1))
				})
			})
			Context("Value after end of array", func() {
				It("Should return the value of array an index 1", func() {
					arr := array.Searchable[int]{Array: array.Wrap([]int{1})}
					v, i := arr.Search(compare.NumericUnary(10))
					Expect(v).To(Equal(1))
					Expect(i).To(Equal(1))
				})
			})
		})
	})
})
