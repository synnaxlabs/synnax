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
	"strconv"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/uuid"
	"github.com/synnaxlabs/x/validate"
)

// StringZ represents a string schema. It provides methods for validating and converting
// string data. StringZ supports validation of regular strings and UUIDs.
type StringZ struct{ baseZ }

// String creates a new string schema. This is the entry point for creating string
// validation schemas.
func String() StringZ {
	s := StringZ{baseZ: baseZ{dataType: StringT, expectedType: reflect.TypeOf("")}}
	s.wrapper = s
	return s
}

var _ Schema = (*StringZ)(nil)

// Optional marks the string field as optional. Optional fields can be nil or omitted.
func (s StringZ) Optional() StringZ { s.optional = true; return s }

// Shape returns the base shape of the string schema.
func (s StringZ) Shape() Shape { return s.baseZ }

// validateDestinationValue validates that the destination is compatible with string
// data
func (s StringZ) validateDestinationValue(dest reflect.Value) error {
	if dest.Kind() != reflect.Pointer || dest.IsNil() {
		return NewInvalidDestinationTypeError(string(s.dataType), dest)
	}
	destType := dest.Type().Elem()
	for destType.Kind() == reflect.Pointer {
		destType = destType.Elem()
	}
	if destType.Kind() == reflect.String || destType.String() == "uuid.UUID" {
		return nil
	}
	if s.expectedType != nil &&
		(destType.AssignableTo(s.expectedType) ||
			s.expectedType.AssignableTo(destType)) {
		return nil
	}
	return NewInvalidDestinationTypeError(string(s.dataType), dest)
}

// UUID marks the string field as a UUID. This enables UUID-specific validation and
// conversion. The field will be validated to ensure it's a valid UUID format.
func (s StringZ) UUID() StringZ {
	s.expectedType = reflect.TypeOf(uuid.UUID{})
	s.dataType = UUIDT
	return s
}

// UUID creates a new UUID schema. This is a convenience function that creates a string
// schema with UUID validation.
func UUID() StringZ { return String().UUID() }

// Dump converts the given data to a string according to the schema. It validates the
// data and returns an error if the data is invalid. For UUID fields, it ensures the
// string is a valid UUID format. For regular string fields, it accepts:
//   - string values
//   - numeric values (converted to string)
//   - boolean values (converted to string)
func (s StringZ) Dump(data any) (any, error) {
	if data == nil {
		if s.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.RequiredError)
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if s.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.RequiredError)
		}
		val = val.Elem()
	}
	if s.expectedType != nil && s.expectedType == reflect.TypeOf(uuid.UUID{}) {
		switch val.Kind() {
		case reflect.String:
			if _, err := uuid.Parse(val.String()); err != nil {
				return nil, errors.Wrap(
					validate.Error,
					"invalid UUID format: must be a valid UUID string",
				)
			}
			return val.String(), nil
		case reflect.Array:
			if val.Type() == reflect.TypeOf(uuid.UUID{}) {
				return val.Interface().(uuid.UUID).String(), nil
			}
			fallthrough
		default:
			return nil, validate.NewInvalidTypeError(
				"UUID or string",
				types.ValueName(val),
			)
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
	}
	return nil, invalidStringTypeError(val)
}

// Parse converts the given data from a string to the destination type. It validates the
// data and returns an error if the data is invalid. For UUID fields, it ensures the
// string is a valid UUID format and can parse into a UUID type. For regular string
// fields, it accepts:
//   - string values
//   - numeric values (converted to string)
//   - boolean values (converted to string)
func (s StringZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := s.validateDestinationValue(destVal); err != nil {
		return err
	}
	if ok, err := validateNilData(destVal, data, s.baseZ); !ok || err != nil {
		return err
	}
	dataVal := reflect.ValueOf(data)
	if s.expectedType != nil && s.expectedType == reflect.TypeOf(uuid.UUID{}) {
		switch v := data.(type) {
		case string:
			if _, err := uuid.Parse(v); err != nil {
				return invalidUUIDStringError()
			}
			data = v
		case uuid.UUID:
			data = v.String()
		default:
			return newInvalidUUIDTypeError(reflect.ValueOf(dataVal))
		}
	}
	data_, ok := data.(string)
	if !ok {
		if dataVal.Kind() == reflect.Pointer {
			if dataVal.IsNil() {
				return errors.WithStack(validate.RequiredError)
			}
			dataVal = dataVal.Elem()
		}
		switch dataVal.Kind() {
		case reflect.String:
			data_ = dataVal.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			data_ = strconv.FormatInt(dataVal.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64:
			data_ = strconv.FormatUint(dataVal.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			data_ = strconv.FormatFloat(dataVal.Float(), 'f', -1, 64)
		case reflect.Bool:
			data_ = strconv.FormatBool(dataVal.Bool())
		default:
			return invalidStringTypeError(dataVal)
		}
	}
	destVal = destVal.Elem()
	// If the destination is a pointer, we need to allocate it
	if destVal.Kind() == reflect.Pointer {
		if destVal.IsNil() {
			destVal.Set(reflect.New(destVal.Type().Elem()))
		}
		destVal = destVal.Elem()
	}
	// If UUID type is expected, handle both string and UUID destinations
	if s.expectedType != nil && s.expectedType == reflect.TypeOf(uuid.UUID{}) {
		parsedUUID, err := uuid.Parse(data_)
		if err != nil {
			return invalidUUIDStringError()
		}
		if destVal.Kind() == reflect.String {
			destVal.SetString(parsedUUID.String())
			return nil
		}
		if destVal.Type() == reflect.TypeOf(uuid.UUID{}) {
			destVal.Set(reflect.ValueOf(parsedUUID))
			return nil
		}
		return NewInvalidDestinationTypeError(s.expectedType.String(), destVal)
	}
	destVal.SetString(data_)
	return nil
}

func invalidUUIDStringError() error {
	return errors.Wrap(
		validate.Error,
		"invalid UUID format: must be a valid UUID string",
	)
}

func newInvalidUUIDTypeError(value reflect.Value) error {
	return validate.NewInvalidTypeError("UUID or string", types.ValueName(value))
}

func invalidStringTypeError(val reflect.Value) error {
	return validate.NewInvalidTypeError(
		"string or convertible to string",
		types.ValueName(val),
	)
}
