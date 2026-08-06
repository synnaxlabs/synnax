// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

// checkOptionalDefaultInvariant enforces that a field is never both nullable (`?`) and
// carrying a static default. The two are mutually exclusive models for an absent value:
// a nullable field derives its default from absence (null/None), while a defaulted
// field is required and value-filled. Declaring both is contradictory, so the schema is
// rejected at analysis time. The check is structural and applies to every field type,
// not just the cases checkDefaultInvariant can settle.
func checkOptionalDefaultInvariant(c *analysisCtx) {
	for _, typ := range c.table.Types {
		if typ.Namespace != c.namespace {
			continue
		}
		form, ok := typ.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			if f.Default == nil || !f.Optional {
				continue
			}
			c.diag.Add(diagnostics.Errorf(
				nil,
				"field %q in %q is both nullable (`?`) and has a default; the two are "+
					"mutually exclusive. Drop the `?` to make it a required defaulted "+
					"field, or drop the default to make it a plain nullable field.",
				f.Name, typ.Name,
			))
		}
	}
}

// checkDefaultInvariant enforces the default invariant: a required field that carries a
// static default must either default to its type's zero value, or have a type whose
// zero value is not a valid value. When neither holds, value-based default-filling
// could overwrite a deliberate zero (a `precision` of 0, a `visible` of false), so the
// schema is rejected at analysis time.
//
// The check is conservative. It decides only the cases it can settle without
// introspecting validation bounds (booleans and integer enums) and abstains otherwise,
// so it never reports a false positive. Numeric and string bounds are handled in a
// follow-up.
func checkDefaultInvariant(c *analysisCtx) {
	for _, typ := range c.table.Types {
		if typ.Namespace != c.namespace {
			continue
		}
		form, ok := typ.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			// Optional fields are exempt: their absence is itself meaningful, so
			// no static default is overlaid onto a zero value.
			if f.Default == nil || f.Optional {
				continue
			}
			reason, violates := defaultInvariantViolation(f, c.table)
			if !violates {
				continue
			}
			c.diag.Add(diagnostics.Errorf(
				nil,
				"field %q in %q violates the default invariant: %s. Make the field "+
					"optional with `?`, add a bound that excludes the zero value, or set "+
					"the default to the type's zero value.",
				f.Name, typ.Name, reason,
			))
		}
	}
}

// defaultInvariantViolation reports whether a required field's static default is a
// non-zero value whose zero is itself valid. It returns ("", false) for any case it
// cannot decide without validation bounds, so the caller never raises a false positive.
func defaultInvariantViolation(
	f resolution.Field,
	table *resolution.Table,
) (string, bool) {
	switch f.Default.Kind {
	case resolution.ValueKindBool:
		// false is the bool zero value and a valid value, so a `true` default can
		// never coincide with the zero value.
		if f.Default.BoolValue {
			return "a bool default of `true` is non-zero, and `false` is a valid value", true
		}
	case resolution.ValueKindIdent:
		ev, ok := validation.ResolveEnumVariant(f.Default.IdentValue, f.Type, table)
		if !ok {
			// Not an enum variant (e.g. a `create` or `now` boundary default), so it
			// is not a static default subject to the invariant.
			return "", false
		}
		form, ok := ev.Type.Form.(resolution.EnumForm)
		if !ok || !form.IsIntEnum {
			// A string enum has no zero member (its zero is ""), so any default is
			// safe.
			return "", false
		}
		if len(form.Values) > 0 && form.Values[0].Name != ev.Variant.Name {
			return "an integer-enum default that is not the zeroth member leaves the " +
				"zeroth member (the zero value) as a valid, distinct value", true
		}
	}
	return "", false
}
