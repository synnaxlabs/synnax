// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// ReferenceID names a reference: the table whose entries hold it, and its name.
type ReferenceID struct {
	// Table is the name of the table whose entries hold the reference.
	Table string
	// Name is the name of the reference.
	Name string
}

// ReferenceInfo describes a reference that points at a table.
type ReferenceInfo struct {
	// ID names the reference.
	ID ReferenceID
	// OnDelete is what the delete of a target does to the entries that hold it.
	OnDelete OnDelete
	// Kind is the kind of the link from each holder to its targets.
	Kind LinkKind
	// Retargets is true when Table.Retarget can move the reference's holders.
	Retargets bool
}

// References returns each reference that points at the table named table, in the order
// that their tables opened.
func (db *DB) References(table string) []ReferenceInfo {
	refs := db.registry.referencesTo(table)
	out := make([]ReferenceInfo, len(refs))
	for i, r := range refs {
		out[i] = ReferenceInfo{
			ID:        referenceID(r),
			OnDelete:  r.onDeletePolicy(),
			Kind:      r.linkKind(),
			Retargets: r.canRetarget(),
		}
	}
	return out
}

// referenceID returns the ID of r.
func referenceID(r registeredReference) ReferenceID {
	return ReferenceID{Table: r.ownerTable(), Name: r.referenceName()}
}
