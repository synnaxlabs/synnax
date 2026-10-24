// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package telem answers whether a type reference denotes a telem time type, chasing
// aliases and distinct bases. Every generator that renders a time type specially finds
// it through this one package so they cannot drift.
package telem

import (
	"strings"

	"github.com/synnaxlabs/oracle/resolution"
)

// IsTimeStamp reports whether the reference denotes telem's TimeStamp, through aliases
// and distinct bases.
func IsTimeStamp(typeRef resolution.TypeRef, table *resolution.Table) bool {
	return reaches(typeRef, table, "TimeStamp")
}

// IsTimeSpan reports whether the reference denotes telem's TimeSpan, through aliases
// and distinct bases.
func IsTimeSpan(typeRef resolution.TypeRef, table *resolution.Table) bool {
	return reaches(typeRef, table, "TimeSpan")
}

func reaches(typeRef resolution.TypeRef, table *resolution.Table, name string) bool {
	if typeRef.Name == name || strings.HasSuffix(typeRef.Name, "."+name) {
		return true
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		return reaches(form.Target, table, name)
	case resolution.DistinctForm:
		return reaches(form.Base, table, name)
	default:
		return false
	}
}
