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

type category int

const (
	categoryOther category = iota
	categoryString
	categoryNumber
	categoryBoolean
	categoryBinary
)

type primitive struct {
	Name     string   // Oracle type name (e.g., "uuid", "timestamp")
	Category category // Semantic category
}

// All primitives supported by Oracle.
var all = []primitive{
	{Name: "uuid", Category: categoryString},
	{Name: "string", Category: categoryString},
	{Name: "bool", Category: categoryBoolean},
	{Name: "int8", Category: categoryNumber},
	{Name: "int16", Category: categoryNumber},
	{Name: "int32", Category: categoryNumber},
	{Name: "int64", Category: categoryNumber},
	{Name: "uint8", Category: categoryNumber},
	{Name: "uint12", Category: categoryNumber},
	{Name: "uint16", Category: categoryNumber},
	{Name: "uint20", Category: categoryNumber},
	{Name: "uint32", Category: categoryNumber},
	{Name: "uint64", Category: categoryNumber},
	{Name: "float32", Category: categoryNumber},
	{Name: "float64", Category: categoryNumber},
	{Name: "record", Category: categoryOther},
	{Name: "bytes", Category: categoryBinary},
	{Name: "any", Category: categoryOther},
	{Name: "nil", Category: categoryOther},
}

func IsPrimitive(name string) bool {
	for _, p := range all {
		if p.Name == name {
			return true
		}
	}
	return false
}
