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
}

func (StructForm) typeForm() {}

func (f StructForm) IsGeneric() bool { return len(f.TypeParams) > 0 }

func (f StructForm) Field(name string) (Field, bool) {
	return lo.Find(f.Fields, func(fld Field) bool { return fld.Name == name })
}

func (f StructForm) isFieldOmitted(name string) bool {
	return lo.Contains(f.OmittedFields, name)
}

type EnumForm struct {
	// Values are the enum's members, fully resolved. For an extending enum the
	// analyzer expands this to the union of the parents' members plus any
	// members declared in the enum's own body.
	Values []EnumValue
	// Declared holds the members the enum's own body declares, set only when the
	// enum extends. Only a renderer writing the declaration back wants it.
	Declared []EnumValue
	// IsIntEnum reports whether members carry integer (vs string) values. An
	// extending enum matches the kind of the enums it extends.
	IsIntEnum bool
	// Extends names the enums this enum is the union of. Empty for a plain
	// (non-extending) enum.
	Extends []TypeRef
}

func (EnumForm) typeForm() {}

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
	// Enum names the declared enum the discriminator borrows its type from. Its
	// members are the variant names, so plugins emit no discriminator type of their
	// own. The zero TypeRef means the union generates one.
	Enum TypeRef
	// Variants lists every concrete shape the union admits, in declaration order.
	Variants []UnionVariant
	// Declared holds the variants the union's own body declares, set only when
	// Included is non-empty. Only a renderer writing the declaration back wants it.
	Declared []UnionVariant
	// Extends names struct types whose fields are shared across all variants.
	// Resolved later by the analyzer; struct-form is validated there.
	Extends []TypeRef
	// Included names the unions this union takes the variants of. The analyzer
	// moves them out of Extends, which readers take as struct bases.
	Included []TypeRef
}

func (UnionForm) typeForm() {}

// BorrowsEnum reports whether the discriminator takes its type from a declared enum.
func (f UnionForm) BorrowsEnum() bool { return f.Enum.Name != "" }

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
	// Extends names struct types whose fields are flattened into the action's
	// payload. The analyzer resolves these and prepends the inherited fields to
	// Fields, so by code-generation time Fields already holds the unified list.
	Extends []TypeRef
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
	Optional bool
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

// Dealias follows ref through alias declarations to the type it stands for, and returns
// that type with the reference to it. A generic alias's type arguments carry over to
// the target. A distinct type is not an alias, so the walk stops there. ok is false
// when ref is a bare type parameter, or when ref or an alias target does not resolve or
// forms a cycle.
func Dealias(ref TypeRef, table *Table) (Type, TypeRef, bool) {
	visited := make(set.Set[string])
	for {
		if ref.IsTypeParam() || visited.Contains(ref.Name) {
			return Type{}, TypeRef{}, false
		}
		visited.Add(ref.Name)
		resolved, ok := ref.Resolve(table)
		if !ok {
			return Type{}, TypeRef{}, false
		}
		form, isAlias := resolved.Form.(AliasForm)
		if !isAlias {
			return resolved, ref, true
		}
		ref = SubstituteTypeRef(form.Target, TypeArgMap(form.TypeParams, ref.TypeArgs))
	}
}

// IsDistinct reports whether ref, or the type its aliases stand for, is a distinct
// type. Primitives and references the table cannot resolve are not distinct.
func IsDistinct(ref TypeRef, table *Table) bool {
	resolved, _, ok := Dealias(ref, table)
	if !ok {
		return false
	}
	_, isDistinct := resolved.Form.(DistinctForm)
	return isDistinct
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

func refersTo(
	ref TypeRef,
	targetQN string,
	table *Table,
	visited set.Set[string],
) bool {
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
		if form.BorrowsEnum() && refersTo(form.Enum, targetQN, table, visited) {
			return true
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

// PrimitiveBase follows ref through distinct-type and alias chains to the underlying
// primitive type name (e.g. a distinct Key over uint32 resolves to "uint32"). It
// returns the primitive name when ref bottoms out at a primitive, or "" when it does
// not (a struct, union, builtin container, or unresolved reference). A reference whose
// surface name is already a primitive is returned directly. Used to classify named
// numeric and string types for validation, where the constraint applies to the
// underlying primitive.
func PrimitiveBase(ref TypeRef, table *Table) string {
	return primitiveBase(ref, table, set.New[string]())
}

func primitiveBase(ref TypeRef, table *Table, visited set.Set[string]) string {
	if ref.IsTypeParam() {
		if ref.TypeParam != nil && ref.TypeParam.Constraint != nil {
			return primitiveBase(*ref.TypeParam.Constraint, table, visited)
		}
		return ""
	}
	if IsPrimitive(ref.Name) {
		return ref.Name
	}
	if visited.Contains(ref.Name) {
		return ""
	}
	visited.Add(ref.Name)
	resolved, ok := table.Get(ref.Name)
	if !ok {
		return ""
	}
	switch form := resolved.Form.(type) {
	case DistinctForm:
		return primitiveBase(form.Base, table, visited)
	case AliasForm:
		return primitiveBase(form.Target, table, visited)
	}
	return ""
}

// CollectionKind reports the collection kind that ref ultimately resolves to:
// "Array", "Map", or "record". It follows type-parameter constraints and defaults,
// alias targets, and distinct bases. ok is false when ref does not resolve to a
// collection. Generators use it to decide which fields are nilable collections (for
// omitzero tagging and empty defaults) even when the field is typed through an alias
// or a constrained type parameter.
func CollectionKind(ref TypeRef, table *Table) (kind string, ok bool) {
	return collectionKind(ref, table, set.New[string]())
}

func collectionKind(ref TypeRef, table *Table, visited set.Set[string]) (string, bool) {
	if ref.IsTypeParam() {
		if ref.TypeParam == nil {
			return "", false
		}
		if ref.TypeParam.Constraint != nil {
			return collectionKind(*ref.TypeParam.Constraint, table, visited)
		}
		if ref.TypeParam.Default != nil {
			return collectionKind(*ref.TypeParam.Default, table, visited)
		}
		return "", false
	}
	if ref.Name == "Array" || ref.Name == "Map" || ref.Name == "record" {
		return ref.Name, true
	}
	if visited.Contains(ref.Name) {
		return "", false
	}
	visited.Add(ref.Name)
	resolved, ok := table.Get(ref.Name)
	if !ok {
		return "", false
	}
	switch form := resolved.Form.(type) {
	case DistinctForm:
		return collectionKind(form.Base, table, visited)
	case AliasForm:
		return collectionKind(form.Target, table, visited)
	}
	return "", false
}

type TypeParam struct {
	Constraint *TypeRef
	Default    *TypeRef
	Name       string
	Optional   bool
}

// HasDefault returns true if the type parameter has a default value. For languages that
// don't support literal type narrowing or advanced generics (Go, Python, C++, Proto),
// type parameters with defaults should be skipped and substituted with their default
// value instead.
func (tp TypeParam) HasDefault() bool {
	return tp.Default != nil
}

// TypeParams returns the type parameters form declares, or nil when form cannot declare
// any.
func TypeParams(form TypeForm) []TypeParam {
	switch f := form.(type) {
	case StructForm:
		return f.TypeParams
	case AliasForm:
		return f.TypeParams
	case DistinctForm:
		return f.TypeParams
	}
	return nil
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

	for _, ext := range form.Extends {
		parent, extendsRef, ok := Dealias(ext, table)
		if !ok {
			continue
		}
		parentForm, ok := parent.Form.(StructForm)
		if !ok {
			continue
		}

		typeArgMap := TypeArgMap(parentForm.TypeParams, extendsRef.TypeArgs)

		// Copy parent fields before modifying to avoid mutating the table
		parentFieldsOrig := UnifiedFields(parent, table)
		for _, pf := range parentFieldsOrig {
			if form.isFieldOmitted(pf.Name) {
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
			result = append(result, MergeOverrideField(*pf, cf))
		} else {
			result = append(result, cf)
		}
	}
	return result
}

// InheritedFields returns the fields the given parents contribute, keyed by name, with
// each parent's type arguments substituted. The leftmost parent wins a name collision,
// matching UnifiedFields.
func InheritedFields(extends []TypeRef, table *Table) map[string]Field {
	out := make(map[string]Field)
	for _, declared := range extends {
		parent, ext, ok := Dealias(declared, table)
		if !ok {
			continue
		}
		parentForm, ok := parent.Form.(StructForm)
		if !ok {
			continue
		}
		typeArgs := TypeArgMap(parentForm.TypeParams, ext.TypeArgs)
		for _, f := range UnifiedFields(parent, table) {
			if _, seen := out[f.Name]; seen {
				continue
			}
			f.Type = SubstituteTypeRef(f.Type, typeArgs)
			out[f.Name] = f
		}
	}
	return out
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
	// A variant drops a field the union's bases contribute with `-name`.
	var omitted set.Set[string]
	variantType, _, variantOK := Dealias(variant.Type, table)
	variantForm, isStruct := StructForm{}, false
	if variantOK {
		variantForm, isStruct = variantType.Form.(StructForm)
	}
	if isStruct {
		omitted = set.New(variantForm.OmittedFields...)
	}
	appendUnique := func(fields []Field) {
		for _, f := range fields {
			if seen.Contains(f.Name) || omitted.Contains(f.Name) {
				continue
			}
			seen.Add(f.Name)
			result = append(result, f)
		}
	}
	for _, baseRef := range form.Extends {
		base, _, ok := Dealias(baseRef, table)
		if !ok {
			continue
		}
		if _, isStruct := base.Form.(StructForm); !isStruct {
			continue
		}
		appendUnique(UnifiedFields(base, table))
	}
	if !variantOK || !isStruct {
		return result
	}
	appendUnique(UnifiedFields(variantType, table))
	return result
}

// MergeOverrideField resolves a child field that overrides parent into a
// complete field. When the child omits its type it inherits the parent's type,
// optionality, and (unless the child declares its own) default; writing a type
// fully (re)defines both, leaving the override to drop the parent's default by
// omission, matching a from-scratch field declaration. Domains are always
// merged (child expressions win per name), after which any domains the child
// lists via `-@domain` are dropped.
func MergeOverrideField(parent, child Field) Field {
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

// TypeArgMap maps each of params to the argument at its position in args, for
// SubstituteTypeRef. A parameter with no argument is left out.
func TypeArgMap(params []TypeParam, args []TypeRef) map[string]TypeRef {
	typeArgMap := make(map[string]TypeRef, len(params))
	for i, tp := range params {
		if i < len(args) {
			typeArgMap[tp.Name] = args[i]
		}
	}
	return typeArgMap
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
