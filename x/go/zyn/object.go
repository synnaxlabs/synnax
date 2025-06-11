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
	"github.com/synnaxlabs/x/validate"
)

type ObjectZ struct {
	baseZ
	fields map[string]Z
}

func fieldByName(v reflect.Value, field string) reflect.Value {
	snake := lo.SnakeCase(field)
	pascal := lo.PascalCase(field)
	return v.FieldByNameFunc(func(s string) bool {
		return pascal == s || snake == s
	})
}

func (o ObjectZ) Optional() ObjectZ { o.optional = true; return o }

type objectShape struct {
	baseZ
	fields map[string]Shape
}

func (o ObjectZ) Shape() Shape {
	fields := make(map[string]Shape)
	for k, v := range o.fields {
		fields[k] = v.Shape()
	}
	return objectShape{baseZ: o.baseZ, fields: fields}
}

func (o objectShape) Fields() map[string]Shape {
	return o.fields
}

func (o ObjectZ) Field(name string, shape Z) ObjectZ {
	if o.fields == nil {
		o.fields = make(map[string]Z)
	}
	o.fields[name] = shape
	return o
}

func (o ObjectZ) Dump(data any) (any, error) {
	if data == nil {
		if o.optional {
			return nil, nil
		}
		return nil, validate.FieldError{Message: "value is required but was nil"}
	}

	// Check if data is already a map[string]any
	if dataMap, ok := data.(map[string]any); ok {
		// Validate the map against the schema
		result := make(map[string]any)
		for fieldName, schema := range o.fields {
			// Try both original and snake case field names
			fieldData, exists := dataMap[fieldName]
			if !exists {
				fieldData, exists = dataMap[lo.SnakeCase(fieldName)]
			}

			if !exists {
				if schema.Shape().Optional() {
					continue
				}
				return nil, validate.FieldError{Message: "missing required field: " + fieldName}
			}

			fieldData, err := schema.Dump(fieldData)
			if err != nil {
				return nil, validate.FieldError{Message: "invalid field value for " + fieldName + ": " + err.Error()}
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

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			if o.optional {
				return nil, nil
			}
			return nil, validate.FieldError{Message: "value is required but was nil"}
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, validate.FieldError{Message: "invalid type: expected struct or map[string]any"}
	}

	result := make(map[string]any)
	for fieldName, schema := range o.fields {
		field := fieldByName(val, fieldName)
		if !field.IsValid() {
			if schema.Shape().Optional() {
				continue
			}
			return nil, validate.FieldError{Message: "missing required field: " + fieldName}
		}

		fieldData, err := schema.Dump(field.Interface())
		if err != nil {
			return nil, validate.FieldError{Message: "invalid field value for " + fieldName + ": " + err.Error()}
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

func (o ObjectZ) Parse(data any, dest any) error {
	destVal := reflect.ValueOf(dest)
	if err := checkDestVal(destVal, string(ObjectT)); err != nil {
		return err
	}

	// Handle nil data for optional fields
	if data == nil {
		if o.optional {
			return nil
		}
		return validate.FieldError{Message: "value is required but was nil"}
	}

	destVal = destVal.Elem()
	if destVal.Kind() != reflect.Struct {
		return InvalidDestError("object")
	}

	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() != reflect.Map {
		return validate.FieldError{Message: "invalid type: expected map[string]any"}
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return validate.FieldError{Message: "invalid type: expected map[string]any"}
	}

	// Create a map of snake case field names to their original names
	fieldNameMap := make(map[string]string)
	for fieldName := range o.fields {
		fieldNameMap[lo.SnakeCase(fieldName)] = fieldName
	}

	for fieldName, fieldSchema := range o.fields {
		field := fieldByName(destVal, fieldName)
		if !field.IsValid() {
			return validate.FieldError{
				Field:   fieldName,
				Message: "invalid field: " + fieldName,
			}
		}

		// Try both original and snake case field names
		fieldData, exists := dataMap[fieldName]
		if !exists {
			fieldData, exists = dataMap[lo.SnakeCase(fieldName)]
		}

		if !exists {
			if fieldSchema.Shape().Optional() {
				continue
			}
			return validate.FieldError{Field: fieldName, Message: "missing required field"}
		}

		if err := fieldSchema.Parse(fieldData, field.Addr().Interface()); err != nil {
			return validate.FieldError{Field: fieldName, Message: err.Error()}
		}
	}

	return nil
}

func Object(fields map[string]Z) ObjectZ {
	return ObjectZ{
		baseZ:  baseZ{typ: ObjectT},
		fields: fields,
	}
}
