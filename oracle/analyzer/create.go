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

// synthesizeCreateTypes derives a `New` input type for every struct marked `@create`
// (RFC 0043 section 6). The derived type extends the base and omits every `@output`
// field, and carries `@ts use_input` / `@go omit` / `@pb omit`, so the existing
// New-struct codegen produces the input projection: a `z.input` type in TypeScript, a
// pydantic model in Python, and nothing in Go (where the base struct is reused).
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
				Extends:       []resolution.TypeRef{{Name: typ.Name}},
				OmittedFields: omitted,
			},
			Domains: newTypeDomains(typ),
		})
	}
	for _, t := range created {
		if err := c.table.Add(t); err != nil {
			c.diag.Add(diagnostics.Errorf(nil, "failed to synthesize New type: %v", err))
		}
	}
}

// newTypeDomains builds the domains for a synthesized New. It inherits the base type's
// TypeScript and Python output paths (so the New lands in the same files) and adds
// `use_input` for TS. The New is omitted from Go, protobuf, and C++, where the base
// struct is reused rather than a distinct input type.
func newTypeDomains(base resolution.Type) map[string]resolution.Domain {
	domains := map[string]resolution.Domain{
		"go":  {Name: "go", Expressions: resolution.Expressions{{Name: "omit"}}},
		"pb":  {Name: "pb", Expressions: resolution.Expressions{{Name: "omit"}}},
		"cpp": {Name: "cpp", Expressions: resolution.Expressions{{Name: "omit"}}},
	}
	if ts, ok := base.Domains["ts"]; ok {
		exprs := append(resolution.Expressions{}, ts.Expressions...)
		exprs = append(exprs, resolution.Expression{Name: "use_input"})
		domains["ts"] = resolution.Domain{Name: "ts", Expressions: exprs}
	} else {
		domains["ts"] = resolution.Domain{Name: "ts", Expressions: resolution.Expressions{{Name: "use_input"}}}
	}
	if py, ok := base.Domains["py"]; ok {
		domains["py"] = resolution.Domain{Name: "py", Expressions: append(resolution.Expressions{}, py.Expressions...)}
	}
	return domains
}
