package errors_test

import (
	"github.com/arya-analytics/delta/pkg/api/errors"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type structToValidate struct {
	Field string `json:"field" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

var _ = Describe("Validate", func() {
	Describe("Validation", func() {
		Context("Validator Err", func() {
			It("Should return a Response with a single Field error", func() {
				err := validator.New().Struct(structToValidate{Field: "", Email: ""})
				Expect(err).To(HaveOccurred())
				vErr := errors.Validation(err)
				Expect(vErr).To(HaveOccurred())
				Expect(vErr.Type).To(Equal(errors.TypeValidation))
				Expect(vErr.Err).To(HaveLen(2))
				Expect(vErr.Err.(errors.Fields)[0]).To(Equal(errors.Field{Field: "Field", Message: "required"}))
			})
		})
		Context("Field error", func() {
			It("Should wrap the field error", func() {
				err := errors.Validation(errors.Field{Field: "Field", Message: "required"})
				Expect(err).To(HaveOccurred())
				Expect(err.Type).To(Equal(errors.TypeValidation))
				Expect(err.Err).To(HaveLen(1))
				Expect(err.Err.(errors.Fields)[0]).To(Equal(errors.Field{Field: "Field", Message: "required"}))
			})
		})
		Context("Fields error", func() {
			It("Should wrap the fields error", func() {
				err := errors.Validation(errors.Fields{errors.Field{Field: "Field", Message: "required"}})
				Expect(err).To(HaveOccurred())
				Expect(err.Type).To(Equal(errors.TypeValidation))
				Expect(err.Err).To(HaveLen(1))
				Expect(err.Err.(errors.Fields)[0]).To(Equal(errors.Field{Field: "Field", Message: "required"}))
			})
		})
		Context("Any other type of error", func() {
			It("Should return an unexpected error", func() {
				err := errors.Validation(roacherrors.New("error"))
				Expect(err).To(HaveOccurred())
				Expect(err.Type).To(Equal(errors.TypeUnexpected))
				Expect(err.Err.Error()).To(Equal(roacherrors.New("error").Error()))
			})
		})
	})
	Describe("MaybeValidation", func() {
		It("Should return nil if the error is nil", func() {
			err := errors.MaybeValidation(nil)
			Expect(err).To(Equal(errors.Nil))
		})
		It("Should return a Validation error if the error is not nil", func() {
			err := errors.MaybeValidation(errors.Field{Field: "Field", Message: "required"})
			Expect(err).To(HaveOccurred())
			Expect(err.Type).To(Equal(errors.TypeValidation))
		})
	})
})
