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
	"math"
	"reflect"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// NumberZ is a schema for parsing numeric types.
type NumberZ struct {
	baseZ
	coerce bool
}

var _ Schema = (*NumberZ)(nil)

// Number is a schema that validates numeric values.
func Number() NumberZ {
	n := NumberZ{baseZ: baseZ{dataType: NumberT, expectedType: reflect.TypeOf(0)}}
	n.wrapper = n
	return n
}

// Optional marks the number field as optional. Optional fields can be nil or omitted.
func (n NumberZ) Optional() NumberZ { n.optional = true; return n }

// Coerce enables type coercion for the number field. When enabled, the schema will
// attempt to convert values to the expected type. This allows for more flexible type
// conversion but may result in precision loss.
func (n NumberZ) Coerce() NumberZ { n.coerce = true; return n }

// Shape returns the base shape of the number schema.
func (n NumberZ) Shape() Shape { return n.baseZ }

// validateDestination validates that the destination is compatible with numeric data
func (n NumberZ) validateDestination(dest reflect.Value) error {
	if dest.Kind() != reflect.Pointer || dest.IsNil() {
		return NewInvalidDestinationTypeError(string(n.dataType), dest)
	}
	destType := dest.Type().Elem()
	for destType.Kind() == reflect.Pointer {
		destType = destType.Elem()
	}
	if isNumericType(destType) {
		return nil
	}
	if n.expectedType != nil &&
		(destType.AssignableTo(n.expectedType) ||
			n.expectedType.AssignableTo(destType)) {
		return nil
	}
	return NewInvalidDestinationTypeError(string(n.dataType), dest)
}

// Float64 marks the number field as a float64. This enables float64-specific validation
// and conversion.
func (n NumberZ) Float64() NumberZ {
	n.expectedType = reflect.TypeOf(float64(0))
	n.dataType = Float64T
	return n
}

// Float32 marks the number field as a float32. This enables float32-specific validation
// and conversion.
func (n NumberZ) Float32() NumberZ {
	n.expectedType = reflect.TypeOf(float32(0))
	n.dataType = Float32T
	return n
}

// Int marks the number field as an int. This enables int-specific validation and
// conversion.
func (n NumberZ) Int() NumberZ {
	n.expectedType = reflect.TypeOf(0)
	n.dataType = IntT
	return n
}

// Int8 marks the number field as an int8. This enables int8-specific validation and
// conversion.
func (n NumberZ) Int8() NumberZ {
	n.expectedType = reflect.TypeOf(int8(0))
	n.dataType = Int8T
	return n
}

// Int16 marks the number field as an int16. This enables int16-specific validation and
// conversion.
func (n NumberZ) Int16() NumberZ {
	n.expectedType = reflect.TypeOf(int16(0))
	n.dataType = Int16T
	return n
}

// Int32 marks the number field as an int32. This enables int32-specific validation and
// conversion.
func (n NumberZ) Int32() NumberZ {
	n.expectedType = reflect.TypeOf(int32(0))
	n.dataType = Int32T
	return n
}

// Int64 marks the number field as an int64. This enables int64-specific validation and
// conversion.
func (n NumberZ) Int64() NumberZ {
	n.expectedType = reflect.TypeOf(int64(0))
	n.dataType = Int64T
	return n
}

// Uint marks the number field as a uint. This enables uint-specific validation and
// conversion.
func (n NumberZ) Uint() NumberZ {
	n.expectedType = reflect.TypeOf(uint(0))
	n.dataType = UintT
	return n
}

// Uint8 marks the number field as a uint8. This enables uint8-specific validation and
// conversion.
func (n NumberZ) Uint8() NumberZ {
	n.expectedType = reflect.TypeOf(uint8(0))
	n.dataType = Uint8T
	return n
}

// Uint16 marks the number field as a uint16. This enables uint16-specific validation
// and conversion.
func (n NumberZ) Uint16() NumberZ {
	n.expectedType = reflect.TypeOf(uint16(0))
	n.dataType = Uint16T
	return n
}

// Uint32 marks the number field as a uint32. This enables uint32-specific validation
// and conversion.
func (n NumberZ) Uint32() NumberZ {
	n.expectedType = reflect.TypeOf(uint32(0))
	n.dataType = Uint32T
	return n
}

// Uint64 marks the number field as a uint64. This enables uint64-specific validation
// and conversion.
func (n NumberZ) Uint64() NumberZ {
	n.expectedType = reflect.TypeOf(uint64(0))
	n.dataType = Uint64T
	return n
}

// Dump converts the given data to a number according to the schema. It validates the
// data and returns an error if the data is invalid. The function handles type
// conversion and validation based on the expected type. For integer types, it ensures
// the value is within the valid range. For floating-point types, it handles precision
// conversion.
func (n NumberZ) Dump(data any) (any, error) {
	if data == nil {
		if n.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.RequiredError)
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if n.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.RequiredError)
		}
		val = val.Elem()
	}
	// If an expected type is set and coercion is not enabled, validate the input type
	if n.dataType != NumberT && !n.coerce {
		if val.Type() != n.expectedType {
			return nil, validate.NewInvalidTypeError(
				n.expectedType.String(),
				types.ValueName(val),
			)
		}
		return val.Interface(), nil
	}
	// If the source is a custom type, try to convert it to a basic type first
	if val.Type().Kind() != reflect.Float64 &&
		val.Type().Kind() != reflect.Float32 &&
		val.Type().Kind() != reflect.Int &&
		val.Type().Kind() != reflect.Int8 &&
		val.Type().Kind() != reflect.Int16 &&
		val.Type().Kind() != reflect.Int32 &&
		val.Type().Kind() != reflect.Int64 &&
		val.Type().Kind() != reflect.Uint &&
		val.Type().Kind() != reflect.Uint8 &&
		val.Type().Kind() != reflect.Uint16 &&
		val.Type().Kind() != reflect.Uint32 &&
		val.Type().Kind() != reflect.Uint64 {
		if val.CanConvert(reflect.TypeOf(float64(0))) {
			val = val.Convert(reflect.TypeOf(float64(0)))
		} else {
			return nil, cannotConvertToNumberError(val)
		}
	}
	if n.dataType != NumberT {
		switch n.expectedType.Kind() {
		case reflect.Float64, reflect.Float32:
			var floatVal float64
			switch val.Kind() {
			case reflect.Float64, reflect.Float32:
				floatVal = val.Float()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
				reflect.Int64:
				floatVal = float64(val.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
				reflect.Uint64:
				floatVal = float64(val.Uint())
			default:
				return nil, cannotConvertToNumberError(val)
			}
			if n.expectedType.Kind() == reflect.Float32 {
				return float32(floatVal), nil
			}
			return floatVal, nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var intVal int64
			switch val.Kind() {
			case reflect.Float64, reflect.Float32:
				floatVal := val.Float()
				if floatVal != float64(int64(floatVal)) {
					return nil, fractionalPartError(floatVal)
				}
				intVal = int64(floatVal)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
				reflect.Int64:
				intVal = val.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
				reflect.Uint64:
				uintVal := val.Uint()
				if uintVal > uint64(math.MaxInt64) {
					return nil, unsignedIntegerTooLargeError()
				}
				intVal = int64(uintVal)
			default:
				return nil, cannotConvertToNumberError(val)
			}
			if intVal >
				(1<<(n.expectedType.Bits()-1)-1) ||
				intVal < -(1<<(n.expectedType.Bits()-1)) {
				return nil, valueOutOfRangeError(intVal, n.expectedType.String())
			}
			return reflect.ValueOf(intVal).Convert(n.expectedType).Interface(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64:
			var uintVal uint64
			switch val.Kind() {
			case reflect.Float64, reflect.Float32:
				floatVal := val.Float()
				if floatVal != float64(int64(floatVal)) {
					return nil, fractionalPartError(floatVal)
				}
				if floatVal < 0 {
					return nil, negativeToUnsignedError(floatVal)
				}
				uintVal = uint64(floatVal)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				intVal := val.Int()
				if intVal < 0 {
					return nil, negativeToUnsignedError(intVal)
				}
				uintVal = uint64(intVal)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
				reflect.Uint64:
				uintVal = val.Uint()
			default:
				return nil, cannotConvertToNumberError(val)
			}
			if uintVal > (1<<n.expectedType.Bits() - 1) {
				return nil, valueOutOfRangeError(uintVal, n.expectedType.String())
			}
			return reflect.ValueOf(uintVal).Convert(n.expectedType).Interface(), nil
		default:
		}
		return nil, validate.NewInvalidTypeError(
			n.expectedType.String(),
			types.ValueName(val),
		)
	}
	// If no expected type is set, return the value in its most appropriate form
	switch val.Kind() {
	case reflect.Float64, reflect.Float32:
		return val.Float(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint(), nil
	default:
	}
	return nil, cannotConvertToNumberError(val)
}

// Parse converts the given data from a number to the destination type. It validates the
// data and returns an error if the data is invalid. The function handles type
// conversion and validation based on the destination type. For integer types, it
// ensures the value is within the valid range. For floating-point types, it handles
// precision conversion.
func (n NumberZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := n.validateDestination(destVal); err != nil {
		return err
	}
	if ok, err := validateNilData(destVal, data, n.baseZ); !ok || err != nil {
		return err
	}
	destType := destVal.Type().Elem()
	destVal = destVal.Elem()
	if destVal.Kind() == reflect.Pointer {
		if destVal.IsNil() {
			destVal.Set(reflect.New(destType.Elem()))
		}
		destVal = destVal.Elem()
		destType = destType.Elem()
	}
	srcVal := reflect.ValueOf(data)
	srcValName := types.ValueName(srcVal)
	convertibleErr := validate.NewInvalidTypeError(
		"number or convertible to number",
		srcValName,
	)
	if n.dataType != NumberT && !n.coerce {
		if srcVal.Type() != n.expectedType {
			return validate.NewInvalidTypeError(n.expectedType.String(), srcValName)
		}
	}
	if srcVal.Type().Kind() != reflect.Float64 &&
		srcVal.Type().Kind() != reflect.Float32 &&
		srcVal.Type().Kind() != reflect.Int &&
		srcVal.Type().Kind() != reflect.Int8 &&
		srcVal.Type().Kind() != reflect.Int16 &&
		srcVal.Type().Kind() != reflect.Int32 &&
		srcVal.Type().Kind() != reflect.Int64 &&
		srcVal.Type().Kind() != reflect.Uint &&
		srcVal.Type().Kind() != reflect.Uint8 &&
		srcVal.Type().Kind() != reflect.Uint16 &&
		srcVal.Type().Kind() != reflect.Uint32 &&
		srcVal.Type().Kind() != reflect.Uint64 {
		if srcVal.CanConvert(reflect.TypeOf(float64(0))) {
			srcVal = srcVal.Convert(reflect.TypeOf(float64(0)))
		} else {
			return convertibleErr
		}
	}
	// Handle the destination type
	switch destType.Kind() {
	case reflect.Float64, reflect.Float32:
		// For floating point destinations, we can safely convert through float64
		var floatVal float64
		switch srcVal.Kind() {
		case reflect.Float64:
			floatVal = srcVal.Float()
		case reflect.Float32:
			floatVal = srcVal.Float()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			floatVal = float64(srcVal.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64:
			floatVal = float64(srcVal.Uint())
		default:
			return convertibleErr
		}
		destVal.SetFloat(floatVal)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal int64
		switch srcVal.Kind() {
		case reflect.Float64, reflect.Float32:
			floatVal := srcVal.Float()
			if floatVal != float64(int64(floatVal)) {
				return fractionalPartError(floatVal)
			}
			intVal = int64(floatVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal = srcVal.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64:
			uintVal := srcVal.Uint()
			if uintVal > uint64(math.MaxInt64) {
				return unsignedIntegerTooLargeError()
			}
			intVal = int64(uintVal)
		default:
			return convertibleErr
		}
		if intVal > (1<<(destType.Bits()-1)-1) || intVal < -(1<<(destType.Bits()-1)) {
			return errors.Wrap(
				validate.ConversionError,
				"integer value out of range for destination type",
			)
		}
		destVal.SetInt(intVal)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintVal uint64
		switch srcVal.Kind() {
		case reflect.Float64, reflect.Float32:
			floatVal := srcVal.Float()
			if floatVal != float64(int64(floatVal)) {
				return fractionalPartError(floatVal)
			}
			if floatVal < 0 {
				return negativeToUnsignedError(floatVal)
			}
			uintVal = uint64(floatVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal := srcVal.Int()
			if intVal < 0 {
				return negativeToUnsignedError(intVal)
			}
			uintVal = uint64(intVal)
		case reflect.Uint,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintVal = srcVal.Uint()
		default:
			return convertibleErr
		}
		if uintVal > (1<<destType.Bits() - 1) {
			return valueOutOfRangeError(uintVal, destType.Name())
		}
		destVal.SetUint(uintVal)
		return nil
	default:
	}
	return convertibleErr
}

// Uint32 is a schema that validates uint32 numbers.
func Uint32() NumberZ { return Number().Uint32() }

// Uint64 is a schema that validates uint64 numbers.
func Uint64() NumberZ { return Number().Uint64() }

// Int64 is a schema that validates int64 numbers.
func Int64() NumberZ { return Number().Int64() }

// Uint16 is a schema that validates uint16 numbers.
func Uint16() NumberZ { return Number().Uint16() }

func fractionalPartError(float float64) error {
	return errors.Wrapf(
		validate.ConversionError,
		"cannot convert float %v with fractional part to integer",
		float,
	)
}

func negativeToUnsignedError[T types.Numeric](value T) error {
	return errors.Wrapf(
		validate.ConversionError,
		"cannot convert negative value %v to unsigned integer",
		value,
	)
}

func valueOutOfRangeError[T types.Numeric](value T, destinationType string) error {
	return errors.Wrapf(
		validate.ConversionError,
		"integer value %v out of range for destination type %s",
		value,
		destinationType,
	)
}

func cannotConvertToNumberError(val reflect.Value) error {
	return validate.NewInvalidTypeError(
		"number or convertible to number",
		types.ValueName(val),
	)
}

func unsignedIntegerTooLargeError() error {
	return errors.Wrap(
		validate.ConversionError,
		"unsigned integer value too large for conversion to signed integer",
	)
}
