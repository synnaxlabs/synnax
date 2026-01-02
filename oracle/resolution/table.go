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

	"github.com/samber/lo"
)

// Table holds all resolved types from parsed Oracle schema files.
type Table struct {
	Types      []Type
	Imports    map[string]bool
	Namespaces map[string]bool
}

// NewTable creates a new table with built-in types pre-registered.
func NewTable() *Table {
	t := &Table{
		Types:      make([]Type, 0),
		Imports:    make(map[string]bool),
		Namespaces: make(map[string]bool),
	}
	t.registerBuiltins()
	return t
}

func (t *Table) registerBuiltins() {
	primitives := []string{
		"int8", "int16", "int32", "int64",
		"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
		"float32", "float64",
		"bool", "string", "uuid",
		"timestamp", "timespan", "time_range", "time_range_bounded",
		"json", "bytes", "data_type",
	}
	for _, name := range primitives {
		t.Types = append(t.Types, Type{
			Name:          name,
			QualifiedName: name,
			Form:          PrimitiveForm{Name: name},
		})
	}
	t.Types = append(t.Types, Type{
		Name:          "Array",
		QualifiedName: "Array",
		Form:          BuiltinGenericForm{Name: "Array", Arity: 1},
	})
	t.Types = append(t.Types, Type{
		Name:          "Map",
		QualifiedName: "Map",
		Form:          BuiltinGenericForm{Name: "Map", Arity: 2},
	})
}

func (t *Table) Get(qualifiedName string) (Type, bool) {
	return lo.Find(t.Types, func(typ Type) bool {
		return typ.QualifiedName == qualifiedName
	})
}

func (t *Table) MustGet(qualifiedName string) Type {
	typ, ok := t.Get(qualifiedName)
	if !ok {
		panic("type not found: " + qualifiedName)
	}
	return typ
}

func (t *Table) Add(typ Type) error {
	if _, ok := t.Get(typ.QualifiedName); ok {
		return fmt.Errorf("duplicate type: %s", typ.QualifiedName)
	}
	t.Types = append(t.Types, typ)
	return nil
}

// Lookup finds a type by namespace and name.
func (t *Table) Lookup(namespace, name string) (Type, bool) {
	qname := namespace + "." + name
	if typ, ok := t.Get(qname); ok {
		return typ, true
	}
	return lo.Find(t.Types, func(typ Type) bool {
		return typ.Name == name
	})
}

func (t *Table) TypesInNamespace(ns string) []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		return typ.Namespace == ns
	})
}

func (t *Table) TypesWithDomain(domain string) []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		_, ok := typ.Domains[domain]
		return ok
	})
}

func (t *Table) StructTypes() []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		_, ok := typ.Form.(StructForm)
		return ok
	})
}

func (t *Table) EnumTypes() []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		_, ok := typ.Form.(EnumForm)
		return ok
	})
}

func (t *Table) AliasTypes() []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		_, ok := typ.Form.(AliasForm)
		return ok
	})
}

func (t *Table) DistinctTypes() []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		_, ok := typ.Form.(DistinctForm)
		return ok
	})
}

func (t *Table) IsPrimitiveType(name string) bool {
	typ, ok := t.Get(name)
	if !ok {
		return false
	}
	_, isPrimitive := typ.Form.(PrimitiveForm)
	return isPrimitive
}

func (t *Table) IsStringPrimitiveType(name string) bool {
	return name == "string" || name == "uuid"
}

func (t *Table) IsNumberPrimitiveType(name string) bool {
	return lo.Contains([]string{
		"int8", "int16", "int32", "int64",
		"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
		"float32", "float64",
	}, name)
}

func (t *Table) MarkImported(path string) { t.Imports[path] = true }

func (t *Table) IsImported(path string) bool { return t.Imports[path] }

// EnumsInNamespace returns all enum types in the given namespace.
func (t *Table) EnumsInNamespace(ns string) []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		if typ.Namespace != ns {
			return false
		}
		_, ok := typ.Form.(EnumForm)
		return ok
	})
}

// StructsInNamespace returns all struct types in the given namespace.
func (t *Table) StructsInNamespace(ns string) []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		if typ.Namespace != ns {
			return false
		}
		_, ok := typ.Form.(StructForm)
		return ok
	})
}

// TopologicalSort returns a copy of the types slice sorted in topological order
// such that types appear after all types they depend on (within the same namespace).
// Cross-namespace dependencies are assumed to be resolved separately.
// Types with no dependencies retain their original relative order.
func (t *Table) TopologicalSort(types []Type) []Type {
	if len(types) <= 1 {
		return types
	}

	// Build a set of qualified names in the input list for quick lookup
	inSet := make(map[string]bool, len(types))
	for _, typ := range types {
		inSet[typ.QualifiedName] = true
	}

	// Build adjacency list: dependencies[A] = types that A depends on
	dependencies := make(map[string][]string, len(types))
	for _, typ := range types {
		deps := t.collectDependencies(typ)
		// Filter to only include dependencies that are in our input set
		var filteredDeps []string
		for _, dep := range deps {
			if inSet[dep] && dep != typ.QualifiedName {
				filteredDeps = append(filteredDeps, dep)
			}
		}
		dependencies[typ.QualifiedName] = filteredDeps
	}

	// Kahn's algorithm for topological sort
	// First, count incoming edges (how many types depend on each type)
	inDegree := make(map[string]int, len(types))
	for _, typ := range types {
		inDegree[typ.QualifiedName] = 0
	}
	for _, deps := range dependencies {
		for _, dep := range deps {
			inDegree[dep]++ // dep has one more type that depends on it
		}
	}

	// Wait, that's backwards. In Kahn's algorithm:
	// - We want types with no dependencies first
	// - A depends on B means B must come before A
	// So inDegree[A] = number of dependencies A has

	// Recalculate: inDegree[A] = count of types A depends on
	for qname := range inDegree {
		inDegree[qname] = len(dependencies[qname])
	}

	// Queue types with no dependencies
	var queue []string
	for qname, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, qname)
		}
	}

	// Process queue
	var sorted []string
	for len(queue) > 0 {
		// Pop first element
		qname := queue[0]
		queue = queue[1:]
		sorted = append(sorted, qname)

		// For each type that depends on qname, reduce its inDegree
		for _, typ := range types {
			deps := dependencies[typ.QualifiedName]
			for _, dep := range deps {
				if dep == qname {
					inDegree[typ.QualifiedName]--
					if inDegree[typ.QualifiedName] == 0 {
						queue = append(queue, typ.QualifiedName)
					}
					break
				}
			}
		}
	}

	// If we couldn't sort all types, there's a cycle - return original order
	if len(sorted) != len(types) {
		return types
	}

	// Map qualified names back to types
	typeMap := make(map[string]Type, len(types))
	for _, typ := range types {
		typeMap[typ.QualifiedName] = typ
	}

	result := make([]Type, 0, len(sorted))
	for _, qname := range sorted {
		result = append(result, typeMap[qname])
	}
	return result
}

// collectDependencies returns the qualified names of types that the given type depends on.
func (t *Table) collectDependencies(typ Type) []string {
	var deps []string
	seen := make(map[string]bool)

	var addDep func(ref TypeRef)
	addDep = func(ref TypeRef) {
		if ref.Name == "" || ref.IsTypeParam() {
			return
		}
		// Skip primitives and builtins
		if IsPrimitive(ref.Name) || ref.Name == "Array" || ref.Name == "Map" {
			// But still process type args
			for _, arg := range ref.TypeArgs {
				addDep(arg)
			}
			return
		}
		// Try to resolve to get qualified name
		if resolved, ok := t.Get(ref.Name); ok {
			if !seen[resolved.QualifiedName] {
				seen[resolved.QualifiedName] = true
				deps = append(deps, resolved.QualifiedName)
			}
		}
		// Process type arguments
		for _, arg := range ref.TypeArgs {
			addDep(arg)
		}
	}

	switch form := typ.Form.(type) {
	case StructForm:
		if form.Extends != nil {
			addDep(*form.Extends)
		}
		for _, field := range form.Fields {
			addDep(field.Type)
		}
	case AliasForm:
		addDep(form.Target)
	case DistinctForm:
		addDep(form.Base)
	}

	return deps
}
