# Arc Module System - Implementation Design

This document is the implementation blueprint for Arc's module system runtime
infrastructure. It specifies the exact types, interfaces, file locations, and migration
steps required to move from the current scattered, index-based binding architecture to a
unified, name-based module system.

**Scope**: Internal plumbing only. No user-facing `import` syntax. The design must not
close off future `import ( math )` syntax, but that is not a deliverable.

**Success criteria**: A developer can trace the path from "STL function definition" to
"analyzer resolution" to "compiler emission" to "runtime execution" in both Go and C++
without getting lost. Adding a new STL function like `math.sqrt` requires changes in one
package.

---

## 1. Current Architecture (What Exists Today)

### 1.1 Host Function Registration

~390 host functions are registered under a single WASM module `"env"` with flat names
like `channel_read_f64`, `series_element_add_u32`, `math_pow_i64`.

**Compiler side** (`compiler/bindings/imports.go`): `SetupImports()` registers every
host function upfront with the WASM module via `m.AddImport("env", name, funcType)`.
Returns an `ImportIndex` struct with ~40 fields (many `map[string]uint32`) storing the
positional index of each import.

**Go runtime side**: Three files totaling ~10,000 lines:

- `runtime/wasm/bindings/runtime_generated.go` (~4,890 lines): `Runtime` struct with one
  method per host function per type variant
- `compiler/bindings/static_bindings_generated.go` (~4,647 lines): `Bindings` struct
  with function pointer fields and `Bind()` method that registers everything with wazero
  under `"env"`
- `runtime/wasm/bindings/bindings.go` (~440 lines): `BindRuntime()` wires Runtime
  methods to Bindings fields

**C++ runtime side** (`arc/cpp/runtime/wasm/bindings.cpp`): `create_imports()` builds a
positional `vector<wasmtime::Extern>` in the exact same order as the Go side. Uses
`MethodWrapper` template for type conversion.

**The brittleness**: Adding one host function shifts every subsequent index. All three
codebases must stay in lockstep. Previously compiled WASM binaries become incompatible
(though this is a non-issue since Arc always recompiles from source).

### 1.2 Symbol Resolution

Symbols are resolved through `symbol.Resolver` interface:

```go
type Resolver interface {
    Resolve(ctx context.Context, name string) (Symbol, error)
    Search(ctx context.Context, term string) ([]Symbol, error)
}
```

`CompoundResolver` chains `MapResolver` instances in order. Currently assembled in
`core/pkg/service/arc/symbol/resolver.go`:

```
constant.SymbolResolver → op.SymbolResolver → selector.SymbolResolver →
stable.SymbolResolver → status.SymbolResolver → authority.SymbolResolver →
telem.SymbolResolver → stat.SymbolResolver → time.SymbolResolver → channelResolver
```

Each `runtime/` package defines its own `MapResolver` as a package-level var.

### 1.3 Node Factories

`node.MultiFactory` in `core/pkg/service/arc/runtime/task.go`:

```
telem.Factory → selector.Factory → constant.Factory → op.Factory →
stage.Factory → time.Factory → stable.Factory → status.Factory →
authority.Factory → [wasm.Factory if WASM bytecode exists]
```

### 1.4 Compiler Function Resolution

Three separate mechanisms for getting a `uint32` function index:

1. **`ImportIndex`** - Host functions via `ctx.Imports.GetChannelRead(t)` or direct
   field access like `ctx.Imports.Now`
2. **`FunctionIndices`** - User-defined functions via `ctx.FunctionIndices[funcName]`,
   populated as `importCount + i`
3. **`compileBuiltinCall`** - Switch statement hardcoding `len` and `now`

### 1.5 Runtime State

`state.State` is a monolithic struct holding:

- `channel.reads` / `channel.writes` - Channel I/O buffers
- `outputs` - Node output values + timestamps
- `indexes` - Channel-to-index-channel mapping
- `authorityChanges` - Buffered authority changes
- `series` / `seriesHandleCounter` - Transient series handle store
- `strings` / `stringHandleCounter` - Transient string handle store

`runtime/wasm/bindings/Runtime` additionally holds per-node stateful variable maps:
`stateU8[nodeKey][varID]`, `stateI32[nodeKey][varID]`, etc. (12 typed maps).

---

## 2. Module Interface

### 2.1 Design Principles

- A Module IS a `symbol.Resolver` and a `node.Factory` (interface embedding, not getter
  methods returning lists)
- Each module is a stateful object that owns its state slice
- Dependencies between modules are explicit constructor parameters
- The `Module` interface has exactly the methods it needs and no more

### 2.2 The Interface

```go
package stl

// Module is the unit of STL organization. It provides symbols for the analyzer,
// node factories for the scheduler, and host function implementations for the
// WASM runtime. A Module may implement any combination of these roles.
type Module interface {
    // Resolve returns the symbol for the given name, or query.NotFound if this
    // module doesn't own it. The name may be qualified ("math.sqrt") or bare
    // ("set_authority") depending on how the module registers itself.
    symbol.Resolver

    // Create returns a reactive graph node for the given config, or
    // query.NotFound if this module doesn't handle the node type.
    node.Factory

    // BindTo registers this module's host function implementations with the
    // WASM runtime. Called once during runtime setup, before any WASM execution.
    BindTo(ctx context.Context, rt HostRuntime) error
}
```

### 2.3 HostRuntime Interface

Abstracts the WASM runtime (wazero/wasmtime) so modules don't import engine-specific
packages:

```go
// HostRuntime provides the ability to register host functions with the WASM
// engine. Implementations wrap wazero (Go) or wasmtime (C++).
type HostRuntime interface {
    // Export registers a Go function as a host function callable from WASM.
    // The wasmModule parameter is the WASM import module name (e.g., "math").
    // The name parameter is the function name within that module (e.g., "sqrt_f64").
    // The impl parameter must be a Go function with wazero-compatible signature
    // (params/returns limited to uint32, uint64, float32, float64, context.Context).
    Export(wasmModule, name string, impl any) error
}
```

The wazero implementation groups exports by WASM module name internally and uses
`NewHostModuleBuilder` / `NewFunctionBuilder` / `WithFunc` / `Export`.

### 2.4 Module Construction

Modules are constructed with explicit dependencies. Construction order is enforced by
Go's type system:

```go
// In core/pkg/service/arc/runtime/task.go (assembly point):

// Pure modules (no external deps, live in arc/)
channelMod := channel.NewModule()
stateMod   := state.NewModule()
seriesMod  := series.NewModule()
stringMod  := string.NewModule()
mathMod    := math.NewModule()
timeMod    := time.NewModule()
opMod      := op.NewModule()
controlMod := control.NewModule()  // set_authority, stable_for, select, constant
telemMod   := telem.NewModule()    // on, write
stageMod   := stage.NewModule()
statMod    := stat.NewModule()
errors   := error.NewModule()

// Server-dependent modules (live in core/)
statusMod := status.NewModule(statusService)

// Assembly
modules := []stl.Module{
    channelMod, stateMod, seriesMod, stringMod, mathMod, timeMod,
    opMod, controlMod, telemMod, stageMod, statMod, errors,
    statusMod,
}
```

From this one list, the system derives:

- **Symbol resolver**: `stl.CompoundResolver(modules...)` chains all modules
- **Node factory**: `stl.MultiFactory(modules...)` chains all factories
- **WASM host binding**: iterate modules, call `BindTo(ctx, rt)` on each

### 2.5 ModuleResolver (Concrete Helper)

For modules that own a namespace (like `math`), a `ModuleResolver` helper strips the
prefix before delegating to the module's internal `MapResolver`:

```go
// ModuleResolver is a concrete symbol.Resolver that handles qualified name
// resolution for a named module. It strips "name." prefix before delegating.
type ModuleResolver struct {
    Name    string
    Members symbol.MapResolver
}

func (m *ModuleResolver) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
    if !strings.HasPrefix(name, m.Name+".") {
        return symbol.Symbol{}, query.NotFound
    }
    return m.Members.Resolve(ctx, strings.TrimPrefix(name, m.Name+"."))
}
```

For auto-imported symbols (like `set_authority`), modules use a plain `MapResolver` with
bare names. The `symbol.Resolver` interface doesn't change.

---

## 3. Compiler Changes

### 3.1 Two-Phase Compile-Then-Link

The current compiler registers all ~390 imports upfront and assigns indices immediately.
The new design separates compilation from index assignment.

**Phase 1 (compile)**: The compiler compiles all function bodies. When it encounters a
function call (host or local), it calls `ctx.Resolve(qualifiedName, concreteType)` which
returns a temporary handle and records the reference. The Writer emits call instructions
with placeholder operands.

**Phase 2 (link)**: After all functions are compiled, the resolver knows exactly which
host functions were referenced. It partitions references into imports (no compiled body)
vs locals (compiled body exists). Imports get indices 0..N-1, locals get N..N+M-1. A
fixup pass patches all placeholder call operands with real indices.

### 3.2 The Compiler Resolver

Replaces `ImportIndex` + `FunctionIndices` + `compileBuiltinCall` with a single
resolver:

```go
package resolve

// Resolver handles all function index resolution for the compiler.
// It unifies host functions, user-defined functions, and builtins
// into a single lookup mechanism.
type Resolver struct {
    // symbols is the compound symbol resolver from all modules.
    // Used to look up original (polymorphic) symbol definitions.
    symbols symbol.Resolver

    // pending tracks every function reference encountered during Phase 1.
    pending []pendingRef

    // compiled tracks locally-compiled function bodies (by qualified name).
    compiled map[string]compiledFunc

    // handleCounter assigns temporary handles during Phase 1.
    handleCounter uint32
}

type pendingRef struct {
    qualifiedName string      // e.g., "math.sqrt", "channel.read", "myHelper"
    concreteType  types.Type  // Fully resolved type from analyzer
    handle        uint32      // Temporary handle emitted in Phase 1
    callSites     []int       // Byte offsets in Writer where handle was written
}

type compiledFunc struct {
    bodyIndex uint32  // Position in the compiled function list
}
```

### 3.3 Resolve Method

```go
// Resolve returns a temporary handle for a function call and records the
// reference for Phase 2 linking. The concreteType is the fully-resolved
// function type from the analyzer (after constraint solving).
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
```

### 3.4 RegisterLocal Method

Called when the compiler compiles a user-defined function body:

```go
func (r *Resolver) RegisterLocal(name string, bodyIndex uint32) {
    r.compiled[name] = compiledFunc{bodyIndex: bodyIndex}
}
```

### 3.5 Finalize Method (Phase 2)

```go
// Finalize assigns real WASM function indices and returns a patch map
// from temporary handles to real indices.
func (r *Resolver) Finalize(m *wasm.Module) (map[uint32]uint32, error) {
    patches := make(map[uint32]uint32)

    // Partition into imports vs locals
    var imports []pendingRef
    var locals []pendingRef
    for _, ref := range r.pending {
        if _, isLocal := r.compiled[ref.qualifiedName]; isLocal {
            locals = append(locals, ref)
        } else {
            imports = append(imports, ref)
        }
    }

    // Register imports with the WASM module, get indices 0..N-1
    importIndices := make(map[string]uint32) // dedup by concrete WASM name
    for _, ref := range imports {
        wasmModule, wasmName := r.deriveWASMCoordinates(ref)
        key := wasmModule + "/" + wasmName
        if idx, exists := importIndices[key]; exists {
            patches[ref.handle] = idx
            continue
        }
        funcType := r.deriveWASMFuncType(ref.concreteType)
        idx := m.AddImport(wasmModule, wasmName, funcType)
        importIndices[key] = idx
        patches[ref.handle] = idx
    }

    // Assign local function indices: importCount + bodyIndex
    importCount := m.ImportCount()
    for _, ref := range locals {
        compiled := r.compiled[ref.qualifiedName]
        patches[ref.handle] = importCount + compiled.bodyIndex
    }

    return patches, nil
}
```

### 3.6 WASM Name Derivation

The resolver derives WASM import coordinates from the qualified name and concrete type:

```go
func (r *Resolver) deriveWASMCoordinates(ref pendingRef) (string, string) {
    // Look up original symbol to check for type variables
    originalSym, _ := r.symbols.Resolve(context.Background(), ref.qualifiedName)

    // Split qualified name on last dot
    lastDot := strings.LastIndex(ref.qualifiedName, ".")
    wasmModule := ref.qualifiedName[:lastDot]   // e.g., "channel"
    baseName := ref.qualifiedName[lastDot+1:]   // e.g., "read"

    // If original symbol has type variables, append type suffix
    if hasTypeVariables(originalSym.Type) {
        suffix := deriveTypeSuffix(originalSym.Type, ref.concreteType)
        return wasmModule, baseName + "_" + suffix  // e.g., "read_f64"
    }

    return wasmModule, baseName  // e.g., "time", "now"
}
```

### 3.7 Writer Changes

The Writer needs to support placeholder patching. Two approaches:

**Option A (simple)**: Fixed-width LEB128. Write call operands as 5-byte padded LEB128
(the maximum width for a uint32). Phase 2 overwrites in place. This wastes ~3 bytes per
call instruction but avoids offset recalculation.

**Option B (tracked)**: The Writer records the byte offset of each call operand. Phase 2
builds a new byte buffer with correct variable-length LEB128. More complex but produces
smaller binaries.

**Recommendation**: Option A. The wasted bytes are negligible (~1.5KB for 500 calls) and
the implementation is dramatically simpler. The Writer gains one method:

```go
// WriteCallPlaceholder writes a call instruction with a fixed-width 5-byte
// LEB128 operand that can be patched later.
func (w *Writer) WriteCallPlaceholder(handle uint32) int {
    w.WriteOpcode(OpCall)
    offset := w.buf.Len()
    w.WriteLEB128Fixed5(uint64(handle))  // Always 5 bytes
    return offset
}

// PatchCall overwrites a previously written 5-byte call operand.
func (w *Writer) PatchCall(offset int, realIndex uint32) {
    encodeLEB128Fixed5(w.buf.Bytes()[offset:offset+5], uint64(realIndex))
}
```

### 3.8 Compiler Primitive Resolution

Compiler primitives (channel read/write, state load/store, series ops) are called
through the same resolver. The compiler constructs qualified names using conventions:

```go
// Channel read: compiler sees `sensor` (a channel alias), needs to read it
importHandle, _ := ctx.Resolver.Resolve("channel.read", types.Func(types.I32, channelElemType))

// State store: compiler sees `total $= 0` assignment
importHandle, _ := ctx.Resolver.Resolve("state.store", types.Func(types.I32, varType))

// Series arithmetic: compiler sees `data + 5.0` where data is series f64
importHandle, _ := ctx.Resolver.Resolve("series.element_add", types.Func(types.I32, types.F64))
```

Each module's symbol resolver has internal entries for these (with an `Internal` flag on
the symbol so they don't appear in user-facing autocomplete).

### 3.9 What Gets Deleted in the Compiler

| Current File                                     | Lines  | Replacement                            |
| ------------------------------------------------ | ------ | -------------------------------------- |
| `compiler/bindings/imports.go`                   | ~470   | `compiler/resolve/resolver.go` (~250)  |
| `compiler/bindings/helpers.go`                   | ~240   | Absorbed into resolver name derivation |
| `compiler/bindings/static_bindings_generated.go` | ~4,647 | Eliminated entirely                    |
| `compileBuiltinCall` in `expression/compiler.go` | ~50    | Eliminated (regular symbol resolution) |

The `compiler/context/context.go` loses `Imports *bindings.ImportIndex` and
`FunctionIndices map[string]uint32`, gains `Resolver *resolve.Resolver`.

---

## 4. Runtime Changes

### 4.1 Per-Module Host Binding

Replace the single `"env"` host module with per-STL-module host modules. The
`HostRuntime` implementation groups exports by WASM module name:

```go
type wazeroHostRuntime struct {
    ctx     context.Context
    rt      wazero.Runtime
    builders map[string]wazero.HostModuleBuilder
}

func (w *wazeroHostRuntime) Export(wasmModule, name string, impl any) error {
    b, ok := w.builders[wasmModule]
    if !ok {
        b = w.rt.NewHostModuleBuilder(wasmModule)
        w.builders[wasmModule] = b
    }
    b.NewFunctionBuilder().WithFunc(impl).Export(name)
    return nil
}

func (w *wazeroHostRuntime) Instantiate() error {
    for _, b := range w.builders {
        if _, err := b.Instantiate(w.ctx); err != nil {
            return err
        }
    }
    return nil
}
```

### 4.2 Module BindTo Implementations

Each module's `BindTo` registers its host functions with wazero-compatible signatures.

**Example: math module**

```go
func (m *MathModule) BindTo(ctx context.Context, rt stl.HostRuntime) error {
    rt.Export("math", "pow_f64", func(_ context.Context, base, exp float64) float64 {
        return gomath.Pow(base, exp)
    })
    rt.Export("math", "pow_f32", func(_ context.Context, base, exp float32) float32 {
        return float32(gomath.Pow(float64(base), float64(exp)))
    })
    // Integer pow variants...
    return nil
}
```

**Example: channel module (with state)**

```go
type ChannelModule struct {
    state *channelState  // owns channel read/write buffers
}

func (m *ChannelModule) BindTo(ctx context.Context, rt stl.HostRuntime) error {
    // Use generics to reduce boilerplate for i32-compatible types
    bindChannelOps[uint8](rt, m.state, "u8")
    bindChannelOps[uint16](rt, m.state, "u16")
    bindChannelOps[uint32](rt, m.state, "u32")
    bindChannelOps[int8](rt, m.state, "i8")
    bindChannelOps[int16](rt, m.state, "i16")
    bindChannelOps[int32](rt, m.state, "i32")
    // ... f32, f64, i64, u64, str
    return nil
}

func bindChannelOps[T channelNumeric](rt stl.HostRuntime, s *channelState, suffix string) {
    rt.Export("channel", "read_"+suffix, func(_ context.Context, chID uint32) uint32 {
        series, ok := s.ReadChannelValue(chID)
        if !ok || series.Len() == 0 { return 0 }
        return uint32(telem.ValueAt[T](series, -1))
    })
    rt.Export("channel", "write_"+suffix, func(_ context.Context, chID uint32, val uint32) {
        s.WriteChannelValue(chID, telem.NewSeriesV[T](T(val)))
    })
}
```

### 4.3 Go Generics for Type Variants

The `channelNumeric` constraint and similar constraints replace the code generator:

```go
type i32Compatible interface {
    ~uint8 | ~uint16 | ~uint32 | ~int8 | ~int16 | ~int32
}

type i64Compatible interface {
    ~uint64 | ~int64
}
```

Functions that operate on i32-compatible types share a single generic implementation.
The `wazero` runtime infers WASM types from Go function signatures via reflection, so
`func(context.Context, uint32) uint32` automatically maps to `(i32) -> i32`.

Functions for `i64`, `f32`, `f64` types need separate registrations because they map to
different WASM types, but the implementation body is still generic.

### 4.4 State Decomposition

The monolithic `state.State` is decomposed into module-owned state slices. Each module
creates its own state, and the runtime holds references for I/O operations.

```go
// Channel module owns channel I/O buffers
type channelState struct {
    reads  map[uint32]telem.MultiSeries
    writes map[uint32]telem.Series
    indexes map[uint32]uint32
}

// State module owns stateful variable persistence
type variableState struct {
    // Per-node, per-variable storage (12 typed maps)
    currentNodeKey string
    stateU8  map[string]map[uint32]uint8
    stateI32 map[string]map[uint32]int32
    // ... etc
}

// Series module owns transient series handle store
type seriesState struct {
    series  map[uint32]telem.Series
    counter uint32
}

// String module owns transient string handle store
type stringState struct {
    strings map[uint32]string
    counter uint32
}
```

The runtime (in `core/`) orchestrates the flush cycle using module-specific state
handles:

```go
// In dataRuntime.next():
channelMod.State().Ingest(frame)        // populate reads
scheduler.Next(ctx, elapsed, reason)     // execute nodes (WASM calls hit state)
channelMod.State().ClearReads()
authorityChanges := controlMod.State().FlushAuthorityChanges()
flushAuthority(authorityChanges)         // authority BEFORE writes
frame, changed := channelMod.State().Flush()
if changed { sendFrame(frame) }
seriesMod.State().Clear()                // clear transient handles
stringMod.State().Clear()
```

### 4.5 Node Key Injection

The current `Runtime.SetCurrentNodeKey()` is replaced by `context.WithValue`:

```go
// Before each WASM call in nodeImpl.Next():
ctx := context.WithValue(parentCtx, nodeKeyCtxKey, n.ir.Key)
n.wasm.Call(ctx, params...)

// In host function implementations:
func stateLoadF64(varState *variableState) func(context.Context, uint32, float64) float64 {
    return func(ctx context.Context, varID uint32, initValue float64) float64 {
        nodeKey := ctx.Value(nodeKeyCtxKey).(string)
        if m, ok := varState.stateF64[nodeKey]; ok {
            if v, ok := m[varID]; ok {
                return v
            }
        }
        return initValue
    }
}
```

### 4.6 What Gets Deleted in the Runtime

| Current File                                     | Lines       | Replacement                                                 |
| ------------------------------------------------ | ----------- | ----------------------------------------------------------- |
| `runtime/wasm/bindings/runtime_generated.go`     | ~4,890      | Module `BindTo` implementations (~800 total across modules) |
| `runtime/wasm/bindings/bindings.go`              | ~440        | Eliminated (direct `BindTo` calls)                          |
| `runtime/wasm/bindings/generate/main.go`         | ~820        | Eliminated (generics replace codegen)                       |
| `compiler/bindings/static_bindings_generated.go` | ~4,647      | Eliminated (no Bindings struct)                             |
| `runtime/builtin/builtin.go`                     | ~47         | Moved to time module + series module                        |
| `runtime/authority/authority.go`                 | ~99         | Moved to control module                                     |
| `runtime/stable/stable.go`                       | ~127        | Moved to control module                                     |
| `runtime/selector/select.go`                     | ~110        | Moved to control module                                     |
| `runtime/constant/constant.go`                   | ~72         | Moved to control module                                     |
| **Total removed**                                | **~11,252** | **~800 new across modules**                                 |

---

## 5. WASM Import Contract

The WASM binary's import section is the contract between compiler and runtimes. With
name-based binding, order is irrelevant. Both Go and C++ runtimes resolve imports by
`(module, name)` pairs.

### 5.1 Namespace Mapping

| WASM Module | Function Pattern                                            | Example                        |
| ----------- | ----------------------------------------------------------- | ------------------------------ |
| `channel`   | `read_{type}`, `write_{type}`                               | `channel.read_f64`             |
| `state`     | `load_{type}`, `store_{type}`                               | `state.store_i32`              |
| `state`     | `load_series_{type}`, `store_series_{type}`                 | `state.load_series_f64`        |
| `series`    | `create_empty_{type}`, `set_element_{type}`, `index_{type}` | `series.index_u8`              |
| `series`    | `element_{op}_{type}`                                       | `series.element_add_f64`       |
| `series`    | `element_r{op}_{type}`                                      | `series.element_rsub_i32`      |
| `series`    | `series_{op}_{type}`                                        | `series.series_mul_f32`        |
| `series`    | `compare_{cmp}_{type}`                                      | `series.compare_gt_f64`        |
| `series`    | `compare_{cmp}_scalar_{type}`                               | `series.compare_le_scalar_i32` |
| `series`    | `negate_{type}`                                             | `series.negate_f64`            |
| `series`    | `not_u8`, `len`, `slice`                                    | `series.len`                   |
| `math`      | `pow_{type}`                                                | `math.pow_f64`                 |
| `time`      | `now`                                                       | `time.now`                     |
| `string`    | `from_literal`, `concat`, `equal`, `len`                    | `string.concat`                |
| `error`     | `panic`                                                     | `error.panic`                  |

Where `{type}` = `u8|u16|u32|u64|i8|i16|i32|i64|f32|f64` (plus `str` for channel/state).
Where `{op}` = `add|sub|mul|div|mod`. Where `{cmp}` = `gt|lt|ge|le|eq|ne`.

### 5.2 C++ Migration

The C++ runtime migrates from positional `vector<Extern>` to wasmtime's `Linker` API:

```cpp
wasmtime::Linker linker(engine);
linker.define("channel", "read_f64", wasmtime::Func::wrap(store, [&](uint32_t chID) -> double {
    return bindings->channel_read_f64(chID);
}));
// ... for each host function
auto instance = linker.instantiate(store, module);
```

Order becomes irrelevant. The WASM binary declares its imports, and the linker matches
by name. This is a C++-only change that can happen independently.

---

## 6. Package Layout

### 6.1 New Directory Structure

```
arc/go/stl/
    stl.go              // Module interface, HostRuntime interface, CompoundResolver,
                        // MultiFactory helpers
    channel/
        channel.go      // ChannelModule: channel read/write host functions
        state.go        // channelState: read/write buffers, flush, ingest
    state/
        state.go        // StateModule: stateful variable load/store host functions
        state.go        // variableState: per-node typed maps
    series/
        series.go       // SeriesModule: create, index, slice, arithmetic, comparison
        state.go        // seriesState: transient handle store
    string/
        string.go       // StringModule: from_literal, concat, equal, len
        state.go        // stringState: transient handle store
    math/
        math.go         // MathModule: pow (+ future sqrt, sin, cos, abs, clamp)
    time/
        time.go         // TimeModule: now (host func) + interval, wait (graph nodes)
    op/
        op.go           // OpModule: operator graph nodes + series arithmetic host funcs
    control/
        control.go      // ControlModule: set_authority, stable_for, select, constant
    telem/
        telem.go        // TelemModule: on, write (source/sink graph nodes)
    stage/
        stage.go        // StageModule: stage_entry graph node
    stat/
        stat.go         // StatModule: avg, min, max graph nodes
    error/
        error.go        // errorsule: panic host function

arc/go/compiler/
    resolve/
        resolver.go     // Resolver: two-phase compile-then-link
        derive.go       // WASM name derivation, type suffix logic
```

### 6.2 Files That Move

| From                                            | To                                          |
| ----------------------------------------------- | ------------------------------------------- |
| `runtime/builtin/builtin.go` (now, len symbols) | `stl/time/time.go` + `stl/series/series.go` |
| `runtime/authority/authority.go`                | `stl/control/control.go`                    |
| `runtime/stable/stable.go`                      | `stl/control/control.go`                    |
| `runtime/selector/select.go`                    | `stl/control/control.go`                    |
| `runtime/constant/constant.go`                  | `stl/control/control.go`                    |
| `runtime/op/`                                   | `stl/op/op.go`                              |
| `runtime/telem/telem.go`                        | `stl/telem/telem.go`                        |
| `runtime/stat/stat.go`                          | `stl/stat/stat.go`                          |
| `runtime/time/time.go`                          | `stl/time/time.go`                          |
| `runtime/stage/`                                | `stl/stage/stage.go`                        |

### 6.3 Files in `core/` (Server-Dependent)

| File                                      | Content                                                                          |
| ----------------------------------------- | -------------------------------------------------------------------------------- |
| `core/pkg/service/arc/status/set.go`      | `StatusModule`: set_status symbol + factory (stays here, needs `status.Service`) |
| `core/pkg/service/arc/runtime/task.go`    | Module assembly point (constructs all modules, wires resolver/factory/runtime)   |
| `core/pkg/service/arc/symbol/resolver.go` | Simplified: creates `stl.CompoundResolver(modules...)` + channelResolver         |

---

## 7. Implementation Phases

### Phase 1: Foundation (No Behavior Change)

**Goal**: Create `stl/` package with core types. Old and new paths coexist.

1. Create `arc/go/stl/stl.go` with `Module`, `HostRuntime` interfaces
2. Create `arc/go/stl/channel/`, `stl/math/`, etc. as empty module stubs that satisfy
   the interface but delegate to existing code
3. Create `arc/go/compiler/resolve/resolver.go` with the `Resolver` type
4. Write tests for `ModuleResolver`, `HostRuntime` wazero implementation

**Validation**: Existing tests pass unchanged. New types compile.

### Phase 2: Compiler Migration

**Goal**: Replace `ImportIndex` with `Resolver`. WASM binaries use module namespaces.

1. Implement `Resolver.Resolve()`, `RegisterLocal()`, `Finalize()`
2. Implement `deriveWASMCoordinates()` and `deriveTypeSuffix()`
3. Add `WriteCallPlaceholder()` and `PatchCall()` to Writer
4. Update `compiler/context/context.go`: replace `Imports` + `FunctionIndices` with
   `Resolver`
5. Update all compiler callsites:
   - `ctx.Imports.GetChannelRead(t)` → `ctx.Resolver.Resolve("channel.read", ...)`
   - `ctx.Imports.Now` → `ctx.Resolver.Resolve("time.now", ...)`
   - `ctx.Imports.SeriesLen` → `ctx.Resolver.Resolve("series.len", ...)`
   - `ctx.FunctionIndices[name]` → `ctx.Resolver.Resolve(name, ...)`
   - Remove `compileBuiltinCall` entirely
6. Temporarily support both old `"env"` and new module namespaces in the Go runtime (for
   test continuity during migration)

**Validation**: All compiler tests pass. Generated WASM binaries use new import names.

### Phase 3: Go Runtime Migration

**Goal**: Replace generated binding code with module `BindTo` implementations.

1. Implement `wazeroHostRuntime` adapter
2. Implement `BindTo` for each module using Go generics:
   - `channel.BindTo` - channel read/write (generic over numeric types)
   - `state.BindTo` - state load/store (generic over numeric types)
   - `series.BindTo` - all series ops (generic over numeric types and ops)
   - `math.BindTo` - pow variants
   - `time.BindTo` - now
   - `string.BindTo` - string ops
   - `error.BindTo` - panic
3. Update `runtime/wasm/module.go` (`OpenModule`) to iterate modules and call `BindTo`
   instead of using `Bindings.Bind()`
4. Delete `runtime/wasm/bindings/runtime_generated.go`
5. Delete `runtime/wasm/bindings/bindings.go`
6. Delete `runtime/wasm/bindings/generate/main.go`
7. Delete `compiler/bindings/static_bindings_generated.go`

**Validation**: All runtime tests pass. WASM execution works with new binding path.

### Phase 4: STL Symbol + Factory Migration

**Goal**: Move symbol definitions and node factories from `runtime/` packages to `stl/`.

1. Move symbol resolvers and factory implementations to corresponding `stl/` packages
2. Each `stl/` module exposes both symbols and factories through the `Module` interface
3. Update `core/pkg/service/arc/runtime/task.go` to construct modules and derive
   resolver/factory from the module list
4. Update `core/pkg/service/arc/symbol/resolver.go` to use module-based resolution
5. Delete old `runtime/builtin/`, `runtime/authority/`, `runtime/stable/`,
   `runtime/selector/`, `runtime/constant/`

**Validation**: All tests pass. Symbol resolution and node creation work through
modules.

### Phase 5: State Decomposition

**Goal**: Decompose `state.State` into module-owned state slices.

1. Extract `channelState` from `state.State` into `stl/channel/state.go`
2. Extract `variableState` from `Runtime` struct into `stl/state/state.go`
3. Extract `seriesState` from `state.State` into `stl/series/state.go`
4. Extract `stringState` from `state.State` into `stl/string/state.go`
5. Update `core/` runtime to use module state handles for ingest/flush
6. Delete `runtime/wasm/bindings/` directory entirely

**Validation**: All tests pass. State is discoverable through module handles.

### Phase 6: Cleanup

**Goal**: Remove all vestiges of the old architecture.

1. Delete `compiler/bindings/` directory entirely
2. Produce C++ WASM import contract specification (the namespace mapping table)
3. Update documentation
4. Verify no references to old `ImportIndex`, `Bindings`, `BindRuntime` remain

---

## 8. Risk Assessment

### 8.1 Managed Risks

**Test continuity**: Each phase maintains passing tests. Phase 2 temporarily supports
both old and new import names.

**C++ compatibility**: The C++ runtime is NOT updated in this effort. It continues using
positional binding against the old `"env"` namespace. A compatibility shim in the
compiler can optionally emit old-style imports during migration. The C++ migration to
wasmtime `Linker` API happens independently.

**Performance**: Name-based binding has negligible overhead vs positional. wazero and
wasmtime both use hash maps internally for import resolution. The only per-call cost is
`context.WithValue` for node key injection (~10-20ns), which is dwarfed by WASM FFI
overhead.

### 8.2 Open Questions

**Code generation vs pure generics**: The series module has ~300 host functions. Pure
generics reduce this significantly but still require individual `rt.Export()` calls per
type per operation. If the verbosity is unacceptable, a focused code generator for the
series module is fine (Q5 answer: generator complexity is acceptable if authoring
experience is clean).

**`len()` return type**: Currently `i32` for strings, `i64` for series. The QA
recommends standardizing to `i64`. This is a minor breaking change that should be
addressed in Phase 2.

**Graph node polymorphism**: When `math.sqrt{}` appears in a flow statement, the module
provides a dedicated Go graph node factory (not a WASM wrapper). The graph node uses
`state.Node` ports and dispatches internally by data type. This is the same pattern as
existing operator nodes.
