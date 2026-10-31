// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package omit provides utilities for detecting omitted types.
package omit

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// IsType returns true if the type has an "omit" expression in the domain: the
// type does not exist in the domain's language at all.
func IsType(typ resolution.Type, domainName string) bool {
	return hasExpr(typ, domainName, "omit")
}

// IsHand returns true if the type has a "hand" expression in the domain: the
// type exists in the language, hand-written at its declared output path.
// References to it resolve normally; only its declaration is not generated.
func IsHand(typ resolution.Type, domainName string) bool {
	return hasExpr(typ, domainName, "hand")
}

// IsSkipped returns true when no declaration is generated for the type in the
// domain: the type is either omitted or hand-written.
func IsSkipped(typ resolution.Type, domainName string) bool {
	return IsType(typ, domainName) || IsHand(typ, domainName)
}

// IsAbsent reports whether a struct does not exist in the domain's language: it omits
// the language, or has no output there and lives outside a version file.
func IsAbsent(typ resolution.Type, domainName string) bool {
	if IsType(typ, domainName) {
		return true
	}
	if domain, ok := typ.Domains[domainName]; ok {
		if expr, ok := domain.Expressions.Find("output"); ok && len(expr.Values) > 0 {
			return false
		}
	}
	_, _, inVersionFile := paths.VersionFile(typ.FilePath)
	return !inVersionFile
}

// IsField reports whether the field carries a field-level @<domainName> omit: it does
// not exist in that language's generated type.
func IsField(f resolution.Field, domainName string) bool {
	domain, ok := f.Domains[domainName]
	if !ok {
		return false
	}
	_, ok = domain.Expressions.Find("omit")
	return ok
}

// Fields returns a copy of table whose struct types and action payloads drop every
// field that IsField reports for any of domainNames. A field inherits an omit from the
// parent field it overrides. table is not modified.
func Fields(table *resolution.Table, domainNames ...string) *resolution.Table {
	omitted := func(f resolution.Field) bool {
		return lo.ContainsBy(domainNames, func(d string) bool { return IsField(f, d) })
	}
	stripped := &resolution.Table{
		Imports: table.Imports,
		Types:   make([]resolution.Type, len(table.Types)),
	}
	for i, typ := range table.Types {
		if form, ok := typ.Form.(resolution.StructForm); ok {
			dropped := make(set.Set[string])
			for _, f := range resolution.UnifiedFields(typ, table) {
				if omitted(f) {
					dropped.Add(f.Name)
				}
			}
			form.Fields = lo.Reject(form.Fields, func(f resolution.Field, _ int) bool {
				return dropped.Contains(f.Name)
			})
			actions := make([]resolution.Action, len(form.Actions))
			for j, a := range form.Actions {
				a.Fields = lo.Reject(a.Fields, func(f resolution.Field, _ int) bool {
					return omitted(f)
				})
				actions[j] = a
			}
			form.Actions = actions
			typ.Form = form
		}
		stripped.Types[i] = typ
	}
	return stripped
}

func hasExpr(typ resolution.Type, domainName, name string) bool {
	domain, ok := typ.Domains[domainName]
	if !ok {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == name {
			return true
		}
	}
	return false
}
