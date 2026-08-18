// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package primitives provides a unified type mapping system for Oracle primitives.
// It defines all Oracle primitive types and provides a Mapper interface for
// language-specific type conversions.
package primitives

// Category represents the semantic category of a primitive type.
type Category int

const (
	CategoryOther Category = iota
	CategoryString
	CategoryNumber
	CategoryBoolean
	CategoryBinary
)

// Primitive represents an Oracle primitive type with its semantic properties.
type Primitive struct {
	Name     string   // Oracle type name (e.g., "uuid", "timestamp")
	Category Category // Semantic category
}

// All primitives supported by Oracle.
var all = []Primitive{
	{Name: "uuid", Category: CategoryString},
	{Name: "string", Category: CategoryString},
	{Name: "bool", Category: CategoryBoolean},
	{Name: "int8", Category: CategoryNumber},
	{Name: "int16", Category: CategoryNumber},
	{Name: "int32", Category: CategoryNumber},
	{Name: "int64", Category: CategoryNumber},
	{Name: "uint8", Category: CategoryNumber},
	{Name: "uint12", Category: CategoryNumber},
	{Name: "uint16", Category: CategoryNumber},
	{Name: "uint20", Category: CategoryNumber},
	{Name: "uint32", Category: CategoryNumber},
	{Name: "uint64", Category: CategoryNumber},
	{Name: "float32", Category: CategoryNumber},
	{Name: "float64", Category: CategoryNumber},
	{Name: "record", Category: CategoryOther},
	{Name: "bytes", Category: CategoryBinary},
	{Name: "any", Category: CategoryOther},
	{Name: "nil", Category: CategoryOther},
}

func IsPrimitive(name string) bool {
	for _, p := range all {
		if p.Name == name {
			return true
		}
	}
	return false
}
