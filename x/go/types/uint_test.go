package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Uint", func() {
	Describe("BoolToUint8", func() {
		It("Should return 1 if the bool is true", func() {
			Expect(types.BoolToUint8(true)).To(Equal(uint8(1)))
		})
		It("Should return 0 if the bool is false", func() {
			Expect(types.BoolToUint8(false)).To(Equal(uint8(0)))
		})
	})

})
