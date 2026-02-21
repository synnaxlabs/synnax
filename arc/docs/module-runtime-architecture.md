# Arc Module Runtime Architecture

This document captures a deep investigation into how Arc's standard library functions
are currently defined, resolved, compiled, bound, and executed — and lays out the
architectural plan for the module system's runtime infrastructure.

## Table of Contents

1. [The Two Execution Layers](#the-two-execution-layers)
2. [Current Host Function Binding](#current-host-function-binding)
3. [Current Symbol Resolution](#current-symbol-resolution)
4. [The Brittleness Problem](#the-brittleness-problem)
5. [The C++ Runtime Constraint](#the-c-runtime-constraint)
6. [Compiler Primitives vs STL Functions](#compiler-primitives-vs-stl-functions)
7. [Architectural Plan](#architectural-plan)
8. [WASM Import Contract](#wasm-import-contract)
9. [Implementation Phases](#implementation-phases)

---

## The Two Execution Layers

Arc programs execute across two fundamentally different layers.

### The Reactive Layer

A dataflow graph of **nodes** connected by **edges**, executed by a scheduler in
topological strata. Each node transforms input series into output series. Some nodes are
implemented entirely in Go (or C++) without touching WASM:

| Node            | Package              | Purpose                           |
| --------------- | -------------------- | --------------------------------- |
| `set_authority` | `runtime/authority/` | Sets channel control authority    |
| `stable_for`    | `runtime/stable/`    | Debounces a signal for a duration |
| `select`        | `runtime/selector/`  | Routes data by boolean condition  |
| `constant`      | `runtime/constant/`  | Emits a constant value once       |

These nodes implement the `node.Node` interface and are created by `node.Factory`
implementations. They operate directly on `state.Node` inputs/outputs — Go-native, no
WASM.

Other nodes in the graph are **WASM-compiled functions** — user-defined functions and
stages whose bodies are compiled to WebAssembly bytecode. The `wasm.Factory` creates
these by looking up the exported WASM function by name and wrapping it in a `nodeImpl`
that handles input/output plumbing.

### The Compiled/WASM Layer

Inside a WASM-compiled function body, the code executes as WebAssembly instructions.
When the code needs to interact with the outside world — reading a channel, persisting a
stateful variable, manipulating a series, calling `math.pow` — it calls **host
functions** provided by the Go or C++ runtime.

These host functions are the bridge between the sandboxed WASM execution and the Go/C++
runtime. They are registered as WASM imports and called via the `call` instruction.

### Key Insight: Dual Usage

A function like `math.sqrt` needs to work in **both** layers:

- As an **inline call** inside a WASM function body: `x := math.sqrt(16.0)`
- As a **reactive node** in the dataflow graph: `chan -> math.sqrt{} -> output{}`

User-defined functions already work this way — they compile to WASM and can be
instantiated as graph nodes. The STL module system must support the same duality.

---

## Current Host Function Binding

### Compiler Side

`compiler/bindings/imports.go` defines `SetupImports()`, which registers **every** host
function with the WASM module upfront:

```
SetupImports(m *wasm.Module) *ImportIndex
  ├── setupChannelOps()     × 11 types = 22 functions
  ├── setupSeriesOps()      × 10 types = ~200 functions
  ├── setupSeriesUnaryOps()              7 functions
  ├── setupStateOps()       × 11 types = 22 functions
  └── setupGenericOps()                 ~20 functions
                                  Total: ~270+ imports
```

Each registration returns a `uint32` index stored in the `ImportIndex` struct — a
massive struct with ~40 fields (many are `map[string]uint32` keyed by type name).

The compiler emits `call <index>` instructions referencing these positional indices.

### Runtime Side (Go)

`runtime/wasm/bindings/runtime_generated.go` (4,890 lines, code-generated) defines a
`Runtime` struct with ~600 methods — one per host function per type variant. Each method
implements the actual operation (reading channels, manipulating series, etc.).

`runtime/wasm/bindings/bindings.go` (440 lines) wires the Runtime methods to the
compiler's `Bindings` struct via `BindRuntime()` — ~440 lines of
`b.ChannelReadF64 = runtime.ChannelReadF64`.

`compiler/bindings/static_bindings_generated.go` (4,647 lines, code-generated) defines
the `Bindings` struct (function pointer fields) and a `Bind()` method that registers
everything with wazero under the single module name `"env"`.

### Runtime Side (C++)

`arc/cpp/runtime/wasm/bindings.cpp` implements the same host functions in C++ and
registers them with wasmtime. The `create_imports()` function builds a
`vector<wasmtime::Extern>` in the **exact same positional order** as the Go side.

### The Registration Chain

```
Go (Compiler)                    Go (Runtime)                C++ (Runtime)
─────────────                    ────────────                ─────────────
SetupImports() registers         BindRuntime() wires Go      create_imports() builds
imports positionally in           method implementations      positional vector in
the WASM module.                  to Bindings struct.         identical order.
        │                                │                          │
        ▼                                ▼                          ▼
ImportIndex stores              Bindings.Bind() registers    vector<Extern> passed
uint32 indices keyed            all functions under          to wasmtime Instance
by operation + type.            wazero module "env".         constructor.
        │                                │                          │
        ▼                                ▼                          ▼
Compiler emits                  wazero resolves call N       wasmtime resolves
call <index>.                   to the Nth import.           call N to Nth extern.
```

All three must be in lockstep. The WASM binary says "call function at index 47" and all
runtimes must agree on what index 47 means.

---

## Current Symbol Resolution

### The Symbol Struct

```go
type Symbol struct {
    Name string         // Identifier name
    Type types.Type     // Arc type (function signature, primitive, etc.)
    Kind Kind           // KindFunction, KindVariable, KindChannel, etc.
    ID   int            // Variable slot ID (for locals/state)
}
```

### Resolution Chain

Symbols are resolved through a scope tree with a global resolver fallback:

```
1. Current scope children (local variables, parameters)
2. GlobalResolver (CompoundResolver of all STL symbol resolvers)
3. Parent scope (recursively up the lexical chain)
```

The global resolver is a `CompoundResolver` chaining flat `MapResolver` instances:

```go
symbol.CompoundResolver{
    builtin.SymbolResolver,     // now, len
    authority.SymbolResolver,   // set_authority
    stable.SymbolResolver,      // stable_for
    selector.SymbolResolver,    // select
    constant.SymbolResolver,    // constant
    op.SymbolResolver,          // ge, gt, le, lt, eq, ne, add, sub, ...
}
```

Everything is flat — no namespacing, no modules. `now`, `len`, `set_authority`,
`stable_for`, `select`, and all operators coexist in one global pool.

### Hardcoded Builtins

`now()` and `len()` get special treatment. The compiler's `compileBuiltinCall()`
function hardcodes them in a switch statement, bypassing the normal function call path:

```go
switch funcName {
case "len":
    return compileBuiltinLen(ctx, funcCall) // dispatches to SeriesLen or StringLen
case "now":
    return compileBuiltinNow(ctx)           // emits call to Now import
}
```

This means `len` has **compiler-resolved polymorphism** — it picks `SeriesLen` or
`StringLen` based on the argument type at compile time.

---

## The Brittleness Problem

### Index Sensitivity

Adding a single new host function anywhere in `SetupImports()` shifts every subsequent
index. Both Go and C++ runtimes must be updated in lockstep, and any previously compiled
WASM binary becomes incompatible.

### Boilerplate Explosion

The combinatorial expansion of operations × types produces:

| Category                                 | Operations | Types | Functions |
| ---------------------------------------- | ---------- | ----- | --------- |
| Channel read/write                       | 2          | 11    | 22        |
| State load/store                         | 2          | 11    | 22        |
| State series load/store                  | 2          | 10    | 20        |
| Series creation/access                   | 3          | 10    | 30        |
| Series-scalar arithmetic                 | 5          | 10    | 50        |
| Series-scalar reverse arithmetic         | 5          | 10    | 50        |
| Series-series arithmetic                 | 5          | 10    | 50        |
| Series-series comparison                 | 6          | 10    | 60        |
| Series-scalar comparison                 | 6          | 10    | 60        |
| Series negate                            | 1          | 6     | 6         |
| Math pow                                 | 1          | 10    | 10        |
| Generic (len, slice, string, now, panic) | —          | —     | ~10       |
| **Total**                                |            |       | **~390**  |

Each function appears in **three places**: the import registration (Go compiler), the
implementation (Go runtime), and the C++ implementation. That's ~10,400 lines of
generated or manual binding code.

### No Organization

All ~390 functions are registered under a single WASM module name `"env"` with flat
names like `channel_read_f64`, `series_element_add_u32`, `math_pow_i64`. There's no
logical grouping, no way to load only what's needed, and no alignment with the planned
Arc module system.

---

## The C++ Runtime Constraint

The compiled WASM module is serialized as **protobuf** and transmitted from the Go
server to the C++ driver via Freighter:

```
Go Server (compiler + runtime)
    │
    │  Parse → Analyze → Compile → WASM bytes
    │
    ▼
PBModule (protobuf)
    ├── ir (dataflow graph, functions, nodes, edges, strata)
    ├── wasm (compiled WebAssembly bytecode)
    └── output_memory_bases (memory layout for multi-output functions)
    │
    │  Freighter (gRPC/HTTP/WebSocket)
    │
    ▼
C++ Driver (runtime only)
    │
    │  Deserialize → Load WASM → Bind host functions → Execute
    │
    ▼
wasmtime Instance
```

The C++ runtime uses **wasmtime** (not wazero). It deserializes the protobuf, loads the
WASM bytes, and binds host functions. Currently it builds a positional
`vector<wasmtime::Extern>` that must match the Go compiler's import order exactly.

**The Go side is the compiler.** It defines signatures, symbol names, import names —
everything. **The C++ side is a runtime only** (analogous to V8 for JavaScript). It
implements the host function contract defined by the WASM binary's import section.

With name-based WASM imports, the contract becomes self-describing: the WASM binary
declares `import("math", "pow_f64")` and whichever runtime (Go or C++) executes it just
needs to provide a function named `pow_f64` in a host module named `math`. Order becomes
irrelevant. Portability becomes automatic.

---

## Compiler Primitives vs STL Functions

There are two categories of host functions with fundamentally different compiler
treatment:

### Compiler Primitives (Invisible to Users)

These are emitted implicitly by the compiler — users never write function calls for
them:

- **Channel reads**: When you write `sensor` (a channel alias), the compiler emits
  `i32.const(channel_id) + call channel_read_f64`
- **Channel writes**: Assignment to a channel variable emits `call channel_write_f64`
- **State loads**: Reading a `state` variable emits
  `i32.const(var_id) + <init_value> + call state_load_f64`
- **State stores**: Assigning to a `state` variable emits `call state_store_f64`
- **Series operations**: Binary operations on series values emit the appropriate
  `series_element_add_f64`, `series_compare_gt_scalar_f64`, etc.
- **String operations**: String literals emit `string_from_literal`, concatenation emits
  `string_concat`

These have **type-polymorphic dispatch** at compile time — the compiler picks the right
typed variant based on the operand types.

### STL Functions (User-Callable)

These are explicitly called by users (or will be, once the module syntax is added):

- `now()` → `time.now()`: Returns current timestamp
- `len(x)`: Returns length of series or string (compiler-resolved polymorphism)
- `math.pow(x, y)`: Power operation (currently used by `^` operator)
- Future: `math.sqrt`, `math.sin`, `math.cos`, `math.abs`, `math.clamp`, etc.

### Unification Principle

Both categories should use the **same registration infrastructure** (declarative
structs, WASM module namespaces, name-based binding). But the compiler retains
**specialized emission codepaths** for primitives, since they're triggered by language
constructs (variable access, assignment, operators) rather than explicit function calls.

Primitives register with `ArcSymbol: nil` — invisible to the analyzer and users, but
present in the WASM import contract.

---

## Architectural Plan

### New `stl/` Directory

A new `arc/go/stl/` directory becomes the single source of truth for all STL function
definitions:

```
arc/go/stl/
    stl.go          Registry, NewRegistry(), top-level composition
    fn.go           HostFunc, ReactiveNodeDef, Module types
    channel/        channel_read_*, channel_write_* (compiler primitives)
    state/          state_load_*, state_store_* (compiler primitives)
    series/         all series ops (compiler primitives)
    math/           math.pow_* (+ future sqrt, sin, cos, abs, clamp)
    time/           time.now
    string/         string ops (compiler primitives)
    control/        set_authority, stable_for, select, constant (reactive nodes)
    op/             operator symbols (reactive graph analyzer)
    error/          panic
```

### Core Definition Types

```go
// HostFunc defines a single WASM host function.
// Single source of truth for analyzer, compiler, and runtime.
type HostFunc struct {
    WASMModule string            // WASM import module name ("channel", "math")
    WASMName   string            // Function name within module ("read_f64", "pow_f64")
    ArcSymbol  *symbol.Symbol    // Analyzer-visible symbol (nil for primitives)
    Params     []wasm.ValueType  // WASM parameter types
    Results    []wasm.ValueType  // WASM result types
    GoImpl     any               // Go function for wazero binding
}

// ReactiveNodeDef defines a Go-native reactive graph node.
type ReactiveNodeDef struct {
    Symbol  symbol.Symbol                   // Analyzer-visible symbol
    Factory func(deps NodeDeps) node.Factory // Creates node instances
}

// Module groups related functions and nodes.
type Module struct {
    Name          string
    HostFuncs     []HostFunc
    ReactiveNodes []ReactiveNodeDef
}
```

A single `HostFunc` struct serves all four consumers:

1. **Analyzer** reads `ArcSymbol` for type checking
2. **Compiler** reads `WASMModule` + `WASMName` + `Params` + `Results` for import
   registration
3. **Go runtime** reads `GoImpl` for wazero host function binding
4. **C++ runtime** reads the WASM binary's import section (which contains `WASMModule` +
   `WASMName`)

### Registry

```go
type Registry struct {
    modules   []Module
    byWASMKey map[string]*HostFunc  // "module/name" -> HostFunc
    resolver  symbol.Resolver       // combined symbol resolver for analyzer
}
```

The registry provides:

- `SymbolResolver()` — for the analyzer's global resolver chain
- `Lookup(wasmModule, wasmName)` — for the compiler to verify imports exist
- `Modules()` — for the runtime to iterate and build wazero host modules
- `NodeFactories(deps)` — for the scheduler to create reactive node instances

### Compiler: Name-Based Import Tracker

Replace `ImportIndex` (struct with ~40 uint32 fields/maps) with a lazy `Tracker`:

```go
type Tracker struct {
    module  *wasm.Module
    indices map[string]uint32  // "module/name" -> import index
}

// Ensure lazily registers an import and returns its index.
func (t *Tracker) Ensure(wasmModule, wasmName string, ft wasm.FunctionType) uint32
```

The Tracker retains **specialized methods** for compiler primitives that encapsulate the
naming convention and type derivation:

```go
func (t *Tracker) ChannelRead(typ types.Type) (uint32, error)
func (t *Tracker) SeriesArithmetic(op string, typ types.Type) (uint32, error)
func (t *Tracker) MathPow(typ types.Type) (uint32, error)
func (t *Tracker) Now() uint32
```

Each method internally calls `Ensure("channel", "read_f64", ...)` with the deterministic
WASM module + function name. This is the **only-imported optimization**: a program that
never uses series operations never registers series imports in the WASM binary.

### Runtime: Per-Module Host Binding

Replace the single `"env"` host module with per-STL-module host modules:

```go
for _, mod := range cfg.Registry.Modules() {
    builder := wasmRuntime.NewHostModuleBuilder(mod.Name)
    for _, fn := range mod.HostFuncs {
        builder.NewFunctionBuilder().WithFunc(fn.GoImpl).Export(fn.WASMName)
    }
    builder.Instantiate(ctx)
}
```

wazero resolves `import("math", "pow_f64")` to the host module named `"math"` with
export `"pow_f64"`. **Order does not matter.** The C++ side uses wasmtime's `Linker` API
for the same name-based resolution.

### Go Generics for Type Variants

Replace ~10,000 lines of generated code with ~500 lines of generic templates:

```go
type Numeric interface {
    uint8 | uint16 | uint32 | uint64 |
    int8 | int16 | int32 | int64 | float32 | float64
}

func channelRead[T Numeric](s *state.State) func(context.Context, uint32) T {
    return func(_ context.Context, channelID uint32) T {
        series, ok := s.ReadChannelValue(channelID)
        if !ok || series.Len() == 0 { return T(0) }
        return telem.ValueAt[T](series, -1)
    }
}

func elementAdd[T Numeric](s *state.State) func(context.Context, uint32, T) uint32 {
    return func(_ context.Context, handle uint32, scalar T) uint32 {
        series, ok := s.SeriesGet(handle)
        if !ok { return 0 }
        result := telem.Series{DataType: series.DataType}
        op.AddScalar[T](series, scalar, &result)
        return s.SeriesStore(result)
    }
}
```

Each generic template is instantiated per type via a type table:

```go
allFuncs = append(allFuncs, registerForType(typeSpec[uint8]{"u8", wasm.I32}, s)...)
allFuncs = append(allFuncs, registerForType[uint16](typeSpec[uint16]{"u16", wasm.I32}, s)...)
// ... all 10 numeric types
```

No code generator. No generated files. Compile-time type safety. No runtime type
switches.

### What Gets Deleted

| File                                             | Lines       | Replacement                          |
| ------------------------------------------------ | ----------- | ------------------------------------ |
| `compiler/bindings/imports.go`                   | ~470        | `compiler/imports/imports.go` (~200) |
| `compiler/bindings/helpers.go`                   | ~100        | Absorbed into Tracker methods        |
| `compiler/bindings/static_bindings_generated.go` | ~4,647      | Eliminated entirely                  |
| `runtime/wasm/bindings/runtime_generated.go`     | ~4,890      | `stl/` generics (~500)               |
| `runtime/wasm/bindings/bindings.go`              | ~440        | Registry-based binding (~50)         |
| `runtime/wasm/bindings/generate/main.go`         | ~300        | Eliminated entirely                  |
| `runtime/builtin/builtin.go`                     | ~47         | Moved to `stl/time/`                 |
| `runtime/authority/authority.go`                 | ~99         | Moved to `stl/control/`              |
| `runtime/stable/stable.go`                       | ~127        | Moved to `stl/control/`              |
| `runtime/selector/select.go`                     | ~110        | Moved to `stl/control/`              |
| `runtime/constant/constant.go`                   | ~72         | Moved to `stl/control/`              |
| **Total removed**                                | **~11,302** | **~750 new**                         |

---

## WASM Import Contract

The WASM binary's import section is the contract between compiler and runtimes.

### Namespace Mapping

| WASM Module | Function Pattern              | Params             | Results  |
| ----------- | ----------------------------- | ------------------ | -------- |
| `channel`   | `read_{type}`                 | `(i32)`            | `({wt})` |
| `channel`   | `write_{type}`                | `(i32, {wt})`      | `()`     |
| `state`     | `load_{type}`                 | `(i32, {wt})`      | `({wt})` |
| `state`     | `store_{type}`                | `(i32, {wt})`      | `()`     |
| `state`     | `load_series_{type}`          | `(i32, i32)`       | `(i32)`  |
| `state`     | `store_series_{type}`         | `(i32, i32)`       | `()`     |
| `series`    | `create_empty_{type}`         | `(i32)`            | `(i32)`  |
| `series`    | `set_element_{type}`          | `(i32, i32, {wt})` | `(i32)`  |
| `series`    | `index_{type}`                | `(i32, i32)`       | `({wt})` |
| `series`    | `element_{op}_{type}`         | `(i32, {wt})`      | `(i32)`  |
| `series`    | `element_r{op}_{type}`        | `({wt}, i32)`      | `(i32)`  |
| `series`    | `series_{op}_{type}`          | `(i32, i32)`       | `(i32)`  |
| `series`    | `compare_{cmp}_{type}`        | `(i32, i32)`       | `(i32)`  |
| `series`    | `compare_{cmp}_scalar_{type}` | `(i32, {wt})`      | `(i32)`  |
| `series`    | `negate_{type}`               | `(i32)`            | `(i32)`  |
| `series`    | `not_u8`                      | `(i32)`            | `(i32)`  |
| `series`    | `len`                         | `(i32)`            | `(i64)`  |
| `series`    | `slice`                       | `(i32, i32, i32)`  | `(i32)`  |
| `math`      | `pow_{type}`                  | `({wt}, {wt})`     | `({wt})` |
| `time`      | `now`                         | `()`               | `(i64)`  |
| `string`    | `from_literal`                | `(i32, i32)`       | `(i32)`  |
| `string`    | `concat`                      | `(i32, i32)`       | `(i32)`  |
| `string`    | `equal`                       | `(i32, i32)`       | `(i32)`  |
| `string`    | `len`                         | `(i32)`            | `(i32)`  |
| `error`     | `panic`                       | `(i32, i32)`       | `()`     |

Where `{type}` is `u8|u16|u32|u64|i8|i16|i32|i64|f32|f64` (plus `str` for
channel/state). Where `{wt}` is the WASM type: `i32` for 8/16/32-bit integers, `i64` for
64-bit integers, `f32`, `f64`. Where `{op}` is `add|sub|mul|div|mod`. Where `{cmp}` is
`gt|lt|ge|le|eq|ne`.

---

## Implementation Phases

### Phase 1: Foundation

Create `arc/go/stl/` with core types and `arc/go/compiler/imports/` with the Tracker.
Old and new paths coexist. No behavior change.

### Phase 2: Compiler Migration

Port `compiler/context/` from `ImportIndex` to `Tracker`. Update all compiler callsites.
WASM binaries start using module namespaces. Temporarily support both old and new import
names in the runtime for test continuity.

### Phase 3: Go Runtime Migration

Implement generic host functions in `stl/`. Replace `runtime/wasm/module.go` with
per-module host builders. Replace `runtime/wasm/bindings/` with registry-based binding.
Delete generated files and the code generator.

### Phase 4: STL Symbol Migration

Move symbol definitions and reactive node factories from `runtime/` packages to `stl/`.
Delete old packages.

### Phase 5: Cleanup

Delete `compiler/bindings/` and `runtime/wasm/bindings/` entirely. Produce C++ contract
specification. Update documentation.
