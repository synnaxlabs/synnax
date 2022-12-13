package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Validate", func() {
	Describe("Ternay", func() {
		It("Should accumulate the error if the condition is met", func() {
			v := validate.New("demo")
			v.Ternaryf(true, "error")
			executed := false
			v.Funcf(func() bool {
				executed = true
				return true
			}, "error")
			Expect(v.Error()).To(HaveOccurred())
			Expect(executed).To(BeFalse())
		})
	})
})
