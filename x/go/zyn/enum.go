// Copyright 2025 Synnax Labs, Inc.
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
const EnumT Type = "enum"

// EnumZ represents an enum schema.
// It provides methods for validating and converting enumerated values.
type EnumZ struct {
	baseZ
	values []any
}

var _ Z = (*EnumZ)(nil)

// Optional marks the enum field as optional.
// Optional fields can be nil or omitted.
func (e EnumZ) Optional() EnumZ { e.optional = true; return e }

// Shape returns the base shape of the enum schema.
func (e EnumZ) Shape() Shape { return e.baseZ }

// Values adds the given values to the enum schema.
// The values must be of the same type.
func (e EnumZ) Values(values ...any) EnumZ {
	if e.values == nil {
		e.values = make([]any, 0)
	}
	e.values = append(e.values, values...)
	return e
}

// Dump converts the given data to an enum value according to the schema.
// It validates the data and returns an error if the data is invalid.
// The function ensures the value is one of the allowed enum values.
func (e EnumZ) Dump(data any) (any, error) {
	if data == nil {
		if e.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.RequiredError)
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			if e.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.RequiredError)
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

// Parse converts the given data from an enum value to the destination type.
// It validates the data and returns an error if the data is invalid.
// The function ensures the value is one of the allowed enum values.
func (e EnumZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := validateDestinationValue(destVal, string(EnumT)); err != nil {
		return err
	}

	if ok, err := validateNilData(destVal, data, e.baseZ); !ok || err != nil {
		return err
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return errors.WithStack(validate.RequiredError)
		}
		val = val.Elem()
	}

	destVal = destVal.Elem()
	if destVal.Kind() == reflect.Ptr {
		if destVal.IsNil() {
			destVal.Set(reflect.New(destVal.Type().Elem()))
		}
		destVal = destVal.Elem()
	}

	// Try to convert the value to the destination type first
	if val.Type().ConvertibleTo(destVal.Type()) {
		convertedVal := val.Convert(destVal.Type())
		// Check if the converted value is in the allowed values
		for _, v := range e.values {
			// Try direct comparison first
			if reflect.DeepEqual(v, convertedVal.Interface()) {
				destVal.Set(convertedVal)
				return nil
			}
			// Try comparing underlying values
			if reflect.DeepEqual(reflect.ValueOf(v).Convert(destVal.Type()).Interface(), convertedVal.Interface()) {
				destVal.Set(convertedVal)
				return nil
			}
		}
	}

	return invalidEnumValueError(val.Interface(), e.values)
}

// Enum creates a new enum schema with the given values.
// This is the entry point for creating enum validation schemas.
func Enum[T comparable](values ...T) EnumZ {
	anyValues := make([]any, len(values))
	for i, v := range values {
		anyValues[i] = v
	}
	return EnumZ{
		baseZ:  baseZ{typ: EnumT},
		values: anyValues,
	}
}

func invalidEnumValueError(value any, allowedValues []any) error {
	return errors.Wrapf(validate.Error, "invalid enum value %v, allowed values are %v", value, allowedValues)
}
