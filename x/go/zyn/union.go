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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// UnionZ represents a union schema. It validates that data matches at least one of the
// provided schemas, trying each in order and returning the first successful result.
type UnionZ struct {
	baseZ
	schemas []Schema
}

var _ Schema = (*UnionZ)(nil)

// Optional marks the union as optional, allowing nil values.
func (u UnionZ) Optional() UnionZ { u.optional = true; return u }

// UnionShape represents the shape of a union schema. It can be obtained via type
// assertion on the Shape interface returned by UnionZ.Shape().
type UnionShape struct {
	baseZ
	variants []Shape
}

// Variants returns the shapes of each schema in the union.
func (u UnionShape) Variants() []Shape { return u.variants }

// Shape returns the shape of the union schema.
func (u UnionZ) Shape() Shape {
	variants := make([]Shape, len(u.schemas))
	for i, s := range u.schemas {
		variants[i] = s.Shape()
	}
	return UnionShape{baseZ: u.baseZ, variants: variants}
}

// Parse tries each schema in order and returns the first successful parse. The
// destination is only modified on success. If all schemas fail, an error is returned.
func (u UnionZ) Parse(data, dest any) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Pointer || destVal.IsNil() {
		return NewInvalidDestinationTypeError("any", destVal)
	}
	if data == nil {
		if u.optional {
			return nil
		}
		return errors.WithStack(validate.ErrRequired)
	}
	destElemType := destVal.Elem().Type()
	if destElemType.Kind() == reflect.Interface {
		for _, schema := range u.schemas {
			if err := schema.Validate(data); err == nil {
				destVal.Elem().Set(reflect.ValueOf(data))
				return nil
			}
		}
	} else {
		for _, schema := range u.schemas {
			tempDest := reflect.New(destElemType)
			if err := schema.Parse(data, tempDest.Interface()); err == nil {
				destVal.Elem().Set(tempDest.Elem())
				return nil
			}
		}
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"data did not match any of %d union variants",
		len(u.schemas),
	)
}

// Dump tries each schema in order and returns the first successful dump. If all
// schemas fail, an error is returned.
func (u UnionZ) Dump(data any) (any, error) {
	if data == nil {
		if u.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.ErrRequired)
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if u.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.ErrRequired)
		}
		data = val.Elem().Interface()
	}
	for _, schema := range u.schemas {
		if result, err := schema.Dump(data); err == nil {
			return result, nil
		}
	}
	return nil, errors.Wrapf(
		validate.ErrValidation,
		"data did not match any of %d union variants",
		len(u.schemas),
	)
}

// Union creates a new union schema that tries each of the provided schemas in order.
// At least two schemas must be provided.
func Union(schemas ...Schema) UnionZ {
	if len(schemas) < 2 {
		panic("union must have at least 2 schemas")
	}
	u := UnionZ{
		baseZ:   baseZ{dataType: UnionT},
		schemas: schemas,
	}
	u.wrapper = u
	return u
}
