# Arc Module System - Next Steps

## Context

Branch `sy-3495-arc-modules` implements the front-end of the Arc module system:
- Grammar for `import ( math time )` syntax
- Member access expressions like `math.sqrt(x)`
- Import validation (duplicate/unknown module errors)
- Unused import warnings
- Modules as first-class `Symbol` objects with a `Resolver` field

**All analyzer tests pass.** The module system works through type-checking.

## What's Missing

The compiler and runtime don't handle module function calls yet. When you write:

```arc
import ( math )

func main() f64 {
    return math.sqrt(16.0)
}
```

This parses and type-checks correctly, but compilation will fail or produce incorrect WASM because:
1. The compiler doesn't recognize `math.sqrt` as a stdlib call
2. The runtime doesn't provide host functions for math operations

## Task 1: Compiler Support for Module Calls

Location: `arc/go/compiler/`

When compiling a function call expression, check if it's a module member call (e.g., `math.sqrt`). If so, emit a call to an imported host function instead of a local function.

### Key Files
- `compiler/expression/expression.go` - Expression compilation
- `compiler/wasm/wasm.go` - WASM generation

### Approach
1. In expression compilation, detect when a `postfixExpression` has:
   - A `primaryExpression` that's an identifier resolving to `KindModule`
   - A `memberAccess` suffix
   - A `functionCallSuffix`

2. Look up the member in the module's resolver to get its type signature

3. Emit a WASM `call` instruction to an imported function with a naming convention like:
   - `math.sqrt` → import `"env" "math_sqrt"`
   - `time.now` → import `"env" "time_now"`

4. Ensure the WASM module's import section declares these functions

### Reference
See how the compiler handles regular function calls in `compiler/expression/expression.go` and adapt for external calls.

## Task 2: Runtime Host Functions

Location: `arc/cpp/runtime/` (C++ WASM runtime)

The C++ runtime that executes Arc WASM modules needs to provide host functions that implement stdlib operations.

### Key Files
- `arc/cpp/runtime/wasm/bindings.cpp` - Host function bindings
- `arc/cpp/runtime/wasm/module.cpp` - WASM module loading

### Functions to Implement

**Math module:**
```cpp
// All take f64, return f64
double math_sqrt(double x);
double math_pow(double base, double exp);
double math_abs(double x);
double math_sin(double x);
double math_cos(double x);
double math_tan(double x);
double math_floor(double x);
double math_ceil(double x);
double math_min(double a, double b);
double math_max(double a, double b);
```

**Time module:**
```cpp
// Returns nanoseconds since epoch
int64_t time_now();
// Returns nanoseconds elapsed
int64_t time_elapsed(int64_t since);
```

### Approach
1. Add the host functions to the Wasmtime linker in `bindings.cpp`
2. Use standard C++ `<cmath>` for math functions
3. Use `std::chrono` for time functions
4. Follow the existing pattern for how other host functions are registered

## Task 3: Integration Test

Create an end-to-end test that:
1. Compiles Arc code using `math.sqrt`
2. Executes the WASM with the runtime
3. Verifies the result is correct

Location: `arc/go/text/text_test.go` or create a new integration test file.

## Module Definitions Reference

The stdlib modules are defined in `arc/go/text/text.go` in the `defaultModules()` function:

```go
"math": symbol.MapResolver{
    "sqrt": symbol.Symbol{
        Name: "sqrt",
        Kind: symbol.KindFunction,
        Type: types.Function(types.FunctionProperties{
            Inputs:  types.Params{{Name: "x", Type: types.F64()}},
            Outputs: types.Params{{Name: "output", Type: types.F64()}},
        }),
    },
    // ... more functions
},
"time": symbol.MapResolver{
    "now": symbol.Symbol{...},
    "elapsed": symbol.Symbol{...},
},
```

## Architecture Notes

- Modules are `Symbol` objects with `Kind == symbol.KindModule`
- Module symbols have a `Resolver` field containing their members
- Member access resolves through: scope.Resolve("math") → symbol.Resolver.Resolve("sqrt")
- The type system already validates argument types for stdlib calls

## Running Tests

```bash
cd arc/go
ginkgo -r  # Run all tests
ginkgo -r --focus "math.sqrt"  # Run specific tests
```

## Future Work (Lower Priority)

After compiler/runtime support:
- Import aliases: `import ( m = math )`
- Hierarchical modules: `import ( math.fft )`
- LSP autocomplete for module members
- Additional stdlib modules (io, string, etc.)
