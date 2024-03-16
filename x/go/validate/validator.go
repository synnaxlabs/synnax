// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate

import (
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/types"
	"reflect"
)

type Validator struct {
	scope string
	errutil.Catch
}

func New(scope string) *Validator {
	return &Validator{scope: scope, Catch: *errutil.NewCatch()}
}

// Ternary adds the error with the given message to the validator if the condition
// is true.
func (v *Validator) Ternary(cond bool, msg string) bool {
	v.Exec(func() error {
		return lo.Ternary(cond, v.New(msg), nil)
	})
	return v.Error() != nil
}

func (v *Validator) Ternaryf(cond bool, format string, args ...any) bool {
	v.Exec(func() error {
		err := lo.Ternary(cond, v.Newf(format, args...), nil)
		return err
	})
	return v.Error() != nil
}

func (v *Validator) New(msg string) error {
	return errors.Wrapf(Error, "[%s] - "+msg, v.scope)
}

func (v *Validator) Newf(format string, args ...any) error {
	return errors.Wrapf(Error, "[%s] - "+format, append([]any{v.scope}, args...)...)
}

func (v *Validator) Funcf(f func() bool, format string, args ...any) bool {
	v.Exec(func() error {
		return lo.Ternary(f(), v.Newf(format, args...), nil)
	})
	return v.Error() != nil
}

func (v *Validator) Func(f func() bool, msg string) bool {
	v.Exec(func() error {
		return lo.Ternary(f(), v.New(msg), nil)
	})
	return v.Error() != nil
}

var (
	Error = errors.New("validation error")
)

type FieldError struct {
	Field   string
	Message string
}

func (f FieldError) Error() string { return f.Field + ": " + f.Message }

func NewFieldError(field, message string) FieldError {
	errors.Cause()
}

func NotNil(v *Validator, field string, value any) bool {
	isNil := value == nil || (reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil())
	return v.Ternaryf(isNil, "%s must be non-nil", field)
}

func Positive[T types.Numeric](v *Validator, field string, value T) bool {
	return v.Ternaryf(value <= 0, "%s must be positive", field)
}

func GreaterThan[T types.Numeric](v *Validator, field string, value T, threshold T) bool {
	return v.Ternaryf(value <= threshold, "%s must be greater than %d", field, threshold)
}

func GreaterThanEq[T types.Numeric](v *Validator, field string, value T, threshold T) bool {
	return v.Ternaryf(
		value < threshold,
		"%s must be greater than or equal to %d", field, threshold)
}

func NonZero[T types.Numeric](v *Validator, field string, value T) bool {
	return v.Ternaryf(
		value == 0,
		"%s must be non-zero", field)
}

func NonZeroable(v *Validator, field string, value override.Zeroable) bool {
	return v.Ternaryf(
		value.IsZero(),
		"%s must be non-zero", field,
	)
}

func NonNegative[T types.Numeric](v *Validator, field string, value T) bool {
	return v.Ternaryf(
		value < 0,
		"%s must be non-negative", field)
}

func NotEmptySlice[T any](v *Validator, field string, value []T) bool {
	return v.Ternaryf(
		len(value) == 0,
		"%s must be non-empty", field)
}

func NotEmptyString[T ~string](v *Validator, field string, value T) bool {
	return v.Ternaryf(value == "", "%s must be set", field)
}

func MapDoesNotContainF[K comparable, V any](
	v *Validator,
	value K,
	m map[K]V,
	format string,
	args ...any,
) bool {
	return v.Funcf(func() bool {
		_, ok := m[value]
		return ok
	}, format, args...)
}

func MapContainsf[K comparable, V any](
	v *Validator,
	value K,
	m map[K]V,
	format string,
	args ...any,
) bool {
	return v.Funcf(func() bool {
		_, ok := m[value]
		return !ok
	}, format, args...)
}
