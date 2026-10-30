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
	"slices"

	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

// generatedLangs are the language domains whose omission rules the analyzer validates.
// pb is excluded: proto messages reference schema types by name only, so omission there
// never dangles.
var generatedLangs = []string{"go", "ts", "py", "cpp"}

// validateDomainOmits enforces cross-language omission consistency over the resolved
// table:
//
//  1. A type that generates in a language must not reference a type omitted in
//     that language (@lang omit means the type does not exist there). Fields
//     with their own @lang omit or @lang type override escape the check, as do
//     references to @lang hand types, which exist hand-written. Nor may it
//     reference a type with no output in that language. A struct may extend a
//     parent absent from the language; it then declares the inherited fields.
//  2. A file-level @lang output where every declared type omits the language —
//     with no hand-written anchors or actions — generates nothing and should
//     be removed.
//  3. A field that overrides an inherited field cannot omit a language the parent
//     field keeps.
func validateDomainOmits(table *resolution.Table, diag *diagnostics.Files) {
	for _, lang := range generatedLangs {
		validateOmitRefs(table, lang, diag)
	}
	validateOmitOverrides(table, diag)
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
					d := diagnostics.Errorf(
						nil,
						"%s generates for %s but %s references %s, which is omitted in %s",
						t.QualifiedName,
						lang,
						context,
						resolved.QualifiedName,
						lang,
					)
					diag.Report(t.Namespace+".oracle", d)
					return
				}
				if !hasOutput(resolved, lang, table) {
					d := diagnostics.Errorf(
						nil,
						"%s generates for %s but %s references %s, which has no %s "+
							"output",
						t.QualifiedName,
						lang,
						context,
						resolved.QualifiedName,
						lang,
					)
					diag.Report(t.Namespace+".oracle", d)
				}
			}
			visit(ref)
		}
		switch form := t.Form.(type) {
		case resolution.UnionForm:
			if form.BorrowsEnum() {
				check(form.Enum, "its discriminator enum")
			}
		case resolution.AliasForm:
			check(form.Target, "its alias target")
		case resolution.DistinctForm:
			check(form.Base, "its base type")
		case resolution.StructForm:
			// A struct with a parent absent from lang declares its inherited fields
			// itself, so only those fields must exist there.
			fields := form.Fields
			if slices.ContainsFunc(form.Extends, func(ref resolution.TypeRef) bool {
				parent, ok := ref.Resolve(table)
				return ok && omit.IsAbsent(parent, lang)
			}) {
				fields = resolution.UnifiedFields(t, table)
			} else {
				for _, ext := range form.Extends {
					check(ext, "its extends clause")
				}
			}
			for _, f := range fields {
				if fieldEscapesLang(f, lang) {
					continue
				}
				check(f.Type, "field "+f.Name)
			}
		}
	}
}

// fieldEscapesLang reports whether the field's own domain removes or replaces it in the
// language: an omit expression or a type override.
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

// validateNoVersionTag errors on any @go version declaration: the tag is
// retired. Membership in the resource's current version file marks a type
// versioned; the versions directory is the sole authority.
func validateNoVersionTag(c *analysisCtx, types []resolution.Type) {
	if dom, ok := c.fileDomains["go"]; ok {
		if _, has := dom.Expressions.Find("version"); has {
			c.report(diagnostics.Errorf(nil,
				"%s declares the retired @go version tag; membership in the "+
					"current version file marks a type versioned",
				c.namespace,
			))
		}
	}
	for _, t := range types {
		dom, ok := t.Domains["go"]
		if !ok {
			continue
		}
		if _, has := dom.Expressions.Find("version"); has {
			c.report(diagnostics.Errorf(nil,
				"%s declares the retired @go version tag; membership in the "+
					"current version file marks a type versioned",
				t.QualifiedName,
			))
		}
	}
}

// validateImex errors on malformed @go imex declarations: the marker is per-type and
// takes no argument or `body`. The Go imex plugin rejects a marked type whose version
// chain leaves it unversioned.
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
		if !ok || len(expr.Values) == 0 ||
			len(expr.Values) == 1 && expr.Values[0].IdentValue == "body" {
			continue
		}
		c.report(diagnostics.Errorf(nil,
			"%s has a malformed @go imex; the marker takes no argument or body",
			t.QualifiedName,
		))
	}
}

// validateDeadOutputs errors when a file declares a language output that nothing uses:
// every type omits the language and none is hand-written.
func validateDeadOutputs(c *analysisCtx, types []resolution.Type) {
	var hasActions bool
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
		var alive bool
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

// validateFieldOmits errors on an illegal field-level omit. A field may omit any
// generated language but Go, which has @go marshal omit and @go wire omit instead. A
// key field may not leave any language or the wire, a field may not omit a language its
// type does not generate in, and a field must stay in at least one language.
func validateFieldOmits(c *analysisCtx, types []resolution.Type) {
	for _, t := range types {
		form, ok := t.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			name := t.QualifiedName + "." + f.Name
			if omit.IsField(f, "go") {
				c.report(diagnostics.Errorf(nil,
					"%s omits go; use @go marshal omit or @go wire omit instead", name,
				))
			}
			if wire := getFieldString(f, "go", "wire"); wire != "" && wire != "omit" {
				c.report(diagnostics.Errorf(nil,
					"%s has @go wire %s; the only value is omit", name, wire,
				))
			}
			isKey := key.HasKey(f)
			if isKey && getFieldString(f, "go", "wire") == "omit" {
				c.report(diagnostics.Errorf(nil,
					"%s is a key and cannot carry @go wire omit", name,
				))
			}
			var generating, omitted int
			for _, lang := range fieldOmitLangs {
				generates := generatesIn(t, lang)
				if generates {
					generating++
				}
				if lang == "go" || !omit.IsField(f, lang) {
					continue
				}
				omitted++
				if isKey {
					c.report(diagnostics.Errorf(nil,
						"%s is a key and cannot omit %s", name, lang,
					))
				}
				if !generates {
					c.report(diagnostics.Errorf(nil,
						"%s omits %s, but %s does not generate in %s",
						name, lang, t.QualifiedName, lang,
					))
				}
			}
			if omitted > 0 && omitted == generating {
				c.report(diagnostics.Errorf(nil,
					"%s is omitted in every language %s generates in; remove the field",
					name, t.QualifiedName,
				))
			}
		}
	}
}

// fieldOmitLangs are the languages a field-level omit can name: the generated
// languages plus pb, whose messages hold the field too.
var fieldOmitLangs = append(slices.Clone(generatedLangs), "pb")

// generatesIn reports whether t generates a declaration in lang. A pb message needs a
// pb domain and a Go output, which sets its package.
func generatesIn(t resolution.Type, lang string) bool {
	if omit.IsSkipped(t, lang) {
		return false
	}
	if lang != "pb" {
		return getPath(t, lang) != ""
	}
	_, hasPB := t.Domains["pb"]
	return hasPB && getPath(t, "go") != ""
}

// validateOmitOverrides errors when a field overrides an inherited field and omits a
// language the parent field keeps. The parent would still contribute the field, so the
// omit would not take effect.
func validateOmitOverrides(table *resolution.Table, diag *diagnostics.Files) {
	for _, t := range table.Types {
		form, ok := t.Form.(resolution.StructForm)
		if !ok || len(form.Extends) == 0 {
			continue
		}
		inherited := make(map[string]resolution.Field)
		for _, ext := range form.Extends {
			parent, _, ok := resolution.Dealias(ext, table)
			if !ok {
				continue
			}
			for _, pf := range resolution.UnifiedFields(parent, table) {
				if _, seen := inherited[pf.Name]; !seen {
					inherited[pf.Name] = pf
				}
			}
		}
		for _, f := range form.Fields {
			pf, ok := inherited[f.Name]
			if !ok {
				continue
			}
			for _, lang := range fieldOmitLangs {
				if omit.IsField(f, lang) && !omit.IsField(pf, lang) {
					diag.Report(t.Namespace+".oracle", diagnostics.Errorf(nil,
						"%s.%s overrides an inherited field and cannot omit %s; "+
							"omit it on the parent field or drop it with -%s",
						t.QualifiedName, f.Name, lang, f.Name,
					))
				}
			}
		}
	}
}

func getFieldString(f resolution.Field, lang, expr string) string {
	dom, ok := f.Domains[lang]
	if !ok {
		return ""
	}
	e, ok := dom.Expressions.Find(expr)
	if !ok || len(e.Values) == 0 {
		return ""
	}
	if v := e.Values[0].StringValue; v != "" {
		return v
	}
	return e.Values[0].IdentValue
}

// hasOutput reports whether the type exists in the language at some output path. An
// enum without its own path takes the path of a struct in the same schema file.
// Built-in and version-file types have no declared path, since plugins place them.
func hasOutput(t resolution.Type, lang string, table *resolution.Table) bool {
	switch t.Form.(type) {
	case resolution.PrimitiveForm, resolution.BuiltinGenericForm:
		return true
	}
	if getPath(t, lang) != "" {
		return true
	}
	if _, _, ok := paths.VersionFile(t.FilePath); ok {
		return true
	}
	if _, ok := t.Form.(resolution.EnumForm); !ok {
		return false
	}
	return slices.ContainsFunc(table.StructTypes(), func(s resolution.Type) bool {
		return s.Namespace == t.Namespace && s.FilePath == t.FilePath &&
			getPath(s, lang) != ""
	})
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
