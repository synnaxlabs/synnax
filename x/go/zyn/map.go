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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// MapZ represents a map schema. It validates that each key-value pair in a map conforms
// to the key and value schemas respectively.
type MapZ struct {
	baseZ
	key   Schema
	value Schema
}

var _ Schema = (*MapZ)(nil)

// Optional marks the map as optional, allowing nil values.
func (m MapZ) Optional() MapZ { m.optional = true; return m }

// MapShape represents the shape of a map schema. It can be obtained via type assertion
// on the Shape interface returned by MapZ.Shape().
type MapShape struct {
	baseZ
	key   Shape
	value Shape
}

// Key returns the shape of the map's key schema.
func (ms MapShape) Key() Shape { return ms.key }

// Value returns the shape of the map's value schema.
func (ms MapShape) Value() Shape { return ms.value }

// Shape returns the shape of the map schema.
func (m MapZ) Shape() Shape {
	return MapShape{baseZ: m.baseZ, key: m.key.Shape(), value: m.value.Shape()}
}

// Parse converts the given data from a map to the destination type. It expects data
// to be a map[string]any (e.g. from JSON) and dest to be a pointer to a map.
func (m MapZ) Parse(data, dest any) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Pointer || destVal.IsNil() {
		return NewInvalidDestinationTypeError("map", destVal)
	}
	if destVal.Elem().Kind() != reflect.Map {
		return NewInvalidDestinationTypeError("map", destVal)
	}
	ok, err := validateNilData(destVal, data, m.baseZ)
	if !ok || err != nil {
		return err
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return NewInvalidDestinationTypeError(
			"map[string]any",
			reflect.ValueOf(data),
		)
	}
	destType := destVal.Elem().Type()
	keyType := destType.Key()
	valType := destType.Elem()
	result := reflect.MakeMapWithSize(destType, len(dataMap))
	for k, v := range dataMap {
		parsedKey := reflect.New(keyType)
		if err := m.key.Parse(k, parsedKey.Interface()); err != nil {
			return validate.PathedError(err, k)
		}
		parsedVal := reflect.New(valType)
		if err := m.value.Parse(v, parsedVal.Interface()); err != nil {
			return validate.PathedError(err, k)
		}
		result.SetMapIndex(parsedKey.Elem(), parsedVal.Elem())
	}
	destVal.Elem().Set(result)
	return nil
}

// Dump converts a map to its standardized map[string]any representation. Each key is
// dumped through the key schema and each value through the value schema.
func (m MapZ) Dump(data any) (any, error) {
	if data == nil {
		if m.optional {
			return nil, nil
		}
		return nil, errors.WithStack(validate.ErrRequired)
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			if m.optional {
				return nil, nil
			}
			return nil, errors.WithStack(validate.ErrRequired)
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Map {
		return nil, validate.NewInvalidTypeError(
			"map",
			fmt.Sprintf("%v", val.Type()),
		)
	}
	result := make(map[string]any, val.Len())
	iter := val.MapRange()
	for iter.Next() {
		dumpedKey, err := m.key.Dump(iter.Key().Interface())
		if err != nil {
			return nil, validate.PathedError(
				err,
				fmt.Sprintf("%v", iter.Key().Interface()),
			)
		}
		keyStr, ok := dumpedKey.(string)
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"map key must serialize to string, got %T",
				dumpedKey,
			)
		}
		dumpedVal, err := m.value.Dump(iter.Value().Interface())
		if err != nil {
			return nil, validate.PathedError(
				err,
				fmt.Sprintf("%v", iter.Key().Interface()),
			)
		}
		result[keyStr] = dumpedVal
	}
	return result, nil
}

// Validate checks that the data is a valid map without parsing into a destination.
func (m MapZ) Validate(data any) error {
	if data == nil {
		if m.optional {
			return nil
		}
		return errors.WithStack(validate.ErrRequired)
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return NewInvalidDestinationTypeError(
			"map[string]any",
			reflect.ValueOf(data),
		)
	}
	for k, v := range dataMap {
		if err := m.key.Validate(k); err != nil {
			return validate.PathedError(err, k)
		}
		if err := m.value.Validate(v); err != nil {
			return validate.PathedError(err, k)
		}
	}
	return nil
}

// Map creates a new map schema with the given key and value schemas. Each key-value
// pair in the map will be validated according to the respective schemas.
func Map(key, value Schema) MapZ {
	m := MapZ{
		baseZ: baseZ{
			dataType:     MapT,
			expectedType: reflect.TypeFor[map[string]any](),
		},
		key:   key,
		value: value,
	}
	m.wrapper = m
	return m
}
