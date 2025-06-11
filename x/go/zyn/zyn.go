// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn

type Type string

const (
	StringT  Type = "string"
	BoolT    Type = "bool"
	NumberT  Type = "number"
	IntT     Type = "int"
	UintT    Type = "uint"
	Float32T Type = "float"
	Float64T Type = "double"
	Uint8T   Type = "uint8"
	Uint16T  Type = "uint16"
	Uint32T  Type = "uint32"
	Uint64T  Type = "uint64"
	Int8T    Type = "int8"
	Int16T   Type = "int16"
	Int32T   Type = "int32"
	Int64T   Type = "int64"
	UUIDT    Type = "uuid"
	ObjectT  Type = "object"
)

type Z interface {
	Parse(data any, dest any) error
	Dump(data any) (any, error)
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

type baseZ struct {
	optional bool
	typ      Type
}

func (b baseZ) Shape() Shape { return b }

func (b baseZ) Optional() bool { return b.optional }

func (b baseZ) Type() Type { return b.typ }

func (b baseZ) Fields() map[string]Shape { return nil }
