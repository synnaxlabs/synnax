package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Enum", func() {
	Describe("NewInclusiveBoundsChecker", func() {
		Context("Integer bounds", func() {
			It("Should accept values within bounds", func() {
				checker := validate.NewInclusiveBoundsChecker(1, 10)
				Expect(checker(5)).ToNot(HaveOccurred())
				Expect(checker(1)).ToNot(HaveOccurred())  // lower bound
				Expect(checker(10)).ToNot(HaveOccurred()) // upper bound
			})

			It("Should reject values outside bounds", func() {
				checker := validate.NewInclusiveBoundsChecker(1, 10)
				Expect(checker(0)).To(HaveOccurred())
				Expect(checker(11)).To(HaveOccurred())
			})
		})

		Context("Float bounds", func() {
			It("Should accept values within bounds", func() {
				checker := validate.NewInclusiveBoundsChecker(1.0, 10.0)
				Expect(checker(5.5)).ToNot(HaveOccurred())
				Expect(checker(1.0)).ToNot(HaveOccurred())  // lower bound
				Expect(checker(10.0)).ToNot(HaveOccurred()) // upper bound
			})

			It("Should reject values outside bounds", func() {
				checker := validate.NewInclusiveBoundsChecker(1.0, 10.0)
				Expect(checker(0.9)).To(HaveOccurred())
				Expect(checker(10.1)).To(HaveOccurred())
			})
		})

		Context("Error messages", func() {
			It("Should include type and bounds in error message", func() {
				checker := validate.NewInclusiveBoundsChecker(1, 10)
				err := checker(0)
				Expect(err.Error()).To(ContainSubstring("int"))
				Expect(err.Error()).To(ContainSubstring("1"))
				Expect(err.Error()).To(ContainSubstring("10"))
			})
		})
	})
})
