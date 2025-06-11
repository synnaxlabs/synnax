// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package zyn provides a type-safe schema validation and serialization system for Go.
// It allows defining schemas for data structures and provides methods to validate,
// serialize, and deserialize data according to those schemas.
package zyn

import (
	"reflect"
	"strconv"

	"github.com/synnaxlabs/x/validate"
)

// BoolZ represents a boolean schema.
// It provides methods for validating and converting boolean data.
// BoolZ supports conversion from various types to boolean values.
type BoolZ struct{ baseZ }

// Optional marks the boolean field as optional.
// Optional fields can be nil or omitted.
func (b BoolZ) Optional() BoolZ { b.optional = true; return b }

// Shape returns the base shape of the boolean schema.
func (b BoolZ) Shape() Shape { return b.baseZ }

// Dump converts the given data to a boolean according to the schema.
// It validates the data and returns an error if the data is invalid.
// The function accepts:
//   - boolean values
//   - string values ("true", "false", "1", "0")
//   - numeric values (non-zero is true, zero is false)
func (b BoolZ) Dump(data any) (any, error) {
	if data == nil {
		if b.optional {
			return nil, nil
		}
		return nil, validate.FieldError{Message: "value is required but was nil"}
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			if b.optional {
				return nil, nil
			}
			return nil, validate.FieldError{Message: "value is required but was nil"}
		}
		val = val.Elem()
	}

	var boolVal bool
	switch val.Kind() {
	case reflect.Bool:
		boolVal = val.Bool()
	case reflect.String:
		var err error
		boolVal, err = strconv.ParseBool(val.String())
		if err != nil {
			return nil, validate.FieldError{Message: "invalid boolean string: must be 'true', 'false', '1', or '0'"}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		boolVal = val.Int() != 0
	case reflect.Float32, reflect.Float64:
		boolVal = val.Float() != 0
	default:
		return nil, validate.FieldError{Message: "invalid type: expected boolean, string, number, or nil"}
	}

	return boolVal, nil
}

// Parse converts the given data from a boolean to the destination type.
// It validates the data and returns an error if the data is invalid.
// The function accepts:
//   - boolean values
//   - string values ("true", "false", "1", "0")
//   - numeric values (non-zero is true, zero is false)
func (b BoolZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := validateDestinationValue(destVal, string(BoolT)); err != nil {
		return err
	}

	if ok, err := validateNilData(destVal, data, b.baseZ); !ok || err != nil {
		return err
	}

	destVal = destVal.Elem()
	// If the destination is a pointer, we need to allocate it
	if destVal.Kind() == reflect.Ptr {
		if destVal.IsNil() {
			destVal.Set(reflect.New(destVal.Type().Elem()))
		}
		destVal = destVal.Elem()
	}

	var boolVal bool
	switch v := data.(type) {
	case bool:
		boolVal = v
	case string:
		var err error
		boolVal, err = strconv.ParseBool(v)
		if err != nil {
			return validate.FieldError{Message: "invalid boolean string: must be 'true', 'false', '1', or '0'"}
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		boolVal = reflect.ValueOf(v).Int() != 0
	case float32, float64:
		boolVal = reflect.ValueOf(v).Float() != 0
	default:
		return validate.FieldError{Message: "invalid type: expected boolean, string, number, or nil"}
	}

	destVal.SetBool(boolVal)
	return nil
}

// Bool creates a new boolean schema.
// This is the entry point for creating boolean validation schemas.
func Bool() BoolZ {
	return BoolZ{}
}
