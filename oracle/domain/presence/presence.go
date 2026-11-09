// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package presence reads the @presence mark, which a type takes when its value carries
// its own absence, such as patch.Change.
package presence

import "github.com/synnaxlabs/oracle/resolution"

// Domain is the name of the mark.
const Domain = "presence"

// Of returns the type that ref names when that type carries its own absence. An
// optional field of such a type holds the value itself, not an optional wrapper.
func Of(ref resolution.TypeRef, table *resolution.Table) (resolution.Type, bool) {
	if ref.IsTypeParam() {
		return resolution.Type{}, false
	}
	typ, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return resolution.Type{}, false
	}
	_, marked := typ.Domains[Domain]
	return typ, marked
}

// Field reports whether f is an optional field that holds a value that carries its own
// absence.
func Field(f resolution.Field, table *resolution.Table) bool {
	_, ok := Of(f.Type, table)
	return ok && f.Optional
}
