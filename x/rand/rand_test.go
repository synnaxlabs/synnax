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
		Describe("Field", func() {
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
})
