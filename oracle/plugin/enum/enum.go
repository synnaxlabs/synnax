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
// Returns a deduplicated slice based on QualifiedName.
func CollectReferenced(structs []resolution.Struct) []resolution.Enum {
	seen := make(map[string]bool)
	var enums []resolution.Enum
	for _, s := range structs {
		for _, f := range s.Fields {
			if f.TypeRef.Kind == resolution.TypeKindEnum && f.TypeRef.EnumRef != nil {
				if !seen[f.TypeRef.EnumRef.QualifiedName] {
					seen[f.TypeRef.EnumRef.QualifiedName] = true
					enums = append(enums, *f.TypeRef.EnumRef)
				}
			}
		}
	}
	return enums
}

// FindOutputPath finds the output path for an enum.
// First checks if the enum has its own output domain, then falls back to
// searching for a struct in the same namespace that has an output domain.
// domainName specifies which domain to look up (e.g., "ts", "py").
func FindOutputPath(e resolution.Enum, table *resolution.Table, domainName string) string {
	if path := output.GetEnumPath(e, domainName); path != "" {
		return path
	}
	for _, s := range table.AllStructs() {
		if s.Namespace == e.Namespace {
			if path := output.GetPath(s, domainName); path != "" {
				return path
			}
		}
	}
	return ""
}

// CollectWithOwnOutput collects enums that have their own output domain defined.
// These are standalone enums not just referenced by structs.
func CollectWithOwnOutput(allEnums []resolution.Enum, domainName string) []resolution.Enum {
	var result []resolution.Enum
	for _, e := range allEnums {
		if output.GetEnumPath(e, domainName) != "" && !output.IsEnumOmitted(e, domainName) {
			result = append(result, e)
		}
	}
	return result
}

// FindPBOutputPath finds the pb output path for an enum using the new pb/ pattern.
// Derives from @go output + "/pb/" for structs in the same namespace.
func FindPBOutputPath(e resolution.Enum, table *resolution.Table) string {
	// Check if enum's namespace has a struct with @pb flag
	for _, s := range table.AllStructs() {
		if s.Namespace == e.Namespace {
			if path := output.GetPBPath(s); path != "" {
				return path
			}
		}
	}
	return ""
}
