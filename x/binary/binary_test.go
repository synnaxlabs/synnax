package binary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
)

var _ = Describe("Binary", func() {
	Describe("MakeCopy", func() {
		It("Should return a copy of the given byte slice", func() {
			bytes := []byte("hello")
			copied := binary.MakeCopy(bytes)
			Expect(copied).To(Equal(bytes))
			Expect(copied).ToNot(BeIdenticalTo(bytes))
		})
	})

})
