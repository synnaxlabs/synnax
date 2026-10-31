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
	@pb omit
}
Ordered struct<T> {
	eq  T?
	in  T[]?
	gt  T?
	gte T?
	lt  T?
	lte T?
	@pb omit
}
Contains struct<T> {
	any T[]?
	@pb omit
}
`

// AddFilterSchema adds FilterSchema at "schemas/x/filter" with an output for every
// language.
func AddFilterSchema(loader *MockFileLoader) {
	loader.Add("schemas/x/filter", `
@go output "x/go/filter"
@py output "x/py/x/filter"
@ts output "x/ts/src/filter"
@cpp output "x/cpp/filter"
@pb
`+FilterSchema)
}

// ThingFilter declares a filter over Thing with an output for every language. It needs
// AddFilterSchema.
const ThingFilter = `
import "schemas/x/filter"

@go output "core/pkg/service/thing"
@py output "client/py/synnax/thing"
@ts output "client/ts/src/thing"
@cpp output "client/cpp/thing"
@pb

Key = uuid

Thing struct {
	key  Key    { @key }
	name string
}

Where filter Thing {
	key
	name
}
`
