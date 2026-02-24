// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package diff

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/oracle/resolution"
)

// ChangeKind classifies how a field changed between two schema versions.
type ChangeKind int

const (
	FieldUnchanged ChangeKind = iota
	FieldAdded
	FieldRemoved
	FieldTypeChanged
)

// FieldDiff describes the change to a single field between schema versions.
type FieldDiff struct {
	Name    string
	Kind    ChangeKind
	OldType string
	NewType string
}

// TypeDiff describes the changes to a struct type between schema versions.
type TypeDiff struct {
	TypeName string
	Changed  bool
	Fields   []FieldDiff
}

// FormatTypeRef produces a canonical string representation of a TypeRef
// suitable for equality comparison.
func FormatTypeRef(ref resolution.TypeRef) string {
	var b strings.Builder
	b.WriteString(ref.Name)
	if len(ref.TypeArgs) > 0 {
		b.WriteString("<")
		for i, arg := range ref.TypeArgs {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(FormatTypeRef(arg))
		}
		b.WriteString(">")
	}
	if ref.ArraySize != nil {
		fmt.Fprintf(&b, "[%d]", *ref.ArraySize)
	}
	return b.String()
}

// DiffStructs compares two versions of the same struct type and returns a
// TypeDiff describing which fields were added, removed, or changed.
func DiffStructs(
	old, new resolution.Type,
	oldTable, newTable *resolution.Table,
) TypeDiff {
	oldFields := resolution.UnifiedFields(old, oldTable)
	newFields := resolution.UnifiedFields(new, newTable)

	oldMap := make(map[string]resolution.Field, len(oldFields))
	for _, f := range oldFields {
		oldMap[f.Name] = f
	}
	newMap := make(map[string]resolution.Field, len(newFields))
	for _, f := range newFields {
		newMap[f.Name] = f
	}

	td := TypeDiff{TypeName: old.QualifiedName}

	// Check old fields: unchanged or removed or type changed
	for _, of := range oldFields {
		nf, exists := newMap[of.Name]
		if !exists {
			td.Fields = append(td.Fields, FieldDiff{
				Name:    of.Name,
				Kind:    FieldRemoved,
				OldType: FormatTypeRef(of.Type),
			})
			td.Changed = true
			continue
		}
		oldTypeStr := FormatTypeRef(of.Type)
		newTypeStr := FormatTypeRef(nf.Type)
		if oldTypeStr != newTypeStr {
			td.Fields = append(td.Fields, FieldDiff{
				Name:    of.Name,
				Kind:    FieldTypeChanged,
				OldType: oldTypeStr,
				NewType: newTypeStr,
			})
			td.Changed = true
		} else {
			td.Fields = append(td.Fields, FieldDiff{
				Name: of.Name,
				Kind: FieldUnchanged,
			})
		}
	}

	// Check new fields not in old: added
	for _, nf := range newFields {
		if _, exists := oldMap[nf.Name]; !exists {
			td.Fields = append(td.Fields, FieldDiff{
				Name:    nf.Name,
				Kind:    FieldAdded,
				NewType: FormatTypeRef(nf.Type),
			})
			td.Changed = true
		}
	}

	return td
}

// DiffTables compares all struct types across two resolution tables and returns
// TypeDiffs for types that have changed. Types present only in the new table
// are treated as added (not returned as diffs). Types present only in the old
// table are treated as removed and included in the result.
func DiffTables(old, new *resolution.Table) []TypeDiff {
	oldStructs := old.StructTypes()
	newStructs := new.StructTypes()

	newMap := make(map[string]resolution.Type, len(newStructs))
	for _, t := range newStructs {
		newMap[t.QualifiedName] = t
	}
	oldMap := make(map[string]resolution.Type, len(oldStructs))
	for _, t := range oldStructs {
		oldMap[t.QualifiedName] = t
	}

	var diffs []TypeDiff

	for _, ot := range oldStructs {
		nt, exists := newMap[ot.QualifiedName]
		if !exists {
			diffs = append(diffs, TypeDiff{
				TypeName: ot.QualifiedName,
				Changed:  true,
				Fields: []FieldDiff{{
					Name: ot.QualifiedName,
					Kind: FieldRemoved,
				}},
			})
			continue
		}
		td := DiffStructs(ot, nt, old, new)
		if td.Changed {
			diffs = append(diffs, td)
		}
	}

	return diffs
}
