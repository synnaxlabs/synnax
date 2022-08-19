package httputil_test

import (
	"github.com/arya-analytics/x/httputil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EncoderDecoder", func() {
	Describe("Determine", func() {
		It("Should determine the encoder based on a content-type", func() {
			ecd, err := httputil.DetermineEncoderDecoder("application/json")
			Expect(err).ToNot(HaveOccurred())
			Expect(ecd.ContentType()).To(Equal("application/json"))
		})
		It("Should return an error if the content-type is not supported", func() {
			_, err := httputil.DetermineEncoderDecoder("application/octet-stream")
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("SupportedContentTypes", func() {
		It("Should return the supported content types", func() {
			contentTypes := httputil.SupportedContentTypes()
			Expect(contentTypes).To(ContainElement("application/json"))
			Expect(contentTypes).To(ContainElement("application/msgpack"))
		})
	})
})
