// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bindings

import (
	"fmt"
	"sort"

	"github.com/synnaxlabs/arc/compiler/wasm"
)

// ImportDef defines a single WASM import.
type ImportDef struct {
	Module   string
	Name     string
	FuncType wasm.FunctionType
	Index    uint32
}

// ImportRegistry manages named imports organized by module.
// It provides a clean abstraction for registering host function imports
// and looking them up by module:name during compilation.
//
// The registry supports:
//   - Registration of imports with automatic index assignment
//   - Lookup by module:name for code generation
//   - Deterministic output order (by index) when writing to WASM module
//
// This replaces the fragile position-based import system with name-based resolution.
type ImportRegistry struct {
	imports map[string]map[string]ImportDef // module -> name -> def
	indices map[string]uint32               // "module:name" -> index
	nextIdx uint32
}

// NewImportRegistry creates a new import registry.
func NewImportRegistry() *ImportRegistry {
	return &ImportRegistry{
		imports: make(map[string]map[string]ImportDef),
		indices: make(map[string]uint32),
	}
}

// Register adds an import and returns its index.
// If the import already exists (same module:name), returns the existing index.
func (r *ImportRegistry) Register(module, name string, ft wasm.FunctionType) uint32 {
	key := module + ":" + name
	if idx, exists := r.indices[key]; exists {
		return idx // Already registered
	}

	idx := r.nextIdx
	r.nextIdx++

	if r.imports[module] == nil {
		r.imports[module] = make(map[string]ImportDef)
	}
	r.imports[module][name] = ImportDef{
		Module:   module,
		Name:     name,
		FuncType: ft,
		Index:    idx,
	}
	r.indices[key] = idx
	return idx
}

// Lookup returns the index for a module:name import.
func (r *ImportRegistry) Lookup(module, name string) (uint32, bool) {
	idx, ok := r.indices[module+":"+name]
	return idx, ok
}

// MustLookup panics if import not found (compile-time safety).
func (r *ImportRegistry) MustLookup(module, name string) uint32 {
	idx, ok := r.Lookup(module, name)
	if !ok {
		panic(fmt.Sprintf("import not registered: %s:%s", module, name))
	}
	return idx
}

// Count returns the total number of registered imports.
func (r *ImportRegistry) Count() uint32 {
	return r.nextIdx
}

// WriteToModule writes all imports to the WASM module in index order.
// This ensures deterministic output regardless of registration order.
func (r *ImportRegistry) WriteToModule(m *wasm.Module) {
	// Collect all defs and sort by index
	defs := make([]ImportDef, 0, r.nextIdx)
	for _, modImports := range r.imports {
		for _, def := range modImports {
			defs = append(defs, def)
		}
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Index < defs[j].Index
	})

	// Write to module in order
	for _, def := range defs {
		m.AddImport(def.Module, def.Name, def.FuncType)
	}
}

// Modules returns a list of all registered module names.
func (r *ImportRegistry) Modules() []string {
	modules := make([]string, 0, len(r.imports))
	for mod := range r.imports {
		modules = append(modules, mod)
	}
	sort.Strings(modules)
	return modules
}

// ModuleFunctions returns all function names in a module.
func (r *ImportRegistry) ModuleFunctions(module string) []string {
	modImports, ok := r.imports[module]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(modImports))
	for name := range modImports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
