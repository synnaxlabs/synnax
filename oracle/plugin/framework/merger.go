// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package framework provides shared utilities for Oracle code generation plugins.
package framework

import "github.com/synnaxlabs/oracle/resolution"

// MergeTypes combines two type slices, deduplicating by QualifiedName.
// This is used to merge standalone enums with those referenced by structs,
// or to merge typedefs that share the same output path.
func MergeTypes(a, b []resolution.Type) []resolution.Type {
	seen := make(map[string]bool, len(a))
	for _, t := range a {
		seen[t.QualifiedName] = true
	}
	result := append([]resolution.Type{}, a...)
	for _, t := range b {
		if !seen[t.QualifiedName] {
			result = append(result, t)
			seen[t.QualifiedName] = true
		}
	}
	return result
}

// MergeTypesByName combines two type slices, deduplicating by Name.
// Some contexts (like Python) use Name instead of QualifiedName for deduplication.
func MergeTypesByName(a, b []resolution.Type) []resolution.Type {
	seen := make(map[string]bool, len(a))
	for _, t := range a {
		seen[t.Name] = true
	}
	result := append([]resolution.Type{}, a...)
	for _, t := range b {
		if !seen[t.Name] {
			result = append(result, t)
			seen[t.Name] = true
		}
	}
	return result
}
