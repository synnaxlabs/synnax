// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package zyn provides a type-safe schema validation and serialization system for Go.
// It allows defining schemas for data structures and provides methods to validate,
// serialize, and deserialize data according to those schemas.
package zyn

import (
	"slices"
)

// Type represents the type of a schema.
// It is used to identify the kind of validation and conversion rules to apply.
type Type string

func (t Type) String() string {
	return string(t)
}

const (
	// StringT represents a string type in the schema.
	StringT Type = "string"
	// BoolT represents a boolean type in the schema.
	BoolT Type = "bool"
	// NumberT represents a generic number type in the schema.
	NumberT Type = "number"
	// ObjectT represents an object/struct type in the schema.
	ObjectT Type = "object"
	// UUIDT represents a UUID type in the schema.
	UUIDT Type = "uuid"

	// IntT represents an int type in the schema.
	IntT Type = "int"
	// Int8T represents an int8 type in the schema.
	Int8T Type = "int8"
	// Int16T represents an int16 type in the schema.
	Int16T Type = "int16"
	// Int32T represents an int32 type in the schema.
	Int32T Type = "int32"
	// Int64T represents an int64 type in the schema.
	Int64T Type = "int64"

	// UintT represents a uint type in the schema.
	UintT Type = "uint"
	// Uint8T represents a uint8 type in the schema.
	Uint8T Type = "uint8"
	// Uint16T represents a uint16 type in the schema.
	Uint16T Type = "uint16"
	// Uint32T represents a uint32 type in the schema.
	Uint32T Type = "uint32"
	// Uint64T represents a uint64 type in the schema.
	Uint64T Type = "uint64"

	// Float32T represents a float32 type in the schema.
	Float32T Type = "float"
	// Float64T represents a float64 type in the schema.
	Float64T Type = "double"
)

var (
	IntegerTypes = []Type{
		IntT,
		Int8T,
		Int16T,
		Int32T,
		Int64T,
		UintT,
		Uint8T,
		Uint16T,
		Uint32T,
		Uint64T,
	}
	IntegerTypeZ       = Enum(IntegerTypes...)
	FloatingPointTypes = []Type{
		Float32T,
		Float64T,
	}
	FloatingPointTypeZ = Enum(FloatingPointTypes...)
	NumericTypes       = slices.Concat(
		[]Type{NumberT},
		IntegerTypes,
		FloatingPointTypes,
	)
	NumericTypeZ   = Enum(NumericTypes...)
	PrimitiveTypes = slices.Concat(
		[]Type{StringT, BoolT, UUIDT},
		IntegerTypes,
	)
	PrimitiveTypeZ = Enum(PrimitiveTypes...)
	Types          = slices.Concat(
		[]Type{ObjectT},
		IntegerTypes,
	)
	TypesZ = Enum(Types...)
)

// Z is a schema that provides methods for validating and converting data.
type Z interface {
	// Parse converts the given data from a standardized format to the destination type.
	// It validates the data and returns an error if the data is invalid.
	Parse(data any, dest any) error
	// Dump converts the given data to a standardized format according to the schema.
	// It validates the data and returns an error if the data is invalid.
	Dump(data any) (any, error)
	// Shape returns the base shape of the schema.
	Shape() Shape
}

// Shape provides information about the structure of a zyn schema.
type Shape interface {
	// Optional is true if the schema can be nil.
	Optional() bool
	// Type returns a string representation of the schema's type.
	Type() Type
	// Fields is only valid for object schemas, and returns a map of the field
	// names to the schemas for each field.
	Fields() map[string]Shape
}

// baseZ provides the base implementation for all schema types.
type baseZ struct {
	optional bool
	typ      Type
}

// Shape returns the base shape of the schema.
func (b baseZ) Shape() Shape { return b }

// Optional returns whether the schema is optional.
func (b baseZ) Optional() bool { return b.optional }

// Type returns the type of the schema.
func (b baseZ) Type() Type { return b.typ }

// Fields returns nil as baseZ is not an object schema.
func (b baseZ) Fields() map[string]Shape { return nil }
