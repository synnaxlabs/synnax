// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package telem finds the telem types that generators render specially, chasing aliases
// and distinct bases. Every generator finds these types through this one package so
// they cannot drift.
package telem

import (
	"strings"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/resolution"
)

// IsTimeStamp reports whether the reference denotes telem's TimeStamp in the language
// of domainName, through aliases and distinct bases. A type hand-written in that
// language is its own type there, so the chain stops at it.
func IsTimeStamp(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	domainName string,
) bool {
	_, ok := TimeStampRef(typeRef, table, domainName)
	return ok
}

// TimeStampRef returns the reference to telem's TimeStamp that typeRef denotes, under
// the rules of IsTimeStamp. ok is false when typeRef does not denote it.
func TimeStampRef(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	domainName string,
) (resolution.TypeRef, bool) {
	return find(typeRef, table, domainName, "TimeStamp")
}

// IsTimeSpan is IsTimeStamp for telem's TimeSpan.
func IsTimeSpan(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	domainName string,
) bool {
	_, ok := find(typeRef, table, domainName, "TimeSpan")
	return ok
}

// IsRate is IsTimeStamp for telem's Rate.
func IsRate(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	domainName string,
) bool {
	_, ok := find(typeRef, table, domainName, "Rate")
	return ok
}

func find(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	domainName, name string,
) (resolution.TypeRef, bool) {
	if typeRef.Name == name || strings.HasSuffix(typeRef.Name, "."+name) {
		return typeRef, true
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok || omit.IsHand(resolved, domainName) {
		return resolution.TypeRef{}, false
	}
	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		return find(form.Target, table, domainName, name)
	case resolution.DistinctForm:
		return find(form.Base, table, domainName, name)
	default:
		return resolution.TypeRef{}, false
	}
}
