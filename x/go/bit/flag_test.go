package bit_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/bit"
)

var _ = Describe("Flag", func() {
	Describe("FlagPos", func() {
		It("Should set a bit flag at a particular position", func() {
			var (
				b    uint8
				flag bit.FlagPos = 8
			)
			b = flag.Set(b, true)
			Expect(flag.Get(b)).To(BeTrue())
			Expect(b).To(Equal(255))
		})
	})

})
