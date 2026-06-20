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
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

// synthesizeCreateTypes derives a `New` input type for every struct marked `@create`.
// The derived type extends the base (mirroring its generic type parameters) and omits
// every `@output` field, and carries `@ts use_input` plus `omit` for Go, Python,
// protobuf, and C++, so the existing New-struct codegen produces the input projection
// only where it adds value: a `z.input` type in TypeScript, and nothing elsewhere (where
// the base struct is reused, since its defaulted fields are already optional at
// construction).
//
// Defaulted fields become optional automatically through `z.input`, so they need no
// per-field handling here. If a hand-written `New` already exists in the namespace it is
// respected and nothing is synthesized.
func synthesizeCreateTypes(c *analysisCtx) {
	var created []resolution.Type
	for _, typ := range c.table.Types {
		if typ.Namespace != c.namespace {
			continue
		}
		if _, ok := typ.Domains["create"]; !ok {
			continue
		}
		form, ok := typ.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		if _, exists := c.table.Get(c.namespace + ".New"); exists {
			continue
		}
		var omitted []string
		for _, f := range form.Fields {
			if _, ok := f.Domains["output"]; ok {
				omitted = append(omitted, f.Name)
			}
		}
		created = append(created, resolution.Type{
			Name:          "New",
			Namespace:     c.namespace,
			QualifiedName: c.namespace + ".New",
			FilePath:      c.filePath,
			Form: resolution.StructForm{
				TypeParams:    form.TypeParams,
				Extends:       []resolution.TypeRef{{Name: typ.QualifiedName}},
				OmittedFields: omitted,
			},
			Domains: newTypeDomains(typ),
		})
		// Only one New can exist per namespace; stop after synthesizing it so a
		// second @create struct in the same namespace can't collide on table.Add.
		break
	}
	for _, t := range created {
		if err := c.table.Add(t); err != nil {
			c.diag.Add(diagnostics.Errorf(nil, "failed to synthesize New type: %v", err))
		}
	}
}

// newTypeDomains builds the domains for a synthesized New. It inherits the base type's
// TypeScript output path (so the New lands in the same file) and adds `use_input` and
// `type_only` for TS, so the New emits only as an input-typed type referencing the base
// schema, never as its own runtime zod const. The New is omitted from Go, Python,
// protobuf, and C++, where the base struct is reused rather than a distinct input type:
// in those languages defaulted fields are already optional at construction, so a New
// projection adds nothing the base does not already provide.
//
// The base's `name` expression is deliberately dropped from the TS domain: the New is a
// distinct sibling that emits under its own name (`New`), so it must not adopt the base's
// renamed identifier (which would collide with the base).
func newTypeDomains(base resolution.Type) map[string]resolution.Domain {
	domains := map[string]resolution.Domain{
		"go":  {Name: "go", Expressions: resolution.Expressions{{Name: "omit"}}},
		"pb":  {Name: "pb", Expressions: resolution.Expressions{{Name: "omit"}}},
		"cpp": {Name: "cpp", Expressions: resolution.Expressions{{Name: "omit"}}},
		"py":  {Name: "py", Expressions: resolution.Expressions{{Name: "omit"}}},
	}
	tsExprs := inheritedNewExpressions(base, "ts")
	tsExprs = append(tsExprs, resolution.Expression{Name: "use_input"}, resolution.Expression{Name: "type_only"})
	domains["ts"] = resolution.Domain{Name: "ts", Expressions: tsExprs}
	return domains
}

// inheritedNewExpressions returns the base type's expressions for the given domain with
// the `name` expression removed, so a synthesized New inherits output paths, omission,
// and behavioral flags but emits under its own name rather than the base's. Returns an
// empty slice when the base has no such domain.
func inheritedNewExpressions(base resolution.Type, domain string) resolution.Expressions {
	d, ok := base.Domains[domain]
	if !ok {
		return resolution.Expressions{}
	}
	out := make(resolution.Expressions, 0, len(d.Expressions))
	for _, e := range d.Expressions {
		if e.Name == "name" {
			continue
		}
		out = append(out, e)
	}
	return out
}
