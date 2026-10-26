// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package matcher lists the field matchers of a retrieved type, so that go/query, which
// generates them, and go/filter, which calls them, agree on their names.
package matcher

import (
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/key"
	plugindomain "github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/pluralize"
	"github.com/synnaxlabs/x/set"
)

// Kind is the signature of a matcher.
type Kind int

const (
	// Variadic matches a field equal to any of the values: MatchNames(vals ...string).
	Variadic Kind = iota
	// Single matches a field equal to one value: MatchInternal(v bool).
	Single
	// Element matches a slice field that holds a value: MatchTag(v string).
	Element
)

// Scalar reports whether a matcher of kind k takes one value.
func (k Kind) Scalar() bool { return k != Variadic }

// Matcher is one field matcher.
type Matcher struct {
	Field resolution.Field
	// Func is the name of the matcher function.
	Func string
	// Access is the selector that reads the field from an entry, as Access returns it.
	Access string
	Kind   Kind
}

// List returns the matchers of typ in field order: one for each field with @filter or
// named by a bare line of a filter over typ. A @key field has none, since MatchKeys
// matches it.
func List(typ resolution.Type, table *resolution.Table) ([]Matcher, error) {
	bare := bareLines(typ, table)
	var out []Matcher
	fields, _ := resolution.RecordFields(typ, table)
	for _, f := range fields {
		filtered := plugindomain.HasDomainFromField(f, "filter")
		if _, isKey := f.Domains["key"]; isKey {
			if filtered {
				return nil, errors.Newf(
					"%s: @filter on the @key field %s duplicates MatchKeys",
					typ.QualifiedName,
					f.Name,
				)
			}
			continue
		}
		if !filtered && !bare.Contains(f.Name) {
			continue
		}
		access, ok := Access(typ, f, table)
		if !ok {
			return nil, errors.Newf(
				"%s: a variant drops field %s, so no matcher can read it",
				typ.QualifiedName,
				f.Name,
			)
		}
		m := newMatcher(f, table)
		m.Access = access
		out = append(out, m)
	}
	return out, nil
}

// Access returns the selector that reads field f from an entry of typ: Name on a
// struct, and a generated method such as Base().Name or Type() on a union. It returns
// false for a union field that no generated method reads.
func Access(
	typ resolution.Type,
	f resolution.Field,
	table *resolution.Table,
) (string, bool) {
	form, ok := typ.Form.(resolution.UnionForm)
	if !ok {
		return naming.GetFieldName(f), true
	}
	if f.Name == form.Discriminator {
		method, ok := naming.UnionTagMethod(form)
		return method + "()", ok
	}
	for _, b := range naming.UnionBases(form, table) {
		base, _, ok := resolution.Dealias(b.Ref, table)
		if ok && lo.ContainsBy(
			resolution.UnifiedFields(base, table),
			func(bf resolution.Field) bool { return bf.Name == f.Name },
		) {
			return b.Method + "()." + naming.GetFieldName(f), true
		}
	}
	return "", false
}

// bareLines returns the names of the bare lines of every filter over typ.
func bareLines(typ resolution.Type, table *resolution.Table) set.Set[string] {
	names := make(set.Set[string])
	for _, t := range table.Types {
		if t.Filter == nil {
			continue
		}
		target, _, ok := resolution.Dealias(t.Filter.Target, table)
		if !ok || target.QualifiedName != typ.QualifiedName {
			continue
		}
		for _, line := range t.Filter.Lines {
			if !line.HasType() {
				names.Add(line.Name)
			}
		}
	}
	return names
}

func newMatcher(f resolution.Field, table *resolution.Table) Matcher {
	goName := naming.GetFieldName(f)
	switch {
	case key.ResolvePrimitive(f.Type, table) == "bool":
		return Matcher{Field: f, Func: "Match" + goName, Kind: Single}
	case f.Type.Name == "Array" && len(f.Type.TypeArgs) == 1:
		return Matcher{Field: f, Func: "Match" + Singular(goName), Kind: Element}
	case plugindomain.HasExprFromField(f, "filter", "scalar"):
		return Matcher{Field: f, Func: "Match" + goName, Kind: Single}
	}
	return Matcher{Field: f, Func: "Match" + Plural(goName), Kind: Variadic}
}

// Plural returns the plural form of a name. A name that is already plural
// ("Properties") gets "List" appended, so the two forms stay distinct.
func Plural(name string) string {
	plural := pluralize.String(name)
	if plural == name {
		return name + "List"
	}
	return plural
}

// Singular returns a best-effort singular form of a plural noun ("Integrations" to
// "Integration"). It returns a name that matches no plural rule unchanged.
func Singular(name string) string {
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
