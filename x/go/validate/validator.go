// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate

import (
	"math"
	"reflect"
	"unicode/utf8"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/types"
)

type Validator struct {
	scope string
	errs  []error
}

func New(scope string) *Validator { return &Validator{scope: scope} }

// Error returns the accumulated validation errors.
func (v *Validator) Error() error { return errors.Join(v.errs...) }

// Exec runs a function and adds any error it returns to the accumulated errors.
func (v *Validator) Exec(fn func() error) {
	if err := fn(); err != nil {
		v.errs = append(v.errs, err)
	}
}

// ruleError is a validation failure. It matches ErrValidation without adding the text
// of ErrValidation to its message.
type ruleError struct{ error }

func (e ruleError) Unwrap() error { return e.error }

func (ruleError) Is(target error) bool { return target == ErrValidation }

// Ternary adds the error with the given message to the validator if the condition is
// true.
func (v *Validator) Ternary(path string, cond bool, msg string) bool {
	v.Exec(func() error {
		if !cond {
			return nil
		}
		return PathedError(ruleError{errors.New(msg)}, path)
	})
	return len(v.errs) > 0
}

func (v *Validator) Ternaryf(field string, cond bool, format string, args ...any) bool {
	v.Exec(func() error {
		if !cond {
			return nil
		}
		return PathedError(ruleError{errors.Newf(format, args...)}, field)
	})
	return len(v.errs) > 0
}

// NotNil returns true and attaches an error to v if the value is nil.
func (v *Validator) NotNil(field string, value any) bool {
	isNil := value == nil
	if !isNil {
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface,
			reflect.Map, reflect.Pointer, reflect.Slice:
			isNil = rv.IsNil()
		}
	}
	return v.Ternary(field, isNil, "must be non-nil")
}

// Positive returns true and attaches an error to v if the value is not greater than
// zero or is NaN.
func (v *Validator) Positive[T types.Numeric](field string, value T) bool {
	return v.Ternary(field, isNaN(value) || value <= 0, "must be positive")
}

// InBounds returns true and attaches an error to v if the value is outside the
// half-open interval [lower, upper) or is NaN.
func (v *Validator) InBounds[T types.Numeric](
	field string,
	value, lower, upper T,
) bool {
	return v.Ternaryf(
		field,
		isNaN(value) || value < lower || value >= upper,
		"must be in bounds [%v, %v)",
		lower,
		upper,
	)
}

// GreaterThan returns true and attaches an error to v if the value is not greater than
// the threshold or is NaN.
func (v *Validator) GreaterThan[T types.Numeric](
	field string,
	value T,
	threshold T,
) bool {
	return v.Ternaryf(
		field,
		isNaN(value) || value <= threshold,
		"must be greater than %v",
		threshold,
	)
}

// GreaterThanEq returns true and attaches an error to v if the value is less than the
// threshold or is NaN.
func (v *Validator) GreaterThanEq[T types.Numeric](
	field string,
	value T,
	threshold T,
) bool {
	return v.Ternaryf(
		field,
		isNaN(value) || value < threshold,
		"must be greater than or equal to %v",
		threshold,
	)
}

// LessThanEq returns true and attaches an error to v if the value is greater than the
// threshold or is NaN.
func (v *Validator) LessThanEq[T types.Numeric](
	field string,
	value T,
	threshold T,
) bool {
	return v.Ternaryf(
		field,
		isNaN(value) || value > threshold,
		"must be less than or equal to %v",
		threshold,
	)
}

func isNaN[T types.Numeric](value T) bool { return math.IsNaN(float64(value)) }

// NonZero returns true and attaches an error to v if the value is zero.
func (v *Validator) NonZero[T comparable](field string, value T) bool {
	var zero T
	return v.Ternary(field, value == zero, "must be non-zero")
}

// NonZeroable returns true and attaches an error to v if the value reports itself as
// zero.
func (v *Validator) NonZeroable(field string, value override.Zeroable) bool {
	return v.Ternary(field, value.IsZero(), "must be non-zero")
}

// NotEmptySlice returns true and attaches an error to v if the slice is empty.
func (v *Validator) NotEmptySlice[T any](field string, value []T) bool {
	return v.Ternary(field, len(value) == 0, "must be non-empty")
}

// NotEmptyString returns true and attaches an error to v if the string is empty.
func (v *Validator) NotEmptyString[T ~string](field string, value T) bool {
	return v.Ternary(field, value == "", "required")
}

// MinLength returns true and attaches an error to v if the string has fewer than min
// Unicode code points.
func (v *Validator) MinLength[T ~string](field string, value T, min int) bool {
	return v.Ternaryf(
		field,
		utf8.RuneCountInString(string(value)) < min,
		"must be at least %d characters long",
		min,
	)
}

// MaxLength returns true and attaches an error to v if the string has more than max
// Unicode code points.
func (v *Validator) MaxLength[T ~string](field string, value T, max int) bool {
	return v.Ternaryf(
		field,
		utf8.RuneCountInString(string(value)) > max,
		"must be at most %d characters long",
		max,
	)
}
