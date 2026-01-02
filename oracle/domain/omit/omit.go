// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package omit provides utilities for detecting omitted types.
package omit

import "github.com/synnaxlabs/oracle/resolution"

// IsType returns true if the type has an "omit" expression in the domain.
func IsType(typ resolution.Type, domainName string) bool {
	domain, ok := typ.Domains[domainName]
	if !ok {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == "omit" {
			return true
		}
	}
	return false
}
