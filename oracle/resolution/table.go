// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

import (
	"fmt"
	"os"

	"github.com/samber/lo"
)

// DebugTypeDef enables debug logging for typedef lookups
var DebugTypeDef = os.Getenv("ORACLE_DEBUG_TYPEDEF") != ""

// Table holds all resolved types from parsed Oracle schema files.
// It serves as the central registry for struct, enum, and typedef definitions
// that plugins use for code generation.
type Table struct {
	Structs    []Struct
	Enums      []Enum
	TypeDefs   []TypeDef
	Imports    map[string]bool
	Namespaces map[string]bool
}

// NewTable creates an empty resolution table.
func NewTable() *Table {
	return &Table{Imports: make(map[string]bool), Namespaces: make(map[string]bool)}
}

// LookupStruct finds a struct by namespace and name.
// It first tries an exact qualified name match, then falls back to name-only.
func (t *Table) LookupStruct(namespace, name string) (Struct, bool) {
	qname := namespace + "." + name
	strct, ok := t.GetStruct(qname)
	if ok {
		return strct, true
	}
	return lo.Find(t.Structs, func(item Struct) bool {
		return item.Name == name
	})
}

// LookupEnum finds an enum by namespace and name.
// It first tries an exact qualified name match, then falls back to name-only.
func (t *Table) LookupEnum(namespace, name string) (Enum, bool) {
	qname := namespace + "." + name
	enum, ok := t.GetEnum(qname)
	if ok {
		return enum, true
	}
	return lo.Find(t.Enums, func(item Enum) bool {
		return item.Name == name
	})
}

// GetStruct returns the struct with the given qualified name.
func (t *Table) GetStruct(qname string) (Struct, bool) {
	return lo.Find(t.Structs, func(item Struct) bool {
		return item.QualifiedName == qname
	})
}

// MustGetStruct returns the struct with the given qualified name or panics.
func (t *Table) MustGetStruct(qname string) Struct { return lo.Must(t.GetStruct(qname)) }

// GetEnum returns the enum with the given qualified name.
func (t *Table) GetEnum(qname string) (Enum, bool) {
	return lo.Find(t.Enums, func(item Enum) bool {
		return item.QualifiedName == qname
	})
}

// MustGetEnum returns the enum with the given qualified name or panics.
func (t *Table) MustGetEnum(qname string) Enum {
	return lo.Must(t.GetEnum(qname))
}

// AddStruct adds a struct entry to the table.
func (t *Table) AddStruct(e Struct) { t.Structs = append(t.Structs, e) }

// AddEnum adds an enum entry to the table.
func (t *Table) AddEnum(e Enum) { t.Enums = append(t.Enums, e) }

// MarkImported records that a file path has been imported.
func (t *Table) MarkImported(path string) { t.Imports[path] = true }

// IsImported returns true if the file path has been imported.
func (t *Table) IsImported(path string) bool { return t.Imports[path] }

// AllStructs returns all struct entries in the table.
func (t *Table) AllStructs() []Struct { return t.Structs }

// AllEnums returns all enum entries in the table.
func (t *Table) AllEnums() []Enum { return t.Enums }

// StructsInNamespace returns all structs in the given namespace.
func (t *Table) StructsInNamespace(ns string) []Struct {
	return lo.Filter(t.Structs, func(item Struct, _ int) bool {
		return item.Namespace == ns
	})
}

// EnumsInNamespace returns all enums in the given namespace.
func (t *Table) EnumsInNamespace(ns string) []Enum {
	return lo.Filter(t.Enums, func(item Enum, _ int) bool {
		return item.Namespace == ns
	})
}

// LookupTypeDef finds a type definition by namespace and name.
// It first tries an exact qualified name match, then falls back to name-only.
func (t *Table) LookupTypeDef(namespace, name string) (TypeDef, bool) {
	qname := namespace + "." + name
	if DebugTypeDef {
		fmt.Fprintf(os.Stderr, "[TYPEDEF] LookupTypeDef(%q, %q) called, looking for %q\n", namespace, name, qname)
	}
	td, ok := t.GetTypeDef(qname)
	if ok {
		if DebugTypeDef {
			fmt.Fprintf(os.Stderr, "[TYPEDEF] LookupTypeDef(%q, %q): exact match found -> %s\n", namespace, name, td.QualifiedName)
		}
		return td, true
	}
	// Fallback to name-only search
	fallback, found := lo.Find(t.TypeDefs, func(item TypeDef) bool {
		return item.Name == name
	})
	if DebugTypeDef {
		if found {
			fmt.Fprintf(os.Stderr, "[TYPEDEF] LookupTypeDef(%q, %q): exact match FAILED for %q, fallback found -> %s (namespace=%s)\n",
				namespace, name, qname, fallback.QualifiedName, fallback.Namespace)
		} else {
			fmt.Fprintf(os.Stderr, "[TYPEDEF] LookupTypeDef(%q, %q): NOT FOUND (no exact match, no fallback)\n", namespace, name)
		}
	}
	return fallback, found
}

// GetTypeDef returns the type definition with the given qualified name.
func (t *Table) GetTypeDef(qname string) (TypeDef, bool) {
	return lo.Find(t.TypeDefs, func(item TypeDef) bool {
		return item.QualifiedName == qname
	})
}

// MustGetTypeDef returns the type definition with the given qualified name or panics.
func (t *Table) MustGetTypeDef(qname string) TypeDef { return lo.Must(t.GetTypeDef(qname)) }

// AddTypeDef adds a type definition entry to the table.
func (t *Table) AddTypeDef(e TypeDef) {
	if DebugTypeDef {
		fmt.Fprintf(os.Stderr, "[TYPEDEF] AddTypeDef: %s (namespace=%s)\n", e.QualifiedName, e.Namespace)
	}
	t.TypeDefs = append(t.TypeDefs, e)
}

// AllTypeDefs returns all type definition entries in the table.
func (t *Table) AllTypeDefs() []TypeDef { return t.TypeDefs }

// TypeDefsInNamespace returns all type definitions in the given namespace.
func (t *Table) TypeDefsInNamespace(ns string) []TypeDef {
	return lo.Filter(t.TypeDefs, func(item TypeDef, _ int) bool {
		return item.Namespace == ns
	})
}
