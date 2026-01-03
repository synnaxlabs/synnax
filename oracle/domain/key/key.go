// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package key provides utilities for extracting key fields from Oracle schemas.
package key

import "github.com/synnaxlabs/oracle/resolution"

// Field represents a key field extracted from a struct.
type Field struct {
	Name      string
	Primitive string
	Generate  bool
}

// SkipFunc is a predicate that determines whether to skip a type when collecting keys.
type SkipFunc func(resolution.Type) bool

// Collect gathers all unique key fields from the given struct types.
func Collect(types []resolution.Type, table *resolution.Table, skip SkipFunc) []Field {
	seen := make(map[string]bool)
	var result []Field
	for _, typ := range types {
		if skip != nil && skip(typ) {
			continue
		}
		form, ok := typ.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			if _, hasKey := f.Domains["key"]; hasKey {
				if !seen[f.Name] {
					seen[f.Name] = true
					result = append(result, Field{
						Name:      f.Name,
						Primitive: ResolvePrimitive(f.Type, table),
					})
				}
			}
		}
	}
	return result
}

// HasKey checks if a field has the @key annotation.
func HasKey(field resolution.Field) bool {
	_, hasKey := field.Domains["key"]
	return hasKey
}

// HasGenerate checks if a field has @key generate annotation.
func HasGenerate(field resolution.Field) bool {
	domain, hasKey := field.Domains["key"]
	if !hasKey {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == "generate" {
			return true
		}
	}
	return false
}

// ResolvePrimitive extracts the underlying primitive from a TypeRef.
func ResolvePrimitive(ref resolution.TypeRef, table *resolution.Table) string {
	if ref.IsTypeParam() {
		return ""
	}
	if resolution.IsPrimitive(ref.Name) {
		return ref.Name
	}
	// Follow DistinctForm to base type
	if typ, ok := table.Get(ref.Name); ok {
		if form, ok := typ.Form.(resolution.DistinctForm); ok {
			return ResolvePrimitive(form.Base, table)
		}
	}
	return ""
}
