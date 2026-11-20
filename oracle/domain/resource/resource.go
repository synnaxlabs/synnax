// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package resource extracts @resource domain data: the resource type name that the
// generators stamp onto a type's resource IDs.
package resource

import (
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/resolution"
)

// Data contains the extracted resource information for a schema.
type Data struct {
	KeyField   *key.Field
	TypeName   string
	StructName string
}

// SkipFunc is a predicate that determines whether to skip a type.
type SkipFunc func(resolution.Type) bool

// Extract finds resource metadata from struct or union types with an @resource domain.
// A struct takes the first of keyFields. A union takes the key of its base structs,
// wherever they are generated. Returns nil if no resource domain or key is found.
func Extract(
	types []resolution.Type,
	keyFields []key.Field,
	skip SkipFunc,
	table *resolution.Table,
) *Data {
	for _, typ := range types {
		if skip != nil && skip(typ) {
			continue
		}
		fields := keyFields
		switch form := typ.Form.(type) {
		case resolution.StructForm:
		case resolution.UnionForm:
			fields = baseKeys(form, table)
		default:
			continue
		}
		domain, ok := typ.Domains["resource"]
		if !ok {
			continue
		}
		var typeName string
		for _, expr := range domain.Expressions {
			if expr.Name == "type" && len(expr.Values) > 0 {
				typeName = expr.Values[0].StringValue
				break
			}
		}
		if typeName == "" || len(fields) == 0 {
			continue
		}
		return &Data{
			TypeName:   typeName,
			StructName: typ.Name,
			KeyField:   &fields[0],
		}
	}
	return nil
}

// baseKeys returns the key fields of a union's base structs, including the fields
// those bases inherit.
func baseKeys(form resolution.UnionForm, table *resolution.Table) []key.Field {
	var result []key.Field
	for _, ext := range form.Extends {
		base, _, ok := resolution.Dealias(ext, table)
		if !ok {
			continue
		}
		for _, f := range resolution.UnifiedFields(base, table) {
			if key.HasKey(f) {
				result = append(result, key.Field{
					Name:      f.Name,
					Primitive: key.ResolvePrimitive(f.Type, table),
					Type:      f.Type,
				})
			}
		}
	}
	return result
}
