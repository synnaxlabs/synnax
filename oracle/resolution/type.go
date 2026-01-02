// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

import "github.com/samber/lo"

// Type is the universal representation of any type in Oracle.
type Type struct {
	Name          string
	Namespace     string
	QualifiedName string
	FilePath      string
	Form          TypeForm
	Domains       map[string]Domain
	AST           any
}

// TypeForm is the interface for all type forms.
type TypeForm interface {
	typeForm()
}

// StructForm represents a struct with fields.
type StructForm struct {
	Fields        []Field
	TypeParams    []TypeParam
	Extends       *TypeRef
	OmittedFields []string
	IsRecursive   bool
	HasKeyDomain  bool
}

func (StructForm) typeForm() {}

func (f StructForm) IsGeneric() bool { return len(f.TypeParams) > 0 }

func (f StructForm) TypeParam(name string) (TypeParam, bool) {
	return lo.Find(f.TypeParams, func(tp TypeParam) bool { return tp.Name == name })
}

func (f StructForm) Field(name string) (Field, bool) {
	return lo.Find(f.Fields, func(fld Field) bool { return fld.Name == name })
}

func (f StructForm) IsFieldOmitted(name string) bool {
	return lo.Contains(f.OmittedFields, name)
}

// EnumForm represents an enumeration.
type EnumForm struct {
	Values    []EnumValue
	IsIntEnum bool
}

func (EnumForm) typeForm() {}

// DistinctForm represents a distinct named type wrapping another type.
type DistinctForm struct {
	Base       TypeRef
	TypeParams []TypeParam
}

func (DistinctForm) typeForm() {}

// AliasForm represents a transparent type alias.
type AliasForm struct {
	Target     TypeRef
	TypeParams []TypeParam
}

func (AliasForm) typeForm() {}

func (f AliasForm) IsGeneric() bool { return len(f.TypeParams) > 0 }

func (f AliasForm) TypeParam(name string) (TypeParam, bool) {
	return lo.Find(f.TypeParams, func(tp TypeParam) bool { return tp.Name == name })
}

// PrimitiveForm represents a built-in primitive type.
type PrimitiveForm struct {
	Name string
}

func (PrimitiveForm) typeForm() {}

// BuiltinGenericForm represents a built-in generic type (Array, Map).
type BuiltinGenericForm struct {
	Name  string
	Arity int
}

func (BuiltinGenericForm) typeForm() {}

// Field represents a field within a struct.
type Field struct {
	Name           string
	Type           TypeRef
	Domains        map[string]Domain
	IsOptional     bool
	IsHardOptional bool
	OmitIfUnset    bool
	AST            any
}

// EnumValue represents a single value in an enumeration.
type EnumValue struct {
	Name    string
	Value   any
	Domains map[string]Domain
}

func (v EnumValue) StringValue() string {
	if s, ok := v.Value.(string); ok {
		return s
	}
	return ""
}

func (v EnumValue) IntValue() int64 {
	if i, ok := v.Value.(int64); ok {
		return i
	}
	return 0
}

// TypeRef is a reference to a type, possibly with type arguments.
type TypeRef struct {
	Name      string
	TypeParam *TypeParam
	TypeArgs  []TypeRef
}

func (r TypeRef) IsTypeParam() bool {
	return r.TypeParam != nil
}

func (r TypeRef) Resolve(table *Table) (Type, bool) {
	return table.Get(r.Name)
}

func (r TypeRef) MustResolve(table *Table) Type {
	return table.MustGet(r.Name)
}

// TypeParam represents a generic type parameter.
type TypeParam struct {
	Name       string
	Constraint *TypeRef
	Default    *TypeRef
	Optional   bool
}

// UnifiedFields returns all fields including inherited ones.
// Child fields override parent fields with the same name, but inherit domains.
func UnifiedFields(typ Type, table *Table) []Field {
	form, ok := typ.Form.(StructForm)
	if !ok {
		return nil
	}
	if form.Extends == nil {
		return form.Fields
	}

	parent, ok := form.Extends.Resolve(table)
	if !ok {
		return form.Fields
	}
	parentForm, ok := parent.Form.(StructForm)
	if !ok {
		return form.Fields
	}

	// Build map of child fields for override detection
	childFieldMap := make(map[string]*Field, len(form.Fields))
	for i := range form.Fields {
		childFieldMap[form.Fields[i].Name] = &form.Fields[i]
	}

	typeArgMap := make(map[string]TypeRef)
	for i, tp := range parentForm.TypeParams {
		if i < len(form.Extends.TypeArgs) {
			typeArgMap[tp.Name] = form.Extends.TypeArgs[i]
		}
	}

	// Build parent field map for domain inheritance
	// IMPORTANT: We must make copies of parent fields before modifying them,
	// otherwise we mutate the original struct's fields which corrupts the table.
	parentFieldsOrig := UnifiedFields(parent, table)
	parentFields := make([]Field, len(parentFieldsOrig))
	for i, pf := range parentFieldsOrig {
		// Copy the field and substitute type refs
		parentFields[i] = pf
		parentFields[i].Type = SubstituteTypeRef(pf.Type, typeArgMap)
	}
	parentFieldMap := make(map[string]*Field, len(parentFields))
	for i := range parentFields {
		parentFieldMap[parentFields[i].Name] = &parentFields[i]
	}

	var result []Field
	for _, pf := range parentFields {
		// Skip if omitted or overridden by child
		if form.IsFieldOmitted(pf.Name) || childFieldMap[pf.Name] != nil {
			continue
		}
		result = append(result, pf)
	}

	// Add child fields, merging domains from parent if overriding
	for _, cf := range form.Fields {
		if pf, isOverride := parentFieldMap[cf.Name]; isOverride {
			// Merge parent domains into child (child takes precedence)
			mergedDomains := make(map[string]Domain, len(pf.Domains)+len(cf.Domains))
			for k, v := range pf.Domains {
				mergedDomains[k] = v
			}
			for k, v := range cf.Domains {
				if existing, ok := mergedDomains[k]; ok {
					mergedDomains[k] = v.Merge(existing)
				} else {
					mergedDomains[k] = v
				}
			}
			cf.Domains = mergedDomains
		}
		result = append(result, cf)
	}
	return result
}

// SubstituteTypeRef replaces type parameters with concrete types.
func SubstituteTypeRef(ref TypeRef, typeArgMap map[string]TypeRef) TypeRef {
	if ref.IsTypeParam() && ref.TypeParam != nil {
		if sub, ok := typeArgMap[ref.TypeParam.Name]; ok {
			return sub
		}
		return ref
	}
	if len(ref.TypeArgs) == 0 {
		return ref
	}
	newArgs := make([]TypeRef, len(ref.TypeArgs))
	for i, arg := range ref.TypeArgs {
		newArgs[i] = SubstituteTypeRef(arg, typeArgMap)
	}
	return TypeRef{Name: ref.Name, TypeArgs: newArgs}
}
