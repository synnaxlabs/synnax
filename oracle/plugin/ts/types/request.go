// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Request returns req as the TypeScript plugins read it. It drops the fields that omit
// TypeScript, and gives the operator of each bare filter line the @ts type of the field
// that the line reads in the record that the client cache holds.
func Request(req *plugin.Request) (*plugin.Request, error) {
	req = req.OmitFields("ts")
	table := req.Resolutions
	for i, typ := range table.Types {
		// A filter with no derived response has no matcher to type.
		if typ.Filter == nil || typ.Filter.Response == "" {
			continue
		}
		record, fields := CachedRecord(typ, table)
		form := resolution.MustForm[resolution.StructForm](typ)
		form.Fields = slices.Clone(form.Fields)
		for _, line := range typ.Filter.Lines {
			if line.HasType() {
				continue
			}
			field, _ := lo.Find(fields, func(f resolution.Field) bool {
				return f.Name == line.Name
			})
			override := domain.GetFieldType(field, "ts")
			if override == "" {
				continue
			}
			arg, ok := overrideRef(override, record.Namespace, table)
			if !ok {
				return nil, errors.Newf(
					"filter %s: line %s reads a field whose @ts type %s is not a type",
					typ.QualifiedName,
					line.Name,
					override,
				)
			}
			idx := slices.IndexFunc(form.Fields, func(f resolution.Field) bool {
				return f.Name == line.Name
			})
			op := form.Fields[idx].Type
			if len(op.TypeArgs) == 1 {
				op.TypeArgs = []resolution.TypeRef{arg}
				form.Fields[idx].Type = op
			}
		}
		typ.Form = form
		table.Types[i] = typ
	}
	return req, nil
}

// CachedRecord returns the record type that the client cache holds for where, which is
// the record that the retrieve response returns, and the fields of that record. where
// must have a derived retrieve response.
func CachedRecord(
	where resolution.Type,
	table *resolution.Table,
) (resolution.Type, []resolution.Field) {
	response := resolution.MustForm[resolution.StructForm](
		table.MustGet(where.Filter.Response),
	)
	// The analyzer derives a response only for a record that is a struct or a union.
	record, _, _ := resolution.Dealias(response.Fields[0].Type.TypeArgs[0], table)
	fields, _ := resolution.RecordFields(record, table)
	return record, fields
}

// overrideRef returns the type that a @ts type override names.
func overrideRef(
	override, namespace string,
	table *resolution.Table,
) (resolution.TypeRef, bool) {
	switch {
	case resolution.IsPrimitive(override):
		return resolution.TypeRef{Name: override}, true
	case override == "number":
		// A TypeScript number is a float64.
		return resolution.TypeRef{Name: "float64"}, true
	}
	typ, ok := table.Get(override)
	if !ok {
		typ, ok = table.Lookup(namespace, override)
	}
	return resolution.TypeRef{Name: typ.QualifiedName}, ok
}
