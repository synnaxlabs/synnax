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

	"github.com/synnaxlabs/x/validate"
)

type BoolZ struct{ baseZ }

func (b BoolZ) Optional() BoolZ { b.optional = true; return b }

func (b BoolZ) Shape() Shape { return b.baseZ }

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

func (b BoolZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := checkDestVal(destVal, string(BoolT)); err != nil {
		return err
	}

	// Handle nil data for optional fields
	if data == nil {
		if b.optional {
			if destVal.Elem().Kind() == reflect.Ptr {
				destVal.Elem().Set(reflect.Zero(destVal.Elem().Type()))
			}
			return nil
		}
		return validate.FieldError{Message: "value is required but was nil"}
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

func Bool() BoolZ {
	return BoolZ{}
}
