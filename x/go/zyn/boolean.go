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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// BoolZ represents a boolean schema. It provides methods for validating and converting
// boolean data. BoolZ supports conversion from various types to boolean values.
type BoolZ struct{ baseZ }

// Optional marks the boolean field as optional. Optional fields can be nil or omitted.
func (b BoolZ) Optional() BoolZ { b.optional = true; return b }

// Shape returns the base shape of the boolean schema.
func (b BoolZ) Shape() Shape { return b.baseZ }

// validateDestination validates that the destination is compatible with boolean data
func (b BoolZ) validateDestination(dest reflect.Value) error {
	if dest.Kind() != reflect.Pointer || dest.IsNil() {
		return NewInvalidDestinationTypeError(string(BoolT), dest)
	}
	destType := dest.Type().Elem()
	for destType.Kind() == reflect.Pointer {
		destType = destType.Elem()
	}
	if destType.Kind() == reflect.Bool {
		return nil
	}
	if b.expectedType != nil &&
		(destType.AssignableTo(b.expectedType) ||
			b.expectedType.AssignableTo(destType)) {
		return nil
	}
	return NewInvalidDestinationTypeError(string(BoolT), dest)
}

// Dump converts the given data to a boolean according to the schema. It validates the
// data and returns an error if the data is invalid. The function accepts:
//   - boolean values
//   - string values ("true", "false", "1", "0")
//   - numeric values (non-zero is true, zero is false)
func (b BoolZ) Dump(data any) (any, error) {
	if data == nil {
		if b.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.RequiredError)
	}
	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() == reflect.Pointer {
		if dataVal.IsNil() {
			if b.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.RequiredError)
		}
		dataVal = dataVal.Elem()
	}
	var boolVal bool
	switch dataVal.Kind() {
	case reflect.Bool:
		boolVal = dataVal.Bool()
	case reflect.String:
		var err error
		boolVal, err = strconv.ParseBool(dataVal.String())
		if err != nil {
			return nil, invalidBooleanStringError(dataVal.String())
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		boolVal = dataVal.Int() != 0
	case reflect.Float32, reflect.Float64:
		boolVal = dataVal.Float() != 0
	default:
		return nil, invalidBooleanTypeError(dataVal)
	}
	return boolVal, nil
}

// Parse converts the given data from a boolean to the destination type. It validates
// the data and returns an error if the data is invalid. The function accepts:
//   - boolean values
//   - string values ("true", "false", "1", "0")
//   - numeric values (non-zero is true, zero is false)
func (b BoolZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := b.validateDestination(destVal); err != nil {
		return err
	}
	if ok, err := validateNilData(destVal, data, b.baseZ); !ok || err != nil {
		return err
	}
	destVal = destVal.Elem()
	// If the destination is a pointer, we need to allocate it
	if destVal.Kind() == reflect.Pointer {
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
			return invalidBooleanStringError(v)
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		boolVal = reflect.ValueOf(v).Int() != 0
	case float32, float64:
		boolVal = reflect.ValueOf(v).Float() != 0
	default:
		return invalidBooleanTypeError(reflect.ValueOf(v))
	}
	destVal.SetBool(boolVal)
	return nil
}

// Bool creates a new boolean schema. This is the entry point for creating boolean
// validation schemas.
func Bool() BoolZ {
	z := BoolZ{baseZ: baseZ{dataType: BoolT, expectedType: reflect.TypeOf(true)}}
	z.wrapper = z
	return z
}

func invalidBooleanStringError(v string) error {
	return errors.Wrapf(
		validate.Error,
		"invalid boolean string '%s': must be 'true', 'false', '1', or '0'",
		v,
	)
}

func invalidBooleanTypeError(v reflect.Value) error {
	return validate.NewInvalidTypeError(
		"boolean, string, number, or nil",
		types.ValueName(v),
	)
}
