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
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// shapes compares what a stored type decodes in an old table with what it decodes in a
// new one. It fills in each generic type with its type arguments, defaults included,
// and looks through aliases and distinct types, so only the filled-in shape counts:
// stored field names and order, field types, optionality, keys, defaults, and enum and
// union members. Struct fields encode by position, so their order counts. Enum members
// and union variants encode by value and name, so the new side may add them.
type shapes struct {
	old, new *resolution.Table
	visiting set.Set[string]
	// covered holds every old type a comparison reached.
	covered set.Set[string]
}

func newShapes(old, new *resolution.Table) shapes {
	return shapes{
		old:      old,
		new:      new,
		visiting: make(set.Set[string]),
		covered:  make(set.Set[string]),
	}
}

// root returns why data that old wrote no longer decodes as new, or "" when it does. A
// type parameter with no default binds to the same placeholder on both sides.
func (s shapes) root(old, new resolution.Type) string {
	oldArgs, newArgs := placeholders(old), placeholders(new)
	if len(oldArgs) != len(newArgs) {
		return old.Name + ": changes its type parameters"
	}
	return s.ref(
		old.Name,
		resolution.TypeRef{Name: old.QualifiedName, TypeArgs: oldArgs},
		resolution.TypeRef{Name: new.QualifiedName, TypeArgs: newArgs},
	)
}

// placeholders returns one argument for each leading type parameter of t that has no
// default. TypeArgMap gives the rest their defaults.
func placeholders(t resolution.Type) []resolution.TypeRef {
	var args []resolution.TypeRef
	for i, p := range resolution.TypeParams(t.Form) {
		if p.HasDefault() {
			break
		}
		args = append(args, resolution.TypeRef{
			TypeParam: &resolution.TypeParam{Name: strconv.Itoa(i + 1)},
		})
	}
	return args
}

// ref compares the instances that two filled-in references stand for. at names the
// position in the stored type for the reason.
func (s shapes) ref(at string, old, new resolution.TypeRef) string {
	oldType, old, oldOK := s.instance(old, s.old)
	newType, new, newOK := s.instance(new, s.new)
	if !oldOK || !newOK {
		if oldOK || newOK || describeRef(old) != describeRef(new) {
			return fmt.Sprintf(
				"%s: changes from %s to %s", at, describe(oldType, old, oldOK),
				describe(newType, new, newOK),
			)
		}
		return ""
	}
	if (old.ArraySize == nil) != (new.ArraySize == nil) ||
		(old.ArraySize != nil && *old.ArraySize != *new.ArraySize) {
		return at + ": changes its array size"
	}
	if oldKind, newKind := kind(oldType), kind(newType); oldKind != newKind {
		return fmt.Sprintf("%s: changes from %s to %s", at, oldKind, newKind)
	}
	key := refKey(old) + "|" + refKey(new)
	if s.visiting.Contains(key) {
		return ""
	}
	s.visiting.Add(key)
	defer s.visiting.Remove(key)
	switch o := oldType.Form.(type) {
	case resolution.StructForm:
		s.coverBases(oldType)
		return s.fields(
			at,
			instanceFields(oldType, old, s.old),
			instanceFields(newType, new, s.new),
		)
	case resolution.EnumForm:
		return enumMembers(at, o, newType.Form.(resolution.EnumForm))
	case resolution.UnionForm:
		return s.union(at, oldType, newType)
	case resolution.BuiltinGenericForm:
		if len(old.TypeArgs) != len(new.TypeArgs) {
			return at + ": changes its type arguments"
		}
		for i := range old.TypeArgs {
			if reason := s.ref(at, old.TypeArgs[i], new.TypeArgs[i]); reason != "" {
				return reason
			}
		}
	}
	return ""
}

// instance follows ref through aliases and distinct types to the type that stores its
// data, and returns that type with the filled-in reference to it. ok is false when ref
// is a placeholder or nil, which no table holds.
func (s shapes) instance(
	ref resolution.TypeRef, table *resolution.Table,
) (resolution.Type, resolution.TypeRef, bool) {
	for {
		if ref.IsTypeParam() || ref.Name == "nil" {
			return resolution.Type{}, ref, false
		}
		t, target, ok := resolution.Dealias(ref, table)
		if !ok {
			panic("unresolved type in an analyzed table: " + ref.Name)
		}
		if table == s.old {
			s.covered.Add(t.QualifiedName)
		}
		distinct, ok := t.Form.(resolution.DistinctForm)
		if !ok {
			return t, target, true
		}
		ref = resolution.SubstituteTypeRef(
			distinct.Base, resolution.TypeArgMap(distinct.TypeParams, target.TypeArgs),
		)
	}
}

// instanceFields returns the stored fields of the struct t with the type arguments of
// ref filled in.
func instanceFields(
	t resolution.Type, ref resolution.TypeRef, table *resolution.Table,
) []resolution.Field {
	args := resolution.TypeArgMap(resolution.TypeParams(t.Form), ref.TypeArgs)
	fields := storedFields(resolution.UnifiedFields(t, table))
	for i := range fields {
		fields[i].Type = resolution.SubstituteTypeRef(fields[i].Type, args)
	}
	return fields
}

// fields compares two filled-in stored field lists.
func (s shapes) fields(at string, oldFields, newFields []resolution.Field) string {
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

// coverBases marks the bases of old as covered, since UnifiedFields flattens them.
func (s shapes) coverBases(old resolution.Type) {
	for _, ext := range old.Form.(resolution.StructForm).Extends {
		base, _, ok := resolution.Dealias(ext, s.old)
		s.covered.Add(ext.Name)
		if !ok {
			continue
		}
		s.covered.Add(base.QualifiedName)
		if _, isStruct := base.Form.(resolution.StructForm); isStruct {
			s.coverBases(base)
		}
	}
}

// cover marks the old struct that ref stands for, and its bases, as covered.
func (s shapes) cover(ref resolution.TypeRef) {
	if t, _, ok := s.instance(ref, s.old); ok {
		if _, isStruct := t.Form.(resolution.StructForm); isStruct {
			s.coverBases(t)
		}
	}
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
	for _, ext := range old.Extends {
		s.cover(ext)
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
		s.cover(ov.Type)
		if reason := s.fields(
			variantAt,
			storedFields(resolution.UnifiedVariantFields(oldType, ov, s.old)),
			storedFields(resolution.UnifiedVariantFields(newType, nv, s.new)),
		); reason != "" {
			return reason
		}
	}
	return ""
}

// kind names the form of t, or the type itself for a primitive or a built-in generic.
func kind(t resolution.Type) string {
	switch t.Form.(type) {
	case resolution.StructForm:
		return "a struct"
	case resolution.EnumForm:
		return "an enum"
	case resolution.UnionForm:
		return "a union"
	case resolution.PrimitiveForm, resolution.BuiltinGenericForm:
		return t.QualifiedName
	}
	panic(fmt.Sprintf("unknown type form %T", t.Form))
}

// describe names a resolved type by its kind, or a placeholder or nil by its reference.
func describe(t resolution.Type, ref resolution.TypeRef, resolved bool) string {
	if resolved {
		return kind(t)
	}
	return describeRef(ref)
}

func describeRef(ref resolution.TypeRef) string {
	if ref.IsTypeParam() {
		return "type parameter " + ref.TypeParam.Name
	}
	return ref.Name
}

// refKey renders a filled-in reference, to detect a comparison that recurses.
func refKey(ref resolution.TypeRef) string {
	var b strings.Builder
	b.WriteString(describeRef(ref))
	if len(ref.TypeArgs) > 0 {
		args := make([]string, len(ref.TypeArgs))
		for i, arg := range ref.TypeArgs {
			args[i] = refKey(arg)
		}
		b.WriteString("<" + strings.Join(args, ",") + ">")
	}
	if ref.ArraySize != nil {
		fmt.Fprintf(&b, "[%d]", *ref.ArraySize)
	}
	return b.String()
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
