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

// Type represents the type of a schema.
// It is used to identify the kind of validation and conversion rules to apply.
type Type string

const (
	// Basic types
	StringT Type = "string" // String type
	BoolT   Type = "bool"   // Boolean type
	NumberT Type = "number" // Generic number type
	ObjectT Type = "object" // Object/struct type
	UUIDT   Type = "uuid"   // UUID type

	// Integer types
	IntT   Type = "int"   // int type
	Int8T  Type = "int8"  // int8 type
	Int16T Type = "int16" // int16 type
	Int32T Type = "int32" // int32 type
	Int64T Type = "int64" // int64 type

	// Unsigned integer types
	UintT   Type = "uint"   // uint type
	Uint8T  Type = "uint8"  // uint8 type
	Uint16T Type = "uint16" // uint16 type
	Uint32T Type = "uint32" // uint32 type
	Uint64T Type = "uint64" // uint64 type

	// Floating-point types
	Float32T Type = "float"  // float32 type
	Float64T Type = "double" // float64 type
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
