// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn

import (
	"reflect"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// EnumT represents an enum type in the schema.
const EnumT DataType = "enum"

// EnumZ represents an enum schema. It provides methods for validating and converting
// enumerated values.
type EnumZ struct {
	baseZ
	values []any
}

var _ Schema = (*EnumZ)(nil)

// Shape returns the base shape of the enum schema.
func (e EnumZ) Shape() Shape { return e.baseZ }

// Validate checks that the data is a valid enum value without parsing into a
// destination.
func (e EnumZ) Validate(data any) error {
	if e.expectedType != nil {
		dest := reflect.New(e.expectedType).Interface()
		return e.Parse(data, dest)
	}
	var dest any
	return e.Parse(data, &dest)
}

// validateDestination validates that the destination is compatible with enum data.
func (e EnumZ) validateDestination(dest reflect.Value) error {
	if dest.Kind() != reflect.Pointer || dest.IsNil() {
		return NewInvalidDestinationTypeError("enum", dest)
	}
	// Get the actual destination type (dereferencing pointer layers)
	destType := dest.Type().Elem()
	for destType.Kind() == reflect.Pointer {
		destType = destType.Elem()
	}
	// Enum can accept destinations that are compatible with its enum values Allow
	// assignment compatibility for custom types
	if e.expectedType != nil &&
		(destType.AssignableTo(e.expectedType) ||
			e.expectedType.AssignableTo(destType)) {
		return nil
	}
	// If no specific expected type, allow basic types that the enum values could
	// convert to
	switch destType.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Bool:
		return nil
	}
	return NewInvalidDestinationTypeError("enum-compatible type", dest)
}

// Dump converts the given data to an enum value according to the schema. It validates
// the data and returns an error if the data is invalid. The function ensures the value
// is one of the allowed enum values.
func (e EnumZ) Dump(data any) (any, error) {
	if data == nil {
		if e.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.ErrRequired)
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if e.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.ErrRequired)
		}
		val = val.Elem()
	}
	// Check if the value is in the allowed values
	for _, v := range e.values {
		if reflect.DeepEqual(v, val.Interface()) {
			return val.Interface(), nil
		}
	}
	return nil, invalidEnumValueError(val.Interface(), e.values)
}

// Parse converts the given data from an enum value to the destination type. It
// validates the data and returns an error if the data is invalid. The function ensures
// the value is one of the allowed enum values.
func (e EnumZ) Parse(data, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := e.validateDestination(destVal); err != nil {
		return err
	}
	if ok, err := validateNilData(destVal, data, e.baseZ); !ok || err != nil {
		return err
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return errors.WithStack(validate.ErrRequired)
		}
		val = val.Elem()
	}
	destVal = destVal.Elem()
	if destVal.Kind() == reflect.Pointer {
		if destVal.IsNil() {
			destVal.Set(reflect.New(destVal.Type().Elem()))
		}
		destVal = destVal.Elem()
	}
	if convertedVal, ok := convertLossless(val, destVal.Type()); ok {
		for _, v := range e.values {
			enumVal, ok := convertLossless(reflect.ValueOf(v), destVal.Type())
			if ok && reflect.DeepEqual(enumVal.Interface(), convertedVal.Interface()) {
				destVal.Set(convertedVal)
				return nil
			}
		}
	}
	return invalidEnumValueError(val.Interface(), e.values)
}

// Enum creates a new enum schema with the given values. This is the entry point for
// creating enum validation schemas.
func Enum[T comparable](values ...T) EnumZ {
	if len(values) == 0 {
		panic("enums must have at least one value")
	}
	anyValues := make([]any, len(values))
	for i, v := range values {
		anyValues[i] = v
	}
	e := EnumZ{
		dataType:     EnumT,
		expectedType: reflect.TypeOf(values[0]),
		values:       anyValues,
	}
	e.wrapper = e
	return e
}

// convertLossless converts v to t only when both are in the same kind family (string,
// bool, or number) and the conversion keeps the value of v.
func convertLossless(v reflect.Value, t reflect.Type) (reflect.Value, bool) {
	if !v.Type().ConvertibleTo(t) {
		return reflect.Value{}, false
	}
	if t.Kind() == reflect.Interface {
		return v.Convert(t), true
	}
	if kindFamily(v.Kind()) != kindFamily(t.Kind()) {
		return reflect.Value{}, false
	}
	converted := v.Convert(t)
	if negative(converted) != negative(v) ||
		!reflect.DeepEqual(converted.Convert(v.Type()).Interface(), v.Interface()) {
		return reflect.Value{}, false
	}
	return converted, true
}

// negative reports whether v holds a number below zero. A same-width conversion between
// signed and unsigned ints round-trips, so only the sign shows that it wrapped.
func negative(v reflect.Value) bool {
	switch {
	case v.CanInt():
		return v.Int() < 0
	case v.CanFloat():
		return v.Float() < 0
	default:
		return false
	}
}

// kindFamily maps every numeric kind to reflect.Float64 and leaves other kinds as is.
func kindFamily(k reflect.Kind) reflect.Kind {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return reflect.Float64
	default:
		return k
	}
}

func invalidEnumValueError(value any, allowedValues []any) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"invalid enum value %v, allowed values are %v",
		value,
		allowedValues,
	)
}
