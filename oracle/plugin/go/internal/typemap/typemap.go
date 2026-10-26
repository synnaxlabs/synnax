// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package typemap provides shared type resolution utilities for Go code generation
// plugins, eliminating duplication of primitive type mapping, alias/distinct
// unwrapping, and Go type name resolution.
package typemap

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// PrimitiveGoType maps an Oracle primitive type name to its Go type string.
// Returns empty string for unsupported primitives.
func PrimitiveGoType(primName string) (string, bool) {
	switch primName {
	case "string":
		return "string", true
	case "bool":
		return "bool", true
	case "int8":
		return "int8", true
	case "int16":
		return "int16", true
	case "int32":
		return "int32", true
	case "int64":
		return "int64", true
	case "uint8":
		return "uint8", true
	case "uint12", "uint16":
		return "uint16", true
	case "uint20", "uint32":
		return "uint32", true
	case "uint64":
		return "uint64", true
	case "float32":
		return "float32", true
	case "float64":
		return "float64", true
	case "uuid":
		return "uuid.UUID", true
	case "bytes":
		return "[]byte", true
	case "record", "any":
		return "interface{}", true
	default:
		return "", false
	}
}

// Nilable reports whether a Go value of ref, through aliases and distinct types, is a
// slice, map, or record. An optional field of such a type holds no pointer, since nil
// already marks it absent. A defaulted type parameter counts as its default.
func Nilable(ref resolution.TypeRef, table *resolution.Table) bool {
	if ref.IsTypeParam() {
		return ref.TypeParam.HasDefault() && Nilable(*ref.TypeParam.Default, table)
	}
	if arrays.IsArray(ref, table) {
		return arrays.Size(ref, table) == nil
	}
	resolved, ok := ref.Resolve(table)
	if !ok {
		return false
	}
	switch form := UnwrapType(resolved, table).Form.(type) {
	case resolution.BuiltinGenericForm:
		return form.Name == "Map"
	case resolution.PrimitiveForm:
		return form.Name == "record" || form.Name == "bytes"
	default:
		return false
	}
}

// IsUUID returns true if the primitive name is "uuid".
func IsUUID(primName string) bool { return primName == "uuid" }

// UnwrapType resolves through alias and distinct type wrappers to find the
// underlying concrete type.
func UnwrapType(typ resolution.Type, table *resolution.Table) resolution.Type {
	actual := typ
	for {
		switch form := actual.Form.(type) {
		case resolution.AliasForm:
			target, ok := form.Target.Resolve(table)
			if !ok {
				return actual
			}
			actual = target
			continue
		case resolution.DistinctForm:
			target, ok := form.Base.Resolve(table)
			if !ok {
				return actual
			}
			actual = target
			continue
		default:
		}
		break
	}
	return actual
}

// UnwrapTypeRef follows ref, which resolves to resolved, through alias and distinct
// declarations to the type they stand for. It returns that type with the type arguments
// it takes, with the arguments ref passes in place of the declarations' type
// parameters. The walk stops at a declaration whose target does not resolve.
func UnwrapTypeRef(
	resolved resolution.Type, ref resolution.TypeRef, table *resolution.Table,
) (resolution.Type, []resolution.TypeRef) {
	for {
		target, ok := wrappedTypeRef(resolved, ref)
		if !ok {
			return resolved, ref.TypeArgs
		}
		next, ok := target.Resolve(table)
		if !ok {
			return resolved, ref.TypeArgs
		}
		resolved, ref = next, target
	}
}

// wrappedTypeRef returns the reference an alias or distinct type wraps, with the type
// arguments ref passes substituted in. ok is false for any other type.
func wrappedTypeRef(
	resolved resolution.Type, ref resolution.TypeRef,
) (resolution.TypeRef, bool) {
	var target resolution.TypeRef
	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		target = form.Target
	case resolution.DistinctForm:
		target = form.Base
	default:
		return resolution.TypeRef{}, false
	}
	typeArgMap := resolution.TypeArgMap(
		resolution.TypeParams(resolved.Form),
		ref.TypeArgs,
	)
	return resolution.SubstituteTypeRef(target, typeArgMap), true
}

// GoTypeRefName renders ref as a Go type, with goTypeNameFn naming each declared type.
// A generic struct or alias takes the type arguments ref passes, except those bound to
// a defaulted type parameter, which Go declarations drop.
func GoTypeRefName(
	ref resolution.TypeRef, table *resolution.Table,
	goTypeNameFn func(resolution.Type) (string, error),
) (string, error) {
	if ref.IsTypeParam() {
		if ref.TypeParam.HasDefault() {
			return GoTypeRefName(*ref.TypeParam.Default, table, goTypeNameFn)
		}
		return ref.TypeParam.Name, nil
	}
	if ref.Name == "Array" && len(ref.TypeArgs) > 0 {
		elem, err := GoTypeRefName(ref.TypeArgs[0], table, goTypeNameFn)
		if err != nil {
			return "", err
		}
		if ref.ArraySize != nil {
			return fmt.Sprintf("[%d]%s", *ref.ArraySize, elem), nil
		}
		return "[]" + elem, nil
	}
	if ref.Name == "Map" && len(ref.TypeArgs) >= 2 {
		key, err := GoTypeRefName(ref.TypeArgs[0], table, goTypeNameFn)
		if err != nil {
			return "", err
		}
		val, err := GoTypeRefName(ref.TypeArgs[1], table, goTypeNameFn)
		if err != nil {
			return "", err
		}
		return "map[" + key + "]" + val, nil
	}
	resolved, ok := ref.Resolve(table)
	if !ok {
		return "", errors.Newf("cannot resolve type %s", ref.Name)
	}
	name, err := goTypeNameFn(resolved)
	if err != nil {
		return "", err
	}
	switch resolved.Form.(type) {
	case resolution.StructForm, resolution.AliasForm:
	default:
		return name, nil
	}
	params := resolution.TypeParams(resolved.Form)
	var args []string
	for i, arg := range ref.TypeArgs {
		if i < len(params) && params[i].HasDefault() {
			continue
		}
		rendered, err := GoTypeRefName(arg, table, goTypeNameFn)
		if err != nil {
			return "", err
		}
		args = append(args, rendered)
	}
	if len(args) == 0 {
		return name, nil
	}
	return name + "[" + strings.Join(args, ", ") + "]", nil
}

// ResolveLeafPrimitive walks through distinct and alias types to find the
// underlying primitive name and, for distinct types, the Go type cast needed.
// This is the unified version of encoderBuilder.resolveLeaf and
// testValueBuilder.resolveLeafPrim.
func ResolveLeafPrimitive(
	typ resolution.Type, table *resolution.Table,
	goTypeNameFn func(resolution.Type) (string, error),
) (primName, goTypeCast string, err error) {
	switch form := typ.Form.(type) {
	case resolution.PrimitiveForm:
		return form.Name, "", nil
	case resolution.DistinctForm:
		base, ok := form.Base.Resolve(table)
		if !ok {
			return "", "", errors.Newf(
				"cannot resolve distinct base %s",
				form.Base.Name,
			)
		}
		basePrim, _, err := ResolveLeafPrimitive(base, table, goTypeNameFn)
		if err != nil {
			return "", "", err
		}
		goType, err := goTypeNameFn(typ)
		if err != nil {
			return "", "", err
		}
		return basePrim, goType, nil
	case resolution.EnumForm:
		goType, err := goTypeNameFn(typ)
		if err != nil {
			return "", "", err
		}
		if form.IsIntEnum {
			return "int64", goType, nil
		}
		return "string", goType, nil
	case resolution.AliasForm:
		target, ok := form.Target.Resolve(table)
		if !ok {
			return "", "", errors.Newf(
				"cannot resolve alias target %s",
				form.Target.Name,
			)
		}
		basePrim, baseCast, err := ResolveLeafPrimitive(target, table, goTypeNameFn)
		if err != nil {
			return "", "", err
		}
		if baseCast != "" {
			goType, err := goTypeNameFn(typ)
			if err != nil {
				return "", "", err
			}
			return basePrim, goType, nil
		}
		return basePrim, "", nil
	default:
		return "", "", errors.Newf(
			"unsupported type form for leaf: %T (%s)",
			form,
			typ.QualifiedName,
		)
	}
}

// ResolveGoSliceElemType renders the Go type of a slice element ref, through alias
// chains. Handles nested arrays (e.g., [][]string).
//
// Distinct types are NOT unwrapped past their declared name: a distinct
// channel.Key over uint32 resolves as "channel.Key", because `make([]X, n)`
// must be assignable to the field's slice type and the field uses the distinct
// name. The one exception is when a distinct type wraps an Array directly
// (e.g., `type Tags = string[]` distinct), where we still recurse into the
// element so nested arrays decode correctly.
func ResolveGoSliceElemType(
	ref resolution.TypeRef, table *resolution.Table,
	goTypeNameFn func(resolution.Type) (string, error),
) (string, error) {
	for {
		resolved, ok := ref.Resolve(table)
		var target resolution.TypeRef
		isWrapper := false
		if ok {
			target, isWrapper = wrappedTypeRef(resolved, ref)
		}
		if !isWrapper {
			return GoTypeRefName(ref, table, goTypeNameFn)
		}
		if target.Name == "Array" && len(target.TypeArgs) > 0 {
			inner, err := ResolveGoSliceElemType(
				target.TypeArgs[0],
				table,
				goTypeNameFn,
			)
			if err != nil {
				return "", err
			}
			return "[]" + inner, nil
		}
		if _, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
			return GoTypeRefName(ref, table, goTypeNameFn)
		}
		ref = target
	}
}

// TypeParamConstraint renders a type parameter's Go constraint: the builtin
// constraint name when one is declared, any otherwise.
func TypeParamConstraint(tp resolution.TypeParam) string {
	if tp.Constraint != nil && resolution.IsConstraint(tp.Constraint.Name) {
		return tp.Constraint.Name
	}
	return "any"
}
