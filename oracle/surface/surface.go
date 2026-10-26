// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package surface writes the shape of analyzed schemas as text: one line per type,
// field, enum member, and union variant. Docs, annotations, and formatting are left
// out, so the reports of two revisions differ only when a shape differs.
package surface

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/resolution"
)

// Write writes the report of every declared type in table, sorted by qualified name.
func Write(w io.Writer, table *resolution.Table) error {
	types := slices.Clone(table.Types)
	slices.SortFunc(types, func(a, b resolution.Type) int {
		return strings.Compare(a.QualifiedName, b.QualifiedName)
	})
	var b strings.Builder
	for _, t := range types {
		writeType(&b, t)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeType(b *strings.Builder, t resolution.Type) {
	qn := t.QualifiedName
	switch f := t.Form.(type) {
	case resolution.PrimitiveForm, resolution.BuiltinGenericForm:
	case resolution.StructForm:
		fmt.Fprintf(
			b,
			"%s%s struct%s\n",
			qn,
			params(f.TypeParams),
			refs(" extends ", f.Extends),
		)
		writeFields(b, qn+".", f.Fields)
		for _, a := range f.Actions {
			fmt.Fprintf(b, "%s action %s\n", qn, a.Name)
			writeFields(b, qn+"!"+a.Name+".", a.Fields)
		}
	case resolution.EnumForm:
		fmt.Fprintf(b, "%s enum%s\n", qn, refs(" extends ", f.Extends))
		for _, v := range f.Values {
			fmt.Fprintf(b, "%s.%s = %v\n", qn, v.Name, quote(v.Value))
		}
	case resolution.UnionForm:
		fmt.Fprintf(
			b, "%s union on %s%s%s\n",
			qn, f.Discriminator, refs(" extends ", f.Extends), refs(" of ", f.Included),
		)
		for _, v := range f.Variants {
			fmt.Fprintf(b, "%s|%s %s\n", qn, v.Name, ref(v.Type))
		}
	case resolution.AliasForm:
		fmt.Fprintf(b, "%s%s = %s\n", qn, params(f.TypeParams), ref(f.Target))
	case resolution.DistinctForm:
		fmt.Fprintf(b, "%s%s distinct %s\n", qn, params(f.TypeParams), ref(f.Base))
	default:
		panic(fmt.Sprintf("surface: unknown form %T of %s", t.Form, qn))
	}
}

func writeFields(b *strings.Builder, prefix string, fields []resolution.Field) {
	for _, f := range fields {
		b.WriteString(prefix + f.Name)
		if f.Optional {
			b.WriteString("?")
		}
		if f.HasType() {
			b.WriteString(" " + ref(f.Type))
		}
		if f.Default != nil {
			b.WriteString(" = " + value(*f.Default))
		}
		b.WriteString("\n")
	}
}

func ref(r resolution.TypeRef) string {
	if r.IsTypeParam() {
		return r.TypeParam.Name
	}
	s := r.Name
	if len(r.TypeArgs) > 0 {
		args := make([]string, len(r.TypeArgs))
		for i, a := range r.TypeArgs {
			args[i] = ref(a)
		}
		s += "<" + strings.Join(args, ", ") + ">"
	}
	if r.ArraySize != nil {
		s += "[" + strconv.FormatInt(*r.ArraySize, 10) + "]"
	}
	return s
}

func refs(label string, rs []resolution.TypeRef) string {
	if len(rs) == 0 {
		return ""
	}
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = ref(r)
	}
	return label + strings.Join(out, ", ")
}

func params(ps []resolution.TypeParam) string {
	if len(ps) == 0 {
		return ""
	}
	out := make([]string, len(ps))
	for i, p := range ps {
		s := p.Name
		if p.Optional {
			s += "?"
		}
		if p.Constraint != nil {
			s += " extends " + ref(*p.Constraint)
		}
		if p.Default != nil {
			s += " = " + ref(*p.Default)
		}
		out[i] = s
	}
	return "<" + strings.Join(out, ", ") + ">"
}

func value(v resolution.ExpressionValue) string {
	switch v.Kind {
	case resolution.ValueKindString:
		return strconv.Quote(v.StringValue)
	case resolution.ValueKindInt:
		return strconv.FormatInt(v.IntValue, 10)
	case resolution.ValueKindFloat:
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 64)
	case resolution.ValueKindBool:
		return strconv.FormatBool(v.BoolValue)
	case resolution.ValueKindIdent:
		return v.IdentValue
	case resolution.ValueKindArray:
		out := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			out[i] = value(e)
		}
		return "[" + strings.Join(out, ", ") + "]"
	case resolution.ValueKindStruct:
		out := make([]string, len(v.Fields))
		for i, f := range v.Fields {
			out[i] = f.Name + " = " + value(f.Value)
		}
		return "{" + strings.Join(out, ", ") + "}"
	default:
		panic(fmt.Sprintf("surface: unknown value kind %d", v.Kind))
	}
}

func quote(v any) any {
	if s, ok := v.(string); ok {
		return strconv.Quote(s)
	}
	return v
}
