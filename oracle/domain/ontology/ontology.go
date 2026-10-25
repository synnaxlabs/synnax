// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ontology provides utilities for extracting @ontology domain data.
// The ontology domain defines how types integrate with the Synnax ontology system.
package ontology

import (
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/resolution"
)

// Data contains the extracted ontology information for a schema.
type Data struct {
	KeyField   *key.Field
	TypeName   string
	StructName string
}

// SkipFunc is a predicate that determines whether to skip a type.
type SkipFunc func(resolution.Type) bool

// Extract finds ontology metadata from struct or union types with an @ontology domain.
// A struct takes the first of keyFields. A union takes the key of its base structs,
// wherever they are generated. Returns nil if no ontology domain or key is found.
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
		domain, ok := typ.Domains["ontology"]
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
