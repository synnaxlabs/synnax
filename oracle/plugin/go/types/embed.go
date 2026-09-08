// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// CanEmbed reports whether a struct's parents can be Go-embedded rather than flattened
// into the child. Beyond the conditions resolver.CanUseInheritance covers, embedding
// fails when two parents promote the same name, because Go rejects an ambiguous
// promoted name both as a selector and as a composite literal key.
func CanEmbed(form resolution.StructForm, table *resolution.Table) bool {
	if !resolver.CanUseInheritance(form, table) {
		return false
	}
	promoted := make(set.Set[string])
	for _, ext := range form.Extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			continue
		}
		for _, name := range promotedGoNames(parent, table) {
			if promoted.Contains(name) {
				return false
			}
			promoted.Add(name)
		}
	}
	return true
}

// promotedGoNames returns the names a parent promotes into a struct that embeds it. An
// embedded parent promotes its own fields and the type names of the parents it embeds
// in turn; a flattened one promotes its whole unified field list.
func promotedGoNames(parent resolution.Type, table *resolution.Table) []string {
	form, ok := parent.Form.(resolution.StructForm)
	if !ok {
		return nil
	}
	if !CanEmbed(form, table) {
		fields := resolution.UnifiedFields(parent, table)
		names := make([]string, 0, len(fields))
		for _, f := range fields {
			names = append(names, naming.GetFieldName(f))
		}
		return names
	}
	defaultOnly := resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)
	names := make([]string, 0, len(form.Extends)+len(form.Fields))
	for _, ext := range form.Extends {
		if grandparent, ok := ext.Resolve(table); ok {
			names = append(names, naming.GetGoName(grandparent))
		}
	}
	for _, f := range form.Fields {
		if !defaultOnly.Contains(f.Name) {
			names = append(names, naming.GetFieldName(f))
		}
	}
	return names
}
