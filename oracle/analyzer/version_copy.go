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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// fileVersion returns the version of the file under analysis, or 0 outside a version
// file.
func fileVersion(c *analysisCtx) int {
	_, n, _ := paths.VersionFile(c.filePath)
	return n
}

// versionBase finds the extends entry of a type named name, declared in version below,
// that names the same type in an earlier version of the chain. It returns the entry's
// index, or -1 when there is none, and false after it reports an error.
func versionBase(
	c *analysisCtx,
	ctx antlr.ParserRuleContext,
	name string,
	extends []resolution.TypeRef,
	below int,
) (int, bool) {
	found := -1
	for i, ext := range extends {
		qualifier, member, qualified := strings.Cut(ext.Name, ".")
		if !qualified || member != name {
			continue
		}
		k, isVersion := paths.Version(qualifier)
		if !isVersion || k >= below {
			continue
		}
		if _, seeded := c.table.Get(ext.Name); !seeded {
			continue
		}
		if found >= 0 {
			c.report(diagnostics.Errorf(ctx,
				"%s: extends more than one earlier version of itself (%s and %s)",
				name, extends[found].Name, ext.Name,
			))
			return -1, false
		}
		if len(ext.TypeArgs) > 0 {
			c.report(diagnostics.Errorf(ctx,
				"%s: an earlier version of itself takes no type arguments", name,
			))
			return -1, false
		}
		found = i
	}
	return found, true
}

// copyStruct applies decl as a patch over the declaration of the same struct in an
// earlier version, when decl extends one. The copy is collected from the earlier
// declaration's source in this file's scope, so it resolves exactly like the same text
// written here.
//
// A typed field replaces the copied field of the same name, and a typeless one merges
// over it. A field omit removes the copied field. New fields and extends entries follow
// the rules of patchEntries and appendRefs.
func copyStruct(
	c *analysisCtx, def *parser.StructFullContext, decl structDecl, below int,
) (structDecl, bool) {
	name := def.IDENT().GetText()
	i, ok := versionBase(c, def, name, decl.form.Extends, below)
	if !ok || i < 0 {
		return decl, ok
	}
	ref := decl.form.Extends[i]
	base, _ := c.table.Get(ref.Name)
	baseDef, isStruct := base.AST.(*parser.StructFullContext)
	if !isStruct {
		c.report(diagnostics.Errorf(def,
			"struct %s: %s is not a struct", name, ref.Name,
		))
		return decl, false
	}
	if len(decl.form.TypeParams) > 0 || baseDef.TypeParams() != nil {
		c.report(diagnostics.Errorf(def,
			"struct %s: a generic struct cannot extend an earlier version of itself",
			name,
		))
		return decl, false
	}
	k, _ := paths.Version(base.Namespace)
	parent, ok := collectStructDecl(c, baseDef, k)
	if !ok {
		return decl, false
	}
	c.copied = append(c.copied, baseDef)
	decl.form.Extends = slices.Delete(decl.form.Extends, i, i+1)
	return patchStruct(parent, decl), true
}

// patchStruct returns child applied as a patch over parent. See copyStruct.
func patchStruct(parent, child structDecl) structDecl {
	out := structDecl{
		form: resolution.StructForm{
			TypeParams: child.form.TypeParams,
			Extends:    appendRefs(parent.form.Extends, child.form.Extends),
			Actions:    child.form.Actions,
		},
		domains: mergeDomains(parent.domains, child.domains),
	}
	out.form.Fields = patchEntries(
		parent.form.Fields,
		child.form.Fields,
		func(f resolution.Field) string { return f.Name },
		func(pf, cf resolution.Field) resolution.Field {
			if cf.HasType() {
				return cf
			}
			return resolution.MergeOverrideField(pf, cf)
		},
		func(name string) bool {
			return slices.Contains(child.form.OmittedFields, name)
		},
	)
	inherited := func(name string) bool {
		return slices.ContainsFunc(parent.form.Fields, func(f resolution.Field) bool {
			return f.Name == name
		})
	}
	for _, omit := range parent.form.OmittedFields {
		if !slices.ContainsFunc(child.form.Fields, func(f resolution.Field) bool {
			return f.Name == omit
		}) {
			out.form.OmittedFields = append(out.form.OmittedFields, omit)
		}
	}
	for _, omit := range child.form.OmittedFields {
		if !inherited(omit) && !slices.Contains(out.form.OmittedFields, omit) {
			out.form.OmittedFields = append(out.form.OmittedFields, omit)
		}
	}
	return out
}

// copyUnion applies decl as a patch over the declaration of the same union in an
// earlier version, when decl extends one. The copy resolves in this file's scope, as in
// copyStruct.
//
// An inline variant over an inline variant of the same name patches its payload as
// copyStruct does. Any other variant of the same name replaces the copied one. New
// variants and extends entries follow the rules of patchEntries and appendRefs.
func copyUnion(
	c *analysisCtx, def parser.IUnionDefContext, decl unionDecl, below int,
) (unionDecl, bool) {
	name := def.AllIDENT()[0].GetText()
	i, ok := versionBase(c, def, name, decl.extends, below)
	if !ok || i < 0 {
		return decl, ok
	}
	ref := decl.extends[i]
	base, _ := c.table.Get(ref.Name)
	baseDef, isUnion := base.AST.(parser.IUnionDefContext)
	if !isUnion {
		c.report(diagnostics.Errorf(def,
			"union %s: %s is not a union", name, ref.Name,
		))
		return decl, false
	}
	k, _ := paths.Version(base.Namespace)
	parent, ok := collectUnionDecl(c, name, baseDef, k)
	if !ok {
		return decl, false
	}
	c.copied = append(c.copied, baseDef)
	// A copied variant reports its diagnostics at this declaration.
	for i := range parent.variants {
		parent.variants[i].variant.AST = def
	}
	variants := patchEntries(
		parent.variants,
		decl.variants,
		func(v variantDecl) string { return v.variant.Name },
		func(pv, cv variantDecl) variantDecl {
			if pv.payload == nil || cv.payload == nil {
				return cv
			}
			payload := patchStruct(*pv.payload, *cv.payload)
			cv.payload = &payload
			cv.variant.Domains = mergeDomains(pv.variant.Domains, cv.variant.Domains)
			return cv
		},
		func(string) bool { return false },
	)
	return unionDecl{
		extends:  appendRefs(parent.extends, slices.Delete(decl.extends, i, i+1)),
		variants: variants,
		domains:  mergeDomains(parent.domains, decl.domains),
	}, true
}

// patchEntries applies child over parent, matching entries by name. An entry that
// parent holds keeps its place, merged with the child entry of its name. A new child
// entry follows the nearest entry written above it in child that parent holds, or ends
// the list when there is none. Parent entries that omitted reports true are left out.
func patchEntries[T any](
	parent, child []T,
	name func(T) string,
	merge func(parent, child T) T,
	omitted func(name string) bool,
) []T {
	inherited := make(set.Set[string], len(parent))
	for _, p := range parent {
		inherited.Add(name(p))
	}
	overrides := make(map[string]T, len(child))
	after := make(map[string][]T)
	anchor := ""
	for _, ch := range child {
		if n := name(ch); inherited.Contains(n) {
			overrides[n] = ch
			anchor = n
		} else {
			after[anchor] = append(after[anchor], ch)
		}
	}
	out := make([]T, 0, len(parent)+len(child))
	for _, p := range parent {
		n := name(p)
		if ch, overridden := overrides[n]; overridden {
			out = append(out, merge(p, ch))
		} else if !omitted(n) {
			out = append(out, p)
		}
		out = append(out, after[n]...)
	}
	return append(out, after[""]...)
}

// appendRefs returns parent followed by each entry of child that parent does not
// already hold.
func appendRefs(parent, child []resolution.TypeRef) []resolution.TypeRef {
	out := slices.Clone(parent)
	for _, ref := range child {
		if !slices.ContainsFunc(out, func(r resolution.TypeRef) bool {
			return r.Name == ref.Name && len(r.TypeArgs) == 0 && len(ref.TypeArgs) == 0
		}) {
			out = append(out, ref)
		}
	}
	return out
}
