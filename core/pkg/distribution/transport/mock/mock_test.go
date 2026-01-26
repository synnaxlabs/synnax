package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mock", func() {
	It("should be true", func() {
		Expect(true).To(BeTrue())
	})
})
