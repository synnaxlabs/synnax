// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package arrays answers array-shape questions about type references, chasing
// aliases and distinct bases. The .proto emitter and every protobuf translator
// derive wrapper-message names through this one package so they cannot drift.
package arrays

import (
	"github.com/synnaxlabs/oracle/resolution"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// IsArray reports whether the reference denotes an array, through aliases and
// distinct bases.
func IsArray(typeRef resolution.TypeRef, table *resolution.Table) bool {
	if typeRef.Name == "Array" {
		return true
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	switch form := resolved.Form.(type) {
	case resolution.BuiltinGenericForm:
		return form.Name == "Array"
	case resolution.AliasForm:
		return IsArray(form.Target, table)
	case resolution.DistinctForm:
		return IsArray(form.Base, table)
	default:
		return false
	}
}

// ElementType returns the array's element reference, through aliases and
// distinct bases.
func ElementType(
	typeRef resolution.TypeRef,
	table *resolution.Table,
) (resolution.TypeRef, bool) {
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return typeRef.TypeArgs[0], true
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return resolution.TypeRef{}, false
	}
	switch form := resolved.Form.(type) {
	case resolution.BuiltinGenericForm:
		if form.Name == "Array" && len(typeRef.TypeArgs) > 0 {
			return typeRef.TypeArgs[0], true
		}
		return resolution.TypeRef{}, false
	case resolution.AliasForm:
		return ElementType(form.Target, table)
	case resolution.DistinctForm:
		return ElementType(form.Base, table)
	default:
		return resolution.TypeRef{}, false
	}
}

// Size returns the declared fixed size, or nil for a dynamic array.
func Size(typeRef resolution.TypeRef, table *resolution.Table) *int64 {
	if typeRef.Name == "Array" && typeRef.ArraySize != nil {
		return typeRef.ArraySize
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return nil
	}
	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		return Size(form.Target, table)
	case resolution.DistinctForm:
		return Size(form.Base, table)
	default:
		return nil
	}
}

// IsFixedSizeUint8 reports whether the reference is a fixed-size uint8 array,
// which protobuf carries as bytes.
func IsFixedSizeUint8(typeRef resolution.TypeRef, table *resolution.Table) bool {
	if Size(typeRef, table) == nil {
		return false
	}
	elemType, ok := ElementType(typeRef, table)
	if !ok {
		return false
	}
	resolved, ok := elemType.Resolve(table)
	if !ok {
		return elemType.Name == "uint8"
	}
	if prim, ok := resolved.Form.(resolution.PrimitiveForm); ok {
		return prim.Name == "uint8"
	}
	return false
}

// IsNested reports whether the reference is an array of arrays.
func IsNested(typeRef resolution.TypeRef, table *resolution.Table) bool {
	if !IsArray(typeRef, table) {
		return false
	}
	elemType, ok := ElementType(typeRef, table)
	if !ok {
		return false
	}
	return IsArray(elemType, table)
}

// OptionalWrapperName derives the wrapper message name for an optional list
// (e.g. "Label" -> "LabelList"), distinct from the nested-array "Wrapper"
// suffix.
func OptionalWrapperName(
	typeRef resolution.TypeRef,
	table *resolution.Table,
) string {
	elemType, ok := ElementType(typeRef, table)
	if !ok {
		return "ListWrapper"
	}
	if resolved, ok := elemType.Resolve(table); ok {
		return resolved.Name + "List"
	}
	if resolution.IsPrimitive(elemType.Name) {
		return cases.Title(language.English).String(elemType.Name) + "List"
	}
	return "ListWrapper"
}

// NestedWrapperName derives the wrapper message name for a nested array's
// inner list.
func NestedWrapperName(
	typeRef resolution.TypeRef,
	table *resolution.Table,
) string {
	elemType, ok := ElementType(typeRef, table)
	if !ok {
		return "ArrayWrapper"
	}
	resolved, ok := elemType.Resolve(table)
	if ok {
		return resolved.Name + "Wrapper"
	}
	if resolution.IsPrimitive(elemType.Name) {
		return cases.Title(language.English).String(elemType.Name) + "Array"
	}
	return "ArrayWrapper"
}
