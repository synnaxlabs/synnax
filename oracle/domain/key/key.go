// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package key provides utilities for extracting key fields from Oracle schemas.
// Key fields are marked with the @key domain and serve as primary identifiers.
package key

import "github.com/synnaxlabs/oracle/resolution"

// Field represents a key field extracted from a struct.
type Field struct {
	Name      string
	Primitive string
}

// SkipFunc is a predicate that determines whether to skip a struct when collecting keys.
type SkipFunc func(resolution.Struct) bool

// Collect gathers all unique key fields from the given structs.
// Fields marked with @key domain are collected, and the skip function can
// be used to exclude certain structs from collection.
func Collect(structs []resolution.Struct, skip SkipFunc) []Field {
	seen := make(map[string]bool)
	var result []Field
	for _, s := range structs {
		if skip != nil && skip(s) {
			continue
		}
		for _, f := range s.Fields {
			if _, hasKey := f.Domains["key"]; hasKey {
				if !seen[f.Name] {
					seen[f.Name] = true
					result = append(result, Field{
						Name:      f.Name,
						Primitive: resolvePrimitive(f.TypeRef),
					})
				}
			}
		}
	}
	return result
}

// resolvePrimitive extracts the underlying primitive from a TypeRef,
// following TypeDef references to their base types.
func resolvePrimitive(typeRef *resolution.TypeRef) string {
	if typeRef == nil {
		return ""
	}
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		return typeRef.Primitive
	case resolution.TypeKindTypeDef:
		if typeRef.TypeDefRef != nil && typeRef.TypeDefRef.BaseType != nil {
			return resolvePrimitive(typeRef.TypeDefRef.BaseType)
		}
	}
	return ""
}
