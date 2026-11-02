// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolver

import (
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// HasDomainOmissions reports whether the struct removes a domain inherited from a
// parent field with `-@domain`. A removal cannot be expressed through language
// inheritance/embedding — the embedded parent still carries the domain — so the
// struct must be flattened to drop it. Typeless overrides are resolved into
// complete fields by the analyzer before generation, so they do not require this.
func HasDomainOmissions(form resolution.StructForm) bool {
	for _, f := range form.Fields {
		if len(f.OmittedDomains) > 0 {
			return true
		}
	}
	return false
}

// DefaultOnlyOverrides returns the names of fields that redeclare a field inherited
// from extends to change only its default. Every language can express such an override
// while still inheriting: the field keeps its declared type and optionality, so only
// the value filled in when it is absent differs. A union variant passes the union's
// bases and its own, since both contribute to the variant it generates.
func DefaultOnlyOverrides(
	extends []resolution.TypeRef,
	fields []resolution.Field,
	table *resolution.Table,
) set.Set[string] {
	out := make(set.Set[string])
	inherited := resolution.InheritedFields(extends, table)
	for _, child := range fields {
		parent, ok := inherited[child.Name]
		if !ok || isStructuralOverride(parent, child) {
			continue
		}
		out.Add(child.Name)
	}
	return out
}

// hasStructuralOverride reports whether form redeclares an inherited field with a
// different type or optionality. Neither Go embedding nor C++ inheritance can restate
// an inherited field, so such a struct must be flattened; leaving it inherited emits a
// second field beside the inherited one.
func hasStructuralOverride(
	form resolution.StructForm,
	table *resolution.Table,
) bool {
	inherited := resolution.InheritedFields(form.Extends, table)
	for _, child := range form.Fields {
		if parent, ok := inherited[child.Name]; ok &&
			isStructuralOverride(parent, child) {
			return true
		}
	}
	return false
}

// isStructuralOverride reports whether child changes more than parent's default. A
// typeless override never can: the analyzer resolves it against the parent before
// generation.
func isStructuralOverride(parent, child resolution.Field) bool {
	return child.Optional != parent.Optional || !sameTypeRef(child.Type, parent.Type)
}

// sameTypeRef reports whether a and b name the same type, comparing array sizes, type
// parameters, and type arguments.
func sameTypeRef(a, b resolution.TypeRef) bool {
	if a.Name != b.Name || len(a.TypeArgs) != len(b.TypeArgs) {
		return false
	}
	if (a.ArraySize == nil) != (b.ArraySize == nil) {
		return false
	}
	if a.ArraySize != nil && *a.ArraySize != *b.ArraySize {
		return false
	}
	if (a.TypeParam == nil) != (b.TypeParam == nil) {
		return false
	}
	if a.TypeParam != nil && a.TypeParam.Name != b.TypeParam.Name {
		return false
	}
	for i := range a.TypeArgs {
		if !sameTypeRef(a.TypeArgs[i], b.TypeArgs[i]) {
			return false
		}
	}
	return true
}

// ParentsExist reports whether every type in extends exists in lang. A struct that
// extends a type absent from its language cannot inherit from it, so it declares the
// inherited fields itself.
func ParentsExist(
	extends []resolution.TypeRef,
	lang string,
	table *resolution.Table,
) bool {
	return !slices.ContainsFunc(extends, func(ref resolution.TypeRef) bool {
		parent, ok := ref.Resolve(table)
		return ok && omit.IsAbsent(parent, lang)
	})
}

// CanUseInheritance checks if a struct can use language inheritance/embedding in lang.
// Returns false if:
// - There are no parent types (Extends is empty)
// - A parent does not exist in lang
// - There are omitted fields (can't omit fields with inheritance)
// - A field removes an inherited domain (must flatten to drop it)
// - A field restates an inherited field's type or optionality
func CanUseInheritance(
	form resolution.StructForm,
	lang string,
	table *resolution.Table,
) bool {
	if len(form.Extends) == 0 {
		return false
	}
	if !ParentsExist(form.Extends, lang, table) {
		return false
	}
	if len(form.OmittedFields) > 0 {
		return false // Can't omit fields with inheritance
	}
	if HasDomainOmissions(form) {
		return false
	}
	if hasStructuralOverride(form, table) {
		return false
	}
	return true
}

// VariantBases splits the types a union variant would inherit in lang, its union's
// bases followed by its own, into the ones it still inherits and the fields it must
// declare itself. A base absent from lang, or whose field the variant drops with
// `-name`, cannot be inherited, because no target language can remove a member from an
// inherited type, so that base's remaining fields are returned for the variant to
// declare. A name an inherited base or the variant itself already supplies wins over
// the flattened base's copy, which is dropped.
func VariantBases(
	form resolution.UnionForm,
	v resolution.UnionVariant,
	lang string,
	table *resolution.Table,
) (inherited []resolution.TypeRef, declared []resolution.Field) {
	payload, ok := v.Type.Resolve(table)
	if !ok || !v.Inline {
		return form.Extends, nil
	}
	pform, ok := payload.Form.(resolution.StructForm)
	if !ok {
		return form.Extends, nil
	}
	bases := append(slices.Clone(form.Extends), pform.Extends...)
	if len(pform.OmittedFields) == 0 && ParentsExist(bases, lang, table) {
		return bases, nil
	}
	drop := set.New(pform.OmittedFields...)
	// taken holds every name the variant already receives: the fields it declares
	// itself, plus everything a base it still inherits contributes. A flattened
	// base's field under one of those names is dropped rather than declared again,
	// since a name arriving twice shadows silently in every target language.
	taken := set.New(lo.Map(pform.Fields, func(f resolution.Field, _ int) string {
		return f.Name
	})...)
	var flattened [][]resolution.Field
	for _, ref := range bases {
		base, baseRef, ok := resolution.Dealias(ref, table)
		if !ok {
			continue
		}
		typeArgs := resolution.TypeArgMap(
			resolution.TypeParams(base.Form),
			baseRef.TypeArgs,
		)
		fields := lo.Map(
			resolution.UnifiedFields(base, table),
			func(f resolution.Field, _ int) resolution.Field {
				f.Type = resolution.SubstituteTypeRef(f.Type, typeArgs)
				return f
			},
		)
		if !ParentsExist([]resolution.TypeRef{ref}, lang, table) ||
			lo.SomeBy(fields, func(f resolution.Field) bool {
				return drop.Contains(f.Name)
			}) {
			flattened = append(flattened, fields)
			continue
		}
		inherited = append(inherited, ref)
		for _, f := range fields {
			taken.Add(f.Name)
		}
	}
	for _, fields := range flattened {
		for _, f := range fields {
			if drop.Contains(f.Name) || taken.Contains(f.Name) {
				continue
			}
			taken.Add(f.Name)
			declared = append(declared, f)
		}
	}
	return inherited, declared
}

// SharedBases returns the bases of a union that every variant inherits whole in lang,
// so each variant holds the base under the base's own name.
func SharedBases(
	form resolution.UnionForm,
	lang string,
	table *resolution.Table,
) []resolution.TypeRef {
	return lo.Filter(form.Extends, func(ext resolution.TypeRef, _ int) bool {
		return lo.EveryBy(form.Variants, func(v resolution.UnionVariant) bool {
			inherited, _ := VariantBases(form, v, lang, table)
			return lo.ContainsBy(inherited, func(r resolution.TypeRef) bool {
				return r.Name == ext.Name
			})
		})
	})
}
