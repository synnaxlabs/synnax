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
	CategoryTemporal
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
	{Name: "timestamp", Category: CategoryTemporal},
	{Name: "timespan", Category: CategoryTemporal},
	{Name: "time_range", Category: CategoryTemporal},
	{Name: "time_range_bounded", Category: CategoryTemporal},
	{Name: "data_type", Category: CategoryOther},
	{Name: "color", Category: CategoryString},
	{Name: "json", Category: CategoryOther},
	{Name: "bytes", Category: CategoryBinary},
}

// primitivesByName is a lookup map for fast access.
var primitivesByName = make(map[string]Primitive)

func init() {
	for _, p := range all {
		primitivesByName[p.Name] = p
	}
}

// All returns all registered primitives.
func All() []Primitive {
	return all
}

// Get returns a primitive by name, and whether it was found.
func Get(name string) (Primitive, bool) {
	p, ok := primitivesByName[name]
	return p, ok
}

// IsPrimitive checks if a name is a registered primitive type.
func IsPrimitive(name string) bool {
	_, ok := primitivesByName[name]
	return ok
}

// IsString returns true if the primitive is string-like.
func IsString(name string) bool {
	p, ok := primitivesByName[name]
	return ok && p.Category == CategoryString
}

// IsNumber returns true if the primitive is numeric.
func IsNumber(name string) bool {
	p, ok := primitivesByName[name]
	return ok && p.Category == CategoryNumber
}

// IsTemporal returns true if the primitive is time-related.
func IsTemporal(name string) bool {
	p, ok := primitivesByName[name]
	return ok && p.Category == CategoryTemporal
}

// IsBoolean returns true if the primitive is boolean.
func IsBoolean(name string) bool {
	p, ok := primitivesByName[name]
	return ok && p.Category == CategoryBoolean
}

// IsBinary returns true if the primitive is binary data.
func IsBinary(name string) bool {
	p, ok := primitivesByName[name]
	return ok && p.Category == CategoryBinary
}
