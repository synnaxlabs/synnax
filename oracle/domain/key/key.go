// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package key

import "github.com/synnaxlabs/oracle/resolution"

type Field struct {
	Name      string
	Primitive string
}

type SkipFunc func(*resolution.StructEntry) bool

func Collect(structs []*resolution.StructEntry, skip SkipFunc) []Field {
	seen := make(map[string]bool)
	var result []Field
	for _, s := range structs {
		if skip != nil && skip(s) {
			continue
		}
		for _, f := range s.Fields {
			if _, hasKey := f.Domains["key"]; hasKey {
				if !seen[f.Name] {
					seen[f.Name] = true
					result = append(result, Field{
						Name:      f.Name,
						Primitive: f.TypeRef.Primitive,
					})
				}
			}
		}
	}
	return result
}
