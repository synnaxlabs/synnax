// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

// FilterSchema is a copy of the operator types in schemas/x/filter.oracle, without
// outputs. Load it at "schemas/x/filter" for specs that declare filters.
const FilterSchema = `
String struct {
	eq    string?
	in    string[]?
	regex string?
}
Bool struct {
	eq bool?
}
Set struct<T> {
	eq T?
	in T[]?
}
Ordered struct<T> {
	eq  T?
	in  T[]?
	gt  T?
	gte T?
	lt  T?
	lte T?
}
Contains struct<T> {
	any T[]?
}
Direction enum {
	ascending  = "ascending"
	descending = "descending"
}
`
