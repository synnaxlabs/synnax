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

// UnionT represents a union type in the schema.
const UnionT DataType = "union"

// UnionZ represents a union schema that can accept values of multiple types.
type UnionZ struct {
	baseZ
	schemas []Schema
}

// Union creates a new union schema that can accept values of multiple types. This is
// the entry point for creating union validation schemas.
func Union(schemas ...Schema) UnionZ {
	var v any
	u := UnionZ{
		baseZ:   baseZ{dataType: UnionT, expectedType: reflect.TypeOf(v)},
		schemas: schemas,
	}
	u.wrapper = u
	return u
}

var _ Schema = (*UnionZ)(nil)

// Optional marks the union field as optional. Optional fields can be nil or omitted.
func (u UnionZ) Optional() UnionZ { u.optional = true; return u }

// Shape returns the base shape of the union schema.
func (u UnionZ) Shape() Shape { return u.baseZ }

// validateDestination validates that the destination is compatible with union data
// Union accepts any destination type since it can contain various types
func (u UnionZ) validateDestination(dest reflect.Value) error {
	if dest.Kind() != reflect.Pointer || dest.IsNil() {
		return NewInvalidDestinationTypeError("union", dest)
	}
	return nil
}

// Dump converts the given data according to the union schema. It tries each schema in
// sequence until one succeeds. Returns an error if no schema can handle the data.
func (u UnionZ) Dump(data any) (any, error) {
	if data == nil {
		if u.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.RequiredError)
	}
	var (
		err  error
		dest any
	)
	valueType := reflect.TypeOf(data)
	for _, z := range u.schemas {
		rt := z.Shape().ReflectType()
		if dest, err = z.Dump(data); err == nil {
			exactMatch := valueType == rt
			if exactMatch {
				break
			}
		}
	}
	return dest, err
}

// Parse converts the given data according to the union schema. It tries each schema in
// sequence until one succeeds. Returns an error if no schema can handle the data.
func (u UnionZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := u.validateDestination(destVal); err != nil {
		return err
	}
	if ok, err := validateNilData(destVal, data, u.baseZ); !ok || err != nil {
		return err
	}
	var (
		err      error
		valueSet bool
	)
	for _, z := range u.schemas {
		rt := z.Shape().ReflectType()
		v := reflect.New(rt)
		pErr := z.Parse(data, v.Interface())
		if pErr == nil {
			destElem := reflect.ValueOf(dest).Elem()
			exactMatch := v.Elem().Type() == destElem.Type()
			convertible := v.Elem().Type().ConvertibleTo(destElem.Type())
			err = nil
			if !convertible || (!exactMatch && valueSet) {
				continue
			}
			destElem.Set(v.Elem().Convert(destElem.Type()))
			valueSet = true
			if exactMatch {
				break
			}
		} else if !valueSet {
			err = pErr
		}
	}
	return err
}
