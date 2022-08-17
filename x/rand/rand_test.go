package rand_test

import (
	"github.com/arya-analytics/x/rand"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rand", func() {
	Describe("ApplySink", func() {
		var (
			m map[int]int
		)
		BeforeEach(func() {
			m = map[int]int{
				1: 2,
				3: 4,
				5: 6,
			}
		})
		Describe("Name", func() {
			It("Should return a random key", func() {
				key := rand.MapKey(m)
				Expect(key > 0).To(BeTrue())
				Expect(key < 6).To(BeTrue())
			})
			It("Should return the zero type when the map is empty", func() {
				m = map[int]int{}
				key := rand.MapKey(m)
				Expect(key).To(BeZero())
			})
		})
		Describe("Value", func() {
			It("Should return a random value", func() {
				value := rand.MapValue(m)
				Expect(value > 0).To(BeTrue())
				Expect(value <= 6).To(BeTrue())
			})
		})
		Describe("Element", func() {
			It("Should return a random element", func() {
				key, value := rand.MapElem(m)
				Expect(key > 0).To(BeTrue())
				Expect(value <= 6).To(BeTrue())
				Expect(key < 6).To(BeTrue())
				Expect(value > 0).To(BeTrue())
			})
		})
	})
	Describe("Slice", func() {
		It("Should return a random element in the slice", func() {
			value := rand.Slice([]int{1, 2, 3, 4, 5, 6})
			Expect(value > 0).To(BeTrue())
		})
		Describe("sub Slice", func() {
			It("Should return random sub-slice", func() {
				value := rand.SubSlice([]int{1, 2, 3, 4, 5, 6}, 2)
				Expect(len(value)).To(Equal(2))
				Expect(value[0] != value[1]).To(BeTrue())
			})
			It("Should return the slice itself", func() {
				slc := []int{1, 2, 3, 4, 5, 6}
				value := rand.SubSlice(slc, 20)
				Expect(len(value)).To(Equal(6))
				Expect(value).To(Equal(slc))
			})
		})
	})
	Describe("Element", func() {
		It("Should return a random element", func() {
			value := rand.Elem(1, 2, 3)
			Expect(value > 0).To(BeTrue())
		})
	})
})
