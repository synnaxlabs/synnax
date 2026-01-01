// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package omit provides utilities for detecting omitted types.
// Types marked with the "omit" expression in a domain are skipped
// during code generation, allowing manual implementations.
package omit

import "github.com/synnaxlabs/oracle/resolution"

// IsStruct returns true if the struct has an "omit" expression in the domain.
func IsStruct(s resolution.Struct, domainName string) bool {
	domain, ok := s.Domains[domainName]
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

// IsEnum returns true if the enum has an "omit" expression in the domain.
func IsEnum(e resolution.Enum, domainName string) bool {
	domain, ok := e.Domains[domainName]
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

// IsTypeDef returns true if the typedef has an "omit" expression in the domain.
func IsTypeDef(td resolution.TypeDef, domainName string) bool {
	domain, ok := td.Domains[domainName]
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
