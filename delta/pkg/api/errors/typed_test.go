package errors_test

import (
	"github.com/arya-analytics/delta/pkg/api/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Typed", func() {
	Describe("Err", func() {
		It("Should return a string representation of the error", func() {
			err := errors.Typed{Type: "type", Err: errors.Field{Field: "field", Message: "Message"}}
			Expect(err.Error()).To(Equal("field: Message"))
		})
		It("Should return 'nil' when the error is of type Nil", func() {
			err := errors.Nil
			Expect(err.Error()).To(Equal("nil"))
		})
	})
	Describe("Occurred", func() {
		It("Should return true when the error is not of type Nil", func() {
			err := errors.Typed{Type: "type", Err: errors.Field{Field: "field", Message: "Message"}}
			Expect(err.Occurred()).To(BeTrue())
		})
	})
})
