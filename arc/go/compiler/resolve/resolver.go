// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package resolve implements two-phase function index resolution for the Arc
// compiler. During compilation, EmitCall and EmitImportCall return temporary
// handles. During linking, FinalizeAndPatch partitions references into imports
// and locals, registers imports with the WASM module, and patches all writers
// with real function indices.
package resolve

import (
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// pendingRef is a recorded function reference awaiting linking. module is
// empty for locally-compiled functions and the WASM import module for host
// imports; name is the function name within that module (or the local
// function name). typeSuffix is appended to the import name when emitting
// the WASM import; empty for monomorphic functions and locals.
type pendingRef struct {
	module       string
	name         string
	typeSuffix   string
	concreteType types.Type
	handle       uint32
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
// Phase 1 (compile): EmitCall / EmitImportCall record references and return
// temporary handles. Phase 2 (link): FinalizeAndPatch assigns real WASM
// function indices and patches all tracked writers.
type Resolver struct {
	pending       []pendingRef
	compiled      map[string]compiledFunc
	writers       []writerPatches
	handleCounter uint32
}

// NewResolver creates an empty Resolver. The Resolver does not consult any
// scope on its own; callers either pass a resolved Symbol to EmitCall, or
// pass explicit (module, name) coordinates to EmitImportCall.
func NewResolver() *Resolver {
	return &Resolver{compiled: make(map[string]compiledFunc)}
}

// RegisterLocal records that a function body was compiled locally and will
// appear in the WASM module's code section at bodyIndex (0-based index into
// the locally-compiled function list). name must match the name later passed
// to EmitCall via a local-targeted Symbol or via EmitLocalCall.
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

// EmitCall records a call to a resolved function and writes a call
// placeholder at the writer's current offset. target's position in the
// symbol tree determines the WASM coordinate:
//
//   - A member of a KindModule (target.Parent.Kind == KindModule) becomes
//     a WASM import under (target.Parent.Name, target.Name). Any type
//     variables in target.Type are reconciled against concreteType to
//     produce a per-instantiation type suffix.
//   - Anything else is treated as a locally-compiled function looked up at
//     link time via RegisterLocal(target.Name, ...).
func (r *Resolver) EmitCall(
	w *wasm.Writer,
	writerID int,
	target *symbol.Symbol,
	concreteType types.Type,
) {
	var module, suffix string
	if target.Parent != nil && target.Parent.Kind == symbol.KindModule {
		module = target.Parent.Name
		suffix = DeriveTypeSuffix(target.Type, concreteType)
	}
	r.record(w, writerID, pendingRef{
		module:       module,
		name:         target.Name,
		typeSuffix:   suffix,
		concreteType: concreteType,
	})
}

// EmitImportCall records a call to a hardcoded host import by explicit
// (module, name) coordinates. Use this for compiler-emitted calls to
// well-known host functions (channels.read, series.set_element, ...) where
// there is no AST identifier to resolve.
func (r *Resolver) EmitImportCall(
	w *wasm.Writer,
	writerID int,
	module, name string,
	concreteType types.Type,
) {
	r.record(w, writerID, pendingRef{
		module:       module,
		name:         name,
		concreteType: concreteType,
	})
}

// EmitImportCallWithSuffix is EmitImportCall with an explicit type suffix.
// Use for host functions whose WASM params are all i32 handles but whose
// import name still needs a type suffix.
func (r *Resolver) EmitImportCallWithSuffix(
	w *wasm.Writer,
	writerID int,
	module, name string,
	concreteType types.Type,
	suffix string,
) {
	r.record(w, writerID, pendingRef{
		module:       module,
		name:         name,
		typeSuffix:   suffix,
		concreteType: concreteType,
	})
}

// EmitLocalCall records a call to a locally-compiled function by name.
// name must match a subsequent RegisterLocal call.
func (r *Resolver) EmitLocalCall(
	w *wasm.Writer,
	writerID int,
	name string,
	concreteType types.Type,
) {
	r.record(w, writerID, pendingRef{
		name:         name,
		concreteType: concreteType,
	})
}

func (r *Resolver) record(w *wasm.Writer, writerID int, ref pendingRef) {
	ref.handle = r.handleCounter
	r.handleCounter++
	r.pending = append(r.pending, ref)
	offset := w.WriteCallPlaceholder(ref.handle)
	r.RecordPlaceholder(writerID, ref.handle, offset)
}

// Finalize partitions pending references into imports and locals, registers
// import entries with the WASM module, and returns a map from temporary
// handles to real WASM function indices.
func (r *Resolver) Finalize(m *wasm.Module) map[uint32]uint32 {
	type importKey struct {
		wasmModule string
		wasmName   string
	}
	importCache := make(map[importKey]uint32)
	patches := make(map[uint32]uint32, len(r.pending))

	var localRefs []pendingRef
	for _, ref := range r.pending {
		if ref.module == "" {
			localRefs = append(localRefs, ref)
			continue
		}
		wasmName := ref.name
		if ref.typeSuffix != "" {
			wasmName = wasmName + "_" + ref.typeSuffix
		}
		key := importKey{wasmModule: ref.module, wasmName: wasmName}
		if idx, ok := importCache[key]; ok {
			patches[ref.handle] = idx
			continue
		}
		ft := DeriveWASMFuncType(ref.concreteType)
		idx := m.AddImport(ref.module, wasmName, ft)
		importCache[key] = idx
		patches[ref.handle] = idx
	}

	for _, ref := range localRefs {
		cf, ok := r.compiled[ref.name]
		if !ok {
			panic("unresolved local function: " + ref.name)
		}
		patches[ref.handle] = m.ImportCount() + cf.bodyIndex
	}

	return patches
}

// FinalizeAndPatch calls Finalize to resolve all function indices, then
// patches every tracked writer's call placeholders with the real indices.
func (r *Resolver) FinalizeAndPatch(m *wasm.Module) {
	patches := r.Finalize(m)
	for _, wp := range r.writers {
		for _, entry := range wp.entries {
			realIdx := patches[entry.handle]
			wp.writer.PatchCall(entry.offset, realIdx)
		}
	}
}
