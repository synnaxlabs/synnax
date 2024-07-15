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
	"context"
	"fmt"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/types"
	"reflect"
	"strings"
)

type Validator struct {
	scope string
	errors.Catcher
}

func New(scope string) *Validator {
	return &Validator{scope: scope, Catcher: *errors.NewCatcher()}
}

// Ternary adds the error with the given message to the validator if the condition
// is true.
func (v *Validator) Ternary(field string, cond bool, msg string) bool {
	v.Exec(func() error {
		return lo.Ternary[error](cond, FieldError{
			Field:   field,
			Message: msg,
		}, nil)
	})
	return v.Error() != nil
}

func (v *Validator) Ternaryf(field string, cond bool, format string, args ...any) bool {
	v.Exec(func() error {
		err := lo.Ternary[error](cond, FieldError{
			Field:   field,
			Message: fmt.Sprintf(format, args...),
		}, nil)
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

var Error = errors.New("validation error")

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (fe FieldError) Error() string { return fe.Field + ":" + fe.Message }

func encode(_ context.Context, err error) (errors.Payload, bool) {
	var fe FieldError
	if errors.As(err, &fe) {
		return errors.Payload{
			Type: "sy.validation.field",
			Data: fe.Error(),
		}, true
	}
	if errors.Is(err, Error) {
		return errors.Payload{
			Type: "sy.validation",
			Data: err.Error(),
		}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	switch p.Type {
	case "sy.validation.field":
		values := strings.Split(p.Data, ": ")
		if len(values) < 2 {
			return errors.Wrapf(Error, p.Data), true
		}
		return FieldError{Field: values[0], Message: values[1]}, true
	case "sy.validation":
		return errors.Wrapf(Error, p.Data), true
	default:
		return nil, false
	}
}

func init() { errors.Register(encode, decode) }

func NotNil(v *Validator, field string, value any) bool {
	isNil := value == nil || (reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil())
	return v.Ternary(field, isNil, "must be non-nil")
}

func Positive[T types.Numeric](v *Validator, field string, value T) bool {
	return v.Ternaryf(field, value <= 0, "must be positive")
}

func GreaterThan[T types.Numeric](v *Validator, field string, value T, threshold T) bool {
	return v.Ternaryf(field, value <= threshold, "must be greater than %v", threshold)
}

func GreaterThanEq[T types.Numeric](v *Validator, field string, value T, threshold T) bool {
	return v.Ternaryf(
		field,
		value < threshold,
		"must be greater than or equal to %v", threshold,
	)
}

func LessThan[T types.Numeric](v *Validator, field string, value T, threshold T) bool {
	return v.Ternaryf(field, value >= threshold, "must be less than %v", threshold)
}

func LessThanEq[T types.Numeric](v *Validator, field string, value T, threshold T) bool {
	return v.Ternaryf(
		field,
		value > threshold,
		"must be less than or equal to %v", threshold)
}

func NonZero[T types.Numeric](v *Validator, field string, value T) bool {
	return v.Ternaryf(
		field,
		value == 0,
		"must be non-zero",
	)
}

func NonZeroable(v *Validator, field string, value override.Zeroable) bool {
	return v.Ternary(field, value.IsZero(), "must be non-zero")
}

func NonNegative[T types.Numeric](v *Validator, field string, value T) bool {
	return v.Ternary(field, value < 0, "field must be non-negative")
}

func NotEmptySlice[T any](v *Validator, field string, value []T) bool {
	return v.Ternary(field, len(value) == 0, "must be non-empty")
}

func NotEmptyString[T ~string](v *Validator, field string, value T) bool {
	return v.Ternary(field, value == "", "field must be set")
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
