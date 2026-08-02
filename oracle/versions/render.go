// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/resolution"
)

// Decl is one version-file declaration to render: a full declaration, or an
// alias line when Alias is set.
type Decl struct {
	Type  resolution.Type
	Alias *Alias
}

// RenderOptions configures version-file rendering.
type RenderOptions struct {
	// Imports are the file's import paths, one line each.
	Imports []string
	// KeepExpression reports whether a domain expression is rendered. Nil
	// keeps everything.
	KeepExpression func(domain string, expr resolution.Expression) bool
	// Qualifier rewrites a reference's namespace qualifier. Returning ""
	// renders the name bare. Nil leaves qualifiers untouched.
	Qualifier func(ns string) string
}

// Render renders declarations into version-file source text. The output is
// structurally canonical but not byte-canonical: callers run the formatter
// over it for canonical bytes.
func Render(decls []Decl, opts RenderOptions) string {
	r := renderer{opts: opts}
	imports := append([]string(nil), opts.Imports...)
	sort.Strings(imports)
	for _, imp := range imports {
		r.linef(`import %q`, imp)
	}
	if len(imports) > 0 {
		r.line("")
	}
	for i, d := range decls {
		if i > 0 {
			r.line("")
		}
		if d.Alias != nil {
			r.linef("%s = v%d.%s", d.Type.Name, d.Alias.Version, d.Alias.Name)
			continue
		}
		r.renderType(d.Type)
	}
	return r.b.String()
}

type renderer struct {
	b      strings.Builder
	opts   RenderOptions
	indent int
}

func (r *renderer) line(s string) {
	if s != "" {
		r.b.WriteString(strings.Repeat("    ", r.indent))
		r.b.WriteString(s)
	}
	r.b.WriteByte('\n')
}

func (r *renderer) linef(format string, args ...any) {
	r.line(fmt.Sprintf(format, args...))
}

func (r *renderer) renderType(t resolution.Type) {
	switch f := t.Form.(type) {
	case resolution.StructForm:
		r.renderStruct(t, f)
	case resolution.EnumForm:
		r.renderEnum(t, f)
	case resolution.UnionForm:
		r.renderUnion(t, f)
	case resolution.DistinctForm:
		r.renderDistinct(t, f)
	case resolution.AliasForm:
		r.renderAlias(t, f)
	}
}

func (r *renderer) renderStruct(t resolution.Type, f resolution.StructForm) {
	head := t.Name + " struct" + r.params(f.TypeParams)
	if len(f.Extends) > 0 {
		head += " extends " + r.refList(f.Extends)
	}
	r.line(head + " {")
	r.indent++
	for _, field := range f.Fields {
		r.renderField(field)
	}
	for _, action := range f.Actions {
		r.line("")
		r.renderAction(action)
	}
	r.renderDomains(t.Domains)
	r.indent--
	r.line("}")
}

func (r *renderer) renderAction(a resolution.Action) {
	r.line("action " + a.Name + " {")
	r.indent++
	for _, field := range a.Fields {
		r.renderField(field)
	}
	r.renderDomains(a.Domains)
	r.indent--
	r.line("}")
}

func (r *renderer) renderField(f resolution.Field) {
	head := f.Name + " " + r.ref(f.Type)
	if f.Optional {
		head += "?"
	}
	if f.Default != nil {
		head += " = " + r.value(*f.Default)
	}
	exprs := r.keptExpressions(f.Domains)
	if len(exprs) == 0 {
		r.line(head)
		return
	}
	r.line(head + " {")
	r.indent++
	for _, e := range exprs {
		r.line(e)
	}
	r.indent--
	r.line("}")
}

func (r *renderer) renderEnum(t resolution.Type, f resolution.EnumForm) {
	head := t.Name + " enum"
	if len(f.Extends) > 0 {
		head += " extends " + r.refList(f.Extends)
	}
	r.line(head + " {")
	r.indent++
	for _, v := range f.Values {
		line := v.Name + " = "
		if f.IsIntEnum {
			line += strconv.FormatInt(v.IntValue(), 10)
		} else {
			line += strconv.Quote(v.StringValue())
		}
		exprs := r.keptExpressions(v.Domains)
		if len(exprs) == 0 {
			r.line(line)
			continue
		}
		r.line(line + " {")
		r.indent++
		for _, e := range exprs {
			r.line(e)
		}
		r.indent--
		r.line("}")
	}
	r.renderDomains(t.Domains)
	r.indent--
	r.line("}")
}

func (r *renderer) renderUnion(t resolution.Type, f resolution.UnionForm) {
	head := t.Name + " union on " + f.Discriminator
	if len(f.Extends) > 0 {
		head += " extends " + r.refList(f.Extends)
	}
	r.line(head + " {")
	r.indent++
	for _, v := range f.Variants {
		r.line(v.Name + " " + r.ref(v.Type))
	}
	r.renderDomains(t.Domains)
	r.indent--
	r.line("}")
}

func (r *renderer) renderDistinct(t resolution.Type, f resolution.DistinctForm) {
	head := t.Name + r.params(f.TypeParams) + " " + r.ref(f.Base)
	r.renderWithBody(head, t.Domains)
}

func (r *renderer) renderAlias(t resolution.Type, f resolution.AliasForm) {
	head := t.Name + r.params(f.TypeParams) + " = " + r.ref(f.Target)
	r.renderWithBody(head, t.Domains)
}

// renderWithBody renders a single-line declaration, appending a domain body
// when any domain expressions survive filtering.
func (r *renderer) renderWithBody(head string, domains map[string]resolution.Domain) {
	exprs := keptDomainExpressions(domains, r.opts.KeepExpression, r)
	if len(exprs) == 0 {
		r.line(head)
		return
	}
	r.line(head + " {")
	r.indent++
	for _, e := range exprs {
		r.line(e)
	}
	r.indent--
	r.line("}")
}

func (r *renderer) renderDomains(domains map[string]resolution.Domain) {
	exprs := keptDomainExpressions(domains, r.opts.KeepExpression, r)
	if len(exprs) == 0 {
		return
	}
	r.line("")
	for _, e := range exprs {
		r.line(e)
	}
}

func (r *renderer) keptExpressions(
	domains map[string]resolution.Domain,
) []string {
	return keptDomainExpressions(domains, r.opts.KeepExpression, r)
}

// keptDomainExpressions renders each surviving domain expression as one
// "@domain expr values" line, in sorted domain order for determinism.
func keptDomainExpressions(
	domains map[string]resolution.Domain,
	keep func(string, resolution.Expression) bool,
	r *renderer,
) []string {
	names := make([]string, 0, len(domains))
	for name := range domains {
		names = append(names, name)
	}
	sort.Strings(names)
	var lines []string
	for _, name := range names {
		dom := domains[name]
		if len(dom.Expressions) == 0 {
			if keep == nil || keep(name, resolution.Expression{}) {
				lines = append(lines, "@"+name)
			}
			continue
		}
		for _, expr := range dom.Expressions {
			if keep != nil && !keep(name, expr) {
				continue
			}
			parts := []string{"@" + name}
			if expr.Name != "" {
				parts = append(parts, expr.Name)
			}
			for _, v := range expr.Values {
				parts = append(parts, r.value(v))
			}
			lines = append(lines, strings.Join(parts, " "))
		}
	}
	return lines
}

func (r *renderer) value(v resolution.ExpressionValue) string {
	switch v.Kind {
	case resolution.ValueKindString:
		if strings.Contains(v.StringValue, "\n") {
			return `"""` + v.StringValue + `"""`
		}
		return strconv.Quote(v.StringValue)
	case resolution.ValueKindInt:
		return strconv.FormatInt(v.IntValue, 10)
	case resolution.ValueKindFloat:
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 64)
	case resolution.ValueKindBool:
		return strconv.FormatBool(v.BoolValue)
	case resolution.ValueKindIdent:
		return r.qualify(v.IdentValue)
	case resolution.ValueKindArray:
		parts := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			parts[i] = r.value(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return ""
	}
}

func (r *renderer) params(params []resolution.TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, p := range params {
		s := p.Name
		if p.Constraint != nil {
			s += " " + r.ref(*p.Constraint)
		}
		if p.Default != nil {
			s += " = " + r.ref(*p.Default)
		}
		parts[i] = s
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

func (r *renderer) refList(refs []resolution.TypeRef) string {
	parts := make([]string, len(refs))
	for i, ref := range refs {
		parts[i] = r.ref(ref)
	}
	return strings.Join(parts, ", ")
}

func (r *renderer) ref(ref resolution.TypeRef) string {
	if ref.TypeParam != nil {
		return ref.TypeParam.Name
	}
	if ref.Name == "Array" && len(ref.TypeArgs) == 1 {
		size := ""
		if ref.ArraySize != nil {
			size = strconv.FormatInt(*ref.ArraySize, 10)
		}
		return r.ref(ref.TypeArgs[0]) + "[" + size + "]"
	}
	if ref.Name == "Map" && len(ref.TypeArgs) == 2 {
		return "map<" + r.ref(ref.TypeArgs[0]) + ", " + r.ref(ref.TypeArgs[1]) + ">"
	}
	name := r.qualify(ref.Name)
	if len(ref.TypeArgs) == 0 {
		return name
	}
	args := make([]string, len(ref.TypeArgs))
	for i, a := range ref.TypeArgs {
		args[i] = r.ref(a)
	}
	return name + "<" + strings.Join(args, ", ") + ">"
}

// qualify rewrites a possibly-qualified name's namespace through the
// Qualifier option.
func (r *renderer) qualify(name string) string {
	if r.opts.Qualifier == nil {
		return name
	}
	prefix, rest, found := strings.Cut(name, ".")
	if !found {
		return name
	}
	ns := r.opts.Qualifier(prefix)
	if ns == "" {
		return rest
	}
	return ns + "." + rest
}
