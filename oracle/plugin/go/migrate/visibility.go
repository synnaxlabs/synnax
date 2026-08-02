// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// visibility.go decides whether a type's developer migrate wrapper is exported.
package migrate

import (
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// wrapperNames decides each type's developer-wrapper name. A wrapper is
// exported when another versioned resource references the type: its migration
// is a cross-package contract. It is unexported when consumption is
// same-package only: a @go migrate entry's gorp wiring, or references from the
// package's own changed types. A type nothing consumes keeps the exported name
// so a generated wrapper is never an unused unexported function.
func wrapperNames(
	diff map[string]schemadiff.TypeDiff,
	oldTable, newTable *resolution.Table,
) map[string]string {
	external := make(set.Set[string])
	local := make(set.Set[string])
	for _, table := range []*resolution.Table{oldTable, newTable} {
		if table == nil {
			continue
		}
		for _, referrer := range table.Types {
			refPath := output.GetPath(referrer, "go")
			if refPath == "" {
				continue
			}
			// Unversioned resources never generate migrations, so their
			// references consume nothing.
			if _, versioned := versioning.Version(referrer); !versioned {
				continue
			}
			td, hasDiff := diff[referrer.QualifiedName]
			changed := hasDiff && td.Kind == schemadiff.TypeChanged
			walkRefs(referrer, table, func(target resolution.Type) {
				targetPath := output.GetPath(target, "go")
				if targetPath == "" {
					return
				}
				if targetPath != refPath {
					external.Add(target.QualifiedName)
				} else if changed && target.QualifiedName != referrer.QualifiedName {
					local.Add(target.QualifiedName)
				}
			})
		}
	}
	names := make(map[string]string)
	for _, table := range []*resolution.Table{oldTable, newTable} {
		if table == nil {
			continue
		}
		for _, typ := range table.Types {
			qn := typ.QualifiedName
			if _, done := names[qn]; done {
				continue
			}
			goName := naming.GetGoName(typ)
			entry := false
			if nt, ok := newTable.Get(qn); ok {
				entry = isMigrateEntry(nt)
			}
			if !external.Contains(qn) && (entry || local.Contains(qn)) {
				names[qn] = "migrate" + goName
			} else {
				names[qn] = "Migrate" + goName
			}
		}
	}
	return names
}

// wrapperNameFor returns the decided wrapper name for qn, defaulting to the
// exported form when the map has no entry.
func wrapperNameFor(wrappers map[string]string, qn, goName string) string {
	if n, ok := wrappers[qn]; ok {
		return n
	}
	return "Migrate" + goName
}

// walkRefs visits every type a migration of typ references: field and extends
// targets, and the element types of arrays. Fields with @go marshal omit never
// reach the wire and are skipped.
func walkRefs(
	typ resolution.Type,
	table *resolution.Table,
	visit func(resolution.Type),
) {
	switch form := typ.Form.(type) {
	case resolution.StructForm:
		for _, ext := range form.Extends {
			visitRef(ext, table, visit)
		}
		for _, f := range form.Fields {
			if domain.GetStringFromField(f, "go", "marshal") == "omit" {
				continue
			}
			visitRef(f.Type, table, visit)
		}
	case resolution.AliasForm:
		visitRef(form.Target, table, visit)
	case resolution.DistinctForm:
		visitRef(form.Base, table, visit)
	}
}

// visitRef resolves ref and its type arguments, visiting each resolved type
// and the element types of array forms.
func visitRef(
	ref resolution.TypeRef,
	table *resolution.Table,
	visit func(resolution.Type),
) {
	for _, arg := range ref.TypeArgs {
		visitRef(arg, table, visit)
	}
	resolved, ok := ref.Resolve(table)
	if !ok {
		return
	}
	visit(resolved)
	if elemRef, isArr := arrayBaseRef(resolved); isArr {
		visitRef(elemRef, table, visit)
	}
}
