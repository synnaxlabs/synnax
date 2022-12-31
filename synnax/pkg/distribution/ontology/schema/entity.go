// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schema

// Entity represents an instance matching a [Schema] (think class and object in OOP).
type Entity struct {
	// Schema is the schema that this entity matches.
	Schema *Schema `json:"schema" msgpack:"schema"`
	// Name is a human-readable name for the entity.
	Name string `json:"name" msgpack:"name"`
	// Data is the data for the entity. Data must match [Schema.Fields].
	Data map[string]any `json:"data" msgpack:"data"`
}

// Get is a strongly-typed getter for an [Entity] field value. Returns true if the
// value was found, false otherwise. Panics if the value is not of the asserted type (
// as defined in the type parameter).
func Get[V Value](d Entity, k string) (v V, ok bool) {
	rv, ok := d.Data[k]
	if !ok {
		return v, false
	}
	v, ok = rv.(V)
	if !ok {
		panic("[schema] - invalid field type")
	}
	return v, true
}

// Set is a strongly-typed setter for an [Entity] field value. Panics if the value is
// not of the asserted type (as defined in the type parameter) or if the field is not
// defined in the [Schema].
func Set[V Value](D Entity, k string, v V) {
	f, ok := D.Schema.Fields[k]
	if !ok {
		panic("[Schema] - field not found")
	}
	if !f.Type.AssertValue(v) {
		panic("[Schema] - invalid field type")
	}
	D.Data[k] = v
}

// NewEntity creates a new entity with the given schema and name and an empty set of
// field data.
func NewEntity(schema *Schema, name string) Entity {
	return Entity{
		Schema: schema,
		Name:   name,
		Data:   make(map[string]any),
	}
}
