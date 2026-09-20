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
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/resolution"
)

// wrapperNames decides each type's developer-wrapper name: a @go migrate entry's
// auto-copy is same-package gorp wiring and stays unexported; every other type keeps
// the exported name so a generated wrapper is never an unused unexported function.
// counterpart maps a qualified name in oldTable to the name newTable files the same
// type under.
func wrapperNames(
	diff map[string]schemadiff.TypeDiff,
	oldTable, newTable *resolution.Table,
	counterpart func(string) string,
) map[string]string {
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
			if nt, ok := newTable.Get(counterpart(qn)); ok {
				entry = isMigrateEntry(nt)
			}
			if entry {
				names[qn] = "migrate" + goName
			} else {
				names[qn] = "Migrate" + goName
			}
		}
	}
	return names
}
