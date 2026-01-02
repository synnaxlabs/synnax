// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package enum provides utilities for working with enums in oracle schemas.
package enum

import (
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

// CollectReferenced collects unique enums referenced by struct fields.
// Returns a deduplicated slice of enum types based on QualifiedName.
func CollectReferenced(structs []resolution.Type, table *resolution.Table) []resolution.Type {
	seen := make(map[string]bool)
	var enums []resolution.Type
	for _, s := range structs {
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			collectEnumsFromTypeRef(f.Type, table, seen, &enums)
		}
	}
	return enums
}

// collectEnumsFromTypeRef recursively collects enums from a type reference.
func collectEnumsFromTypeRef(ref resolution.TypeRef, table *resolution.Table, seen map[string]bool, enums *[]resolution.Type) {
	// Check type args first (for generic types like Array<EnumType>)
	for _, arg := range ref.TypeArgs {
		collectEnumsFromTypeRef(arg, table, seen, enums)
	}

	// Try to resolve the type
	resolved, ok := ref.Resolve(table)
	if !ok {
		return
	}
	if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		if !seen[resolved.QualifiedName] {
			seen[resolved.QualifiedName] = true
			*enums = append(*enums, resolved)
		}
	}
}

// FindOutputPath finds the output path for an enum type.
// First checks if the enum has its own output domain, then falls back to
// searching for a struct in the same namespace that has an output domain.
// domainName specifies which domain to look up (e.g., "ts", "py").
func FindOutputPath(e resolution.Type, table *resolution.Table, domainName string) string {
	if path := output.GetPath(e, domainName); path != "" {
		return path
	}
	for _, s := range table.StructTypes() {
		if s.Namespace == e.Namespace {
			if path := output.GetPath(s, domainName); path != "" {
				return path
			}
		}
	}
	return ""
}

// CollectWithOwnOutput collects enum types that have their own output domain defined.
// These are standalone enums not just referenced by structs.
func CollectWithOwnOutput(allEnums []resolution.Type, domainName string) []resolution.Type {
	var result []resolution.Type
	for _, e := range allEnums {
		if output.GetPath(e, domainName) != "" && !output.IsOmitted(e, domainName) {
			result = append(result, e)
		}
	}
	return result
}

// FindPBOutputPath finds the pb output path for an enum using the new pb/ pattern.
// Derives from @go output + "/pb/" for structs in the same namespace.
func FindPBOutputPath(e resolution.Type, table *resolution.Table) string {
	for _, s := range table.StructTypes() {
		if s.Namespace == e.Namespace {
			if path := output.GetPBPath(s); path != "" {
				return path
			}
		}
	}
	return ""
}
