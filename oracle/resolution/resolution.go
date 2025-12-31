// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package resolution provides the resolution table for Oracle plugins.
package resolution

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/parser"
)

// Table holds all resolved types from parsed Oracle schema files.
// It serves as the central registry for struct and enum definitions
// that plugins use for code generation.
type Table struct {
	Structs    []Struct
	Enums      []Enum
	Imports    map[string]bool
	Namespaces map[string]bool
}

// NewTable creates an empty resolution table.
func NewTable() *Table {
	return &Table{Imports: make(map[string]bool), Namespaces: make(map[string]bool)}
}

// Struct represents a resolved struct definition from an Oracle schema.
// It contains the struct's name, fields, domains, and metadata needed for
// code generation including generic type parameters and inheritance.
type Struct struct {
	AST           parser.IStructDefContext
	Name          string
	Namespace     string
	FilePath      string
	QualifiedName string
	Fields        []Field
	Domains       map[string]Domain
	HasKeyDomain  bool
	TypeParams    []TypeParam
	AliasOf       *TypeRef
	IsRecursive   bool
	Extends       *TypeRef
	OmittedFields []string
}

type TypeParam struct {
	Name       string
	Optional   bool
	Constraint *TypeRef
	Default    *TypeRef
}

func (s *Struct) Field(name string) (Field, bool) {
	return lo.Find(s.Fields, func(item Field) bool {
		return item.Name == name
	})
}

func (s *Struct) IsGeneric() bool { return len(s.TypeParams) > 0 }

func (s *Struct) IsAlias() bool { return s.AliasOf != nil }

func (s *Struct) HasExtends() bool { return s.Extends != nil }

func (s *Struct) IsFieldOmitted(name string) bool {
	for _, f := range s.OmittedFields {
		if f == name {
			return true
		}
	}
	return false
}

func (s *Struct) UnifiedFields() []Field {
	if s.Extends == nil || s.Extends.StructRef == nil {
		return s.Fields
	}
	var (
		parent         = s.Extends.StructRef
		ownFields      = make(map[string]Field)
		parentFieldMap = make(map[string]Field)
		parentFields   = parent.UnifiedFields()
		result         []Field
	)
	for _, f := range s.Fields {
		ownFields[f.Name] = f
	}
	for _, pf := range parentFields {
		parentFieldMap[pf.Name] = pf
	}
	for _, pf := range parentFields {
		if s.IsFieldOmitted(pf.Name) {
			continue
		}
		if _, isOverride := ownFields[pf.Name]; isOverride {
			continue
		}
		result = append(result, s.substituteTypeParams(pf, parent))
	}

	// Add child's own fields, merging parent domains for overrides
	for _, cf := range s.Fields {
		if pf, isOverride := parentFieldMap[cf.Name]; isOverride {
			// Merge: child's type with inherited parent domains
			merged := s.mergeFieldDomains(pf, cf, parent)
			result = append(result, merged)
		} else {
			result = append(result, cf)
		}
	}

	return result
}

// mergeFieldDomains creates a new field with child's type but merged domains.
// Parent domains are inherited. When both have the same domain, expressions are merged
// (child expressions override parent expressions with the same name).
func (s *Struct) mergeFieldDomains(parentField, childField Field, parent *Struct) Field {
	// First substitute type params in parent field if needed
	substitutedParent := s.substituteTypeParams(parentField, parent)
	merged := Field{
		AST:     childField.AST,
		Name:    childField.Name,
		TypeRef: childField.TypeRef, // Child's type wins
		Domains: make(map[string]*Domain),
	}

	// Copy parent domains first
	for name, domain := range substitutedParent.Domains {
		merged.Domains[name] = domain
	}

	// Merge child domains - if same domain exists, merge expressions
	for name, childDomain := range childField.Domains {

		if parentDomain, exists := merged.Domains[name]; exists {
			// Merge expressions: parent first, child overlays
			merged.Domains[name] = mergeDomainExpressions(parentDomain, childDomain)
		} else {
			merged.Domains[name] = childDomain
		}
	}

	return merged
}

// mergeDomainExpressions merges two domains' expressions.
// Parent expressions are kept, child expressions override on name conflict.
func mergeDomainExpressions(parent, child *Domain) *Domain {
	merged := &Domain{
		AST:  child.AST, // Use child's AST for error reporting
		Name: child.Name,
	}

	// Build map of expressions keyed by name
	exprMap := make(map[string]*Expression)
	for _, expr := range parent.Expressions {
		exprMap[expr.Name] = expr
	}
	for _, expr := range child.Expressions {
		exprMap[expr.Name] = expr // Child wins on conflict
	}

	// Convert back to slice, preserving rough order (parent first, then new child)
	for _, expr := range parent.Expressions {
		if e, ok := exprMap[expr.Name]; ok {
			merged.Expressions = append(merged.Expressions, e)
			delete(exprMap, expr.Name)
		}
	}
	// Add any remaining child-only expressions
	for _, expr := range child.Expressions {
		if e, ok := exprMap[expr.Name]; ok {
			merged.Expressions = append(merged.Expressions, e)
			delete(exprMap, expr.Name)
		}
	}

	return merged
}

// substituteTypeParams creates a copy of the field with type parameters replaced
// by the concrete type arguments from the Extends reference.
func (s *Struct) substituteTypeParams(field Field, parent *Struct) Field {
	if len(parent.TypeParams) == 0 || len(s.Extends.TypeArgs) == 0 {
		return field
	}

	typeArgMap := make(map[string]*TypeRef)
	for i, tp := range parent.TypeParams {
		if i < len(s.Extends.TypeArgs) {
			typeArgMap[tp.Name] = s.Extends.TypeArgs[i]
		}
	}

	newTypeRef := substituteTypeRef(field.TypeRef, typeArgMap)
	if newTypeRef == field.TypeRef {
		return field // No substitution needed
	}

	return Field{
		AST:     field.AST,
		Name:    field.Name,
		TypeRef: newTypeRef,
		Domains: field.Domains,
	}
}

func substituteTypeRef(tr *TypeRef, typeArgMap map[string]*TypeRef) *TypeRef {
	if tr == nil {
		return nil
	}

	if tr.Kind == TypeKindTypeParam && tr.TypeParamRef != nil {
		if concrete, ok := typeArgMap[tr.TypeParamRef.Name]; ok {
			// Create a copy with the same modifiers but substituted type
			result := &TypeRef{
				Kind:           concrete.Kind,
				Primitive:      concrete.Primitive,
				StructRef:      concrete.StructRef,
				EnumRef:        concrete.EnumRef,
				TypeParamRef:   concrete.TypeParamRef,
				IsArray:        tr.IsArray || concrete.IsArray,
				IsOptional:     tr.IsOptional || concrete.IsOptional,
				IsHardOptional: tr.IsHardOptional || concrete.IsHardOptional,
				RawType:        concrete.RawType,
				MapKeyType:     concrete.MapKeyType,
				MapValueType:   concrete.MapValueType,
			}
			if len(concrete.TypeArgs) > 0 {
				result.TypeArgs = make([]*TypeRef, len(concrete.TypeArgs))
				for i, arg := range concrete.TypeArgs {
					result.TypeArgs[i] = substituteTypeRef(arg, typeArgMap)
				}
			}
			return result
		}
	}

	// Check if we need to substitute type arguments
	needsSubstitution := false
	for _, arg := range tr.TypeArgs {
		if arg.Kind == TypeKindTypeParam {
			needsSubstitution = true
			break
		}
	}

	// Also check map types
	if tr.Kind == TypeKindMap {
		if tr.MapKeyType != nil && tr.MapKeyType.Kind == TypeKindTypeParam {
			needsSubstitution = true
		}
		if tr.MapValueType != nil && tr.MapValueType.Kind == TypeKindTypeParam {
			needsSubstitution = true
		}
	}

	if !needsSubstitution && len(tr.TypeArgs) == 0 {
		return tr
	}

	// Create a copy with substituted type args
	result := &TypeRef{
		Kind:           tr.Kind,
		Primitive:      tr.Primitive,
		StructRef:      tr.StructRef,
		EnumRef:        tr.EnumRef,
		TypeParamRef:   tr.TypeParamRef,
		IsArray:        tr.IsArray,
		IsOptional:     tr.IsOptional,
		IsHardOptional: tr.IsHardOptional,
		RawType:        tr.RawType,
	}

	if len(tr.TypeArgs) > 0 {
		result.TypeArgs = make([]*TypeRef, len(tr.TypeArgs))
		for i, arg := range tr.TypeArgs {
			result.TypeArgs[i] = substituteTypeRef(arg, typeArgMap)
		}
	}

	if tr.Kind == TypeKindMap {
		result.MapKeyType = substituteTypeRef(tr.MapKeyType, typeArgMap)
		result.MapValueType = substituteTypeRef(tr.MapValueType, typeArgMap)
	}

	return result
}

// TypeParam returns the type parameter with the given name, or nil if not found.
func (s *Struct) TypeParam(name string) TypeParam {
	for _, tp := range s.TypeParams {
		if tp.Name == name {
			return tp
		}
	}
	return nil
}

// Field represents a resolved field within a struct.
type Field struct {
	AST     parser.IFieldDefContext
	Name    string
	TypeRef *TypeRef
	Domains map[string]*Domain
}

// Domain represents a domain annotation on a struct or field.
// The AST can be IDomainContext, IInlineDomainContext, or IFileDomainContext.
type Domain struct {
	AST         antlr.ParserRuleContext
	Name        string
	Expressions []*Expression
}

// Expression represents a single expression within a domain.
type Expression struct {
	AST    parser.IExpressionContext
	Name   string
	Values []ExpressionValue
}

// ExpressionValue holds a parsed value from a domain expression.
type ExpressionValue struct {
	Kind        ValueKind
	StringValue string
	IdentValue  string
	IntValue    int64
	FloatValue  float64
	BoolValue   bool
}

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
	// TypeKindUnresolved represents a type that could not be resolved.
	TypeKindUnresolved
)

// Enum represents a resolved enum definition from an Oracle schema.
type Enum struct {
	AST           parser.IEnumDefContext
	Name          string
	Namespace     string
	FilePath      string
	QualifiedName string
	Values        []*EnumValue
	ValuesByName  map[string]*EnumValue
	IsIntEnum     bool
	Domains       map[string]*Domain
}

// EnumValue represents a single value within an enum.
type EnumValue struct {
	Name        string
	IntValue    int64
	StringValue string
}

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

// LookupStruct finds a struct by namespace and name.
// It first tries an exact qualified name match, then falls back to name-only.
func (t *Table) LookupStruct(namespace, name string) (*Struct, bool) {
	qname := namespace + "." + name
	for _, e := range t.Structs {
		if e.QualifiedName == qname {
			return e, true
		}
	}
	for _, e := range t.Structs {
		if e.Name == name {
			return e, true
		}
	}
	return nil, false
}

// LookupEnum finds an enum by namespace and name.
// It first tries an exact qualified name match, then falls back to name-only.
func (t *Table) LookupEnum(namespace, name string) (*Enum, bool) {
	qname := namespace + "." + name
	for _, e := range t.Enums {
		if e.QualifiedName == qname {
			return e, true
		}
	}
	for _, e := range t.Enums {
		if e.Name == name {
			return e, true
		}
	}
	return nil, false
}

// GetStruct returns the struct with the given qualified name.
func (t *Table) GetStruct(qname string) (*Struct, bool) {
	for _, e := range t.Structs {
		if e.QualifiedName == qname {
			return e, true
		}
	}
	return nil, false
}

// MustGetStruct returns the struct with the given qualified name or panics.
func (t *Table) MustGetStruct(qname string) *Struct {
	e, ok := t.GetStruct(qname)
	if !ok {
		panic("struct not found: " + qname)
	}
	return e
}

// GetEnum returns the enum with the given qualified name.
func (t *Table) GetEnum(qname string) (*Enum, bool) {
	for _, e := range t.Enums {
		if e.QualifiedName == qname {
			return e, true
		}
	}
	return nil, false
}

// MustGetEnum returns the enum with the given qualified name or panics.
func (t *Table) MustGetEnum(qname string) *Enum {
	e, ok := t.GetEnum(qname)
	if !ok {
		panic("enum not found: " + qname)
	}
	return e
}

// AddStruct adds a struct entry to the table.
func (t *Table) AddStruct(e *Struct) { t.Structs = append(t.Structs, e) }

// AddEnum adds an enum entry to the table.
func (t *Table) AddEnum(e *Enum) { t.Enums = append(t.Enums, e) }

// MarkImported records that a file path has been imported.
func (t *Table) MarkImported(path string) { t.Imports[path] = true }

// IsImported returns true if the file path has been imported.
func (t *Table) IsImported(path string) bool { return t.Imports[path] }

// AllStructs returns all struct entries in the table.
func (t *Table) AllStructs() []*Struct { return t.Structs }

// AllEnums returns all enum entries in the table.
func (t *Table) AllEnums() []*Enum { return t.Enums }

// StructsInNamespace returns all structs in the given namespace.
func (t *Table) StructsInNamespace(ns string) []*Struct {
	var r []*Struct
	for _, e := range t.Structs {
		if e.Namespace == ns {
			r = append(r, e)
		}
	}
	return r
}

// EnumsInNamespace returns all enums in the given namespace.
func (t *Table) EnumsInNamespace(ns string) []*Enum {
	var r []*Enum
	for _, e := range t.Enums {
		if e.Namespace == ns {
			r = append(r, e)
		}
	}
	return r
}
