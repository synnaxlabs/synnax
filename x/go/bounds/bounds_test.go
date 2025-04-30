package bounds_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/bounds"
)

var _ = Describe("Bounds", func() {
	DescribeTable("Contains", func(b bounds.Bounds[int], value int, expected bool) {
		Expect(b.Contains(value)).To(Equal(expected))
	},
		Entry("In Middle", bounds.Bounds[int]{Lower: 5, Upper: 10}, 7, true),
		Entry("At Start", bounds.Bounds[int]{Lower: 5, Upper: 10}, 5, true),
		Entry("At End", bounds.Bounds[int]{Lower: 5, Upper: 10}, 10, false),
	)
	Describe("String", func() {
		It("Should return a formatted string", func() {
			Expect(bounds.Bounds[int]{Lower: 3, Upper: 5}.String()).To(Equal("Bounds[3, 5)"))
		})
	})
})
