// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/parser"
)

// Struct represents a resolved struct definition from an Oracle schema.
// It contains the struct's name, fields, domains, and metadata needed for
// code generation including generic type parameters and inheritance.
type Struct struct {
	AST           parser.IStructDefContext
	Name          string
	Namespace     string
	FilePath      string
	QualifiedName string
	Fields        Fields
	Domains       map[string]Domain
	HasKeyDomain  bool
	TypeParams    []TypeParam
	AliasOf       *TypeRef
	IsRecursive   bool
	Extends       *TypeRef
	OmittedFields []string
}

// Field represents a resolved field within a struct.
type Field struct {
	AST     parser.IFieldDefContext
	Name    string
	TypeRef *TypeRef
	Domains map[string]Domain
}

type Fields []Field

func (f Fields) Find(name string) (Field, bool) {
	return lo.Find(f, func(item Field) bool {
		return item.Name == name
	})
}

func (s *Struct) Field(name string) (Field, bool) {
	return s.Fields.Find(name)

}

func (s *Struct) IsGeneric() bool { return len(s.TypeParams) > 0 }

func (s *Struct) IsAlias() bool { return s.AliasOf != nil }

func (s *Struct) HasExtends() bool { return s.Extends != nil }

func (s *Struct) IsFieldOmitted(name string) bool {
	return lo.Contains(s.OmittedFields, name)
}

func (s *Struct) UnifiedFields() Fields {
	if s.Extends == nil || s.Extends.StructRef == nil {
		return s.Fields
	}
	var (
		parent         = s.Extends.StructRef
		ownFields      = make(map[string]Field)
		parentFieldMap = make(map[string]Field)
		result         Fields
	)
	for _, f := range s.Fields {
		ownFields[f.Name] = f
	}
	for _, pf := range parent.UnifiedFields() {
		parentFieldMap[pf.Name] = pf
		if !s.IsFieldOmitted(pf.Name) && !lo.HasKey(ownFields, pf.Name) {
			result = append(result, s.substituteTypeParams(pf, parent))
		}
	}
	for _, childField := range s.Fields {
		fieldToAdd := s.mergeFieldDomains(childField, parentFieldMap[childField.Name], parent)
		result = append(result, fieldToAdd)
	}
	return result
}

func (s *Struct) mergeFieldDomains(childField, parentField Field, parent *Struct) Field {
	substitutedParent := s.substituteTypeParams(parentField, parent)
	merged := Field{
		AST:     childField.AST,
		Name:    childField.Name,
		TypeRef: childField.TypeRef,
		Domains: make(map[string]Domain),
	}
	for name, domain := range substitutedParent.Domains {
		merged.Domains[name] = domain
	}
	for name, childDomain := range childField.Domains {
		merged.Domains[name] = childDomain.Merge(merged.Domains[name])
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
		return field
	}
	return Field{
		AST:     field.AST,
		Name:    field.Name,
		TypeRef: newTypeRef,
		Domains: field.Domains,
	}
}

// TypeParam returns the type parameter with the given name, or nil if not found.
func (s *Struct) TypeParam(name string) (TypeParam, bool) {
	return lo.Find(s.TypeParams, func(item TypeParam) bool {
		return item.Name == name
	})
}
