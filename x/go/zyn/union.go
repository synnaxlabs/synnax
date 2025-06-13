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
const UnionT Type = "union"

// UnionZ represents a union schema that can accept values of multiple types.
type UnionZ struct {
	baseZ
	schemas []Z
}

var _ Z = (*UnionZ)(nil)

// Optional marks the union field as optional.
// Optional fields can be nil or omitted.
func (u UnionZ) Optional() UnionZ { u.optional = true; return u }

// Shape returns the base shape of the union schema.
func (u UnionZ) Shape() Shape { return u.baseZ }

// Dump converts the given data according to the union schema.
// It tries each schema in sequence until one succeeds.
// Returns an error if no schema can handle the data.
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
	for _, z := range u.schemas {
		if dest, err = z.Dump(data); err == nil {
			break
		}
	}
	return dest, err
}

// Parse converts the given data according to the union schema.
// It tries each schema in sequence until one succeeds.
// Returns an error if no schema can handle the data.
func (u UnionZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := validateDestinationValue(destVal, "union"); err != nil {
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
			if !convertible || (!exactMatch && valueSet) {
				continue
			}
			destElem.Set(v.Elem().Convert(destElem.Type()))
			valueSet = true
			err = nil
			if exactMatch {
				break
			}
		}
		err = pErr
	}
	return err
}

// Union creates a new union schema that can accept values of multiple types.
// This is the entry point for creating union validation schemas.
func Union(schemas ...Z) UnionZ {
	return UnionZ{baseZ: baseZ{typ: UnionT}, schemas: schemas}
}
