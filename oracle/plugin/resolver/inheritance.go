// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolver

import "github.com/synnaxlabs/oracle/resolution"

// HasFieldConflicts returns true if multiple parents have overlapping field names.
// This is used to determine if language inheritance/embedding can be used safely.
// When field names conflict across parents, inheritance cannot be used and fields
// must be flattened into the child struct.
func HasFieldConflicts(extends []resolution.TypeRef, table *resolution.Table) bool {
	if len(extends) < 2 {
		return false
	}
	seen := make(map[string]bool)
	for _, ext := range extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			continue
		}
		for _, f := range resolution.UnifiedFields(parent, table) {
			if seen[f.Name] {
				return true
			}
			seen[f.Name] = true
		}
	}
	return false
}

// CanUseInheritance checks if a struct can use language inheritance/embedding.
// Returns false if:
// - There are no parent types (Extends is empty)
// - There are omitted fields (can't omit fields with inheritance)
// - There are field name conflicts between parents
func CanUseInheritance(form resolution.StructForm, table *resolution.Table) bool {
	if len(form.Extends) == 0 {
		return false
	}
	if len(form.OmittedFields) > 0 {
		return false // Can't omit fields with inheritance
	}
	return !HasFieldConflicts(form.Extends, table)
}
