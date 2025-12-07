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

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// ObjectZ represents an object schema. It provides methods for validating and
// converting structured data. ObjectZ supports validation of structs and maps with
// defined field schemas.
type ObjectZ struct {
	baseZ
	fields          map[string]Schema
	caseConversions struct {
		snake  map[string]string
		pascal map[string]string
	}
}

var _ Schema = (*ObjectZ)(nil)

// fieldByName finds a field in a struct by its name, supporting both PascalCase and
// snake_case. Uses pre-computed case conversions from the schema.
func (o ObjectZ) fieldByName(v reflect.Value, field string) reflect.Value {
	snake := o.caseConversions.snake[field]
	pascal := o.caseConversions.pascal[field]
	return v.FieldByNameFunc(func(s string) bool { return pascal == s || snake == s })
}

// Optional marks the object field as optional. Optional fields can be nil or omitted.
func (o ObjectZ) Optional() ObjectZ { o.optional = true; return o }

// objectShape represents the shape of an object schema.
type objectShape struct {
	baseZ
	fields map[string]Shape
}

// Shape returns the base shape of the object schema.
func (o ObjectZ) Shape() Shape {
	fields := make(map[string]Shape)
	for k, v := range o.fields {
		fields[k] = v.Shape()
	}
	return objectShape{baseZ: o.baseZ, fields: fields}
}

// Fields returns a map of field names to their shapes.
func (o objectShape) Fields() map[string]Shape { return o.fields }

func (o ObjectZ) initializeFields() ObjectZ {
	if o.fields == nil {
		o.fields = make(map[string]Schema)
	}
	if o.caseConversions.snake == nil || o.caseConversions.pascal == nil {
		o.caseConversions.snake = make(map[string]string)
		o.caseConversions.pascal = make(map[string]string)
		for name := range o.fields {
			o.caseConversions.snake[name] = lo.SnakeCase(name)
			o.caseConversions.pascal[name] = lo.PascalCase(name)
		}
	}
	return o
}

// Field adds a field to the object schema. The field name can be in PascalCase or
// snake_case. The shape parameter defines the validation rules for the field.
func (o ObjectZ) Field(name string, shape Schema) ObjectZ {
	o = o.initializeFields()
	o.fields[name] = shape
	o.caseConversions.snake[name] = lo.SnakeCase(name)
	o.caseConversions.pascal[name] = lo.PascalCase(name)
	return o
}

// validateDestination validates that the destination is compatible with object data
func (o ObjectZ) validateDestination(dest reflect.Value) error {
	if dest.Kind() != reflect.Pointer || dest.IsNil() {
		return NewInvalidDestinationTypeError(string(ObjectT), dest)
	}
	destType := dest.Type().Elem()
	for destType.Kind() == reflect.Pointer {
		destType = destType.Elem()
	}
	if destType.Kind() != reflect.Struct {
		return NewInvalidDestinationTypeError("struct", dest)
	}
	return nil
}

// Dump converts the given data to an object according to the schema. It validates the
// data and returns an error if the data is invalid. The function accepts:
//   - struct values
//   - map[string]any values
//
// All fields are validated according to their defined schemas. Field names are
// converted to snake_case in the output.
func (o ObjectZ) Dump(data any) (any, error) {
	if data == nil {
		if o.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.RequiredError)
	}
	if dataMap, ok := data.(map[string]any); ok {
		result := make(map[string]any)
		for fieldName, schema := range o.fields {
			fieldData, exists := o.getFieldOnMap(dataMap, fieldName)
			if !exists {
				if schema.Shape().Optional() {
					continue
				}
				return nil, validate.PathedError(validate.RequiredError, fieldName)
			}
			fieldData, err := schema.Dump(fieldData)
			if err != nil {
				return nil, validate.PathedError(err, fieldName)
			}
			if fieldData == nil && schema.Shape().Optional() {
				continue
			}
			result[o.caseConversions.snake[fieldName]] = fieldData
		}
		return result, nil
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if o.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.RequiredError)
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, validate.NewInvalidTypeError(
			"struct or map[string]any",
			types.ValueName(val),
		)
	}
	result := make(map[string]any)
	for fieldName, schema := range o.fields {
		field := o.fieldByName(val, fieldName)
		if !field.IsValid() {
			if schema.Shape().Optional() {
				continue
			}
			return nil, validate.PathedError(validate.RequiredError, fieldName)
		}
		fieldData, err := schema.Dump(field.Interface())
		if err != nil {
			return nil, validate.PathedError(err, fieldName)
		}
		if fieldData == nil && schema.Shape().Optional() {
			continue
		}
		result[o.caseConversions.snake[fieldName]] = fieldData
	}
	return result, nil
}

// Parse converts the given data from an object to the destination type. It validates
// the data and returns an error if the data is invalid. The function expects:
//   - A map[string]any as input
//   - A pointer to a struct as destination
//
// Field names can be in PascalCase or snake_case. All fields are validated according to
// their defined schemas.
func (o ObjectZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := o.validateDestination(destVal); err != nil {
		return err
	}
	// Handle nil data for optional fields
	if data == nil {
		if o.optional {
			return nil
		}
		return errors.WithStack(validate.RequiredError)
	}
	destVal = destVal.Elem()
	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() != reflect.Map {
		return NewInvalidDestinationTypeError("map[string]any", dataVal)
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return NewInvalidDestinationTypeError("map[string]any", destVal)
	}
	for fieldName, fieldSchema := range o.fields {
		field := o.fieldByName(destVal, fieldName)
		if !field.IsValid() {
			continue
		}
		fieldData, exists := o.getFieldOnMap(dataMap, fieldName)
		if !exists {
			if fieldSchema.Shape().Optional() {
				continue
			}
			return validate.PathedError(validate.RequiredError, fieldName)
		}
		if err := fieldSchema.Parse(fieldData, field.Addr().Interface()); err != nil {
			return validate.PathedError(err, fieldName)
		}
	}
	return nil
}

// Object creates a new object schema with the given fields. This is the entry point for
// creating object validation schemas. The fields parameter maps field names to their
// validation schemas.
func Object(fields map[string]Schema) ObjectZ {
	o := ObjectZ{
		baseZ:  baseZ{dataType: ObjectT, expectedType: reflect.TypeOf(struct{}{})},
		fields: fields,
	}
	o.wrapper = o
	return o.initializeFields()
}

func (o ObjectZ) getFieldOnMap(data map[string]any, field string) (any, bool) {
	v, ok := data[o.caseConversions.pascal[field]]
	if !ok {
		v, ok = data[o.caseConversions.snake[field]]
	}
	return v, ok
}
