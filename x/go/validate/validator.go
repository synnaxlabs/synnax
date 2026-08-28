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
	"reflect"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/types"
)

type Validator struct {
	scope string
	err   error
}

func New(scope string) *Validator { return &Validator{scope: scope} }

// Error returns the accumulated validation errors.
func (v *Validator) Error() error { return v.err }

// Exec runs a function and joins any error it returns with the accumulated errors.
func (v *Validator) Exec(fn func() error) { v.err = errors.Join(v.err, fn()) }

// Ternary adds the error with the given message to the validator if the condition is
// true.
func (v *Validator) Ternary(path string, cond bool, msg string) bool {
	v.Exec(func() error {
		return lo.Ternary(cond, PathedError(errors.New(msg), path), nil)
	})
	return v.Error() != nil
}

func (v *Validator) Ternaryf(field string, cond bool, format string, args ...any) bool {
	v.Exec(func() error {
		return lo.Ternary(cond, PathedError(errors.Newf(format, args...), field), nil)
	})
	return v.Error() != nil
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
// zero.
func (v *Validator) Positive[T types.Numeric](field string, value T) bool {
	return v.Ternary(field, value <= 0, "must be positive")
}

// InBounds returns true and attaches an error to v if the value is outside the
// half-open interval [lower, upper).
func (v *Validator) InBounds[T types.Numeric](
	field string,
	value, lower, upper T,
) bool {
	return v.Ternaryf(
		field,
		value < lower || value >= upper,
		"must be in bounds [%v, %v)",
		lower,
		upper,
	)
}

// GreaterThan returns true and attaches an error to v if the value is not greater than
// the threshold.
func (v *Validator) GreaterThan[T types.Numeric](
	field string,
	value T,
	threshold T,
) bool {
	return v.Ternaryf(field, value <= threshold, "must be greater than %v", threshold)
}

// GreaterThanEq returns true and attaches an error to v if the value is less than the
// threshold.
func (v *Validator) GreaterThanEq[T types.Numeric](
	field string,
	value T,
	threshold T,
) bool {
	return v.Ternaryf(
		field,
		value < threshold,
		"must be greater than or equal to %v",
		threshold,
	)
}

// LessThanEq returns true and attaches an error to v if the value is greater than the
// threshold.
func (v *Validator) LessThanEq[T types.Numeric](
	field string,
	value T,
	threshold T,
) bool {
	return v.Ternaryf(
		field,
		value > threshold,
		"must be less than or equal to %v",
		threshold,
	)
}

// NonZero returns true and attaches an error to v if the value is zero.
func (v *Validator) NonZero[T types.Numeric](field string, value T) bool {
	return v.Ternary(field, value == 0, "must be non-zero")
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
