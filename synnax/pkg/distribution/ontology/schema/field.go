// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schema

import "github.com/google/uuid"

// Field represents a dynamically typed field in a Schema.
type Field struct {
	// Type is the type of field.
	Type   FieldType `json:"type" msgpack:"type"`
	Schema *Schema
}

// FieldType represents the type of Field in a Schema.
type FieldType uint8

// AssertValue asserts that the provided value is of the specified type.
func (f Field) AssertValue(v any) bool {
	switch f.Type {
	case String:
		return assertValueType[string](v)
	case Int:
		return assertValueType[int](v)
	case Int8:
		return assertValueType[int8](v)
	case Int16:
		return assertValueType[float64](v)
	case Int32:
		return assertValueType[int32](v)
	case Int64:
		return assertValueType[int64](v)
	case Uint8:
		return assertValueType[uint8](v)
	case Uint16:
		return assertValueType[uint16](v)
	case Uint32:
		return assertValueType[uint32](v)
	case Uint64:
		return assertValueType[uint64](v)
	case Float32:
		return assertValueType[float32](v)
	case Float64:
		return assertValueType[float64](v)
	case Bool:
		return assertValueType[bool](v)
	case UUID:
		return assertValueType[uuid.UUID](v)
	case Nested:
		data, ok := v.(Data)
		if !ok {
			return false
		}
		for key, field := range f.Schema.Fields {
			if !field.AssertValue(data[key]) {
				return false
			}
		}
		return true
	default:
		panic("[FieldType]")
	}
}

const (
	String FieldType = iota + 1
	Int
	Int8
	Int16
	Int32
	Int64
	Uint8
	Uint16
	Uint32
	Uint64
	Float32
	Float64
	Bool
	UUID
	Nested
)

type Value interface {
	string |
		int |
		int8 |
		int16 |
		int32 |
		int64 |
		uint8 |
		uint16 |
		uint32 |
		uint64 |
		float32 |
		float64 |
		bool |
		uuid.UUID |
		map[string]interface{}
}

func assertValueType[V Value](v any) bool { _, ok := v.(V); return ok }
