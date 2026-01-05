// Copyright 2026 Synnax Labs, Inc.
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
	"reflect"
	"slices"
)

// DataType represents the type of a schema. It is used to identify the kind of
// validation and conversion rules to apply.
type DataType string

func (t DataType) String() string { return string(t) }

const (
	// StringT represents a string type in the schema.
	StringT DataType = "string"
	// BoolT represents a boolean type in the schema.
	BoolT DataType = "bool"
	// NumberT represents a generic number type in the schema.
	NumberT DataType = "number"
	// ObjectT represents an object/struct type in the schema.
	ObjectT DataType = "object"
	// UUIDT represents a UUID type in the schema.
	UUIDT DataType = "uuid"
	// IntT represents an int type in the schema.
	IntT DataType = "int"
	// Int8T represents an int8 type in the schema.
	Int8T DataType = "int8"
	// Int16T represents an int16 type in the schema.
	Int16T DataType = "int16"
	// Int32T represents an int32 type in the schema.
	Int32T DataType = "int32"
	// Int64T represents an int64 type in the schema.
	Int64T DataType = "int64"
	// UintT represents a uint type in the schema.
	UintT DataType = "uint"
	// Uint8T represents a uint8 type in the schema.
	Uint8T DataType = "uint8"
	// Uint16T represents a uint16 type in the schema.
	Uint16T DataType = "uint16"
	// Uint32T represents a uint32 type in the schema.
	Uint32T DataType = "uint32"
	// Uint64T represents a uint64 type in the schema.
	Uint64T DataType = "uint64"
	// Float32T represents a float32 type in the schema.
	Float32T DataType = "float32"
	// Float64T represents a float64 type in the schema.
	Float64T DataType = "float64"
)

var (
	StringTypeSchema  = Literal(StringT)
	BoolTypeSchema    = Literal(BoolT)
	NumberTypeSchema  = Literal(NumberT)
	ObjectTypeSchema  = Literal(ObjectT)
	UUIDTypeSchema    = Literal(UUIDT)
	IntTypeSchema     = Literal(IntT)
	Int8TypeSchema    = Literal(Int8T)
	Int16TypeSchema   = Literal(Int16T)
	Int32TypeSchema   = Literal(Int32T)
	Int64TypeSchema   = Literal(Int64T)
	UintTypeSchema    = Literal(UintT)
	Uint8TypeSchema   = Literal(Uint8T)
	Uint16TypeSchema  = Literal(Uint16T)
	Uint32TypeSchema  = Literal(Uint32T)
	Uint64TypeSchema  = Literal(Uint64T)
	Float32TypeSchema = Literal(Float32T)
	Float64TypeSchema = Literal(Float64T)
	IntegerTypes      = []DataType{
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
	IntegerTypeSchema       = Enum(IntegerTypes...)
	FloatingPointTypes      = []DataType{Float32T, Float64T}
	FloatingPointTypeSchema = Enum(FloatingPointTypes...)
	NumericTypes            = slices.Concat(
		[]DataType{NumberT},
		IntegerTypes,
		FloatingPointTypes,
	)
	NumericTypeSchema   = Enum(NumericTypes...)
	PrimitiveTypes      = slices.Concat([]DataType{StringT, BoolT, UUIDT}, NumericTypes)
	PrimitiveTypeSchema = Enum(PrimitiveTypes...)
	DataTypes           = slices.Concat([]DataType{ObjectT}, PrimitiveTypes)
	AnyDataTypeSchema   = Enum(DataTypes...)
)

// Schema is a schema that provides methods for validating and converting data.
type Schema interface {
	// Parse converts the given data from a standardized format to the destination type.
	// It validates the data and returns an error if the data is invalid.
	Parse(data, dest any) error
	// Dump converts the given data to a standardized format according to the schema. It
	// validates the data and returns an error if the data is invalid.
	Dump(data any) (any, error)
	// Shape returns the base shape of the schema.
	Shape() Shape
	Validate(data any) error
}

// Shape provides information about the structure of a zyn schema.
type Shape interface {
	// Optional is true if the schema can be nil.
	Optional() bool
	// DataType returns a string representation of the schema's type.
	DataType() DataType
	// Fields is only valid for object schemas, and returns a map of the field names to
	// the schemas for each field.
	Fields() map[string]Shape
	ReflectType() reflect.Type
}

// baseZ provides the base implementation for all schema types.
type baseZ struct {
	optional     bool
	dataType     DataType
	expectedType reflect.Type
	wrapper      Schema
}

// Shape returns the base shape of the schema.
func (b baseZ) Shape() Shape { return b }

// Optional returns whether the schema is optional.
func (b baseZ) Optional() bool { return b.optional }

// DataType returns the type of the schema.
func (b baseZ) DataType() DataType { return b.dataType }

// Fields returns nil as baseZ is not an object schema.
func (b baseZ) Fields() map[string]Shape { return nil }

func (b baseZ) ReflectType() reflect.Type { return b.expectedType }

func (b baseZ) Validate(data any) error {
	if b.expectedType != nil {
		dest := reflect.New(b.expectedType).Interface()
		return b.wrapper.Parse(data, dest)
	}
	var dest any
	return b.wrapper.Parse(data, &dest)
}
