// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package handwritten

import "github.com/synnaxlabs/oracle/resolution"

func IsStruct(s *resolution.StructEntry, domainName string) bool {
	domain, ok := s.Domains[domainName]
	if !ok {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == "handwritten" {
			return true
		}
	}
	return false
}

func IsEnum(e *resolution.EnumEntry, domainName string) bool {
	domain, ok := e.Domains[domainName]
	if !ok {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == "handwritten" {
			return true
		}
	}
	return false
}
