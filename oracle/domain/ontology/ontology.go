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
			fields = key.Collect(bases(form, table), table, nil)
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

func bases(form resolution.UnionForm, table *resolution.Table) []resolution.Type {
	var result []resolution.Type
	for _, ext := range form.Extends {
		if base, ok := ext.Resolve(table); ok {
			result = append(result, base)
		}
	}
	return result
}
