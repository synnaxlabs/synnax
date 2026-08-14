// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolver

import (
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// HasFieldConflicts returns true if multiple parents have overlapping field names.
// This is used to determine if language inheritance/embedding can be used safely.
// When field names conflict across parents, inheritance cannot be used and fields
// must be flattened into the child struct.
func HasFieldConflicts(extends []resolution.TypeRef, table *resolution.Table) bool {
	if len(extends) < 2 {
		return false
	}
	seen := make(set.Set[string])
	for _, ext := range extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			continue
		}
		for _, f := range resolution.UnifiedFields(parent, table) {
			if seen.Contains(f.Name) {
				return true
			}
			seen.Add(f.Name)
		}
	}
	return false
}

// HasDomainOmissions reports whether the struct removes a domain inherited from a
// parent field with `-@domain`. A removal cannot be expressed through language
// inheritance/embedding — the embedded parent still carries the domain — so the
// struct must be flattened to drop it. Typeless overrides are resolved into
// complete fields by the analyzer before generation, so they do not require this.
func HasDomainOmissions(form resolution.StructForm) bool {
	for _, f := range form.Fields {
		if len(f.OmittedDomains) > 0 {
			return true
		}
	}
	return false
}

// CanUseInheritance checks if a struct can use language inheritance/embedding.
// Returns false if:
// - There are no parent types (Extends is empty)
// - There are omitted fields (can't omit fields with inheritance)
// - There are field name conflicts between parents
// - A field removes an inherited domain (must flatten to drop it)
func CanUseInheritance(form resolution.StructForm, table *resolution.Table) bool {
	if len(form.Extends) == 0 {
		return false
	}
	if len(form.OmittedFields) > 0 {
		return false // Can't omit fields with inheritance
	}
	if HasDomainOmissions(form) {
		return false
	}
	return !HasFieldConflicts(form.Extends, table)
}
