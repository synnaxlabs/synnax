// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package output provides utilities for extracting output paths from oracle schemas.
package output

import "github.com/synnaxlabs/oracle/resolution"

// GetPath extracts the output path from a struct's domain.
// domainName specifies which domain to look up (e.g., "go", "ts", "py").
// Returns an empty string if no output path is defined.
func GetPath(entry resolution.Struct, domainName string) string {
	if domain, ok := entry.Domains[domainName]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "output" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// GetEnumPath extracts the output path from an enum's domain.
// domainName specifies which domain to look up (e.g., "go", "ts", "py").
// Returns an empty string if no output path is defined.
func GetEnumPath(entry resolution.Enum, domainName string) string {
	if domain, ok := entry.Domains[domainName]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "output" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// IsEnumHandwritten checks if an enum has the "handwritten" expression in its domain.
func IsEnumHandwritten(entry resolution.Enum, domainName string) bool {
	if domain, ok := entry.Domains[domainName]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "handwritten" {
				return true
			}
		}
	}
	return false
}
