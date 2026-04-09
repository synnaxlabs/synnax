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
	"fmt"
	"reflect"
	"strconv"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ArrayZ represents an array/slice schema. It validates that each element in a slice
// conforms to the item schema.
type ArrayZ struct {
	baseZ
	item Schema
	min  *int
	max  *int
}

var _ Schema = (*ArrayZ)(nil)

// Optional marks the array as optional, allowing nil values.
func (a ArrayZ) Optional() ArrayZ { a.optional = true; return a }

// Min sets the minimum number of elements allowed in the array.
func (a ArrayZ) Min(n int) ArrayZ { a.min = &n; return a }

// Max sets the maximum number of elements allowed in the array.
func (a ArrayZ) Max(n int) ArrayZ { a.max = &n; return a }

// ArrayShape represents the shape of an array schema. It can be obtained via type
// assertion on the Shape interface returned by ArrayZ.Shape().
type ArrayShape struct {
	baseZ
	item Shape
}

// Item returns the shape of the array's element schema.
func (a ArrayShape) Item() Shape { return a.item }

// Shape returns the shape of the array schema.
func (a ArrayZ) Shape() Shape {
	return ArrayShape{baseZ: a.baseZ, item: a.item.Shape()}
}

func (a ArrayZ) validateLength(length int) error {
	if a.min != nil && length < *a.min {
		return errors.Wrapf(
			validate.ErrValidation,
			"array length %d is less than minimum %d",
			length,
			*a.min,
		)
	}
	if a.max != nil && length > *a.max {
		return errors.Wrapf(
			validate.ErrValidation,
			"array length %d is greater than maximum %d",
			length,
			*a.max,
		)
	}
	return nil
}

// Parse converts the given data from a slice to the destination type. It expects
// data to be a []any (e.g. from JSON) and dest to be a pointer to a slice.
func (a ArrayZ) Parse(data, dest any) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Pointer || destVal.IsNil() {
		return NewInvalidDestinationTypeError("slice", destVal)
	}
	if destVal.Elem().Kind() != reflect.Slice {
		return NewInvalidDestinationTypeError("slice", destVal)
	}
	ok, err := validateNilData(destVal, data, a.baseZ)
	if !ok || err != nil {
		return err
	}
	dataSlice, ok := data.([]any)
	if !ok {
		dataVal := reflect.ValueOf(data)
		if dataVal.Kind() != reflect.Slice {
			return NewInvalidDestinationTypeError("[]any", dataVal)
		}
		dataSlice = make([]any, dataVal.Len())
		for i := range dataSlice {
			dataSlice[i] = dataVal.Index(i).Interface()
		}
	}
	if err := a.validateLength(len(dataSlice)); err != nil {
		return err
	}
	elemType := destVal.Elem().Type().Elem()
	result := reflect.MakeSlice(destVal.Elem().Type(), len(dataSlice), len(dataSlice))
	for i, elem := range dataSlice {
		elemPtr := reflect.New(elemType)
		if err := a.item.Parse(elem, elemPtr.Interface()); err != nil {
			return validate.PathedError(err, strconv.Itoa(i))
		}
		result.Index(i).Set(elemPtr.Elem())
	}
	destVal.Elem().Set(result)
	return nil
}

// Dump converts a slice to its standardized []any representation. Each element is
// dumped according to the item schema.
func (a ArrayZ) Dump(data any) (any, error) {
	if data == nil {
		if a.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.ErrRequired)
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if a.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.ErrRequired)
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Slice {
		return nil, validate.NewInvalidTypeError(
			"slice",
			fmt.Sprintf("%v", val.Type()),
		)
	}
	if err := a.validateLength(val.Len()); err != nil {
		return nil, err
	}
	result := make([]any, val.Len())
	for i := range val.Len() {
		dumped, err := a.item.Dump(val.Index(i).Interface())
		if err != nil {
			return nil, validate.PathedError(err, strconv.Itoa(i))
		}
		result[i] = dumped
	}
	return result, nil
}

// Validate checks that the data is a valid array without parsing into a destination.
func (a ArrayZ) Validate(data any) error {
	if data == nil {
		if a.optional {
			return nil
		}
		return errors.WithStack(validate.ErrRequired)
	}
	dataSlice, ok := data.([]any)
	if !ok {
		dataVal := reflect.ValueOf(data)
		if dataVal.Kind() != reflect.Slice {
			return NewInvalidDestinationTypeError("[]any", dataVal)
		}
		dataSlice = make([]any, dataVal.Len())
		for i := range dataSlice {
			dataSlice[i] = dataVal.Index(i).Interface()
		}
	}
	if err := a.validateLength(len(dataSlice)); err != nil {
		return err
	}
	for i, elem := range dataSlice {
		if err := a.item.Validate(elem); err != nil {
			return validate.PathedError(err, strconv.Itoa(i))
		}
	}
	return nil
}

// Array creates a new array schema with the given item schema. Each element of the
// array will be validated according to the item schema.
func Array(item Schema) ArrayZ {
	a := ArrayZ{
		baseZ: baseZ{dataType: ArrayT, expectedType: reflect.TypeFor[[]any]()},
		item:  item,
	}
	a.wrapper = a
	return a
}
