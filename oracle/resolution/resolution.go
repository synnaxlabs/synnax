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
	"github.com/synnaxlabs/oracle/parser"
)

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
	IsRecursive   bool         // True if this struct references itself in any field

	// Extension support
	Extends       *TypeRef // Parent struct reference (nil if no extension)
	OmittedFields []string // Field names omitted from parent via -fieldName syntax
}

// TypeParam represents a generic type parameter declaration.
// Examples: T, T?, T extends Foo, T? extends Foo, T = Bar, T? = Bar
// The Optional flag (from ?) means fields using this type parameter are absent when not provided
type TypeParam struct {
	Name       string   // e.g., "T", "D"
	Optional   bool     // from "?", fields using this type are absent when not provided
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

// HasExtends returns true if this struct extends another struct.
func (s *StructEntry) HasExtends() bool { return s.Extends != nil }

// IsFieldOmitted returns true if the field name is in the omitted fields list.
func (s *StructEntry) IsFieldOmitted(name string) bool {
	for _, f := range s.OmittedFields {
		if f == name {
			return true
		}
	}
	return false
}

// AllFields returns all effective fields including inherited ones from parent.
// Returns fields in order: inherited parent fields (excluding omitted and overridden),
// followed by this struct's own fields (with inherited domains for overrides).
// For generic parents, type parameters are substituted with concrete type arguments.
func (s *StructEntry) AllFields() []*FieldEntry {
	if s.Extends == nil || s.Extends.StructRef == nil {
		return s.Fields
	}

	parent := s.Extends.StructRef

	// Build map of child's own fields
	ownFields := make(map[string]*FieldEntry)
	for _, f := range s.Fields {
		ownFields[f.Name] = f
	}

	// Build map of parent fields for domain inheritance
	parentFieldMap := make(map[string]*FieldEntry)
	parentFields := parent.AllFields()
	for _, pf := range parentFields {
		parentFieldMap[pf.Name] = pf
	}

	var result []*FieldEntry

	// Add parent fields (excluding omitted and overridden)
	for _, pf := range parentFields {
		// Skip if omitted
		if s.IsFieldOmitted(pf.Name) {
			continue
		}
		// Skip if overridden by child's own field
		if _, isOverride := ownFields[pf.Name]; isOverride {
			continue
		}
		// Substitute type parameters if parent is generic
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
func (s *StructEntry) mergeFieldDomains(parentField, childField *FieldEntry, parent *StructEntry) *FieldEntry {
	// First substitute type params in parent field if needed
	substitutedParent := s.substituteTypeParams(parentField, parent)

	merged := &FieldEntry{
		AST:     childField.AST,
		Name:    childField.Name,
		TypeRef: childField.TypeRef, // Child's type wins
		Domains: make(map[string]*DomainEntry),
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
func mergeDomainExpressions(parent, child *DomainEntry) *DomainEntry {
	merged := &DomainEntry{
		AST:  child.AST, // Use child's AST for error reporting
		Name: child.Name,
	}

	// Build map of expressions keyed by name
	exprMap := make(map[string]*ExpressionEntry)
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
func (s *StructEntry) substituteTypeParams(field *FieldEntry, parent *StructEntry) *FieldEntry {
	if len(parent.TypeParams) == 0 || len(s.Extends.TypeArgs) == 0 {
		return field
	}

	// Build map from type param name to concrete type
	typeArgMap := make(map[string]*TypeRef)
	for i, tp := range parent.TypeParams {
		if i < len(s.Extends.TypeArgs) {
			typeArgMap[tp.Name] = s.Extends.TypeArgs[i]
		}
	}

	// If field type is a type parameter, substitute it
	newTypeRef := substituteTypeRef(field.TypeRef, typeArgMap)
	if newTypeRef == field.TypeRef {
		return field // No substitution needed
	}

	// Create a new FieldEntry with substituted type
	return &FieldEntry{
		AST:     field.AST,
		Name:    field.Name,
		TypeRef: newTypeRef,
		Domains: field.Domains,
	}
}

// substituteTypeRef recursively substitutes type parameters in a TypeRef.
func substituteTypeRef(tr *TypeRef, typeArgMap map[string]*TypeRef) *TypeRef {
	if tr == nil {
		return nil
	}

	// If this is a type parameter reference, substitute it
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
			// Recursively substitute type args of the concrete type
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
	AST         antlr.ParserRuleContext // Can be IDomainContext, IInlineDomainContext, or IFileDomainContext
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
	Kind           TypeKind
	Primitive      string
	StructRef      *StructEntry
	EnumRef        *EnumEntry
	TypeParamRef   *TypeParam // If this references a type parameter (e.g., field value T)
	TypeArgs       []*TypeRef // Type arguments when using a generic (e.g., Status<Foo>)
	IsArray        bool
	IsOptional     bool // Soft optional (?) - Go uses zero value + omitempty
	IsHardOptional bool // Hard optional (??) - Go uses pointer + omitempty
	RawType        string
	// Map type fields (used when Kind == TypeKindMap)
	MapKeyType   *TypeRef // Key type for map<K, V>
	MapValueType *TypeRef // Value type for map<K, V>
}

type TypeKind int

const (
	TypeKindPrimitive TypeKind = iota
	TypeKindStruct
	TypeKindEnum
	TypeKindTypeParam  // References a type parameter within a generic struct
	TypeKindMap        // Map type: map<K, V>
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
	Domains       map[string]*DomainEntry
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
