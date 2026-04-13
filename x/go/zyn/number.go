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
	n := NumberZ{baseZ: baseZ{dataType: NumberT, expectedType: reflect.TypeFor[int]()}}
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
	n.expectedType = reflect.TypeFor[float64]()
	n.dataType = Float64T
	return n
}

// Float32 marks the number field as a float32. This enables float32-specific validation
// and conversion.
func (n NumberZ) Float32() NumberZ {
	n.expectedType = reflect.TypeFor[float32]()
	n.dataType = Float32T
	return n
}

// Int marks the number field as an int. This enables int-specific validation and
// conversion.
func (n NumberZ) Int() NumberZ {
	n.expectedType = reflect.TypeFor[int]()
	n.dataType = IntT
	return n
}

// Int8 marks the number field as an int8. This enables int8-specific validation and
// conversion.
func (n NumberZ) Int8() NumberZ {
	n.expectedType = reflect.TypeFor[int8]()
	n.dataType = Int8T
	return n
}

// Int16 marks the number field as an int16. This enables int16-specific validation and
// conversion.
func (n NumberZ) Int16() NumberZ {
	n.expectedType = reflect.TypeFor[int16]()
	n.dataType = Int16T
	return n
}

// Int32 marks the number field as an int32. This enables int32-specific validation and
// conversion.
func (n NumberZ) Int32() NumberZ {
	n.expectedType = reflect.TypeFor[int32]()
	n.dataType = Int32T
	return n
}

// Int64 marks the number field as an int64. This enables int64-specific validation and
// conversion.
func (n NumberZ) Int64() NumberZ {
	n.expectedType = reflect.TypeFor[int64]()
	n.dataType = Int64T
	return n
}

// Uint marks the number field as a uint. This enables uint-specific validation and
// conversion.
func (n NumberZ) Uint() NumberZ {
	n.expectedType = reflect.TypeFor[uint]()
	n.dataType = UintT
	return n
}

// Uint8 marks the number field as a uint8. This enables uint8-specific validation and
// conversion.
func (n NumberZ) Uint8() NumberZ {
	n.expectedType = reflect.TypeFor[uint8]()
	n.dataType = Uint8T
	return n
}

// Uint16 marks the number field as a uint16. This enables uint16-specific validation
// and conversion.
func (n NumberZ) Uint16() NumberZ {
	n.expectedType = reflect.TypeFor[uint16]()
	n.dataType = Uint16T
	return n
}

// Uint32 marks the number field as a uint32. This enables uint32-specific validation
// and conversion.
func (n NumberZ) Uint32() NumberZ {
	n.expectedType = reflect.TypeFor[uint32]()
	n.dataType = Uint32T
	return n
}

// Uint64 marks the number field as a uint64. This enables uint64-specific validation
// and conversion.
func (n NumberZ) Uint64() NumberZ {
	n.expectedType = reflect.TypeFor[uint64]()
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
		return nil, errors.WithStack(validate.ErrRequired)
	}
	if result, ok := n.dumpFast(data); ok {
		return result, nil
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if n.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.ErrRequired)
		}
		val = val.Elem()
	}
	return n.dumpReflect(val)
}

func (n NumberZ) dumpFast(data any) (any, bool) {
	if n.dataType == NumberT {
		switch v := data.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return int64(v), true
		case int64:
			return v, true
		case int32:
			return int64(v), true
		case uint32:
			return uint64(v), true
		case uint64:
			return v, true
		}
		return nil, false
	}
	if n.coerce {
		return nil, false
	}
	switch n.dataType {
	case Float64T:
		if v, ok := data.(float64); ok {
			return v, true
		}
	case Float32T:
		if v, ok := data.(float32); ok {
			return v, true
		}
	case IntT:
		if v, ok := data.(int); ok {
			return v, true
		}
	case Int64T:
		if v, ok := data.(int64); ok {
			return v, true
		}
	case Int32T:
		if v, ok := data.(int32); ok {
			return int64(v), true
		}
	case Uint32T:
		if v, ok := data.(uint32); ok {
			return uint64(v), true
		}
	case Uint64T:
		if v, ok := data.(uint64); ok {
			return v, true
		}
	case Uint16T:
		if v, ok := data.(uint16); ok {
			return uint64(v), true
		}
	}
	return nil, false
}

func (n NumberZ) dumpReflect(val reflect.Value) (any, error) {
	if n.dataType != NumberT && !n.coerce {
		if val.Type() != n.expectedType {
			return nil, validate.NewInvalidTypeError(
				n.expectedType.String(),
				types.ValueName(val),
			)
		}
		return val.Interface(), nil
	}
	if !isNumericType(val.Type()) {
		if val.CanConvert(reflect.TypeFor[float64]()) {
			val = val.Convert(reflect.TypeFor[float64]())
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
	if ok := n.parseFast(data, dest); ok {
		return nil
	}
	return n.parseReflect(data, dest)
}

func (n NumberZ) parseFast(data any, dest any) bool {
	if data == nil || dest == nil {
		return false
	}
	if n.dataType != NumberT && !n.coerce {
		switch d := dest.(type) {
		case *float64:
			if d == nil {
				return false
			}
			if v, ok := data.(float64); ok && n.dataType == Float64T {
				*d = v
				return true
			}
		case *int:
			if d == nil {
				return false
			}
			if v, ok := data.(int); ok && n.dataType == IntT {
				*d = v
				return true
			}
		case *int64:
			if d == nil {
				return false
			}
			if v, ok := data.(int64); ok && n.dataType == Int64T {
				*d = v
				return true
			}
		case *uint32:
			if d == nil {
				return false
			}
			if v, ok := data.(uint32); ok && n.dataType == Uint32T {
				*d = v
				return true
			}
		case *uint64:
			if d == nil {
				return false
			}
			if v, ok := data.(uint64); ok && n.dataType == Uint64T {
				*d = v
				return true
			}
		}
		return false
	}
	switch d := dest.(type) {
	case *float64:
		if d == nil {
			return false
		}
		switch v := data.(type) {
		case float64:
			*d = v
			return true
		case int:
			*d = float64(v)
			return true
		case int64:
			*d = float64(v)
			return true
		}
	case *int:
		if d == nil {
			return false
		}
		if v, ok := data.(int); ok {
			*d = v
			return true
		}
	case *int64:
		if d == nil {
			return false
		}
		if v, ok := data.(int64); ok {
			*d = v
			return true
		}
	case *uint32:
		if d == nil {
			return false
		}
		if v, ok := data.(uint32); ok {
			*d = v
			return true
		}
	case *uint64:
		if d == nil {
			return false
		}
		if v, ok := data.(uint64); ok {
			*d = v
			return true
		}
	}
	return false
}

func (n NumberZ) parseReflect(data any, dest any) error {
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
	if n.dataType != NumberT && !n.coerce {
		if srcVal.Type() != n.expectedType {
			return validate.NewInvalidTypeError(
				n.expectedType.String(),
				types.ValueName(srcVal),
			)
		}
	}
	if !isNumericType(srcVal.Type()) {
		if srcVal.CanConvert(reflect.TypeFor[float64]()) {
			srcVal = srcVal.Convert(reflect.TypeFor[float64]())
		} else {
			return cannotConvertToNumberError(srcVal)
		}
	}
	switch destType.Kind() {
	case reflect.Float64, reflect.Float32:
		var floatVal float64
		switch srcVal.Kind() {
		case reflect.Float64, reflect.Float32:
			floatVal = srcVal.Float()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			floatVal = float64(srcVal.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64:
			floatVal = float64(srcVal.Uint())
		default:
			return cannotConvertToNumberError(srcVal)
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
			return cannotConvertToNumberError(srcVal)
		}
		if intVal > (1<<(destType.Bits()-1)-1) || intVal < -(1<<(destType.Bits()-1)) {
			return errors.Wrap(
				validate.ErrConversion,
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
			return cannotConvertToNumberError(srcVal)
		}
		if uintVal > (1<<destType.Bits() - 1) {
			return valueOutOfRangeError(uintVal, destType.Name())
		}
		destVal.SetUint(uintVal)
		return nil
	default:
	}
	return cannotConvertToNumberError(srcVal)
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
		validate.ErrConversion,
		"cannot convert float %v with fractional part to integer",
		float,
	)
}

func negativeToUnsignedError[T types.Numeric](value T) error {
	return errors.Wrapf(
		validate.ErrConversion,
		"cannot convert negative value %v to unsigned integer",
		value,
	)
}

func valueOutOfRangeError[T types.Numeric](value T, destinationType string) error {
	return errors.Wrapf(
		validate.ErrConversion,
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
		validate.ErrConversion,
		"unsigned integer value too large for conversion to signed integer",
	)
}
