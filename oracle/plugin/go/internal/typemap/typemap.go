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

// UnwrapTypeRef resolves through alias and distinct type ref wrappers,
// tracking effective type arguments as it goes. Returns the final resolved
// type and the effective type arguments at that level.
func UnwrapTypeRef(
	resolved resolution.Type, ref resolution.TypeRef, table *resolution.Table,
) (resolution.Type, []resolution.TypeRef) {
	actual := resolved
	effectiveTypeArgs := ref.TypeArgs
	for {
		switch form := actual.Form.(type) {
		case resolution.AliasForm:
			target, ok := form.Target.Resolve(table)
			if !ok {
				return actual, effectiveTypeArgs
			}
			if len(form.Target.TypeArgs) > 0 {
				effectiveTypeArgs = form.Target.TypeArgs
			}
			actual = target
			continue
		case resolution.DistinctForm:
			target, ok := form.Base.Resolve(table)
			if !ok {
				return actual, effectiveTypeArgs
			}
			if len(form.Base.TypeArgs) > 0 {
				effectiveTypeArgs = form.Base.TypeArgs
			}
			actual = target
			continue
		default:
		}
		break
	}
	return actual, effectiveTypeArgs
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
			return "", "", errors.Newf("cannot resolve distinct base %s", form.Base.Name)
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
			return "", "", errors.Newf("cannot resolve alias target %s", form.Target.Name)
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
		return "", "", errors.Newf("unsupported type form for leaf: %T (%s)", form, typ.QualifiedName)
	}
}

// ResolveGoSliceElemType resolves through alias/distinct chains to find the
// Go type string for a slice element. Handles nested arrays (e.g., [][]string).
func ResolveGoSliceElemType(
	typ resolution.Type, table *resolution.Table,
	goTypeNameFn func(resolution.Type) (string, error),
) (string, error) {
	actual := typ
	for {
		var baseRef resolution.TypeRef
		switch form := actual.Form.(type) {
		case resolution.AliasForm:
			baseRef = form.Target
		case resolution.DistinctForm:
			baseRef = form.Base
		default:
			return goTypeNameFn(actual)
		}
		target, ok := baseRef.Resolve(table)
		if !ok {
			return "", errors.Newf("cannot resolve type %s", baseRef.Name)
		}
		if bg, ok := target.Form.(resolution.BuiltinGenericForm); ok && bg.Name == "Array" && len(baseRef.TypeArgs) > 0 {
			innerElem, ok := baseRef.TypeArgs[0].Resolve(table)
			if !ok {
				return "", errors.Newf("cannot resolve array element %s", baseRef.TypeArgs[0].Name)
			}
			innerGoType, err := ResolveGoSliceElemType(innerElem, table, goTypeNameFn)
			if err != nil {
				return "", err
			}
			return "[]" + innerGoType, nil
		}
		actual = target
	}
}
