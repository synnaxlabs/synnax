package errors_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
)

var _ = Describe("Field", func() {
	Describe("Err", func() {
		It("Should return a string representation of the Field error", func() {
			Expect(errors.Field{Field: "field", Message: "Message"}.Error()).To(Equal("field: Message"))
		})
	})
	Describe("Fields", func() {
		Describe("Err", func() {
			It("Should return a string representation of the Field error", func() {
				Expect(errors.Fields{
					{Field: "field", Message: "Message"},
					{Field: "field2", Message: "message2"},
				}.Error()).To(Equal("field: Message\nfield2: message2"))
			})
		})
	})
})
