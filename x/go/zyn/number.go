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

	"github.com/synnaxlabs/x/validate"
)

// NumberZ is a schema for parsing numeric types.
type NumberZ struct {
	baseZ
	expectedType reflect.Type
	coerce       bool
}

// Optional marks the number field as optional. Optional fields can be nil or omitted.
func (n NumberZ) Optional() NumberZ { n.optional = true; return n }

// Coerce enables type coercion for the number field.
// When enabled, the schema will attempt to convert values to the expected type.
// This allows for more flexible type conversion but may result in precision loss.
func (n NumberZ) Coerce() NumberZ { n.coerce = true; return n }

// Shape returns the base shape of the number schema.
func (n NumberZ) Shape() Shape { return n.baseZ }

// Float64 marks the number field as a float64.
// This enables float64-specific validation and conversion.
func (n NumberZ) Float64() NumberZ {
	n.expectedType = reflect.TypeOf(float64(0))
	n.typ = Float64T
	return n
}

// Float32 marks the number field as a float32.
// This enables float32-specific validation and conversion.
func (n NumberZ) Float32() NumberZ {
	n.expectedType = reflect.TypeOf(float32(0))
	n.typ = Float32T
	return n
}

// Int marks the number field as an int.
// This enables int-specific validation and conversion.
func (n NumberZ) Int() NumberZ {
	n.expectedType = reflect.TypeOf(int(0))
	n.typ = IntT
	return n
}

// Int8 marks the number field as an int8.
// This enables int8-specific validation and conversion.
func (n NumberZ) Int8() NumberZ {
	n.expectedType = reflect.TypeOf(int8(0))
	n.typ = Int8T
	return n
}

// Int16 marks the number field as an int16.
// This enables int16-specific validation and conversion.
func (n NumberZ) Int16() NumberZ {
	n.expectedType = reflect.TypeOf(int16(0))
	n.typ = Int16T
	return n
}

// Int32 marks the number field as an int32.
// This enables int32-specific validation and conversion.
func (n NumberZ) Int32() NumberZ {
	n.expectedType = reflect.TypeOf(int32(0))
	n.typ = Int32T
	return n
}

// Int64 marks the number field as an int64.
// This enables int64-specific validation and conversion.
func (n NumberZ) Int64() NumberZ {
	n.expectedType = reflect.TypeOf(int64(0))
	n.typ = Int64T
	return n
}

// Uint marks the number field as a uint.
// This enables uint-specific validation and conversion.
func (n NumberZ) Uint() NumberZ {
	n.expectedType = reflect.TypeOf(uint(0))
	n.typ = UintT
	return n
}

// Uint8 marks the number field as a uint8.
// This enables uint8-specific validation and conversion.
func (n NumberZ) Uint8() NumberZ {
	n.expectedType = reflect.TypeOf(uint8(0))
	n.typ = Uint8T
	return n
}

// Uint16 marks the number field as a uint16.
// This enables uint16-specific validation and conversion.
func (n NumberZ) Uint16() NumberZ {
	n.expectedType = reflect.TypeOf(uint16(0))
	n.typ = Uint16T
	return n
}

// Uint32 marks the number field as a uint32.
// This enables uint32-specific validation and conversion.
func (n NumberZ) Uint32() NumberZ {
	n.expectedType = reflect.TypeOf(uint32(0))
	n.typ = Uint32T
	return n
}

// Uint64 marks the number field as a uint64.
// This enables uint64-specific validation and conversion.
func (n NumberZ) Uint64() NumberZ {
	n.expectedType = reflect.TypeOf(uint64(0))
	n.typ = Uint64T
	return n
}

// Dump converts the given data to a number according to the schema.
// It validates the data and returns an error if the data is invalid.
// The function handles type conversion and validation based on the expected type.
// For integer types, it ensures the value is within the valid range.
// For floating-point types, it handles precision conversion.
func (n NumberZ) Dump(data any) (any, error) {
	if data == nil {
		if n.optional {
			return nil, nil
		}
		return nil, validate.FieldError{Message: "value is required but was nil"}
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			if n.optional {
				return nil, nil
			}
			return nil, validate.FieldError{Message: "value is required but was nil"}
		}
		val = val.Elem()
	}

	// If an expected type is set and coercion is not enabled, validate the input type
	if n.expectedType != nil && !n.coerce {
		if val.Type() != n.expectedType {
			return nil, invalidTypeError(n.expectedType, val)
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
		// Try to convert to float64 first
		if val.CanConvert(reflect.TypeOf(float64(0))) {
			val = val.Convert(reflect.TypeOf(float64(0)))
		} else {
			return nil, validate.FieldError{Message: "invalid type: expected number or convertible to number"}
		}
	}

	// If an expected type is set, convert to that type
	if n.expectedType != nil {
		// Handle the conversion based on the expected type
		switch n.expectedType.Kind() {
		case reflect.Float64, reflect.Float32:
			var floatVal float64
			switch val.Kind() {
			case reflect.Float64, reflect.Float32:
				floatVal = val.Float()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				floatVal = float64(val.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				floatVal = float64(val.Uint())
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
					return nil, validate.FieldError{Message: "cannot convert float with fractional part to integer"}
				}
				intVal = int64(floatVal)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				intVal = val.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				uintVal := val.Uint()
				if uintVal > uint64(1<<63-1) {
					return nil, validate.FieldError{Message: "unsigned integer value too large for signed integer type"}
				}
				intVal = int64(uintVal)
			}
			// Check for overflow
			if intVal > (1<<(n.expectedType.Bits()-1)-1) || intVal < -(1<<(n.expectedType.Bits()-1)) {
				return nil, validate.FieldError{Message: "integer value out of range for destination type"}
			}
			return reflect.ValueOf(intVal).Convert(n.expectedType).Interface(), nil

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var uintVal uint64
			switch val.Kind() {
			case reflect.Float64, reflect.Float32:
				floatVal := val.Float()
				if floatVal != float64(int64(floatVal)) {
					return nil, validate.FieldError{Message: "cannot convert float with fractional part to unsigned integer"}
				}
				if floatVal < 0 {
					return nil, validate.FieldError{Message: "cannot convert negative value to unsigned integer"}
				}
				uintVal = uint64(floatVal)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				intVal := val.Int()
				if intVal < 0 {
					return nil, validate.FieldError{Message: "cannot convert negative value to unsigned integer"}
				}
				uintVal = uint64(intVal)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				uintVal = val.Uint()
			}
			if uintVal > (1<<n.expectedType.Bits() - 1) {
				return nil, validate.FieldError{Message: "unsigned integer value out of range for destination type"}
			}
			return reflect.ValueOf(uintVal).Convert(n.expectedType).Interface(), nil
		}
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
		return nil, validate.FieldError{Message: "invalid type: expected number or convertible to number"}
	}
}

// Parse converts the given data from a number to the destination type.
// It validates the data and returns an error if the data is invalid.
// The function handles type conversion and validation based on the destination type.
// For integer types, it ensures the value is within the valid range.
// For floating-point types, it handles precision conversion.
func (n NumberZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := validateDestinationValue(destVal, string(n.typ)); err != nil {
		return err
	}
	if ok, err := validateNilData(destVal, data, n.baseZ); !ok || err != nil {
		return err
	}

	destType := destVal.Type().Elem()
	destVal = destVal.Elem()

	// If the destination is a pointer, we need to allocate it
	if destVal.Kind() == reflect.Ptr {
		if destVal.IsNil() {
			destVal.Set(reflect.New(destType.Elem()))
		}
		destVal = destVal.Elem()
		destType = destType.Elem()
	}

	// Handle the source value based on its type
	srcVal := reflect.ValueOf(data)

	// If an expected type is set and coercion is not enabled, validate the input type
	if n.expectedType != nil && !n.coerce {
		if srcVal.Type() != n.expectedType {
			return validate.FieldError{Message: "invalid type: expected " + n.expectedType.String()}
		}
	}

	// If the source is a custom type, try to convert it to a basic type first
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
		// Try to convert to float64 first
		if srcVal.CanConvert(reflect.TypeOf(float64(0))) {
			srcVal = srcVal.Convert(reflect.TypeOf(float64(0)))
		} else {
			return validate.FieldError{Message: "invalid type: expected number or convertible to number"}
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
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			floatVal = float64(srcVal.Uint())
		}
		destVal.SetFloat(floatVal)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// For integer destinations, we need to be careful about precision
		var intVal int64
		switch srcVal.Kind() {
		case reflect.Float64, reflect.Float32:
			// Check if the float has a fractional part
			floatVal := srcVal.Float()
			if floatVal != float64(int64(floatVal)) {
				return validate.FieldError{Message: "cannot convert float with fractional part to integer"}
			}
			intVal = int64(floatVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal = srcVal.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintVal := srcVal.Uint()
			if uintVal > uint64(1<<63-1) {
				return validate.FieldError{Message: "unsigned integer value too large for signed integer type"}
			}
			intVal = int64(uintVal)
		}
		// Check for overflow
		if intVal > (1<<(destType.Bits()-1)-1) || intVal < -(1<<(destType.Bits()-1)) {
			return validate.FieldError{Message: "integer value out of range for destination type"}
		}
		destVal.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// For unsigned integer destinations, we need to be careful about negative values
		var uintVal uint64
		switch srcVal.Kind() {
		case reflect.Float64, reflect.Float32:
			// Check if the float has a fractional part
			floatVal := srcVal.Float()
			if floatVal != float64(int64(floatVal)) {
				return validate.FieldError{Message: "cannot convert float with fractional part to unsigned integer"}
			}
			// Check if the float is negative
			if floatVal < 0 {
				return validate.FieldError{Message: "cannot convert negative value to unsigned integer"}
			}
			uintVal = uint64(floatVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Check if the int is negative
			intVal := srcVal.Int()
			if intVal < 0 {
				return validate.FieldError{Message: "cannot convert negative value to unsigned integer"}
			}
			uintVal = uint64(intVal)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintVal = srcVal.Uint()
		}
		// Check for overflow
		if uintVal > (1<<destType.Bits() - 1) {
			return validate.FieldError{Message: "unsigned integer value out of range for destination type"}
		}
		destVal.SetUint(uintVal)
	}

	return nil
}

// Number is a schema that validates numeric values.
func Number() NumberZ { return NumberZ{baseZ: baseZ{typ: NumberT}} }

// Uint32 is a schema that validates uint32 numbers.
func Uint32() NumberZ { return Number().Uint32() }

// Uint64 is a schema that validates uint64 numbers.
func Uint64() NumberZ { return Number().Uint64() }

// Float32 is a schema that validates float32 numbers.
func Float32() NumberZ { return Number().Float32() }

// Float64 is a schema that validates float64 numbers.
func Float64() NumberZ { return Number().Float64() }

// Int is a schema that validates integer numbers.
func Int() NumberZ { return Number().Int() }

// Int8 is a schema that validates int8 numbers.
func Int8() NumberZ { return Number().Int8() }

// Int16 is a schema that validates int16 numbers.
func Int16() NumberZ { return Number().Int16() }

// Int32 is a schema that validates int32 numbers.
func Int32() NumberZ { return Number().Int32() }

// Int64 is a schema that validates int64 numbers.
func Int64() NumberZ { return Number().Int64() }

// Uint is a schema that validates uint numbers.
func Uint() NumberZ { return Number().Uint() }

// Uint8 is a schema that validates uint8 numbers.
func Uint8() NumberZ { return Number().Uint8() }

// Uint16 is a schema that validates uint16 numbers.
func Uint16() NumberZ { return Number().Uint16() }
