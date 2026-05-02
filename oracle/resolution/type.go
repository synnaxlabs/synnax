// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/set"
)

type Type struct {
	Domains       map[string]Domain
	Form          TypeForm
	AST           any
	Name          string
	Namespace     string
	QualifiedName string
	FilePath      string
}

type TypeForm interface {
	typeForm()
}

type StructForm struct {
	Fields        []Field
	Actions       []Action
	TypeParams    []TypeParam
	Extends       []TypeRef
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

// Action is a named typed mutation declared inside a struct body. The struct's
// generated codec includes a discriminated-union envelope that dispatches by
// Name; per-action handlers live in hand-written sibling files.
type Action struct {
	AST     any
	Domains map[string]Domain
	Name    string
	Fields  []Field
}

type Field struct {
	AST            any
	Domains        map[string]Domain
	Name           string
	Type           TypeRef
	IsOptional     bool
	IsHardOptional bool
	OmitIfUnset    bool
}

type EnumValue struct {
	Domains map[string]Domain
	Value   any
	Name    string
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
	TypeParam *TypeParam
	ArraySize *int64
	Name      string
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

// RefersTo reports whether ref directly or transitively references a type
// identified by targetQualifiedName. The search follows type arguments, struct
// field types, alias targets, and distinct bases, so mutual-recursion cycles
// (A → B → A) are detected, not just direct self-reference (A → A). A visited
// set prevents infinite loops on non-target cycles.
//
// This is the shared primitive behind the analyzer's IsRecursive detection
// and the C++ plugin's decision between std::optional<T> and
// x::mem::indirect<T> for hard-optional fields.
func RefersTo(ref TypeRef, targetQualifiedName string, table *Table) bool {
	return refersTo(ref, targetQualifiedName, table, set.New[string]())
}

func refersTo(ref TypeRef, targetQN string, table *Table, visited set.Set[string]) bool {
	if ref.Name == targetQN {
		return true
	}
	for _, arg := range ref.TypeArgs {
		if refersTo(arg, targetQN, table, visited) {
			return true
		}
	}
	if visited.Contains(ref.Name) {
		return false
	}
	visited.Add(ref.Name)
	resolved, ok := table.Get(ref.Name)
	if !ok {
		return false
	}
	switch form := resolved.Form.(type) {
	case StructForm:
		for _, f := range form.Fields {
			if refersTo(f.Type, targetQN, table, visited) {
				return true
			}
		}
	case AliasForm:
		return refersTo(form.Target, targetQN, table, visited)
	case DistinctForm:
		return refersTo(form.Base, targetQN, table, visited)
	}
	return false
}

type TypeParam struct {
	Constraint *TypeRef
	Default    *TypeRef
	Name       string
	Optional   bool
}

// HasDefault returns true if the type parameter has a default value.
// For languages that don't support literal type narrowing or advanced generics (Go, Python,
// C++, Proto), type parameters with defaults should be skipped and substituted with
// their default value instead.
func (tp TypeParam) HasDefault() bool {
	return tp.Default != nil
}

// NonDefaultedTypeParams returns the subset of type params that do NOT have a
// default value, preserving input order. Languages that can't express literal
// narrowing or advanced generics (Go, Python, C++, Proto) substitute defaulted
// params with their default value at codegen time and therefore must exclude
// them from emitted signatures and instantiations.
func NonDefaultedTypeParams(tps []TypeParam) []TypeParam {
	out := make([]TypeParam, 0, len(tps))
	for _, tp := range tps {
		if tp.HasDefault() {
			continue
		}
		out = append(out, tp)
	}
	return out
}

func UnifiedFields(typ Type, table *Table) []Field {
	form, ok := typ.Form.(StructForm)
	if !ok {
		return nil
	}
	if len(form.Extends) == 0 {
		return form.Fields
	}

	childFieldMap := make(map[string]*Field, len(form.Fields))
	for i := range form.Fields {
		childFieldMap[form.Fields[i].Name] = &form.Fields[i]
	}

	// Collect fields from all parents (left-to-right, first wins on conflict)
	seenFields := make(set.Set[string])
	var allParentFields []Field

	for _, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(table)
		if !ok {
			continue
		}
		parentForm, ok := parent.Form.(StructForm)
		if !ok {
			continue
		}

		// Build type argument substitution map for this parent
		typeArgMap := make(map[string]TypeRef)
		for i, tp := range parentForm.TypeParams {
			if i < len(extendsRef.TypeArgs) {
				typeArgMap[tp.Name] = extendsRef.TypeArgs[i]
			}
		}

		// Copy parent fields before modifying to avoid mutating the table
		parentFieldsOrig := UnifiedFields(parent, table)
		for _, pf := range parentFieldsOrig {
			if form.IsFieldOmitted(pf.Name) {
				continue
			}
			if seenFields.Contains(pf.Name) {
				continue // First parent wins
			}
			substitutedField := pf
			substitutedField.Type = SubstituteTypeRef(pf.Type, typeArgMap)
			seenFields.Add(pf.Name)
			allParentFields = append(allParentFields, substitutedField)
		}
	}

	// Build parent field map for domain merging during override
	parentFieldMap := make(map[string]*Field, len(allParentFields))
	for i := range allParentFields {
		parentFieldMap[allParentFields[i].Name] = &allParentFields[i]
	}

	// Collect parent fields that are not overridden by child
	var result []Field
	for _, pf := range allParentFields {
		if childFieldMap[pf.Name] != nil {
			continue // Child overrides this field
		}
		result = append(result, pf)
	}

	// Add child fields with domain merging for overrides
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
	return TypeRef{Name: ref.Name, TypeArgs: newArgs, ArraySize: ref.ArraySize}
}
