package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Utilities", func() {
	It("should be true", func() {
		Expect(true).To(BeTrue())
	})

})
