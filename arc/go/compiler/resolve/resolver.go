// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package resolve implements two-phase function index resolution for the Arc
// compiler. During compilation, Resolve returns temporary handles. During
// linking, FinalizeAndPatch partitions references into imports and locals,
// registers imports with the WASM module, and patches all writers with real
// function indices.
package resolve

import (
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

type pendingRef struct {
	qualifiedName string
	concreteType  types.Type
	typeSuffix    string
	handle        uint32
}

type compiledFunc struct {
	bodyIndex uint32
}

type patchEntry struct {
	handle uint32
	offset int
}

type writerPatches struct {
	writer  *wasm.Writer
	entries []patchEntry
}

// Resolver handles all function index resolution for the compiler.
// Phase 1 (compile): Resolve returns temporary handles and records references.
// Phase 2 (link): FinalizeAndPatch assigns real WASM function indices and
// patches all tracked writers.
type Resolver struct {
	symbols       symbol.Resolver
	pending       []pendingRef
	compiled      map[string]compiledFunc
	writers       []writerPatches
	handleCounter uint32
}

// NewResolver creates a new Resolver backed by the given symbol resolver.
func NewResolver(symbols symbol.Resolver) *Resolver {
	return &Resolver{
		symbols:  symbols,
		compiled: make(map[string]compiledFunc),
	}
}

// Resolve returns a temporary handle for the named function and records the
// reference for later linking. The concreteType is the monomorphized function
// type at the call site.
func (r *Resolver) Resolve(name string, concreteType types.Type) (uint32, error) {
	handle := r.handleCounter
	r.handleCounter++
	r.pending = append(r.pending, pendingRef{
		qualifiedName: name,
		concreteType:  concreteType,
		handle:        handle,
	})
	return handle, nil
}

// ResolveWithSuffix is like Resolve but uses an explicit type suffix instead of
// deriving it from type variables. Use this for functions where the WASM params
// are all i32 handles but the import name still needs a type suffix (e.g.,
// series_create_empty_f64).
func (r *Resolver) ResolveWithSuffix(name string, concreteType types.Type, suffix string) (uint32, error) {
	handle := r.handleCounter
	r.handleCounter++
	r.pending = append(r.pending, pendingRef{
		qualifiedName: name,
		concreteType:  concreteType,
		typeSuffix:    suffix,
		handle:        handle,
	})
	return handle, nil
}

// RegisterLocal records that a function body was compiled locally and will
// appear in the WASM module's code section at the given bodyIndex (0-based
// index into the locally-compiled function list).
func (r *Resolver) RegisterLocal(name string, bodyIndex uint32) {
	r.compiled[name] = compiledFunc{bodyIndex: bodyIndex}
}

// TrackWriter registers a writer for patch tracking and returns its ID.
func (r *Resolver) TrackWriter(w *wasm.Writer) int {
	id := len(r.writers)
	r.writers = append(r.writers, writerPatches{writer: w})
	return id
}

// RecordPlaceholder records that a call placeholder was written at the given
// offset in the tracked writer.
func (r *Resolver) RecordPlaceholder(writerID int, handle uint32, offset int) {
	r.writers[writerID].entries = append(r.writers[writerID].entries, patchEntry{
		handle: handle,
		offset: offset,
	})
}

// Finalize partitions pending references into imports and locals, registers
// import entries with the WASM module, and returns a map from temporary handles
// to real WASM function indices.
func (r *Resolver) Finalize(m *wasm.Module) (map[uint32]uint32, error) {
	type importKey struct {
		wasmModule string
		wasmName   string
	}
	importCache := make(map[importKey]uint32)
	patches := make(map[uint32]uint32, len(r.pending))

	var localRefs []pendingRef
	for _, ref := range r.pending {
		if _, ok := r.compiled[ref.qualifiedName]; ok {
			localRefs = append(localRefs, ref)
			continue
		}
		wasmMod, wasmName := DeriveWASMCoordinates(r.symbols, ref)
		key := importKey{wasmModule: wasmMod, wasmName: wasmName}
		if idx, ok := importCache[key]; ok {
			patches[ref.handle] = idx
			continue
		}
		ft := DeriveWASMFuncType(ref.concreteType)
		idx := m.AddImport(wasmMod, wasmName, ft)
		importCache[key] = idx
		patches[ref.handle] = idx
	}

	for _, ref := range localRefs {
		cf := r.compiled[ref.qualifiedName]
		patches[ref.handle] = m.ImportCount() + cf.bodyIndex
	}

	return patches, nil
}

// FinalizeAndPatch calls Finalize to resolve all function indices, then patches
// every tracked writer's call placeholders with the real indices.
func (r *Resolver) FinalizeAndPatch(m *wasm.Module) error {
	patches, err := r.Finalize(m)
	if err != nil {
		return err
	}
	for _, wp := range r.writers {
		for _, entry := range wp.entries {
			realIdx := patches[entry.handle]
			wp.writer.PatchCall(entry.offset, realIdx)
		}
	}
	return nil
}
