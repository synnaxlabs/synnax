// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

// Package resolution provides the resolution table for Oracle plugins.
package resolution

import "github.com/synnaxlabs/oracle/parser"

type Table struct {
	Structs    []*StructEntry
	Enums      []*EnumEntry
	Imports    map[string]bool
	Namespaces map[string]bool
}

func NewTable() *Table {
	return &Table{
		Structs:    make([]*StructEntry, 0),
		Enums:      make([]*EnumEntry, 0),
		Imports:    make(map[string]bool),
		Namespaces: make(map[string]bool),
	}
}

type StructEntry struct {
	AST           parser.IStructDefContext
	Name          string
	Namespace     string
	FilePath      string
	QualifiedName string
	Fields        []*FieldEntry
	Domains       map[string]*DomainEntry
	HasIDDomain   bool
	TypeParams    []*TypeParam // Generic type parameters (e.g., <T, U extends schema>)
	AliasOf       *TypeRef     // If non-nil, this struct is an alias of another type
}

// TypeParam represents a generic type parameter declaration.
// Examples: T, T extends schema, T extends schema = never
type TypeParam struct {
	Name       string   // e.g., "T", "D"
	Constraint *TypeRef // from "extends X", nil = any
	Default    *TypeRef // from "= X", nil = required
}

func (s *StructEntry) Field(name string) *FieldEntry {
	for _, f := range s.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// IsGeneric returns true if this struct has type parameters.
func (s *StructEntry) IsGeneric() bool { return len(s.TypeParams) > 0 }

// IsAlias returns true if this struct is an alias of another type.
func (s *StructEntry) IsAlias() bool { return s.AliasOf != nil }

// TypeParam returns the type parameter with the given name, or nil if not found.
func (s *StructEntry) TypeParam(name string) *TypeParam {
	for _, tp := range s.TypeParams {
		if tp.Name == name {
			return tp
		}
	}
	return nil
}

type FieldEntry struct {
	AST     parser.IFieldDefContext
	Name    string
	TypeRef *TypeRef
	Domains map[string]*DomainEntry
}

type DomainEntry struct {
	AST         parser.IDomainDefContext
	Name        string
	Expressions []*ExpressionEntry
}

type ExpressionEntry struct {
	AST    parser.IExpressionContext
	Name   string
	Values []ExpressionValue
}

type ExpressionValue struct {
	Kind        ValueKind
	StringValue string
	IdentValue  string
	IntValue    int64
	FloatValue  float64
	BoolValue   bool
}

type ValueKind int

const (
	ValueKindString ValueKind = iota
	ValueKindInt
	ValueKindFloat
	ValueKindBool
	ValueKindIdent
)

type TypeRef struct {
	Kind         TypeKind
	Primitive    string
	StructRef    *StructEntry
	EnumRef      *EnumEntry
	TypeParamRef *TypeParam   // If this references a type parameter (e.g., field value T)
	TypeArgs     []*TypeRef   // Type arguments when using a generic (e.g., Status<Foo>)
	IsArray      bool
	IsOptional   bool
	IsNullable   bool
	RawType      string
}

type TypeKind int

const (
	TypeKindPrimitive TypeKind = iota
	TypeKindStruct
	TypeKindEnum
	TypeKindTypeParam // References a type parameter within a generic struct
	TypeKindUnresolved
)

type EnumEntry struct {
	AST           parser.IEnumDefContext
	Name          string
	Namespace     string
	FilePath      string
	QualifiedName string
	Values        []*EnumValue
	ValuesByName  map[string]*EnumValue
	IsIntEnum     bool
}

type EnumValue struct {
	Name        string
	IntValue    int64
	StringValue string
}

var Primitives = map[string]bool{
	"uuid": true, "string": true, "bool": true,
	"int8": true, "int16": true, "int32": true, "int64": true,
	"uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"timestamp": true, "timespan": true, "time_range": true, "time_range_bounded": true,
	"json": true, "bytes": true,
}

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

func (t *Table) LookupStruct(namespace, name string) (*StructEntry, bool) {
	qname := namespace + "." + name
	// First pass: exact qualified name match (takes priority)
	for _, e := range t.Structs {
		if e.QualifiedName == qname {
			return e, true
		}
	}
	// Second pass: name-only match (fallback for unqualified references)
	for _, e := range t.Structs {
		if e.Name == name {
			return e, true
		}
	}
	return nil, false
}

func (t *Table) LookupEnum(namespace, name string) (*EnumEntry, bool) {
	qname := namespace + "." + name
	// First pass: exact qualified name match (takes priority)
	for _, e := range t.Enums {
		if e.QualifiedName == qname {
			return e, true
		}
	}
	// Second pass: name-only match (fallback for unqualified references)
	for _, e := range t.Enums {
		if e.Name == name {
			return e, true
		}
	}
	return nil, false
}

func (t *Table) GetStruct(qname string) (*StructEntry, bool) {
	for _, e := range t.Structs {
		if e.QualifiedName == qname {
			return e, true
		}
	}
	return nil, false
}

func (t *Table) MustGetStruct(qname string) *StructEntry {
	e, ok := t.GetStruct(qname)
	if !ok {
		panic("struct not found: " + qname)
	}
	return e
}

func (t *Table) GetEnum(qname string) (*EnumEntry, bool) {
	for _, e := range t.Enums {
		if e.QualifiedName == qname {
			return e, true
		}
	}
	return nil, false
}

func (t *Table) MustGetEnum(qname string) *EnumEntry {
	e, ok := t.GetEnum(qname)
	if !ok {
		panic("enum not found: " + qname)
	}
	return e
}

func (t *Table) AddStruct(e *StructEntry)    { t.Structs = append(t.Structs, e) }
func (t *Table) AddEnum(e *EnumEntry)        { t.Enums = append(t.Enums, e) }
func (t *Table) MarkImported(path string)    { t.Imports[path] = true }
func (t *Table) IsImported(path string) bool { return t.Imports[path] }

func (t *Table) AllStructs() []*StructEntry { return t.Structs }
func (t *Table) AllEnums() []*EnumEntry     { return t.Enums }

func (t *Table) StructsInNamespace(ns string) []*StructEntry {
	var r []*StructEntry
	for _, e := range t.Structs {
		if e.Namespace == ns {
			r = append(r, e)
		}
	}
	return r
}

func (t *Table) EnumsInNamespace(ns string) []*EnumEntry {
	var r []*EnumEntry
	for _, e := range t.Enums {
		if e.Namespace == ns {
			r = append(r, e)
		}
	}
	return r
}
