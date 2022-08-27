package errors

import (
	"github.com/go-playground/validator/v10"
)

// Validation is an error that occurs when a user-supplied value is invalid. For example,
// a user provides an email address without the	@ symbol. Validation attempts to parse
// validation errors from the validator package into a set of Field errors.
func Validation(err error) Typed {
	if _, ok := err.(Fields); ok {
		return Typed{Type: TypeValidation, Err: err}
	}
	if fErr, ok := err.(Field); ok {
		return Typed{Type: TypeValidation, Err: Fields{fErr}}
	}
	if _, ok := err.(validator.ValidationErrors); !ok {
		return Unexpected(err)
	}
	var fields Fields
	for _, e := range err.(validator.ValidationErrors) {
		fields = append(fields, newFieldFromValidator(e))
	}
	return Typed{Type: TypeValidation, Err: fields}
}

// MaybeValidation is a convenience function for returning a Validation error if the error
// is not nil.
func MaybeValidation(err error) Typed {
	if err == nil {
		return Nil
	}
	return Validation(err)
}
