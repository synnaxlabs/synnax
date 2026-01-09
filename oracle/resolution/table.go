// Copyright 2026 Synnax Labs, Inc.
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

type Table struct {
	Types      []Type
	Imports    map[string]bool
	Namespaces map[string]bool
}

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
		"json", "bytes", "data_type", "color", "any",
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

func (t *Table) EnumsInNamespace(ns string) []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		if typ.Namespace != ns {
			return false
		}
		_, ok := typ.Form.(EnumForm)
		return ok
	})
}

func (t *Table) StructsInNamespace(ns string) []Type {
	return lo.Filter(t.Types, func(typ Type, _ int) bool {
		if typ.Namespace != ns {
			return false
		}
		_, ok := typ.Form.(StructForm)
		return ok
	})
}

func (t *Table) TopologicalSort(types []Type) []Type {
	if len(types) <= 1 {
		return types
	}

	inSet := make(map[string]bool, len(types))
	for _, typ := range types {
		inSet[typ.QualifiedName] = true
	}

	dependencies := make(map[string][]string, len(types))
	for _, typ := range types {
		deps := t.collectDependencies(typ)
		var filteredDeps []string
		for _, dep := range deps {
			if inSet[dep] && dep != typ.QualifiedName {
				filteredDeps = append(filteredDeps, dep)
			}
		}
		dependencies[typ.QualifiedName] = filteredDeps
	}

	// Kahn's algorithm: inDegree[A] = count of types A depends on
	inDegree := make(map[string]int, len(types))
	for qname := range dependencies {
		inDegree[qname] = len(dependencies[qname])
	}

	// Build initial queue in declaration order by iterating over original slice
	var queue []string
	for _, typ := range types {
		if inDegree[typ.QualifiedName] == 0 {
			queue = append(queue, typ.QualifiedName)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		qname := queue[0]
		queue = queue[1:]
		sorted = append(sorted, qname)

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

	if len(sorted) != len(types) {
		return types // cycle detected
	}

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

func (t *Table) collectDependencies(typ Type) []string {
	var deps []string
	seen := make(map[string]bool)

	var addDep func(ref TypeRef)
	addDep = func(ref TypeRef) {
		if ref.Name == "" || ref.IsTypeParam() {
			return
		}
		if IsPrimitive(ref.Name) || ref.Name == "Array" || ref.Name == "Map" {
			for _, arg := range ref.TypeArgs {
				addDep(arg)
			}
			return
		}
		if resolved, ok := t.Get(ref.Name); ok {
			if !seen[resolved.QualifiedName] {
				seen[resolved.QualifiedName] = true
				deps = append(deps, resolved.QualifiedName)
			}
		}
		for _, arg := range ref.TypeArgs {
			addDep(arg)
		}
	}

	switch form := typ.Form.(type) {
	case StructForm:
		for _, extendsRef := range form.Extends {
			addDep(extendsRef)
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
