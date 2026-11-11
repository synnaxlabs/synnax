// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// shapes compares what a stored type decodes in an old table with what it decodes in a
// new one. Type names, namespaces, docs, rules, and codec markers do not count: only
// stored field names and order, field types, optionality, keys, defaults, and enum and
// union members. Struct fields encode by position, so their order counts. Enum members
// and union variants encode by value and name, so the new side may add them.
type shapes struct {
	old, new *resolution.Table
	visiting set.Set[string]
	// covered holds every old type a comparison reached.
	covered set.Set[string]
	// oldParams and newParams are the type parameters of the types being compared.
	oldParams, newParams []resolution.TypeParam
}

func newShapes(old, new *resolution.Table) shapes {
	return shapes{
		old:      old,
		new:      new,
		visiting: make(set.Set[string]),
		covered:  make(set.Set[string]),
	}
}

// typ returns why data that old wrote no longer decodes as new, or "" when it does. at
// names the position in the stored type for the reason.
func (s shapes) typ(at string, old, new resolution.Type) string {
	key := old.QualifiedName + "|" + new.QualifiedName
	if s.visiting.Contains(key) {
		return ""
	}
	s.visiting.Add(key)
	defer s.visiting.Remove(key)
	s.covered.Add(old.QualifiedName)
	if reflect.TypeOf(old.Form) != reflect.TypeOf(new.Form) {
		return fmt.Sprintf("%s: changes from %T to %T", at, old.Form, new.Form)
	}
	s.oldParams = resolution.TypeParams(old.Form)
	s.newParams = resolution.TypeParams(new.Form)
	if len(s.oldParams) != len(s.newParams) {
		return at + ": changes its type parameters"
	}
	// A reference that omits a defaulted argument stores the default.
	for i, op := range s.oldParams {
		np := s.newParams[i]
		if op.Optional != np.Optional || op.HasDefault() != np.HasDefault() {
			return at + ": changes the default of type parameter " + op.Name
		}
		if op.HasDefault() {
			if reason := s.ref(at, *op.Default, *np.Default); reason != "" {
				return reason
			}
		}
	}
	switch o := old.Form.(type) {
	case resolution.StructForm:
		return s.fields(at, old, new)
	case resolution.EnumForm:
		return enumMembers(at, o, new.Form.(resolution.EnumForm))
	case resolution.UnionForm:
		return s.union(at, old, new)
	case resolution.AliasForm:
		return s.ref(at, o.Target, new.Form.(resolution.AliasForm).Target)
	case resolution.DistinctForm:
		return s.ref(at, o.Base, new.Form.(resolution.DistinctForm).Base)
	case resolution.PrimitiveForm, resolution.BuiltinGenericForm:
		if old.QualifiedName != new.QualifiedName {
			return fmt.Sprintf(
				"%s: changes from %s to %s", at, old.QualifiedName, new.QualifiedName,
			)
		}
	default:
		return fmt.Sprintf("%s: has an unknown form %T", at, old.Form)
	}
	return ""
}

// fields compares the stored fields of two structs, bases included.
func (s shapes) fields(at string, old, new resolution.Type) string {
	oldFields := storedFields(resolution.UnifiedFields(old, s.old))
	newFields := storedFields(resolution.UnifiedFields(new, s.new))
	if !slices.EqualFunc(oldFields, newFields, func(a, b resolution.Field) bool {
		return a.Name == b.Name
	}) {
		return fmt.Sprintf(
			"%s: stored fields change from %v to %v",
			at, fieldNames(oldFields), fieldNames(newFields),
		)
	}
	for i, of := range oldFields {
		nf, fieldAt := newFields[i], at+"."+of.Name
		switch {
		case of.Optional != nf.Optional:
			return fieldAt + ": changes optionality"
		case domain.HasDomainFromField(of, "key") !=
			domain.HasDomainFromField(nf, "key"):
			return fieldAt + ": changes @key"
		case !reflect.DeepEqual(of.Default, nf.Default):
			return fieldAt + ": changes its default"
		}
		if reason := s.ref(fieldAt, of.Type, nf.Type); reason != "" {
			return reason
		}
	}
	return ""
}

func (s shapes) union(at string, oldType, newType resolution.Type) string {
	old := oldType.Form.(resolution.UnionForm)
	new := newType.Form.(resolution.UnionForm)
	if old.Discriminator != new.Discriminator {
		return at + ": changes its discriminator"
	}
	if old.BorrowsEnum() != new.BorrowsEnum() {
		return at + ": changes its discriminator type"
	}
	if old.BorrowsEnum() {
		if reason := s.ref(at, old.Enum, new.Enum); reason != "" {
			return reason
		}
	}
	if len(old.Extends) != len(new.Extends) {
		return at + ": changes its bases"
	}
	for i := range old.Extends {
		if reason := s.ref(at, old.Extends[i], new.Extends[i]); reason != "" {
			return reason
		}
	}
	for _, ov := range old.Variants {
		variantAt := at + "." + ov.Name
		nv, ok := new.Variant(ov.Name)
		if !ok {
			return variantAt + ": variant removed"
		}
		if ov.Inline != nv.Inline {
			return variantAt + ": changes between inline and named"
		}
		// A variant can drop fields that the union's bases give it.
		oldFields := fieldNames(storedFields(
			resolution.UnifiedVariantFields(oldType, ov, s.old),
		))
		newFields := fieldNames(storedFields(
			resolution.UnifiedVariantFields(newType, nv, s.new),
		))
		if !slices.Equal(oldFields, newFields) {
			return fmt.Sprintf(
				"%s: stored fields change from %v to %v",
				variantAt,
				oldFields,
				newFields,
			)
		}
		if reason := s.ref(variantAt, ov.Type, nv.Type); reason != "" {
			return reason
		}
	}
	return ""
}

func (s shapes) ref(at string, old, new resolution.TypeRef) string {
	if len(old.TypeArgs) != len(new.TypeArgs) {
		return at + ": changes its type arguments"
	}
	for i := range old.TypeArgs {
		if reason := s.ref(at, old.TypeArgs[i], new.TypeArgs[i]); reason != "" {
			return reason
		}
	}
	if (old.ArraySize == nil) != (new.ArraySize == nil) ||
		(old.ArraySize != nil && *old.ArraySize != *new.ArraySize) {
		return at + ": changes its array size"
	}
	if old.IsTypeParam() != new.IsTypeParam() {
		return at + ": changes between a type parameter and a concrete type"
	}
	if old.IsTypeParam() {
		if paramIndex(s.oldParams, old) != paramIndex(s.newParams, new) {
			return at + ": changes its type parameter"
		}
		return ""
	}
	// The analyzer fills an omitted type argument with nil, which no table holds.
	if old.Name == "nil" || new.Name == "nil" {
		if old.Name != new.Name {
			return fmt.Sprintf("%s: changes from %s to %s", at, old.Name, new.Name)
		}
		return ""
	}
	return s.typ(at, s.old.MustGet(old.Name), s.new.MustGet(new.Name))
}

// paramIndex returns the position of the type parameter that ref holds in params.
// Stored data binds parameters by position. It panics when params does not declare the
// parameter.
func paramIndex(params []resolution.TypeParam, ref resolution.TypeRef) int {
	i := slices.IndexFunc(params, func(p resolution.TypeParam) bool {
		return p.Name == ref.TypeParam.Name
	})
	if i < 0 {
		panic("type parameter not in scope: " + ref.TypeParam.Name)
	}
	return i
}

// enumMembers requires every old member in new with the same value. Stored data holds
// member values, so a new member leaves old data readable.
func enumMembers(at string, old, new resolution.EnumForm) string {
	if old.IsIntEnum != new.IsIntEnum {
		return at + ": changes between int and string members"
	}
	for _, ov := range old.Values {
		i := slices.IndexFunc(new.Values, func(v resolution.EnumValue) bool {
			return v.Name == ov.Name
		})
		if i < 0 {
			return fmt.Sprintf("%s: member %s removed", at, ov.Name)
		}
		if new.Values[i].Value != ov.Value {
			return fmt.Sprintf("%s: member %s changes its value", at, ov.Name)
		}
	}
	return ""
}

// storedFields drops the fields that @go marshal omit keeps out of the stored bytes.
func storedFields(fields []resolution.Field) []resolution.Field {
	return slices.DeleteFunc(slices.Clone(fields), func(f resolution.Field) bool {
		return domain.GetStringFromField(f, "go", "marshal") == "omit"
	})
}

func fieldNames(fields []resolution.Field) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}
