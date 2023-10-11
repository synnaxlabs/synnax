// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/synnaxlabs/x/validate"
)

// Validation is an error that occurs when a user-supplied value is invalid. For example,
// a user provides an email address without the	@ symbol. Validation attempts to parse
// validation errors from the validator package into a set of Field errors.
func Validation(err error) Typed {
	var fields Fields
	if errors.As(err, &fields) {
		return Typed{Type: TypeValidation, Err: err}
	}
	var fErr Field
	if errors.As(err, &fErr) {
		return Typed{Type: TypeValidation, Err: Fields{fErr}}
	}
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		if errors.Is(err, validate.Error) {
			return Typed{Type: TypeValidation, Err: Fields{Field{Message: err.Error()}}}
		}
		return Unexpected(err)
	}
	var f Fields
	for _, e := range validationErrors {
		f = append(f, newFieldFromValidator(e))
	}
	return Typed{Type: TypeValidation, Err: f}
}

// MaybeValidation is a convenience function for returning a Validation error if the error
// is not nil.
func MaybeValidation(err error) Typed {
	if err == nil {
		return Nil
	}
	return Validation(err)
}
