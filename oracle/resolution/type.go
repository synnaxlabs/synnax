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
	"maps"

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
	// Synthetic marks analyzer-fabricated types (inline union variant
	// payloads) that resolve like any other type but are excluded from
	// standalone code generation.
	Synthetic bool
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
	// Values are the enum's members, fully resolved. For an extending enum the
	// analyzer expands this to the union of the parents' members plus any
	// members declared in the enum's own body.
	Values []EnumValue
	// IsIntEnum reports whether members carry integer (vs string) values. An
	// extending enum matches the kind of the enums it extends.
	IsIntEnum bool
	// Extends names the enums this enum is the union of. Empty for a plain
	// (non-extending) enum.
	Extends []TypeRef
}

func (EnumForm) typeForm() {}

// IsExtension reports whether the enum extends one or more other enums.
func (f EnumForm) IsExtension() bool { return len(f.Extends) > 0 }

// UnionForm describes a discriminated union: a closed set of struct variants
// distinguished by the string value of a single shared discriminator field.
// All variants share an optional set of base fields contributed by Extends.
//
// The discriminator field is owned by the union declaration and must not be
// declared by any variant struct or by any base in Extends. Plugins read
// Discriminator and the per-variant Name to emit the discriminator field in
// their target language (Zod literal, Go tagged field, Pydantic Literal, etc.).
type UnionForm struct {
	// Discriminator is the JSON field name whose string value selects a variant.
	Discriminator string
	// Variants lists every concrete shape the union admits, in declaration order.
	Variants []UnionVariant
	// Extends names struct types whose fields are shared across all variants.
	// Resolved later by the analyzer; struct-form is validated there.
	Extends []TypeRef
}

func (UnionForm) typeForm() {}

func (f UnionForm) Variant(name string) (UnionVariant, bool) {
	return lo.Find(f.Variants, func(v UnionVariant) bool { return v.Name == name })
}

// UnionVariant is one concrete shape inside a UnionForm. Name is both the
// variant's identifier in source and the string the discriminator field takes
// in JSON. Type references the variant's payload struct.
type UnionVariant struct {
	AST     any
	Domains map[string]Domain
	Name    string
	Type    TypeRef
	// Inline reports that the payload was declared inline in the union body.
	// Type then references a Synthetic struct whose fields generators flatten
	// into the variant member instead of embedding a named payload type.
	Inline bool
}

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
	AST     any
	Domains map[string]Domain
	Name    string
	Type    TypeRef
	// Default is the field's default value, declared inline as `name type = value`.
	// It is nil when the field has no default.
	Default *ExpressionValue
	// Optional reports whether the field is optional, declared with a trailing `?`
	// on the field. An optional field admits an explicit absence (a Go pointer, a
	// TypeScript `.optional()`, a Python `| None`, a C++ `std::optional`, a proto3
	// `optional`) that round-trips faithfully rather than collapsing to the zero
	// value.
	Optional    bool
	OmitIfUnset bool
	// OmittedDomains names domains to drop from the inherited parent field during
	// override resolution, declared with the `-@domain` syntax.
	OmittedDomains []string
}

// HasType reports whether the field declares a type. It is false only for a
// partial override in an extending struct: a field that names an inherited field
// and omits its type to inherit the parent's type, optionality, and (unless the
// override sets its own) default.
func (f Field) HasType() bool {
	return f.Type.Name != "" || f.Type.TypeParam != nil
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
// field types and extends bases, union bases and variant payloads, alias
// targets, and distinct bases, so mutual-recursion cycles (A → B → A) are
// detected, not just direct self-reference (A → A). A visited set prevents
// infinite loops on non-target cycles.
//
// This is the shared primitive behind the analyzer's IsRecursive detection
// and the C++ plugin's decision between std::optional<T> and
// x::mem::indirect<T> for optional fields.
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
		for _, ext := range form.Extends {
			if refersTo(ext, targetQN, table, visited) {
				return true
			}
		}
		for _, f := range form.Fields {
			if refersTo(f.Type, targetQN, table, visited) {
				return true
			}
		}
	case UnionForm:
		for _, ext := range form.Extends {
			if refersTo(ext, targetQN, table, visited) {
				return true
			}
		}
		for _, v := range form.Variants {
			if refersTo(v.Type, targetQN, table, visited) {
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

	parentFieldMap := make(map[string]*Field, len(allParentFields))
	for i := range allParentFields {
		parentFieldMap[allParentFields[i].Name] = &allParentFields[i]
	}

	// Parent fields the child does not override, in parent order. Overrides are
	// emitted below in child-declaration order.
	var result []Field
	for _, pf := range allParentFields {
		if childFieldMap[pf.Name] != nil {
			continue
		}
		result = append(result, pf)
	}

	for _, cf := range form.Fields {
		if pf, isOverride := parentFieldMap[cf.Name]; isOverride {
			result = append(result, mergeOverrideField(*pf, cf))
		} else {
			result = append(result, cf)
		}
	}
	return result
}

// UnifiedVariantFields returns the flattened field list for one variant of a
// discriminated union: fields contributed by the union's Extends (with their
// own inheritance resolved), followed by the variant struct's own fields (also
// with its inheritance resolved). Duplicate names are dropped after the first
// occurrence, matching UnifiedFields' first-wins semantics for struct mixins.
//
// The discriminator field is NOT included in the returned list. Callers know
// its name from UnionForm.Discriminator and its value from variant.Name, and
// are responsible for emitting it in the form their target language requires
// (a Zod literal, a Go tagged field, a Pydantic Literal, etc.).
//
// If union is not a UnionForm, or the variant resolves to a non-struct type,
// the result is the best effort possible: the bases that did resolve, and
// nothing for the variant.
func UnifiedVariantFields(union Type, variant UnionVariant, table *Table) []Field {
	form, ok := union.Form.(UnionForm)
	if !ok {
		return nil
	}
	seen := make(set.Set[string])
	var result []Field
	appendUnique := func(fields []Field) {
		for _, f := range fields {
			if seen.Contains(f.Name) {
				continue
			}
			seen.Add(f.Name)
			result = append(result, f)
		}
	}
	for _, baseRef := range form.Extends {
		base, ok := baseRef.Resolve(table)
		if !ok {
			continue
		}
		if _, isStruct := base.Form.(StructForm); !isStruct {
			continue
		}
		appendUnique(UnifiedFields(base, table))
	}
	variantType, ok := variant.Type.Resolve(table)
	if !ok {
		return result
	}
	if _, isStruct := variantType.Form.(StructForm); !isStruct {
		return result
	}
	appendUnique(UnifiedFields(variantType, table))
	return result
}

// mergeOverrideField resolves a child field that overrides parent into a
// complete field. When the child omits its type it inherits the parent's type,
// optionality, and (unless the child declares its own) default; writing a type
// fully (re)defines both, leaving the override to drop the parent's default by
// omission, matching a from-scratch field declaration. Domains are always
// merged (child expressions win per name), after which any domains the child
// lists via `-@domain` are dropped.
func mergeOverrideField(parent, child Field) Field {
	out := child
	if !child.HasType() {
		out.Type = parent.Type
		// A typeless override inherits the parent's nullability unless it forces
		// its own with a standalone `?` marker (key?).
		if !child.Optional {
			out.Optional = parent.Optional
		}
		if child.Default == nil {
			out.Default = parent.Default
		}
	}
	domains := make(map[string]Domain, len(parent.Domains)+len(child.Domains))
	maps.Copy(domains, parent.Domains)
	for k, v := range child.Domains {
		if existing, ok := domains[k]; ok {
			domains[k] = v.Merge(existing)
		} else {
			domains[k] = v
		}
	}
	for _, name := range child.OmittedDomains {
		delete(domains, name)
	}
	out.Domains = domains
	return out
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
