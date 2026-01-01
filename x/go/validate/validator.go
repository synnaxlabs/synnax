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
	errors.Catcher
}

func New(scope string) *Validator {
	return &Validator{scope: scope, Catcher: *errors.NewCatcher()}
}

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
func NotNil(v *Validator, field string, value any) bool {
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

func Positive[T types.Numeric](v *Validator, field string, value T) bool {
	return v.Ternary(field, value <= 0, "must be positive")
}

func GreaterThanEq[T types.Numeric](
	v *Validator,
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

func LessThanEq[T types.Numeric](
	v *Validator,
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

func NonZero[T types.Numeric](v *Validator, field string, value T) bool {
	return v.Ternary(field, value == 0, "must be non-zero")
}

func NonZeroable(v *Validator, field string, value override.Zeroable) bool {
	return v.Ternary(field, value.IsZero(), "must be non-zero")
}

func NotEmptySlice[T any](v *Validator, field string, value []T) bool {
	return v.Ternary(field, len(value) == 0, "must be non-empty")
}

func NotEmptyString[T ~string](v *Validator, field string, value T) bool {
	return v.Ternary(field, value == "", "required")
}
