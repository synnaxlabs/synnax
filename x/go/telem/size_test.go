package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Size", func() {
	Describe("String", func() {
		It("Should return the correct string", func() {
			s := telem.Size(0)
			Expect(s.String()).To(Equal("0B"))
		})
	})
})
