// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

// ValueKind identifies the type of an expression value.
type ValueKind int

const (
	ValueKindString ValueKind = iota
	ValueKindInt
	ValueKindFloat
	ValueKindBool
	ValueKindIdent
)

// TypeRef represents a resolved type reference in a field or type parameter.
type TypeRef struct {
	Kind      TypeKind
	Primitive string
	StructRef *Struct
	EnumRef   *Enum
	// TypeDefRef points to the type definition when Kind is TypeKindTypeDef.
	TypeDefRef *TypeDef
	// TypeParamRef points to the type parameter when Kind is TypeKindTypeParam.
	TypeParamRef *TypeParam
	// TypeArgs holds type arguments when using a generic type (e.g., Status<Foo>).
	TypeArgs []*TypeRef
	IsArray  bool
	// IsOptional indicates soft optional (?) - Go uses zero value + omitempty.
	IsOptional bool
	// IsHardOptional indicates hard optional (??) - Go uses pointer + omitempty.
	IsHardOptional bool
	RawType        string
	// MapKeyType is the key type for map<K, V> (used when Kind == TypeKindMap).
	MapKeyType *TypeRef
	// MapValueType is the value type for map<K, V> (used when Kind == TypeKindMap).
	MapValueType *TypeRef
}

// TypeKind identifies the category of a type reference.
type TypeKind int

const (
	// TypeKindPrimitive represents a built-in primitive type.
	TypeKindPrimitive TypeKind = iota
	// TypeKindStruct represents a struct type reference.
	TypeKindStruct
	// TypeKindEnum represents an enum type reference.
	TypeKindEnum
	// TypeKindTypeParam represents a reference to a type parameter within a generic struct.
	TypeKindTypeParam
	// TypeKindMap represents a map type: map<K, V>.
	TypeKindMap
	// TypeKindTypeDef represents a reference to a top-level type definition.
	TypeKindTypeDef
	// TypeKindUnresolved represents a type that could not be resolved.
	TypeKindUnresolved
)

// Primitives is the set of built-in primitive type names recognized by Oracle.
var Primitives = map[string]bool{
	"uuid": true, "string": true, "bool": true,
	"int8": true, "int16": true, "int32": true, "int64": true,
	"uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"timestamp": true, "timespan": true, "time_range": true, "time_range_bounded": true,
	"json": true, "bytes": true,
}

// IsPrimitive returns true if the name is a built-in primitive type.
func IsPrimitive(name string) bool { return Primitives[name] }

// StringPrimitives identifies primitives that map to string-like types.
var StringPrimitives = map[string]bool{"string": true, "uuid": true}

// NumberPrimitives identifies primitives that map to number types.
var NumberPrimitives = map[string]bool{
	"int8": true, "int16": true, "int32": true, "int64": true,
	"uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
}

// IsStringPrimitive checks if the primitive is a string-like type.
func IsStringPrimitive(name string) bool { return StringPrimitives[name] }

// IsNumberPrimitive checks if the primitive is a number type.
func IsNumberPrimitive(name string) bool { return NumberPrimitives[name] }
