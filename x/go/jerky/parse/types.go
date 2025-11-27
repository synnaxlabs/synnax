// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package parse provides AST parsing functionality for jerky code generation.
package parse

import "go/token"

// TypeKind represents the category of a Go type.
type TypeKind int

const (
	// KindPrimitive represents primitive types (bool, int, string, etc.).
	KindPrimitive TypeKind = iota
	// KindNamed represents named types (uuid.UUID, custom type wrappers).
	KindNamed
	// KindSlice represents slice types ([]T).
	KindSlice
	// KindMap represents map types (map[K]V).
	KindMap
	// KindPointer represents pointer types (*T).
	KindPointer
	// KindStruct represents embedded struct types.
	KindStruct
)

// GoType represents a Go type with full resolution information.
type GoType struct {
	// Kind is the category of the type.
	Kind TypeKind
	// Name is the full type name (e.g., "uuid.UUID", "string", "[]byte").
	Name string
	// PackagePath is the import path for named types (e.g., "github.com/google/uuid").
	PackagePath string
	// PackageName is the short package name (e.g., "uuid").
	PackageName string
	// Underlying is the underlying type for named types (e.g., uint32 for type Key uint32).
	Underlying *GoType
	// Elem is the element type for slices, maps, and pointers.
	Elem *GoType
	// Key is the key type for maps.
	Key *GoType
	// IsJerky indicates if this type is managed by jerky.
	IsJerky bool
}

// String returns a string representation of the type.
func (t GoType) String() string {
	return t.Name
}

// StructTags represents parsed struct field tags.
type StructTags struct {
	JSON    string
	Msgpack string
	Raw     string
}

// ParsedField represents a single field in a parsed struct.
type ParsedField struct {
	// Name is the Go field name (e.g., "Key").
	Name string
	// GoType is the full type information.
	GoType GoType
	// Tags contains parsed struct tags.
	Tags StructTags
	// FieldNumber is the proto field number (1-indexed, based on declaration order).
	FieldNumber int
	// Position is the source location for error reporting.
	Position token.Position
}

// ParsedStruct represents a fully analyzed Go struct annotated with //go:generate jerky.
type ParsedStruct struct {
	// Name is the struct name (e.g., "Medication").
	Name string
	// PackagePath is the full import path (e.g., "github.com/synnaxlabs/synnax/core/pkg/service/medication").
	PackagePath string
	// PackageName is the short package name (e.g., "medication").
	PackageName string
	// SourceFile is the absolute path to the source file.
	SourceFile string
	// Fields contains all struct fields in declaration order.
	Fields []ParsedField
	// Position is the source location of the struct for error reporting.
	Position token.Position
}

// FieldNames returns a slice of all field names.
func (p ParsedStruct) FieldNames() []string {
	names := make([]string, len(p.Fields))
	for i, f := range p.Fields {
		names[i] = f.Name
	}
	return names
}

// JerkyDependencies returns fully qualified names of jerky-managed types this struct depends on.
// Only includes types where IsJerky is true on the GoType.
func (p ParsedStruct) JerkyDependencies() []string {
	seen := make(map[string]bool)
	var deps []string

	var collectDeps func(t GoType)
	collectDeps = func(t GoType) {
		if t.IsJerky && t.PackagePath != "" {
			key := t.PackagePath + "." + typeName(t.Name)
			if !seen[key] {
				seen[key] = true
				deps = append(deps, key)
			}
		}
		if t.Elem != nil {
			collectDeps(*t.Elem)
		}
		if t.Key != nil {
			collectDeps(*t.Key)
		}
	}

	for _, f := range p.Fields {
		collectDeps(f.GoType)
	}
	return deps
}

// typeName extracts just the type name from a potentially qualified name.
func typeName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return name
}
