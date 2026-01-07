// Copyright 2026 Synnax Labs, Inc.
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

// GetPath extracts the output path from a type's domain.
func GetPath(typ resolution.Type, domainName string) string {
	if domain, ok := typ.Domains[domainName]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "output" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// IsOmitted checks if a type has the "omit" expression in its domain.
func IsOmitted(typ resolution.Type, domainName string) bool {
	if domain, ok := typ.Domains[domainName]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "omit" {
				return true
			}
		}
	}
	return false
}

// HasPB checks if a type has the @pb directive.
func HasPB(typ resolution.Type) bool {
	_, hasPB := typ.Domains["pb"]
	return hasPB
}

// GetPBPath returns the pb output path for a type.
func GetPBPath(typ resolution.Type) string {
	if !HasPB(typ) {
		return ""
	}
	goPath := GetPath(typ, "go")
	if goPath == "" {
		return ""
	}
	return goPath + "/pb"
}
