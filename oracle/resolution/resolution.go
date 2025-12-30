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
	Structs    map[string]*StructEntry
	Enums      map[string]*EnumEntry
	Imports    map[string]bool
	Namespaces map[string]bool
}

func NewTable() *Table {
	return &Table{
		Structs:    make(map[string]*StructEntry),
		Enums:      make(map[string]*EnumEntry),
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
}

func (s *StructEntry) Field(name string) *FieldEntry {
	for _, f := range s.Fields {
		if f.Name == name {
			return f
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
	Kind       TypeKind
	Primitive  string
	StructRef  *StructEntry
	EnumRef    *EnumEntry
	IsArray    bool
	IsOptional bool
	IsNullable bool
	RawType    string
}

type TypeKind int

const (
	TypeKindPrimitive TypeKind = iota
	TypeKindStruct
	TypeKindEnum
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
	if e, ok := t.Structs[namespace+"."+name]; ok {
		return e, true
	}
	if e, ok := t.Structs[name]; ok {
		return e, true
	}
	return nil, false
}

func (t *Table) LookupEnum(namespace, name string) (*EnumEntry, bool) {
	if e, ok := t.Enums[namespace+"."+name]; ok {
		return e, true
	}
	if e, ok := t.Enums[name]; ok {
		return e, true
	}
	return nil, false
}

func (t *Table) AddStruct(e *StructEntry)    { t.Structs[e.QualifiedName] = e }
func (t *Table) AddEnum(e *EnumEntry)        { t.Enums[e.QualifiedName] = e }
func (t *Table) MarkImported(path string)    { t.Imports[path] = true }
func (t *Table) IsImported(path string) bool { return t.Imports[path] }

func (t *Table) AllStructs() []*StructEntry {
	r := make([]*StructEntry, 0, len(t.Structs))
	for _, e := range t.Structs {
		r = append(r, e)
	}
	return r
}

func (t *Table) AllEnums() []*EnumEntry {
	r := make([]*EnumEntry, 0, len(t.Enums))
	for _, e := range t.Enums {
		r = append(r, e)
	}
	return r
}

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
