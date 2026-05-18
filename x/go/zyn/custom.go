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

// CustomT represents a custom-parsed type in the schema.
const CustomT DataType = "custom"

// CustomZ is a schema for typed values that are produced by a caller-supplied parser.
// Use it when a wire-format primitive (typically a string or number) needs to be
// converted into a domain type whose parsing rules don't fit the built-in zyn schemas,
// e.g. a hex color string into a typed Color. Dump is the inverse, used on the way
// out; when no Dump function is provided, the value is returned as-is.
type CustomZ[T any] struct {
	baseZ
	parse func(any) (T, error)
	dump  func(T) (any, error)
}

// Custom creates a new custom schema parameterised by the destination type T. The
// parse function converts wire data into a T; it should return a clear error for any
// unsupported input shape so the surrounding zyn error path can wrap it.
func Custom[T any](parse func(any) (T, error)) CustomZ[T] {
	c := CustomZ[T]{
		baseZ: baseZ{dataType: CustomT, expectedType: reflect.TypeFor[T]()},
		parse: parse,
	}
	c.wrapper = c
	return c
}

// WithDump attaches a dump function that converts a T back into wire data. When
// absent, Dump returns the T as-is — appropriate when the wire representation is
// identical to T (e.g. a typed enum that marshals as its underlying string).
func (c CustomZ[T]) WithDump(dump func(T) (any, error)) CustomZ[T] {
	c.dump = dump
	return c
}

// Optional marks the field as optional. Optional fields can be nil or omitted; when
// absent, the destination is left at its zero value.
func (c CustomZ[T]) Optional() CustomZ[T] { c.optional = true; return c }

// Shape returns the base shape of the schema.
func (c CustomZ[T]) Shape() Shape { return c.baseZ }

// Validate runs Parse against a discardable destination.
func (c CustomZ[T]) Validate(data any) error {
	var dest T
	return c.Parse(data, &dest)
}

// Parse invokes the caller-supplied parser and writes the result into dest.
func (c CustomZ[T]) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Pointer || destVal.IsNil() {
		return NewInvalidDestinationTypeError(string(c.dataType), destVal)
	}
	if ok, err := validateNilData(destVal, data, c.baseZ); !ok || err != nil {
		return err
	}
	parsed, err := c.parse(data)
	if err != nil {
		return err
	}
	destElem := destVal.Elem()
	if destElem.Kind() == reflect.Pointer {
		if destElem.IsNil() {
			destElem.Set(reflect.New(destElem.Type().Elem()))
		}
		destElem = destElem.Elem()
	}
	parsedVal := reflect.ValueOf(parsed)
	if !parsedVal.Type().AssignableTo(destElem.Type()) {
		if !parsedVal.Type().ConvertibleTo(destElem.Type()) {
			return NewInvalidDestinationTypeError(c.expectedType.String(), destVal)
		}
		parsedVal = parsedVal.Convert(destElem.Type())
	}
	destElem.Set(parsedVal)
	return nil
}

// Dump converts the typed value back to its wire representation. When no dump
// function is supplied, the value is returned unchanged.
func (c CustomZ[T]) Dump(data any) (any, error) {
	if data == nil {
		if c.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.ErrRequired)
	}
	if c.dump == nil {
		return data, nil
	}
	t, ok := data.(T)
	if !ok {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"expected %s, got %T", c.expectedType, data,
		)
	}
	return c.dump(t)
}
