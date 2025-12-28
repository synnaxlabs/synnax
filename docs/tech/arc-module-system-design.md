# Arc Module System Design

**Status**: Approved
**Date**: 2024-12-28

---

## Overview

This document defines the module system for Arc's standard library. User-defined modules
are deferred to Phase 2.

---

## Syntax

```arc
// Optional comments

import (
    math
    time
)

func main() {
    x := math.sqrt(16.0)
    t := time.now()
}
```

### Rules

1. One `import` block per file
2. Block must precede all declarations
3. One module per line
4. Unused imports are compile errors
5. Unknown modules are compile errors
6. Circular imports are forbidden

### Hierarchical Modules

Dots separate hierarchy levels. The last segment becomes the qualifier:

```arc
import (
    math
    math.fft
)

x := math.sqrt(4)        // qualifier: math
y := fft.transform(data) // qualifier: fft
```

---

## Resolution

1. Parser extracts module names from import block
2. Analyzer validates against stdlib registry
3. Each module added to scope as a named entry
4. Member access resolves within that module's symbols

```arc
import (
    math
    mathutils
)

x := math.sqrt(16.0)     // scope["math"].symbols["sqrt"]
y := mathutils.sqrt(9.0) // scope["mathutils"].symbols["sqrt"]
```

No collision—each module is its own namespace.

### Errors

| Condition | Message |
|-----------|---------|
| Unknown module | `unknown module "foo"` |
| Duplicate import | `duplicate import "math"` |
| Unused import | `imported "math" but never used` |
| Circular import | `circular import detected: a → b → a` |

---

## Standard Library

### Phase 1

| Module | Functions |
|--------|-----------|
| `math` | `sqrt`, `pow`, `abs`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `floor`, `ceil`, `round`, `min`, `max`, `log`, `log10`, `exp`, `PI`, `E` |
| `time` | `now`, `interval`, `wait`, `elapsed` |

### Future

| Module | Purpose |
|--------|---------|
| `math.fft` | FFT utilities |
| `math.stats` | Statistics |
| `physics.fluids` | Fluid properties |
| `control.pid` | PID controllers |

---

## Implementation

### Go Structure

```
arc/go/runtime/
├── math/
│   └── math.go
├── time/
│   └── time.go
└── registry.go
```

### Registry

```go
var StdlibModules = map[string]symbol.Resolver{
    "math": math.SymbolResolver,
    "time": time.SymbolResolver,
}
```

### Module Definition

```go
var SymbolResolver = symbol.MapResolver{
    "sqrt": symbol.Symbol{
        Name: "sqrt",
        Kind: symbol.KindFunction,
        Type: types.Function(types.FunctionProperties{
            Inputs:  types.Params{{Name: "x", Type: types.F64()}},
            Outputs: types.Params{{Name: "result", Type: types.F64()}},
        }),
    },
}
```

### Grammar

```antlr
// Lexer
IMPORT: 'import';

// Parser
program: importBlock? topLevelItem* EOF;
importBlock: IMPORT LPAREN importItem+ RPAREN;
importItem: modulePath;
modulePath: IDENTIFIER (DOT IDENTIFIER)*;
```

### WASM Integration

Stdlib functions are implemented as WASM host functions. Naming convention:
`<module>_<function>_<type>`

```
math_sqrt_f64
math_pow_f64
time_now
```

#### Runtime Binding

Both runtimes must provide matching host function implementations:

| Runtime | File | Mechanism |
|---------|------|-----------|
| Go (wazero) | `arc/go/runtime/wasm/bindings/` | Name-based linking via `HostModuleBuilder.Export()` |
| C++ (wasmtime) | `arc/cpp/runtime/wasm/bindings.cpp` | Positional vector (requires rework) |

#### C++ Rework

The C++ runtime currently uses positional import matching—functions must be registered
in exact order matching the compiler. This is fragile as stdlib grows.

Migrate to wasmtime's `Linker` API for name-based linking:

```cpp
// Current (positional):
std::vector<wasmtime::Extern> imports;
imports.push_back(sqrt_func);  // Must be index N

// Target (name-based):
wasmtime::Linker linker(engine);
linker.define("env", "math_sqrt_f64", sqrt_func);  // Order irrelevant
```

Name-based linking resolves imports by `(module, name)` pairs at instantiation time.
Runtime call performance is identical—linking is a one-time cost.

---

## Deferred Decisions

| Topic | Phase |
|-------|-------|
| User-defined modules | 2 |
| Visibility (`pub`) | 2 |
| Import aliasing (`as`) | 2 |
| Module versioning | 3 |
