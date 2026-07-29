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
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

// generatedLangs are the language domains whose omission rules the analyzer
// validates. pb is excluded: proto messages reference schema types by name
// only, so omission there never dangles.
var generatedLangs = []string{"go", "ts", "py", "cpp"}

// validateDomainOmits enforces cross-language omission consistency over the
// resolved table:
//
//  1. A type that generates in a language must not reference a type omitted in
//     that language (@lang omit means the type does not exist there). Fields
//     with their own @lang omit or @lang type override escape the check, as do
//     references to @lang hand types, which exist hand-written.
//  2. A file-level @lang output where every declared type omits the language —
//     with no hand-written anchors or actions — generates nothing and should
//     be removed.
func validateDomainOmits(table *resolution.Table, diag *diagnostics.Files) {
	for _, lang := range generatedLangs {
		validateOmitRefs(table, lang, diag)
	}
}

func validateOmitRefs(
	table *resolution.Table,
	lang string,
	diag *diagnostics.Files,
) {
	for _, t := range table.Types {
		if getPath(t, lang) == "" || omit.IsSkipped(t, lang) {
			continue
		}
		form, ok := t.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		check := func(ref resolution.TypeRef, context string) {
			var visit func(r resolution.TypeRef)
			visit = func(r resolution.TypeRef) {
				for _, arg := range r.TypeArgs {
					visit(arg)
				}
				resolved, ok := r.Resolve(table)
				if !ok {
					return
				}
				if omit.IsType(resolved, lang) {
					d := diagnostics.Errorf(nil,
						"%s generates for %s but %s references %s, which is omitted in %s",
						t.QualifiedName, lang, context, resolved.QualifiedName, lang,
					)
					diag.Report(t.Namespace+".oracle", d)
				}
			}
			visit(ref)
		}
		for _, ext := range form.Extends {
			check(ext, "its extends clause")
		}
		for _, f := range form.Fields {
			if fieldEscapesLang(f, lang) {
				continue
			}
			check(f.Type, "field "+f.Name)
		}
	}
}

// fieldEscapesLang reports whether the field's own domain removes or replaces
// it in the language: an omit expression or a type override.
func fieldEscapesLang(f resolution.Field, lang string) bool {
	dom, ok := f.Domains[lang]
	if !ok {
		return false
	}
	for _, expr := range dom.Expressions {
		if expr.Name == "omit" || expr.Name == "type" {
			return true
		}
	}
	return false
}

// validateFileVersion errors when @go version is declared file-level.
// Versioned-ness is a per-type property (it tracks persistence), so the
// declaration must sit on each type it applies to.
func validateFileVersion(c *analysisCtx) {
	dom, ok := c.fileDomains["go"]
	if !ok {
		return
	}
	if _, has := dom.Expressions.Find("version"); has {
		d := diagnostics.Errorf(nil,
			"%s declares @go version file-level; declare it per type instead",
			c.namespace,
		)
		c.report(d)
	}
}

// validateVersionArgs errors on malformed @go version declarations: the first
// value must be an integer, and the only allowed extra argument is a single
// `pinned` marker.
func validateVersionArgs(c *analysisCtx, types []resolution.Type) {
	for _, t := range types {
		dom, ok := t.Domains["go"]
		if !ok {
			continue
		}
		expr, ok := dom.Expressions.Find("version")
		if !ok {
			continue
		}
		valid := len(expr.Values) >= 1 && len(expr.Values) <= 2 &&
			expr.Values[0].Kind == resolution.ValueKindInt
		if valid && len(expr.Values) == 2 {
			valid = expr.Values[1].IdentValue == "pinned"
		}
		if valid {
			continue
		}
		d := diagnostics.Errorf(nil,
			"%s has a malformed @go version; expected `version <int>` or `version <int> pinned`",
			t.QualifiedName,
		)
		c.report(d)
	}
}

// validateImex errors on malformed @go imex declarations: the marker is
// per-type, takes no arguments, and requires a @go version on the same type.
func validateImex(c *analysisCtx, types []resolution.Type) {
	if dom, ok := c.fileDomains["go"]; ok {
		if _, has := dom.Expressions.Find("imex"); has {
			c.report(diagnostics.Errorf(nil,
				"%s declares @go imex file-level; declare it per type instead",
				c.namespace,
			))
			return
		}
	}
	for _, t := range types {
		dom, ok := t.Domains["go"]
		if !ok {
			continue
		}
		expr, ok := dom.Expressions.Find("imex")
		if !ok {
			continue
		}
		if len(expr.Values) > 0 {
			c.report(diagnostics.Errorf(nil,
				"%s has a malformed @go imex; the marker takes no arguments",
				t.QualifiedName,
			))
		}
		if _, versioned := dom.Expressions.Find("version"); !versioned {
			c.report(diagnostics.Errorf(nil,
				"%s declares @go imex without @go version", t.QualifiedName,
			))
		}
	}
}

// validateDeadOutputs errors when a file declares a language output that
// nothing uses: every type omits the language and none is hand-written.
func validateDeadOutputs(
	c *analysisCtx,
	types []resolution.Type,
) {
	hasActions := false
	for _, t := range types {
		if sf, ok := t.Form.(resolution.StructForm); ok && len(sf.Actions) > 0 {
			hasActions = true
		}
	}
	for _, lang := range generatedLangs {
		dom, ok := c.fileDomains[lang]
		if !ok {
			continue
		}
		if _, hasOutput := dom.Expressions.Find("output"); !hasOutput {
			continue
		}
		if hasActions && (lang == "go" || lang == "ts") {
			continue
		}
		alive := false
		for _, t := range types {
			if omit.IsType(t, lang) {
				continue
			}
			// Generating or hand-written types both justify the output.
			alive = true
			break
		}
		if len(types) > 0 && !alive {
			d := diagnostics.Errorf(nil,
				"every type in %s omits %s; remove the @%s output declaration",
				c.namespace, lang, lang,
			)
			c.report(d)
		}
	}
}

// getPath returns the type's output path for the language, or "".
func getPath(t resolution.Type, lang string) string {
	dom, ok := t.Domains[lang]
	if !ok {
		return ""
	}
	expr, ok := dom.Expressions.Find("output")
	if !ok || len(expr.Values) == 0 {
		return ""
	}
	return expr.Values[0].StringValue
}
