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
	TypeName   string
	StructName string
	KeyField   *key.Field
}

// SkipFunc is a predicate that determines whether to skip a type.
type SkipFunc func(resolution.Type) bool

// Extract finds ontology metadata from types with an @ontology domain.
// Returns nil if no ontology domain is found or if there are no key fields.
func Extract(types []resolution.Type, keyFields []key.Field, skip SkipFunc) *Data {
	if len(keyFields) == 0 {
		return nil
	}
	for _, typ := range types {
		if skip != nil && skip(typ) {
			continue
		}
		// Only process struct types
		_, ok := typ.Form.(resolution.StructForm)
		if !ok {
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
		if typeName == "" {
			continue
		}
		return &Data{
			TypeName:   typeName,
			StructName: typ.Name,
			KeyField:   &keyFields[0],
		}
	}
	return nil
}
