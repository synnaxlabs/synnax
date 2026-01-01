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

// GetTypeDefPath extracts the output path from a typedef's domain.
// domainName specifies which domain to look up (e.g., "go", "ts", "py").
// Returns an empty string if no output path is defined.
func GetTypeDefPath(entry resolution.TypeDef, domainName string) string {
	if domain, ok := entry.Domains[domainName]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "output" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// IsEnumOmitted checks if an enum has the "omit" expression in its domain.
func IsEnumOmitted(entry resolution.Enum, domainName string) bool {
	if domain, ok := entry.Domains[domainName]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "omit" {
				return true
			}
		}
	}
	return false
}

// HasPB checks if a struct has the @pb directive (flag, no parameters).
// This enables pb/ subdirectory generation.
func HasPB(entry resolution.Struct) bool {
	_, hasPB := entry.Domains["pb"]
	return hasPB
}

// HasPBEnum checks if an enum's containing struct has the @pb directive.
func HasPBEnum(entry resolution.Enum) bool {
	_, hasPB := entry.Domains["pb"]
	return hasPB
}

// GetPBPath returns the pb output path for a struct.
// When @pb flag is present, derives from @go output + "/pb/".
// Returns empty string if @pb not present or @go output not defined.
func GetPBPath(entry resolution.Struct) string {
	if !HasPB(entry) {
		return ""
	}
	goPath := GetPath(entry, "go")
	if goPath == "" {
		return ""
	}
	return goPath + "/pb"
}

// GetPBEnumPath returns the pb output path for an enum.
// When @pb flag is present, derives from @go output + "/pb/".
func GetPBEnumPath(entry resolution.Enum) string {
	if !HasPBEnum(entry) {
		return ""
	}
	goPath := GetEnumPath(entry, "go")
	if goPath == "" {
		return ""
	}
	return goPath + "/pb"
}
