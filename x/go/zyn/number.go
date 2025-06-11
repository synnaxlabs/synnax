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

type NumberZ struct {
	baseZ
	expectedType reflect.Type
}

func (n NumberZ) Optional() NumberZ { n.optional = true; return n }

func (n NumberZ) Shape() Shape { return n.baseZ }

func (n NumberZ) Float64() NumberZ {
	n.expectedType = reflect.TypeOf(float64(0))
	n.typ = Float64T
	return n
}

func (n NumberZ) Float32() NumberZ {
	n.expectedType = reflect.TypeOf(float32(0))
	n.typ = Float32T
	return n
}

func (n NumberZ) Int() NumberZ {
	n.expectedType = reflect.TypeOf(int(0))
	n.typ = IntT
	return n
}

func (n NumberZ) Int8() NumberZ {
	n.expectedType = reflect.TypeOf(int8(0))
	n.typ = Int8T
	return n
}

func (n NumberZ) Int16() NumberZ {
	n.expectedType = reflect.TypeOf(int16(0))
	n.typ = Int16T
	return n
}

func (n NumberZ) Int32() NumberZ {
	n.expectedType = reflect.TypeOf(int32(0))
	n.typ = Int32T
	return n
}

func (n NumberZ) Int64() NumberZ {
	n.expectedType = reflect.TypeOf(int64(0))
	n.typ = Int64T
	return n
}

func (n NumberZ) Uint() NumberZ {
	n.expectedType = reflect.TypeOf(uint(0))
	n.typ = UintT
	return n
}

func (n NumberZ) Uint8() NumberZ {
	n.expectedType = reflect.TypeOf(uint8(0))
	n.typ = Uint8T
	return n
}

func (n NumberZ) Uint16() NumberZ {
	n.expectedType = reflect.TypeOf(uint16(0))
	n.typ = Uint16T
	return n
}

func (n NumberZ) Uint32() NumberZ {
	n.expectedType = reflect.TypeOf(uint32(0))
	n.typ = Uint32T
	return n
}

func (n NumberZ) Uint64() NumberZ {
	n.expectedType = reflect.TypeOf(uint64(0))
	n.typ = Uint64T
	return n
}

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

	// If an expected type is set, validate the input type
	if n.expectedType != nil {
		if val.Type() != n.expectedType {
			return nil, validate.FieldError{Message: "invalid type: expected " + n.expectedType.String()}
		}
		return val.Interface(), nil
	}

	// Handle the source value based on its type
	switch val.Kind() {
	case reflect.Float64, reflect.Float32:
		return val.Float(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint(), nil
	default:
		// Try to convert to float64 first
		if val.CanConvert(reflect.TypeOf(float64(0))) {
			return val.Convert(reflect.TypeOf(float64(0))).Float(), nil
		}
		return nil, validate.FieldError{Message: "invalid type: expected number or convertible to number"}
	}
}

func Number() NumberZ { return NumberZ{baseZ: baseZ{typ: NumberT}} }

func Uint32() NumberZ  { return Number().Uint32() }
func Uint64() NumberZ  { return Number().Uint64() }
func Float32() NumberZ { return Number().Float32() }
func Float64() NumberZ { return Number().Float64() }
func Int() NumberZ     { return Number().Int() }
func Int8() NumberZ    { return Number().Int8() }
func Int16() NumberZ   { return Number().Int16() }
func Int32() NumberZ   { return Number().Int32() }
func Int64() NumberZ   { return Number().Int64() }
func Uint() NumberZ    { return Number().Uint() }
func Uint8() NumberZ   { return Number().Uint8() }
func Uint16() NumberZ  { return Number().Uint16() }

func (n NumberZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := checkDestVal(destVal, string(n.typ)); err != nil {
		return err
	}

	if data == nil {
		if n.optional {
			if destVal.Elem().Kind() == reflect.Ptr {
				destVal.Elem().Set(reflect.Zero(destVal.Elem().Type()))
			}
			return nil
		}
		return validate.FieldError{Message: "value is required but was nil"}
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

	// If an expected type is set, validate the input type
	if n.expectedType != nil {
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
