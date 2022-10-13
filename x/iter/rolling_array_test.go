package iter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/iter"
)

var _ = Describe("RollingArray", func() {
	Describe("Initialize", func() {
		It("Should initialize an ordered array with the given capacity", func() {
			arr := iter.NewRollingArray[int](10)
			Expect(arr.Capacity).To(Equal(10))
		})
	})
	Describe("Append", func() {
		Context("Array is not full - Appended values don't exceed capacity", func() {
			It("Should append the value to the array and increase its size", func() {
				arr := iter.NewRollingArray[int](10)
				arr.Append(1)
				Expect(arr.Size).To(Equal(1))
				Expect(arr.Get(0)).To(Equal(1))
			})
		})
		Context("Array is not full - Appended values exceed capacity", func() {
			It("Should append the value to the array", func() {
				arr := iter.NewRollingArray[int](3)
				arr.Append(1, 2)
				Expect(arr.Size).To(Equal(2))
				arr.Append(4, 5)
				Expect(arr.Get(0)).To(Equal(2))
				Expect(arr.Get(1)).To(Equal(4))
				Expect(arr.Get(2)).To(Equal(5))
			})
		})
		Context("Array is full - Appended values don't cause rollover", func() {
			It("Should append the value to the array", func() {
				arr := iter.NewRollingArray[int](3)
				arr.Append(1, 2, 3)
				Expect(arr.Size).To(Equal(3))
				arr.Append(4)
				Expect(arr.Get(0)).To(Equal(2))
				Expect(arr.Get(1)).To(Equal(3))
				Expect(arr.Get(2)).To(Equal(4))
			})
		})
		Context("Array is full - Appended values cause rollover", func() {
			It("Should append the value to the array", func() {
				arr := iter.NewRollingArray[int](3)
				arr.Append(1, 2, 3)
				arr.Append(4)
				arr.Append(3, 4, 5)
				Expect(arr.Get(0)).To(Equal(3))
				Expect(arr.Get(1)).To(Equal(4))
				Expect(arr.Get(2)).To(Equal(5))
			})
		})
	})
})
