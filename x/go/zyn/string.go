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
	"strconv"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/validate"
)

type StringZ struct {
	baseZ
	expectedType reflect.Type
}

var _ Z = (*StringZ)(nil)

func (s StringZ) Optional() StringZ { s.optional = true; return s }

func (s StringZ) Shape() Shape { return s.baseZ }

func (s StringZ) UUID() StringZ {
	s.expectedType = reflect.TypeOf(uuid.UUID{})
	s.typ = UUIDT
	return s
}

func UUID() StringZ { return String().UUID() }

func (s StringZ) Dump(data any) (any, error) {
	if data == nil {
		if s.optional {
			return nil, nil
		}
		return nil, validate.FieldError{Message: "value is required but was nil"}
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			if s.optional {
				return nil, nil
			}
			return nil, validate.FieldError{Message: "value is required but was nil"}
		}
		val = val.Elem()
	}

	// If UUID type is expected, validate the input type
	if s.expectedType != nil && s.expectedType == reflect.TypeOf(uuid.UUID{}) {
		switch val.Kind() {
		case reflect.String:
			// Try to parse as UUID
			if _, err := uuid.Parse(val.String()); err != nil {
				return nil, validate.FieldError{Message: "invalid UUID format: must be a valid UUID string"}
			}
			return val.String(), nil
		case reflect.Array:
			// Check if it's a UUID type
			if val.Type() == reflect.TypeOf(uuid.UUID{}) {
				return val.Interface().(uuid.UUID).String(), nil
			}
			return nil, validate.FieldError{Message: "invalid UUID type: expected UUID or string"}
		default:
			return nil, validate.FieldError{Message: "invalid UUID type: expected UUID or string"}
		}
	}

	switch val.Kind() {
	case reflect.String:
		return val.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil
	default:
		return nil, validate.FieldError{Message: "invalid type: expected string or convertible to string"}
	}
}

func (s StringZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return InvalidDestError("string")
	}

	if destVal.IsNil() {
		return validate.FieldError{Message: "destination pointer is nil"}
	}

	// Handle nil data for optional fields
	if data == nil {
		if s.optional {
			if destVal.Elem().Kind() == reflect.Ptr {
				destVal.Elem().Set(reflect.Zero(destVal.Elem().Type()))
			}
			return nil
		}
		return validate.FieldError{Message: "value is required but was nil"}
	}

	// If UUID type is expected, validate the input type
	if s.expectedType != nil && s.expectedType == reflect.TypeOf(uuid.UUID{}) {
		switch v := data.(type) {
		case string:
			if _, err := uuid.Parse(v); err != nil {
				return validate.FieldError{Message: "invalid UUID format: must be a valid UUID string"}
			}
			data = v
		case uuid.UUID:
			data = v.String()
		default:
			return validate.FieldError{Message: "invalid UUID type: expected UUID or string"}
		}
	}

	data_, ok := data.(string)
	if !ok {
		// Try to convert to string
		val := reflect.ValueOf(data)
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return validate.FieldError{Message: "value is required but was nil"}
			}
			val = val.Elem()
		}
		switch val.Kind() {
		case reflect.String:
			data_ = val.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			data_ = strconv.FormatInt(val.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			data_ = strconv.FormatUint(val.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			data_ = strconv.FormatFloat(val.Float(), 'f', -1, 64)
		case reflect.Bool:
			data_ = strconv.FormatBool(val.Bool())
		default:
			return validate.FieldError{Message: "invalid type: expected string or convertible to string"}
		}
	}

	destVal = destVal.Elem()
	// If the destination is a pointer, we need to allocate it
	if destVal.Kind() == reflect.Ptr {
		if destVal.IsNil() {
			destVal.Set(reflect.New(destVal.Type().Elem()))
		}
		destVal = destVal.Elem()
	}

	destVal.SetString(data_)
	return nil
}

func String() StringZ {
	return StringZ{baseZ: baseZ{typ: StringT}}
}
