// Copyright 2026 Synnax Labs, Inc.
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

// IsType returns true if the type has an "omit" expression in the domain: the
// type does not exist in the domain's language at all.
func IsType(typ resolution.Type, domainName string) bool {
	return hasExpr(typ, domainName, "omit")
}

// IsHand returns true if the type has a "hand" expression in the domain: the
// type exists in the language, hand-written at its declared output path.
// References to it resolve normally; only its declaration is not generated.
func IsHand(typ resolution.Type, domainName string) bool {
	return hasExpr(typ, domainName, "hand")
}

// IsSkipped returns true when no declaration is generated for the type in the
// domain: the type is either omitted or hand-written.
func IsSkipped(typ resolution.Type, domainName string) bool {
	return IsType(typ, domainName) || IsHand(typ, domainName)
}

func hasExpr(typ resolution.Type, domainName, name string) bool {
	domain, ok := typ.Domains[domainName]
	if !ok {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == name {
			return true
		}
	}
	return false
}
