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
	"maps"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// DiscriminatedUnionZ represents a discriminated union schema. It selects the correct
// object schema based on the value of a discriminator field, providing O(1) lookup
// instead of the O(n) sequential matching used by UnionZ.
type DiscriminatedUnionZ struct {
	baseZ
	discriminator       string
	discriminatorSnake  string
	discriminatorPascal string
	discriminatorCamel  string
	variants            map[string]ObjectZ
}

var _ Schema = (*DiscriminatedUnionZ)(nil)

// Optional marks the discriminated union as optional, allowing nil values.
func (d DiscriminatedUnionZ) Optional() DiscriminatedUnionZ {
	d.optional = true
	return d
}

// DiscriminatedUnionShape represents the shape of a discriminated union schema.
type DiscriminatedUnionShape struct {
	baseZ
	discriminator string
	variants      map[string]Shape
}

// Discriminator returns the name of the discriminator field.
func (d DiscriminatedUnionShape) Discriminator() string { return d.discriminator }

// Variants returns a map from discriminator values to their object shapes.
func (d DiscriminatedUnionShape) Variants() map[string]Shape { return d.variants }

// Shape returns the shape of the discriminated union schema.
func (d DiscriminatedUnionZ) Shape() Shape {
	variants := make(map[string]Shape, len(d.variants))
	for k, v := range d.variants {
		variants[k] = v.Shape()
	}
	return DiscriminatedUnionShape{
		baseZ:         d.baseZ,
		discriminator: d.discriminator,
		variants:      variants,
	}
}

func (d DiscriminatedUnionZ) getDiscriminatorFromMap(
	data map[string]any,
) (string, error) {
	if v, ok := data[d.discriminatorPascal]; ok {
		return fmt.Sprintf("%v", v), nil
	}
	if v, ok := data[d.discriminatorCamel]; ok {
		return fmt.Sprintf("%v", v), nil
	}
	if v, ok := data[d.discriminatorSnake]; ok {
		return fmt.Sprintf("%v", v), nil
	}
	if v, ok := data[d.discriminator]; ok {
		return fmt.Sprintf("%v", v), nil
	}
	return "", d.missingDiscriminatorError()
}

func (d DiscriminatedUnionZ) getDiscriminatorFromStruct(
	val reflect.Value,
) (string, error) {
	field := val.FieldByNameFunc(func(s string) bool {
		return s == d.discriminatorPascal ||
			s == d.discriminatorCamel ||
			s == d.discriminatorSnake ||
			s == d.discriminator
	})
	if !field.IsValid() {
		return "", d.missingDiscriminatorError()
	}
	return fmt.Sprintf("%v", field.Interface()), nil
}

func (d DiscriminatedUnionZ) lookupVariant(
	discriminatorValue string,
) (ObjectZ, error) {
	variant, ok := d.variants[discriminatorValue]
	if !ok {
		keys := slices.Collect(maps.Keys(d.variants))
		sort.Strings(keys)
		return ObjectZ{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown discriminator value %q for field %q, expected one of [%s]",
			discriminatorValue,
			d.discriminator,
			strings.Join(keys, ", "),
		)
	}
	return variant, nil
}

func (d DiscriminatedUnionZ) missingDiscriminatorError() error {
	return errors.Wrapf(
		validate.ErrValidation,
		"discriminator field %q not found in data",
		d.discriminator,
	)
}

// Parse reads the discriminator field from the data, selects the matching object
// schema, and delegates parsing to it.
func (d DiscriminatedUnionZ) Parse(data, dest any) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Pointer || destVal.IsNil() {
		return NewInvalidDestinationTypeError("struct", destVal)
	}
	if data == nil {
		if d.optional {
			return nil
		}
		return errors.WithStack(validate.ErrRequired)
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return NewInvalidDestinationTypeError("map[string]any", reflect.ValueOf(data))
	}
	discriminatorValue, err := d.getDiscriminatorFromMap(dataMap)
	if err != nil {
		return err
	}
	variant, err := d.lookupVariant(discriminatorValue)
	if err != nil {
		return err
	}
	return variant.Parse(data, dest)
}

// Dump extracts the discriminator field from the data, selects the matching object
// schema, and delegates dumping to it.
func (d DiscriminatedUnionZ) Dump(data any) (any, error) {
	if data == nil {
		if d.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.ErrRequired)
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if d.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.ErrRequired)
		}
		val = val.Elem()
	}
	if dataMap, ok := data.(map[string]any); ok {
		discriminatorValue, err := d.getDiscriminatorFromMap(dataMap)
		if err != nil {
			return nil, err
		}
		variant, err := d.lookupVariant(discriminatorValue)
		if err != nil {
			return nil, err
		}
		return variant.Dump(data)
	}
	if val.Kind() != reflect.Struct {
		return nil, validate.NewInvalidTypeError(
			"struct or map[string]any",
			fmt.Sprintf("%v", val.Type()),
		)
	}
	discriminatorValue, err := d.getDiscriminatorFromStruct(val)
	if err != nil {
		return nil, err
	}
	variant, err := d.lookupVariant(discriminatorValue)
	if err != nil {
		return nil, err
	}
	return variant.Dump(data)
}

// Validate checks that the data matches one of the discriminated union variants.
func (d DiscriminatedUnionZ) Validate(data any) error {
	if data == nil {
		if d.optional {
			return nil
		}
		return errors.WithStack(validate.ErrRequired)
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return NewInvalidDestinationTypeError("map[string]any", reflect.ValueOf(data))
	}
	discriminatorValue, err := d.getDiscriminatorFromMap(dataMap)
	if err != nil {
		return err
	}
	variant, err := d.lookupVariant(discriminatorValue)
	if err != nil {
		return err
	}
	return variant.Validate(dataMap)
}

// findFieldSchema looks up a field in an ObjectZ by trying the given name directly,
// then its snake_case, camelCase, and PascalCase variants.
func findFieldSchema(obj ObjectZ, name string) (Schema, bool) {
	if s, ok := obj.fields[name]; ok {
		return s, true
	}
	snake := lo.SnakeCase(name)
	if s, ok := obj.fields[snake]; ok {
		return s, true
	}
	camel := lo.CamelCase(name)
	if s, ok := obj.fields[camel]; ok {
		return s, true
	}
	pascal := lo.PascalCase(name)
	if s, ok := obj.fields[pascal]; ok {
		return s, true
	}
	return nil, false
}

// DiscriminatedUnion creates a new discriminated union schema. The discriminator
// parameter names the field used to distinguish between variants. Each ObjectZ schema
// must have a Literal value for the discriminator field.
func DiscriminatedUnion(discriminator string, schemas ...ObjectZ) DiscriminatedUnionZ {
	if len(schemas) < 2 {
		panic("discriminated union requires at least 2 schemas")
	}
	variants := make(map[string]ObjectZ, len(schemas))
	for _, schema := range schemas {
		fieldSchema, ok := findFieldSchema(schema, discriminator)
		if !ok {
			panic(fmt.Sprintf(
				"discriminated union field %q not found in schema",
				discriminator,
			))
		}
		enumZ, ok := fieldSchema.(EnumZ)
		if !ok {
			panic(fmt.Sprintf(
				"discriminated union field %q must be a Literal schema",
				discriminator,
			))
		}
		if len(enumZ.values) != 1 {
			panic(fmt.Sprintf(
				"discriminated union field %q must be a Literal schema (single value), got enum with %d values",
				discriminator,
				len(enumZ.values),
			))
		}
		key := fmt.Sprintf("%v", enumZ.values[0])
		if _, exists := variants[key]; exists {
			panic(fmt.Sprintf(
				"duplicate discriminator value %q",
				key,
			))
		}
		variants[key] = schema
	}
	d := DiscriminatedUnionZ{
		baseZ:               baseZ{dataType: DiscriminatedUnionT},
		discriminator:       discriminator,
		discriminatorSnake:  lo.SnakeCase(discriminator),
		discriminatorPascal: lo.PascalCase(discriminator),
		discriminatorCamel:  lo.CamelCase(discriminator),
		variants:            variants,
	}
	d.wrapper = d
	return d
}
