// Copyright 2025 Synnax Labs, Inc.
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

// SkipFunc is a predicate that determines whether to skip a struct.
type SkipFunc func(*resolution.Struct) bool

// Extract finds ontology metadata from structs with an @ontology domain.
// Returns nil if no ontology domain is found or if there are no key fields.
func Extract(structs []*resolution.Struct, keyFields []key.Field, skip SkipFunc) *Data {
	if len(keyFields) == 0 {
		return nil
	}
	for _, s := range structs {
		if skip != nil && skip(s) {
			continue
		}
		domain, ok := s.Domains["ontology"]
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
			StructName: s.Name,
			KeyField:   &keyFields[0],
		}
	}
	return nil
}
