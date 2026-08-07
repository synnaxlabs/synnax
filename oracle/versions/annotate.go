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
	"context"
	"maps"
	"slices"

	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Annotate stamps chain-derived @go version expressions onto the live table:
// every type a chain's current file enumerates receives its chain's current
// version, and the pinned marker where the file declares it. The version
// files are the authority; the injected expressions are how the generators
// read it. A type that still hand-declares a version must agree with its
// chain.
func (r *Resolver) Annotate(
	ctx context.Context, table *resolution.Table,
) error {
	for _, livePath := range slices.Sorted(maps.Keys(r.chains)) {
		chain := r.chains[livePath]
		current := chain.Current()
		f, err := r.File(ctx, livePath, current)
		if err != nil {
			return err
		}
		surf, err := r.Surface(ctx, livePath, current)
		if err != nil {
			return err
		}
		pinned := filePinnedNames(f)
		for i := range table.Types {
			t := &table.Types[i]
			if t.Namespace != chain.Resource ||
				t.FilePath != livePath+".oracle" {
				continue
			}
			def, member := surf[t.Name]
			if !member {
				continue
			}
			injectPersistence(t, def.Type)
			if declared, ok := declaredVersion(*t); ok {
				if declared != current {
					return errors.Newf(
						"%s.%s declares @go version %d but its chain's current file is v%d",
						chain.Resource, t.Name, declared, current,
					)
				}
				continue
			}
			injectVersion(t, current, pinned.Contains(t.Name))
		}
	}
	return nil
}

// injectPersistence carries the version file's @go marshal and migrate
// declarations onto a live type that does not declare them itself: the file
// owns the persistence facts, and the generators read them off the live
// table. Field-level marshal tags travel too — an omitted field is a
// per-version codec fact the version file records.
func injectPersistence(t *resolution.Type, def resolution.Type) {
	injectFieldMarshal(t, def)
	src, ok := def.Domains["go"]
	if !ok {
		return
	}
	for _, name := range []string{"marshal", "migrate", "imex"} {
		expr, has := src.Expressions.Find(name)
		if !has {
			continue
		}
		if dom, ok := t.Domains["go"]; ok {
			if _, declared := dom.Expressions.Find(name); declared {
				continue
			}
		}
		domains := maps.Clone(t.Domains)
		if domains == nil {
			domains = make(map[string]resolution.Domain)
		}
		dom := domains["go"]
		dom.Name = "go"
		dom.Expressions = append(slices.Clone(dom.Expressions), expr)
		domains["go"] = dom
		t.Domains = domains
	}
}

// filePinnedNames extracts a file's @go pinned type names.
func filePinnedNames(f *File) set.Set[string] {
	pinned := make(set.Set[string])
	for _, t := range f.Defined {
		if dom, ok := t.Domains["go"]; ok {
			if _, has := dom.Expressions.Find("pinned"); has {
				pinned.Add(t.Name)
			}
		}
	}
	return pinned
}

// declaredVersion reads a hand-declared @go version from a live type.
func declaredVersion(t resolution.Type) (int, bool) {
	dom, ok := t.Domains["go"]
	if !ok {
		return 0, false
	}
	expr, ok := dom.Expressions.Find("version")
	if !ok || len(expr.Values) == 0 {
		return 0, false
	}
	return int(expr.Values[0].IntValue), true
}

// injectVersion adds a @go version expression (and optionally pinned) to the
// type, cloning the shared domain maps first.
func injectVersion(t *resolution.Type, version int, pinned bool) {
	domains := maps.Clone(t.Domains)
	if domains == nil {
		domains = make(map[string]resolution.Domain)
	}
	dom := domains["go"]
	dom.Name = "go"
	values := []resolution.ExpressionValue{{
		Kind:     resolution.ValueKindInt,
		IntValue: int64(version),
	}}
	if pinned {
		values = append(values, resolution.ExpressionValue{
			Kind:       resolution.ValueKindIdent,
			IdentValue: "pinned",
		})
	}
	dom.Expressions = append(
		slices.Clone(dom.Expressions),
		resolution.Expression{Name: "version", Values: values},
	)
	domains["go"] = dom
	t.Domains = domains
}

// injectFieldMarshal copies each field's @go marshal expression from the
// version file's declaration onto the matching live field.
func injectFieldMarshal(t *resolution.Type, def resolution.Type) {
	form, ok := t.Form.(resolution.StructForm)
	if !ok {
		return
	}
	defForm, ok := def.Form.(resolution.StructForm)
	if !ok {
		return
	}
	byName := make(map[string]resolution.Expression, len(defForm.Fields))
	for _, f := range defForm.Fields {
		dom, ok := f.Domains["go"]
		if !ok {
			continue
		}
		if expr, has := dom.Expressions.Find("marshal"); has {
			byName[f.Name] = expr
		}
	}
	if len(byName) == 0 {
		return
	}
	fields := slices.Clone(form.Fields)
	for i := range fields {
		expr, has := byName[fields[i].Name]
		if !has {
			continue
		}
		domains := maps.Clone(fields[i].Domains)
		if domains == nil {
			domains = make(map[string]resolution.Domain)
		}
		dom := domains["go"]
		dom.Name = "go"
		if _, declared := dom.Expressions.Find("marshal"); !declared {
			dom.Expressions = append(slices.Clone(dom.Expressions), expr)
		}
		domains["go"] = dom
		fields[i].Domains = domains
	}
	form.Fields = fields
	t.Form = form
}
