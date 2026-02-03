# Arc Module System - Implementation Plan

## Current Status

| Component | Status | Notes |
|-----------|--------|-------|
| **Grammar** | ✓ | Complete with imports and aliases |
| **Parser** | ✓ | Regenerated with import/alias support |
| **Analyzer** | ✓ | Fully integrated with import analysis |
| **Stdlib Definitions** | ✓ | math and time modules defined |
| **Compiler** | 🔄 | Needs architectural refactor for modules |
| **Runtime (C++)** | 🔄 | Needs name-based import resolution |

## Architectural Vision

### Problem: Fragile Index-Based Import Resolution

The current system uses **position-based** import resolution:

```
Compiler (Go)                         Runtime (C++)
─────────────────                     ─────────────────
imports[0] = channel_read_u8          imports[0] = channel_read_u8
imports[1] = channel_write_u8         imports[1] = channel_write_u8
...                                   ...
imports[42] = series_add_f64          imports[42] = series_add_f64
...                                   ...
imports[489] = panic                  imports[489] = panic

Compiler emits: call 42               Runtime: imports[42] must match!
```

**Problems:**
- Adding/reordering functions breaks everything silently
- No version checking - wrong function gets called
- ~490 functions must be in exact same order in Go and C++
- "call 42" is meaningless for debugging
- Tight coupling between compiler and runtime

### Solution: Name-Based Import Resolution

WASM already supports named imports. We should use them properly:

```
WASM Module contains:
  import "arc.core" "channel_read_u8" (func $0)
  import "math" "sqrt" (func $1)
  import "time" "now" (func $2)

Runtime resolves by NAME:
  registry["arc.core:channel_read_u8"] → Bindings::channel_read_u8
  registry["math:sqrt"] → std::sqrt
  registry["time:now"] → TimeStamp::now()

Build imports vector by reading WASM import names, not hardcoded order.
```

**Benefits:**
- Order doesn't matter - registry is a map
- Adding new functions is safe
- Missing imports detected with clear errors
- Self-documenting: "call math:sqrt" vs "call 42"
- Compiler and runtime can evolve independently

### Module Organization

Reorganize imports from flat "env" namespace into logical modules:

| Module | Purpose | Example Functions |
|--------|---------|-------------------|
| `arc.core` | Channel I/O, panic | `channel_read_u8`, `channel_write_f64`, `panic` |
| `arc.series` | Series operations | `create_empty_f64`, `index_f64`, `add_f64` |
| `arc.state` | Stateful variables | `load_u64`, `store_u64`, `load_series_f64` |
| `arc.string` | String operations | `from_literal`, `concat`, `equal`, `len` |
| `math` | Math stdlib | `sqrt`, `sin`, `cos`, `pow`, `abs`, `floor`, `ceil` |
| `time` | Time stdlib | `now`, `elapsed` |

## Implementation Plan

### Phase 1: Compiler Refactor

#### 1.1 Create Import Registry System

**New file: `compiler/bindings/registry.go`**

```go
// ImportRegistry manages named imports organized by module.
type ImportRegistry struct {
    imports map[string]map[string]ImportDef  // module -> name -> def
    indices map[string]uint32                 // "module:name" -> index
    nextIdx uint32
}

type ImportDef struct {
    Module   string
    Name     string
    FuncType wasm.FunctionType
    Index    uint32
}

func NewImportRegistry() *ImportRegistry {
    return &ImportRegistry{
        imports: make(map[string]map[string]ImportDef),
        indices: make(map[string]uint32),
    }
}

// Register adds an import and returns its index.
func (r *ImportRegistry) Register(module, name string, ft wasm.FunctionType) uint32 {
    key := module + ":" + name
    if idx, exists := r.indices[key]; exists {
        return idx  // Already registered
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

// WriteToModule writes all imports to the WASM module.
func (r *ImportRegistry) WriteToModule(m *wasm.Module) {
    // Sort by index to ensure deterministic output
    defs := make([]ImportDef, r.nextIdx)
    for _, modImports := range r.imports {
        for _, def := range modImports {
            defs[def.Index] = def
        }
    }
    for _, def := range defs {
        m.AddImport(def.Module, def.Name, def.FuncType)
    }
}
```

#### 1.2 Refactor Core Imports

**Update: `compiler/bindings/core.go`** (renamed from imports.go)

```go
// RegisterCoreImports registers arc.core module functions.
func RegisterCoreImports(r *ImportRegistry) {
    // Channel operations - organized by type
    for _, typ := range types.Numerics {
        registerChannelOps(r, typ)
    }
    registerChannelOps(r, types.String())

    // Panic
    r.Register("arc.core", "panic", wasm.FunctionType{
        Params: []wasm.ValueType{wasm.I32, wasm.I32},  // ptr, len
    })
}

func registerChannelOps(r *ImportRegistry, t types.Type) {
    wasmType := wasm.ConvertType(t)

    r.Register("arc.core", fmt.Sprintf("channel_read_%s", t), wasm.FunctionType{
        Params:  []wasm.ValueType{wasm.I32},
        Results: []wasm.ValueType{wasmType},
    })

    r.Register("arc.core", fmt.Sprintf("channel_write_%s", t), wasm.FunctionType{
        Params:  []wasm.ValueType{wasm.I32, wasmType},
    })
}
```

**New file: `compiler/bindings/series.go`**

```go
// RegisterSeriesImports registers arc.series module functions.
func RegisterSeriesImports(r *ImportRegistry) {
    for _, typ := range types.Numerics {
        registerSeriesOps(r, typ)
    }
    registerSeriesUnaryOps(r)
}

func registerSeriesOps(r *ImportRegistry, t types.Type) {
    wasmType := wasm.ConvertType(t)
    typeName := t.String()

    // Create/access
    r.Register("arc.series", fmt.Sprintf("create_empty_%s", typeName), ...)
    r.Register("arc.series", fmt.Sprintf("set_element_%s", typeName), ...)
    r.Register("arc.series", fmt.Sprintf("index_%s", typeName), ...)

    // Arithmetic
    for _, op := range []string{"add", "sub", "mul", "div", "mod"} {
        r.Register("arc.series", fmt.Sprintf("element_%s_%s", op, typeName), ...)
        r.Register("arc.series", fmt.Sprintf("element_r%s_%s", op, typeName), ...)
        r.Register("arc.series", fmt.Sprintf("series_%s_%s", op, typeName), ...)
    }

    // Comparison
    for _, op := range []string{"gt", "lt", "ge", "le", "eq", "ne"} {
        r.Register("arc.series", fmt.Sprintf("compare_%s_%s", op, typeName), ...)
        r.Register("arc.series", fmt.Sprintf("compare_%s_scalar_%s", op, typeName), ...)
    }
}
```

**New file: `compiler/bindings/state.go`**

```go
// RegisterStateImports registers arc.state module functions.
func RegisterStateImports(r *ImportRegistry) {
    for _, typ := range types.Numerics {
        registerStateOps(r, typ)
    }
    registerStateOps(r, types.String())
}

func registerStateOps(r *ImportRegistry, t types.Type) {
    wasmType := wasm.ConvertType(t)
    typeName := t.String()

    r.Register("arc.state", fmt.Sprintf("load_%s", typeName), wasm.FunctionType{
        Params:  []wasm.ValueType{wasm.I32, wasm.I32, wasmType},
        Results: []wasm.ValueType{wasmType},
    })

    r.Register("arc.state", fmt.Sprintf("store_%s", typeName), wasm.FunctionType{
        Params: []wasm.ValueType{wasm.I32, wasm.I32, wasmType},
    })

    // Series state
    r.Register("arc.state", fmt.Sprintf("load_series_%s", typeName), ...)
    r.Register("arc.state", fmt.Sprintf("store_series_%s", typeName), ...)
}
```

**New file: `compiler/bindings/string.go`**

```go
// RegisterStringImports registers arc.string module functions.
func RegisterStringImports(r *ImportRegistry) {
    r.Register("arc.string", "from_literal", wasm.FunctionType{
        Params:  []wasm.ValueType{wasm.I32, wasm.I32},
        Results: []wasm.ValueType{wasm.I32},
    })
    r.Register("arc.string", "concat", wasm.FunctionType{
        Params:  []wasm.ValueType{wasm.I32, wasm.I32},
        Results: []wasm.ValueType{wasm.I32},
    })
    r.Register("arc.string", "equal", wasm.FunctionType{
        Params:  []wasm.ValueType{wasm.I32, wasm.I32},
        Results: []wasm.ValueType{wasm.I32},
    })
    r.Register("arc.string", "len", wasm.FunctionType{
        Params:  []wasm.ValueType{wasm.I32},
        Results: []wasm.ValueType{wasm.I32},
    })
}
```

#### 1.3 Add Stdlib Module Imports

**New file: `compiler/bindings/math.go`**

```go
// RegisterMathImports registers math stdlib module functions.
func RegisterMathImports(r *ImportRegistry) {
    // f64 -> f64 functions
    for _, name := range []string{"sqrt", "sin", "cos", "tan", "asin", "acos", "atan",
                                   "abs", "floor", "ceil", "round", "exp", "log", "log10"} {
        r.Register("math", name, wasm.FunctionType{
            Params:  []wasm.ValueType{wasm.F64},
            Results: []wasm.ValueType{wasm.F64},
        })
    }

    // (f64, f64) -> f64 functions
    for _, name := range []string{"pow", "min", "max", "atan2"} {
        r.Register("math", name, wasm.FunctionType{
            Params:  []wasm.ValueType{wasm.F64, wasm.F64},
            Results: []wasm.ValueType{wasm.F64},
        })
    }

    // Constants (as zero-arg functions)
    r.Register("math", "pi", wasm.FunctionType{
        Results: []wasm.ValueType{wasm.F64},
    })
    r.Register("math", "e", wasm.FunctionType{
        Results: []wasm.ValueType{wasm.F64},
    })
}
```

**New file: `compiler/bindings/time.go`**

```go
// RegisterTimeImports registers time stdlib module functions.
func RegisterTimeImports(r *ImportRegistry) {
    r.Register("time", "now", wasm.FunctionType{
        Results: []wasm.ValueType{wasm.I64},  // nanoseconds
    })

    r.Register("time", "elapsed", wasm.FunctionType{
        Params:  []wasm.ValueType{wasm.I64},  // since timestamp
        Results: []wasm.ValueType{wasm.I64},  // duration nanoseconds
    })
}
```

#### 1.4 Update Compiler Context

**Update: `compiler/context/context.go`**

```go
type Context struct {
    // ... existing fields ...

    // Replace ImportIndex with ImportRegistry
    Registry *bindings.ImportRegistry

    // FunctionIndices now includes "module:name" keys for stdlib
    FunctionIndices map[string]uint32
}
```

#### 1.5 Update Expression Compiler

**Update: `compiler/expression/compiler.go`**

```go
func compileFunctionCallExpr(ctx Context, funcName string, ...) error {
    // Check if it's a module-qualified call (e.g., "math.sqrt")
    if strings.Contains(funcName, ".") {
        parts := strings.SplitN(funcName, ".", 2)
        module, name := parts[0], parts[1]

        idx, ok := ctx.Registry.Lookup(module, name)
        if !ok {
            return errors.Newf("unknown function %s.%s", module, name)
        }

        // Compile arguments
        for _, arg := range args {
            if err := compileExpr(ctx, arg); err != nil {
                return err
            }
        }

        ctx.Writer.WriteCall(idx)
        return nil
    }

    // Existing logic for user-defined functions
    funcIdx, ok := ctx.FunctionIndices[funcName]
    if !ok {
        return errors.Newf("function %s not found", funcName)
    }
    // ...
}
```

#### 1.6 Update Main Compiler

**Update: `compiler/compiler.go`**

```go
func Compile(program *ir.Program) ([]byte, error) {
    module := wasm.NewModule()

    // Create import registry and register all modules
    registry := bindings.NewImportRegistry()
    bindings.RegisterCoreImports(registry)
    bindings.RegisterSeriesImports(registry)
    bindings.RegisterStateImports(registry)
    bindings.RegisterStringImports(registry)
    bindings.RegisterMathImports(registry)
    bindings.RegisterTimeImports(registry)

    // Write imports to WASM module
    registry.WriteToModule(module)

    // Create compilation context
    ctx := &context.Context{
        Registry:        registry,
        FunctionIndices: make(map[string]uint32),
        // ...
    }

    // Assign indices to user functions (after all imports)
    importCount := registry.Count()
    for i, f := range program.Functions {
        ctx.FunctionIndices[f.Key] = importCount + uint32(i)
    }

    // ... rest of compilation
}
```

### Phase 2: C++ Runtime Refactor

#### 2.1 Create Function Registry

**New file: `arc/cpp/runtime/wasm/registry.h`**

```cpp
#pragma once

#include <functional>
#include <string>
#include <unordered_map>
#include <wasmtime.hh>

namespace arc::wasm {

class FunctionRegistry {
public:
    using Factory = std::function<wasmtime::Func(wasmtime::Store&)>;

    // Register a host function
    void register_func(const std::string& module,
                       const std::string& name,
                       Factory factory) {
        std::string key = module + ":" + name;
        factories_[key] = std::move(factory);
    }

    // Build imports vector by reading WASM module's import section
    std::vector<wasmtime::Extern> resolve_imports(
        wasmtime::Store& store,
        const wasmtime::Module& module
    ) const {
        std::vector<wasmtime::Extern> imports;

        for (const auto& import : module.imports()) {
            std::string key = std::string(import.module()) + ":" +
                              std::string(import.name());

            auto it = factories_.find(key);
            if (it == factories_.end()) {
                throw std::runtime_error("Unknown import: " + key);
            }

            imports.push_back(it->second(store));
        }

        return imports;
    }

private:
    std::unordered_map<std::string, Factory> factories_;
};

} // namespace arc::wasm
```

#### 2.2 Register Core Functions

**New file: `arc/cpp/runtime/wasm/core_bindings.cpp`**

```cpp
#include "registry.h"
#include "bindings.h"

namespace arc::wasm {

void register_core_bindings(FunctionRegistry& registry,
                            std::shared_ptr<Bindings> bindings) {
    // Channel operations
    #define REGISTER_CHANNEL_OPS(suffix, type) \
        registry.register_func("arc.core", "channel_read_" #suffix, \
            [bindings](wasmtime::Store& store) { \
                return wasmtime::Func::wrap(store, \
                    wrap(bindings.get(), &Bindings::channel_read_##suffix)); \
            }); \
        registry.register_func("arc.core", "channel_write_" #suffix, \
            [bindings](wasmtime::Store& store) { \
                return wasmtime::Func::wrap(store, \
                    wrap(bindings.get(), &Bindings::channel_write_##suffix)); \
            });

    REGISTER_CHANNEL_OPS(u8, uint8_t)
    REGISTER_CHANNEL_OPS(u16, uint16_t)
    REGISTER_CHANNEL_OPS(u32, uint32_t)
    REGISTER_CHANNEL_OPS(u64, uint64_t)
    REGISTER_CHANNEL_OPS(i8, int8_t)
    REGISTER_CHANNEL_OPS(i16, int16_t)
    REGISTER_CHANNEL_OPS(i32, int32_t)
    REGISTER_CHANNEL_OPS(i64, int64_t)
    REGISTER_CHANNEL_OPS(f32, float)
    REGISTER_CHANNEL_OPS(f64, double)
    REGISTER_CHANNEL_OPS(str, uint32_t)

    #undef REGISTER_CHANNEL_OPS

    // Panic
    registry.register_func("arc.core", "panic",
        [bindings](wasmtime::Store& store) {
            return wasmtime::Func::wrap(store,
                wrap(bindings.get(), &Bindings::panic));
        });
}

} // namespace arc::wasm
```

#### 2.3 Register Series Functions

**New file: `arc/cpp/runtime/wasm/series_bindings.cpp`**

```cpp
void register_series_bindings(FunctionRegistry& registry,
                              std::shared_ptr<Bindings> bindings) {
    #define REGISTER_SERIES_OPS(suffix, type) \
        registry.register_func("arc.series", "create_empty_" #suffix, ...); \
        registry.register_func("arc.series", "set_element_" #suffix, ...); \
        registry.register_func("arc.series", "index_" #suffix, ...); \
        /* Arithmetic */ \
        registry.register_func("arc.series", "element_add_" #suffix, ...); \
        registry.register_func("arc.series", "element_sub_" #suffix, ...); \
        /* ... etc */

    REGISTER_SERIES_OPS(u8, uint8_t)
    REGISTER_SERIES_OPS(u16, uint16_t)
    // ... all types

    #undef REGISTER_SERIES_OPS
}
```

#### 2.4 Register Math Stdlib

**New file: `arc/cpp/runtime/wasm/math_bindings.cpp`**

```cpp
#include <cmath>

void register_math_bindings(FunctionRegistry& registry) {
    // f64 -> f64 functions
    registry.register_func("math", "sqrt", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::sqrt(x); });
    });
    registry.register_func("math", "sin", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::sin(x); });
    });
    registry.register_func("math", "cos", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::cos(x); });
    });
    registry.register_func("math", "tan", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::tan(x); });
    });
    registry.register_func("math", "asin", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::asin(x); });
    });
    registry.register_func("math", "acos", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::acos(x); });
    });
    registry.register_func("math", "atan", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::atan(x); });
    });
    registry.register_func("math", "abs", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::abs(x); });
    });
    registry.register_func("math", "floor", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::floor(x); });
    });
    registry.register_func("math", "ceil", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::ceil(x); });
    });
    registry.register_func("math", "round", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::round(x); });
    });
    registry.register_func("math", "exp", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::exp(x); });
    });
    registry.register_func("math", "log", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::log(x); });
    });
    registry.register_func("math", "log10", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double x) { return std::log10(x); });
    });

    // (f64, f64) -> f64 functions
    registry.register_func("math", "pow", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double b, double e) { return std::pow(b, e); });
    });
    registry.register_func("math", "min", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double a, double b) { return std::min(a, b); });
    });
    registry.register_func("math", "max", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double a, double b) { return std::max(a, b); });
    });
    registry.register_func("math", "atan2", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](double y, double x) { return std::atan2(y, x); });
    });

    // Constants
    registry.register_func("math", "pi", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, []() { return M_PI; });
    });
    registry.register_func("math", "e", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, []() { return M_E; });
    });
}
```

#### 2.5 Register Time Stdlib

**New file: `arc/cpp/runtime/wasm/time_bindings.cpp`**

```cpp
#include "telem/telem.h"

void register_time_bindings(FunctionRegistry& registry) {
    registry.register_func("time", "now", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, []() -> int64_t {
            return telem::TimeStamp::now().nanoseconds();
        });
    });

    registry.register_func("time", "elapsed", [](wasmtime::Store& store) {
        return wasmtime::Func::wrap(store, [](int64_t since) -> int64_t {
            return telem::TimeStamp::now().nanoseconds() - since;
        });
    });
}
```

#### 2.6 Update Module Instantiation

**Update: `arc/cpp/runtime/wasm/module.cpp`**

```cpp
std::pair<std::shared_ptr<Module>, xerrors::Error>
Module::open(const ModuleConfig& cfg) {
    // 1. Create function registry and register all modules
    FunctionRegistry registry;
    register_core_bindings(registry, cfg.bindings);
    register_series_bindings(registry, cfg.bindings);
    register_state_bindings(registry, cfg.bindings);
    register_string_bindings(registry, cfg.bindings);
    register_math_bindings(registry);
    register_time_bindings(registry);

    // 2. Compile WASM module
    auto mod = wasmtime::Module::compile(engine, cfg.wasm_bytes);
    if (!mod) {
        return {nullptr, xerrors::Error("Failed to compile WASM")};
    }

    // 3. Resolve imports BY NAME (not position!)
    std::vector<wasmtime::Extern> imports;
    try {
        imports = registry.resolve_imports(store, *mod);
    } catch (const std::runtime_error& e) {
        return {nullptr, xerrors::Error(e.what())};
    }

    // 4. Create instance
    auto instance = wasmtime::Instance::create(store, *mod, imports);
    // ...
}
```

### Phase 3: Update Stdlib Definitions

**Update: `arc/go/stdlib/stdlib.go`**

```go
package stdlib

import (
    "github.com/synnaxlabs/arc/symbol"
    "github.com/synnaxlabs/arc/types"
)

// Modules maps module names to their symbol resolvers.
// These must match the imports registered in compiler/bindings/*.go
var Modules = map[string]symbol.Resolver{
    "math": mathModule,
    "time": timeModule,
}

var mathModule = symbol.MapResolver{
    // f64 -> f64
    "sqrt":  mathFunc1("sqrt"),
    "sin":   mathFunc1("sin"),
    "cos":   mathFunc1("cos"),
    "tan":   mathFunc1("tan"),
    "asin":  mathFunc1("asin"),
    "acos":  mathFunc1("acos"),
    "atan":  mathFunc1("atan"),
    "abs":   mathFunc1("abs"),
    "floor": mathFunc1("floor"),
    "ceil":  mathFunc1("ceil"),
    "round": mathFunc1("round"),
    "exp":   mathFunc1("exp"),
    "log":   mathFunc1("log"),
    "log10": mathFunc1("log10"),

    // (f64, f64) -> f64
    "pow":   mathFunc2("pow"),
    "min":   mathFunc2("min"),
    "max":   mathFunc2("max"),
    "atan2": mathFunc2("atan2"),

    // Constants
    "pi": symbol.Symbol{
        Name: "pi",
        Kind: symbol.KindFunction,
        Type: types.Function(types.FunctionProperties{
            Outputs: types.Params{{Name: "value", Type: types.F64()}},
        }),
    },
    "e": symbol.Symbol{
        Name: "e",
        Kind: symbol.KindFunction,
        Type: types.Function(types.FunctionProperties{
            Outputs: types.Params{{Name: "value", Type: types.F64()}},
        }),
    },
}

var timeModule = symbol.MapResolver{
    "now": symbol.Symbol{
        Name: "now",
        Kind: symbol.KindFunction,
        Type: types.Function(types.FunctionProperties{
            Outputs: types.Params{{Name: "timestamp", Type: types.TimeStamp()}},
        }),
    },
    "elapsed": symbol.Symbol{
        Name: "elapsed",
        Kind: symbol.KindFunction,
        Type: types.Function(types.FunctionProperties{
            Inputs:  types.Params{{Name: "since", Type: types.TimeStamp()}},
            Outputs: types.Params{{Name: "duration", Type: types.TimeSpan()}},
        }),
    },
}

func mathFunc1(name string) symbol.Symbol {
    return symbol.Symbol{
        Name: name,
        Kind: symbol.KindFunction,
        Type: types.Function(types.FunctionProperties{
            Inputs:  types.Params{{Name: "x", Type: types.F64()}},
            Outputs: types.Params{{Name: "result", Type: types.F64()}},
        }),
    }
}

func mathFunc2(name string) symbol.Symbol {
    return symbol.Symbol{
        Name: name,
        Kind: symbol.KindFunction,
        Type: types.Function(types.FunctionProperties{
            Inputs: types.Params{
                {Name: "a", Type: types.F64()},
                {Name: "b", Type: types.F64()},
            },
            Outputs: types.Params{{Name: "result", Type: types.F64()}},
        }),
    }
}
```

### Phase 4: Integration Testing

#### 4.1 Compiler Tests

**New file: `compiler/bindings/registry_test.go`**

```go
var _ = Describe("ImportRegistry", func() {
    It("Should register and lookup imports", func() {
        r := NewImportRegistry()
        idx := r.Register("math", "sqrt", wasm.FunctionType{...})

        lookupIdx, ok := r.Lookup("math", "sqrt")
        Expect(ok).To(BeTrue())
        Expect(lookupIdx).To(Equal(idx))
    })

    It("Should return same index for duplicate registration", func() {
        r := NewImportRegistry()
        idx1 := r.Register("math", "sqrt", wasm.FunctionType{...})
        idx2 := r.Register("math", "sqrt", wasm.FunctionType{...})
        Expect(idx1).To(Equal(idx2))
    })

    It("Should write imports to module in index order", func() {
        r := NewImportRegistry()
        r.Register("math", "sqrt", ...)
        r.Register("time", "now", ...)
        r.Register("math", "sin", ...)

        m := wasm.NewModule()
        r.WriteToModule(m)

        names := m.ImportNames()
        Expect(names[0]).To(Equal("sqrt"))
        Expect(names[1]).To(Equal("now"))
        Expect(names[2]).To(Equal("sin"))
    })
})
```

#### 4.2 End-to-End Module Tests

**New file: `text/module_test.go`**

```go
var _ = Describe("Module System E2E", func() {
    Describe("Math Module", func() {
        It("Should compile and link math.sqrt", func() {
            src := `
import ( math )

func calculate(x f64) f64 {
    return math.sqrt(x)
}
`
            // Parse and analyze
            t, diag := text.Parse(text.Text{Raw: src})
            Expect(diag).To(BeNil())

            program, diag := text.Analyze(context.Background(), t, nil)
            Expect(diag.Ok()).To(BeTrue())

            // Compile to WASM
            wasm, err := compiler.Compile(program)
            Expect(err).To(BeNil())

            // Verify import section contains "math:sqrt"
            imports := parseImports(wasm)
            Expect(imports).To(ContainElement(Import{
                Module: "math",
                Name:   "sqrt",
            }))
        })

        It("Should compile complex math expressions", func() {
            src := `
import ( math )

func hypotenuse(a f64, b f64) f64 {
    return math.sqrt(math.pow(a, 2.0) + math.pow(b, 2.0))
}
`
            // ... test compilation
        })
    })

    Describe("Time Module", func() {
        It("Should compile time.now and time.elapsed", func() {
            src := `
import ( time )

func measure_duration() i64 {
    start := time.now()
    // ... do work ...
    return time.elapsed(start)
}
`
            // ... test compilation
        })
    })

    Describe("Module Aliases", func() {
        It("Should compile with aliased module", func() {
            src := `
import ( math as m )

func calc() f64 {
    return m.sqrt(16.0)
}
`
            // ... test compilation
        })
    })
})
```

## File Summary

### New Files (Go Compiler)
- `compiler/bindings/registry.go` - Import registry system
- `compiler/bindings/core.go` - arc.core module imports
- `compiler/bindings/series.go` - arc.series module imports
- `compiler/bindings/state.go` - arc.state module imports
- `compiler/bindings/string.go` - arc.string module imports
- `compiler/bindings/math.go` - math stdlib imports
- `compiler/bindings/time.go` - time stdlib imports
- `compiler/bindings/registry_test.go` - Registry tests
- `text/module_test.go` - E2E module tests

### Modified Files (Go Compiler)
- `compiler/compiler.go` - Use registry instead of SetupImports
- `compiler/context/context.go` - Add Registry field
- `compiler/expression/compiler.go` - Handle module-qualified calls
- `stdlib/stdlib.go` - Expand math/time function definitions

### Deleted Files (Go Compiler)
- `compiler/bindings/imports.go` - Replaced by modular files
- `compiler/bindings/gen/main.go` - No longer needed
- `compiler/bindings/static_bindings_generated.go` - No longer needed

### New Files (C++ Runtime)
- `runtime/wasm/registry.h` - Function registry
- `runtime/wasm/core_bindings.cpp` - arc.core bindings
- `runtime/wasm/series_bindings.cpp` - arc.series bindings
- `runtime/wasm/state_bindings.cpp` - arc.state bindings
- `runtime/wasm/string_bindings.cpp` - arc.string bindings
- `runtime/wasm/math_bindings.cpp` - math stdlib bindings
- `runtime/wasm/time_bindings.cpp` - time stdlib bindings

### Modified Files (C++ Runtime)
- `runtime/wasm/module.cpp` - Use registry for import resolution
- `runtime/wasm/bindings.h` - May need minor updates

### Deleted Files (C++ Runtime)
- Large portions of `runtime/wasm/bindings.cpp` - `create_imports()` replaced

## Implementation Order

1. **Go: Create ImportRegistry** - Core abstraction
2. **Go: Refactor existing imports** - Split into modular files
3. **Go: Add math/time imports** - Stdlib compiler support
4. **Go: Update expression compiler** - Handle module.function calls
5. **Go: Tests** - Verify compilation works
6. **C++: Create FunctionRegistry** - Core abstraction
7. **C++: Refactor existing bindings** - Split into modular files
8. **C++: Add math/time bindings** - Stdlib runtime support
9. **C++: Update module.cpp** - Name-based resolution
10. **Integration tests** - Full compile + run tests

## Success Criteria

1. All existing tests pass (no regression)
2. `math.sqrt(16.0)` compiles to `call` with correct index
3. WASM import section shows `import "math" "sqrt"` (not `import "env" "math_sqrt"`)
4. C++ runtime resolves imports by name, not position
5. Adding a new function to math module doesn't break existing WASM
6. Clear error message when import is missing: "Unknown import: math:sqrt"
