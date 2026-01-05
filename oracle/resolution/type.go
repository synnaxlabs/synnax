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

type Type struct {
	Name          string
	Namespace     string
	QualifiedName string
	FilePath      string
	Form          TypeForm
	Domains       map[string]Domain
	AST           any
}

type TypeForm interface {
	typeForm()
}

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

type EnumForm struct {
	Values    []EnumValue
	IsIntEnum bool
}

func (EnumForm) typeForm() {}

type DistinctForm struct {
	Base       TypeRef
	TypeParams []TypeParam
}

func (DistinctForm) typeForm() {}

type AliasForm struct {
	Target     TypeRef
	TypeParams []TypeParam
}

func (AliasForm) typeForm() {}

func (f AliasForm) IsGeneric() bool { return len(f.TypeParams) > 0 }

func (f AliasForm) TypeParam(name string) (TypeParam, bool) {
	return lo.Find(f.TypeParams, func(tp TypeParam) bool { return tp.Name == name })
}

type PrimitiveForm struct {
	Name string
}

func (PrimitiveForm) typeForm() {}

type BuiltinGenericForm struct {
	Name  string
	Arity int
}

func (BuiltinGenericForm) typeForm() {}

type Field struct {
	Name           string
	Type           TypeRef
	Domains        map[string]Domain
	IsOptional     bool
	IsHardOptional bool
	OmitIfUnset    bool
	AST            any
}

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

type TypeParam struct {
	Name       string
	Constraint *TypeRef
	Default    *TypeRef
	Optional   bool
}

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

	// Copy parent fields before modifying to avoid mutating the table.
	parentFieldsOrig := UnifiedFields(parent, table)
	parentFields := make([]Field, len(parentFieldsOrig))
	for i, pf := range parentFieldsOrig {
		parentFields[i] = pf
		parentFields[i].Type = SubstituteTypeRef(pf.Type, typeArgMap)
	}
	parentFieldMap := make(map[string]*Field, len(parentFields))
	for i := range parentFields {
		parentFieldMap[parentFields[i].Name] = &parentFields[i]
	}

	var result []Field
	for _, pf := range parentFields {
		if form.IsFieldOmitted(pf.Name) || childFieldMap[pf.Name] != nil {
			continue
		}
		result = append(result, pf)
	}

	for _, cf := range form.Fields {
		if pf, isOverride := parentFieldMap[cf.Name]; isOverride {
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
