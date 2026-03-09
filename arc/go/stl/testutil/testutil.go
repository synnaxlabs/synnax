// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides shared test infrastructure for STL module unit
// tests. It wraps a real wazero runtime and builds passthrough WASM modules
// so that host-function bindings are exercised through the actual WASM ABI.
package testutil

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Runtime wraps a wazero.Runtime for use in tests. After calling a module's
// NewModule (which instantiates the host module), call Passthrough to build
// a WASM guest module that re-exports the host functions, then use Call to
// invoke them.
type Runtime struct {
	rt           wazero.Runtime
	passthroughs map[string]api.Module
}

// NewRuntime creates a wazero runtime using the interpreter (no JIT
// compilation overhead in tests).
func NewRuntime(ctx context.Context) *Runtime {
	rt := wazero.NewRuntimeWithConfig(
		ctx,
		wazero.NewRuntimeConfigInterpreter(),
	)
	return &Runtime{
		rt:           rt,
		passthroughs: make(map[string]api.Module),
	}
}

// Underlying returns the wazero.Runtime for passing to NewModule
// constructors.
func (r *Runtime) Underlying() wazero.Runtime { return r.rt }

// Close releases all resources held by the runtime.
func (r *Runtime) Close(ctx context.Context) error {
	return r.rt.Close(ctx)
}

// Passthrough creates a WASM guest module that imports all exported
// functions from the named host module and re-exports them as callable
// guest functions. This works around wazero's restriction that prevents
// calling ExportedFunction directly on host modules.
//
// ExportedFunctionDefinitions (metadata only) is NOT restricted on host
// modules, so we auto-discover all function signatures and generate a
// minimal WASM binary that calls through to each import.
func (r *Runtime) Passthrough(ctx context.Context, hostModuleName string) {
	hostMod := r.rt.Module(hostModuleName)
	if hostMod == nil {
		panic(fmt.Sprintf(
			"host module %q not instantiated", hostModuleName,
		))
	}

	defs := hostMod.ExportedFunctionDefinitions()
	imports := make([]funcImport, 0, len(defs))
	for name, def := range defs {
		imports = append(imports, funcImport{
			name:    name,
			params:  def.ParamTypes(),
			results: def.ResultTypes(),
		})
	}
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].name < imports[j].name
	})

	wasmBytes := buildPassthroughWASM(hostModuleName, imports)
	config := wazero.NewModuleConfig().
		WithName("test_" + hostModuleName)
	mod, err := r.rt.InstantiateWithConfig(ctx, wasmBytes, config)
	if err != nil {
		panic(fmt.Sprintf(
			"instantiate passthrough for %q: %v",
			hostModuleName, err,
		))
	}
	r.passthroughs[hostModuleName] = mod
}

// Call invokes a host function (through its passthrough wrapper) by module
// and function name. Panics on lookup or invocation errors so tests get
// clear failures. Call Passthrough before using this.
func (r *Runtime) Call(
	ctx context.Context,
	module, fn string,
	args ...uint64,
) []uint64 {
	mod, ok := r.passthroughs[module]
	if !ok {
		panic(fmt.Sprintf(
			"no passthrough for module %q; call Passthrough first",
			module,
		))
	}
	f := mod.ExportedFunction(fn)
	if f == nil {
		panic(fmt.Sprintf(
			"function %s.%s not found in passthrough", module, fn,
		))
	}
	results, err := f.Call(ctx, args...)
	if err != nil {
		panic(fmt.Sprintf(
			"call %s.%s failed: %v", module, fn, err,
		))
	}
	return results
}

// CallVoid is a convenience wrapper for host functions that return nothing.
func (r *Runtime) CallVoid(
	ctx context.Context,
	module, fn string,
	args ...uint64,
) {
	r.Call(ctx, module, fn, args...)
}

// Argument encoding helpers. These convert typed Go values into the uint64
// representation expected by the WASM ABI.

func U32(v uint32) uint64  { return uint64(v) }
func U64(v uint64) uint64  { return v }
func I32(v int32) uint64   { return uint64(uint32(v)) }
func I64(v int64) uint64   { return uint64(v) }
func F32(v float32) uint64 { return uint64(math.Float32bits(v)) }
func F64(v float64) uint64 { return math.Float64bits(v) }

// Result decoding helpers. These convert the uint64 WASM ABI result back
// into typed Go values.

func AsU32(v uint64) uint32  { return uint32(v) }
func AsU64(v uint64) uint64  { return v }
func AsI32(v uint64) int32   { return int32(uint32(v)) }
func AsI64(v uint64) int64   { return int64(v) }
func AsF32(v uint64) float32 { return math.Float32frombits(uint32(v)) }
func AsF64(v uint64) float64 { return math.Float64frombits(v) }

// funcImport describes a single function to import and re-export.
type funcImport struct {
	name    string
	params  []api.ValueType
	results []api.ValueType
}

// buildPassthroughWASM generates a minimal WASM binary that imports every
// function listed in imports (from moduleName) and defines a wrapper
// function for each that simply forwards all parameters and returns.
func buildPassthroughWASM(moduleName string, imports []funcImport) []byte {
	var buf bytes.Buffer

	// WASM magic number + version 1
	buf.Write([]byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00})

	n := len(imports)

	// Type section (id=1): one func type per import
	var typeSec bytes.Buffer
	writeULEB128(&typeSec, uint32(n))
	for _, imp := range imports {
		typeSec.WriteByte(0x60) // func type tag
		writeULEB128(&typeSec, uint32(len(imp.params)))
		for _, p := range imp.params {
			typeSec.WriteByte(p)
		}
		writeULEB128(&typeSec, uint32(len(imp.results)))
		for _, r := range imp.results {
			typeSec.WriteByte(r)
		}
	}
	writeSection(&buf, 1, typeSec.Bytes())

	// Import section (id=2)
	var impSec bytes.Buffer
	writeULEB128(&impSec, uint32(n))
	for i, imp := range imports {
		writeWASMString(&impSec, moduleName)
		writeWASMString(&impSec, imp.name)
		impSec.WriteByte(0x00)            // import kind: function
		writeULEB128(&impSec, uint32(i))  // type index
	}
	writeSection(&buf, 2, impSec.Bytes())

	// Function section (id=3): wrapper funcs with same types
	var funcSec bytes.Buffer
	writeULEB128(&funcSec, uint32(n))
	for i := range imports {
		writeULEB128(&funcSec, uint32(i))
	}
	writeSection(&buf, 3, funcSec.Bytes())

	// Export section (id=7)
	var expSec bytes.Buffer
	writeULEB128(&expSec, uint32(n))
	for i, imp := range imports {
		writeWASMString(&expSec, imp.name)
		expSec.WriteByte(0x00)                // export kind: function
		writeULEB128(&expSec, uint32(n+i))    // func index (after imports)
	}
	writeSection(&buf, 7, expSec.Bytes())

	// Code section (id=10): each wrapper calls the corresponding import
	var codeSec bytes.Buffer
	writeULEB128(&codeSec, uint32(n))
	for i, imp := range imports {
		var body bytes.Buffer
		writeULEB128(&body, 0) // no locals
		for j := range imp.params {
			body.WriteByte(0x20) // local.get
			writeULEB128(&body, uint32(j))
		}
		body.WriteByte(0x10) // call
		writeULEB128(&body, uint32(i)) // import function index
		body.WriteByte(0x0b) // end

		writeULEB128(&codeSec, uint32(body.Len()))
		codeSec.Write(body.Bytes())
	}
	writeSection(&buf, 10, codeSec.Bytes())

	return buf.Bytes()
}

func writeSection(buf *bytes.Buffer, id byte, content []byte) {
	buf.WriteByte(id)
	writeULEB128(buf, uint32(len(content)))
	buf.Write(content)
}

func writeWASMString(buf *bytes.Buffer, s string) {
	writeULEB128(buf, uint32(len(s)))
	buf.WriteString(s)
}

func writeULEB128(buf *bytes.Buffer, v uint32) {
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if v == 0 {
			break
		}
	}
}
