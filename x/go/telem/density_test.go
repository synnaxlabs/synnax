package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Density", func() {
	Describe("SampleCount", func() {
		It("Should return the number of samples within the number of bytes", func() {
			Expect(telem.Bit64.SampleCount(16)).To(Equal(int64(2)))
		})
		It("Should panic if the density if unknown", func() {
			Expect(func() {
				telem.UnknownDensity.SampleCount(16)
			}).To(PanicWith("attempted to call SampleCount() on undefined density"))
		})
	})

	Describe("Size", func() {
		It("Should return the number of bytes occupied by the given sample count", func() {
			Expect(telem.Bit64.Size(2)).To(Equal(telem.Size(16)))
		})
		It("Should panic if the density if unknown", func() {
			Expect(func() {
				telem.UnknownDensity.Size(16)
			}).To(PanicWith("attempted to call Size() on undefined density"))
		})
	})

})
