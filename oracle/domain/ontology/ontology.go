// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/resolution"
)

type Data struct {
	TypeName   string
	StructName string
	KeyField   *key.Field
}

type SkipFunc func(*resolution.StructEntry) bool

func Extract(structs []*resolution.StructEntry, keyFields []key.Field, skip SkipFunc) *Data {
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
