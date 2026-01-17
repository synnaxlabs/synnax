// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package primitives

// Import represents an import/include required for a type mapping.
type Import struct {
	// Category is the import category (e.g., "external", "internal", "system", "typing").
	Category string
	// Path is the import path or module name.
	Path string
	// Name is the specific import name (for "from X import Y" patterns).
	Name string
}

// Mapping defines how a primitive maps to a target language.
type Mapping struct {
	TargetType string
	ZeroValue  string
	Imports    []Import
}

// Mapper provides language-specific primitive type mappings.
// Each language plugin implements this interface with its own mappings.
type Mapper interface {
	// Map returns the language-specific mapping for a primitive.
	Map(name string) Mapping
}

// tableMapper implements Mapper using a lookup table.
type tableMapper struct {
	mappings     map[string]Mapping
	fallbackType string
}

// NewMapper creates a Mapper from a mapping table and fallback type.
func NewMapper(mappings map[string]Mapping, fallbackType string) Mapper {
	return &tableMapper{mappings: mappings, fallbackType: fallbackType}
}

func (m *tableMapper) Map(name string) Mapping {
	if mapping, ok := m.mappings[name]; ok {
		return mapping
	}
	return Mapping{TargetType: m.fallbackType}
}
