// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package match holds the canonical naming convention for the generated
// service-package Match functions and the predicate metadata both the
// go/query plugin (which emits them) and the go/api plugin (which calls
// them) consume. Centralizing the convention here makes naming drift
// between the two generators a compile-time impossibility.
package match

import (
	"strings"

	"github.com/synnaxlabs/x/pluralize"
)

// Field describes the filter-relevant shape of a struct field for the
// purpose of deriving its Match function name. Both plugins build a Field
// from their own resolved view of the schema and pass it to {@link Func}.
type Field struct {
	// GoName is the field's PascalCase Go identifier (e.g. "Integrations").
	GoName string
	// IsKey is true for the @key field. Its Match function is always
	// "MatchKeys" regardless of GoName.
	IsKey bool
	// IsBool is true for bool primitives. Match functions for bool fields
	// take a single value and use the field name as-is.
	IsBool bool
	// IsSlice is true when the underlying field type is a slice. The
	// Match function semantically tests "list contains value" and uses
	// the singularized field name.
	IsSlice bool
	// IsScalar is true for @filter scalar fields (single-value match)
	// that are not slices. Match functions take a single value and use
	// the field name as-is.
	IsScalar bool
}

// Func returns the canonical Match function name for a filter field. The
// returned name is the public identifier emitted by go/query and called
// by go/api in the same service package.
func Func(f Field) string {
	if f.IsKey {
		return "MatchKeys"
	}
	if f.IsBool || f.IsScalar {
		return "Match" + f.GoName
	}
	if f.IsSlice {
		return "Match" + Singularize(f.GoName)
	}
	return "Match" + PluralizeDistinct(f.GoName)
}

// PluralizeDistinct returns the plural form of a name. If the plural is
// the same as the singular (already-plural words like "Properties"), it
// appends "List" to avoid colliding with the singular function name.
func PluralizeDistinct(name string) string {
	plural := pluralize.String(name)
	if plural == name {
		return name + "List"
	}
	return plural
}

// Singularize returns a best-effort singular form of a name. Handles the
// common English plural endings (-ies, -ches/-shes/-xes/-ses, -s); leaves
// the input unchanged if no rule matches.
func Singularize(name string) string {
	switch {
	case strings.HasSuffix(name, "ies") && len(name) > 3:
		return name[:len(name)-3] + "y"
	case strings.HasSuffix(name, "ches"),
		strings.HasSuffix(name, "shes"),
		strings.HasSuffix(name, "xes"),
		strings.HasSuffix(name, "ses"):
		return name[:len(name)-2]
	case strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") && len(name) > 1:
		return name[:len(name)-1]
	}
	return name
}
