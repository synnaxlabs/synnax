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

// ObjectZ represents an object schema.
// It provides methods for validating and converting structured data.
// ObjectZ supports validation of structs and maps with defined field schemas.
type ObjectZ struct {
	baseZ
	fields map[string]Schema
}

var _ Schema = (*ObjectZ)(nil)

// fieldByName finds a field in a struct by its name, supporting both PascalCase and snake_case.
func fieldByName(v reflect.Value, field string) reflect.Value {
	snake := lo.SnakeCase(field)
	pascal := lo.PascalCase(field)
	return v.FieldByNameFunc(func(s string) bool {
		return pascal == s || snake == s
	})
}

// Optional marks the object field as optional.
// Optional fields can be nil or omitted.
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
func (o objectShape) Fields() map[string]Shape {
	return o.fields
}

// Field adds a field to the object schema.
// The field name can be in PascalCase or snake_case.
// The shape parameter defines the validation rules for the field.
func (o ObjectZ) Field(name string, shape Schema) ObjectZ {
	if o.fields == nil {
		o.fields = make(map[string]Schema)
	}
	o.fields[name] = shape
	return o
}

// validateDestination validates that the destination is compatible with object data
func (o ObjectZ) validateDestination(dest reflect.Value) error {
	if dest.Kind() != reflect.Ptr || dest.IsNil() {
		return NewInvalidDestinationTypeError(string(ObjectT), dest)
	}
	destType := dest.Type().Elem()
	for destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}
	if destType.Kind() != reflect.Struct {
		return NewInvalidDestinationTypeError("struct", dest)
	}
	return nil
}

// Dump converts the given data to an object according to the schema.
// It validates the data and returns an error if the data is invalid.
// The function accepts:
//   - struct values
//   - map[string]any values
//
// All fields are validated according to their defined schemas.
// Field names are converted to snake_case in the output.
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
			fieldData, exists := getFieldOnMap(dataMap, fieldName)
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

			snakeCaseName := lo.SnakeCase(fieldName)
			result[snakeCaseName] = fieldData
		}
		return result, nil
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			if o.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.RequiredError)
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, validate.NewInvalidTypeError("struct or map[string]any", types.ValueName(val))
	}

	result := make(map[string]any)
	for fieldName, schema := range o.fields {
		field := fieldByName(val, fieldName)
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

		// Skip nil optional fields
		if fieldData == nil && schema.Shape().Optional() {
			continue
		}

		// Convert field name to snake case for output
		snakeCaseName := lo.SnakeCase(fieldName)
		result[snakeCaseName] = fieldData
	}

	return result, nil
}

// Parse converts the given data from an object to the destination type.
// It validates the data and returns an error if the data is invalid.
// The function expects:
//   - A map[string]any as input
//   - A pointer to a struct as destination
//
// Field names can be in PascalCase or snake_case.
// All fields are validated according to their defined schemas.
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
		return NewInvalidDestinationTypeError("map[string]any", destVal)
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return NewInvalidDestinationTypeError("map[string]any", destVal)
	}

	// Create a map of snake case field names to their original names
	fieldNameMap := make(map[string]string)
	for fieldName := range o.fields {
		fieldNameMap[lo.SnakeCase(fieldName)] = fieldName
	}

	for fieldName, fieldSchema := range o.fields {
		field := fieldByName(destVal, fieldName)
		if !field.IsValid() {
			continue
			//return validate.PathedError(
			//	errors.Wrap(validate.Error, "invalid field"),
			//	fieldName,
			//)
		}

		// Try both original and snake case field names
		fieldData, exists := getFieldOnMap(dataMap, fieldName)
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

// Object creates a new object schema with the given fields.
// This is the entry point for creating object validation schemas.
// The fields parameter maps field names to their validation schemas.
func Object(fields map[string]Schema) ObjectZ {
	o := ObjectZ{
		baseZ:  baseZ{dataType: ObjectT, expectedType: reflect.TypeOf(struct{}{})},
		fields: fields,
	}
	o.wrapper = o
	return o
}

func getFieldOnMap(data map[string]any, field string) (any, bool) {
	v, ok := data[lo.PascalCase(field)]
	if !ok {
		v, ok = data[lo.SnakeCase(field)]
	}
	return v, ok
}
