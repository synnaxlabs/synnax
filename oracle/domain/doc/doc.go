// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package doc provides utilities for extracting documentation from oracle schemas.
package doc

import "github.com/synnaxlabs/oracle/resolution"

// Get extracts documentation from a domain map.
// It looks for a "doc" domain and returns the first expression's value or name.
// Returns an empty string if no documentation is defined.
func Get(domains map[string]resolution.Domain) string {
	if domain, ok := domains["doc"]; ok {
		if len(domain.Expressions) > 0 {
			expr := domain.Expressions[0]
			if len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
			return expr.Name
		}
	}
	return ""
}
