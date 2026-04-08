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

import (
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// MergeTypes combines two type slices, deduplicating by QualifiedName.
// This is used to merge standalone enums with those referenced by structs,
// or to merge typedefs that share the same output path.
func MergeTypes(a, b []resolution.Type) []resolution.Type {
	seen := make(set.Set[string], len(a))
	for _, t := range a {
		seen.Add(t.QualifiedName)
	}
	result := append([]resolution.Type{}, a...)
	for _, t := range b {
		if !seen.Contains(t.QualifiedName) {
			result = append(result, t)
			seen.Add(t.QualifiedName)
		}
	}
	return result
}

// MergeTypesByName combines two type slices, deduplicating by Name.
// Some contexts (like Python) use Name instead of QualifiedName for deduplication.
func MergeTypesByName(a, b []resolution.Type) []resolution.Type {
	seen := make(set.Set[string], len(a))
	for _, t := range a {
		seen.Add(t.Name)
	}
	result := append([]resolution.Type{}, a...)
	for _, t := range b {
		if !seen.Contains(t.Name) {
			result = append(result, t)
			seen.Add(t.Name)
		}
	}
	return result
}
