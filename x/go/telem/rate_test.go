package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Rate", func() {
	Describe("Rate", func() {
		Describe("Period", func() {
			It("Should return the correct period for the data rate", func() {
				Expect(telem.Rate(1).Period()).To(Equal(telem.Second))
			})
		})
		Describe("Distance", func() {
			It("Should return the number of samples that fit in the span", func() {
				Expect(telem.Rate(10).SampleCount(telem.Second)).To(Equal(10))
			})
		})
		Describe("SpanTo", func() {
			It("Should return the span of the provided samples", func() {
				Expect(telem.Rate(10).Span(10)).To(Equal(telem.Second))
			})
		})
		Describe("SizeSpan", func() {
			It("Should return the span of the provided number of bytes", func() {
				Expect(telem.Rate(10).SizeSpan(16, telem.Bit64)).To(Equal(200 * telem.Millisecond))
			})
		})
	})
})
