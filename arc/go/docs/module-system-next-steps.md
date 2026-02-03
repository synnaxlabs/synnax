# Arc Module System - Implementation Status & Next Steps

## Current Status Summary

| Component | Status | Notes |
|-----------|--------|-------|
| **Grammar** | ✓ | Complete with imports and aliases |
| **Parser** | ✓ | Regenerated with import/alias support |
| **Analyzer** | ✓ | Fully integrated with import analysis |
| **Stdlib Definitions** | ✓ | math and time modules defined |
| **Compiler** | 0% | No module call support |
| **Runtime** | 0% | No stdlib host functions |

## Completed Work

### Phase 1: Analyzer Integration ✓

1. **Fixed imports.go API mismatches:**
   - Changed `AddError`/`AddWarning` to use `Add(diagnostics.Error(...))`
   - Changed `ResolvePrefix` to `Search` method
   - Fixed test file to use `SeverityError`/`SeverityWarning`

2. **Created stdlib module definitions:**
   - Created `arc/go/stdlib/stdlib.go` with math and time modules
   - Defined function signatures matching spec

3. **Integrated into AnalyzeProgram:**
   - Import the imports package in analyzer.go
   - Call `imports.Analyze()` before declaration collection
   - Pass stdlib modules map

4. **Added module member access support:**
   - Updated expression analyzer to handle `math.sqrt(16.0)` style calls
   - Resolves module symbol, then member via module's Resolver
   - Validates function call arguments against member's type

5. **All tests pass:** 850+ tests across 49 suites

### Phase 2: Grammar Enhancement ✓

1. **Added alias syntax to grammar:**
   ```antlr
   importItem : modulePath (AS IDENTIFIER)?
   ```

2. **Added AS token to lexer:**
   ```antlr
   AS : 'as' ;
   ```

3. **Regenerated parser** with alias support

4. **Updated imports.go to handle aliases:**
   - Detects `AS` token and uses alias identifier as qualifier
   - Original module name not accessible when alias provided
   - Duplicate alias detection works correctly

5. **Added comprehensive alias tests:**
   - `import ( math as m )` - use alias to access members
   - Reject using original name when alias is provided
   - Reject duplicate aliases
   - Reject alias conflicting with non-aliased import

## Remaining Work

### Phase 3: Compiler Support (Priority: High)

The compiler needs to:

1. **Detect Module Member Calls** - When compiling a function call, check if the callee is a module member:
   ```go
   // In compiler/expression/expression.go
   // When we see: math.sqrt(x)
   // 1. Resolve "math" -> symbol with Kind == KindModule
   // 2. Resolve "sqrt" via module's Resolver
   // 3. Emit WASM import call instead of local function call
   ```

2. **Emit WASM Imports** - Add imported functions to the WASM module:
   ```go
   // Naming convention: module_function
   // math.sqrt -> import "env" "math_sqrt"
   // time.now  -> import "env" "time_now"
   ```

3. **Track Import Dependencies** - Ensure only used imports are declared in WASM

**Key Files:**
- `compiler/expression/expression.go` - Expression compilation
- `compiler/bindings/imports.go` - WASM import declarations
- `compiler/wasm/module.go` - WASM module structure

### Phase 4: Runtime Implementation (Priority: High)

The C++ runtime needs host functions for all stdlib operations.

**Location:** `arc/cpp/runtime/wasm/bindings.cpp`

**Math Functions (all f64 -> f64):**
```cpp
double math_sqrt(double x) { return std::sqrt(x); }
double math_sin(double x)  { return std::sin(x); }
double math_cos(double x)  { return std::cos(x); }
double math_tan(double x)  { return std::tan(x); }
double math_abs(double x)  { return std::abs(x); }
double math_floor(double x) { return std::floor(x); }
double math_ceil(double x)  { return std::ceil(x); }
double math_pow(double base, double exp) { return std::pow(base, exp); }
double math_min(double a, double b) { return std::min(a, b); }
double math_max(double a, double b) { return std::max(a, b); }
```

**Time Functions:**
```cpp
int64_t time_now() {
    using namespace std::chrono;
    return duration_cast<nanoseconds>(
        system_clock::now().time_since_epoch()
    ).count();
}

int64_t time_elapsed(int64_t since) {
    return time_now() - since;
}
```

These need to be registered with Wasmtime in `create_imports()`.

### Phase 5: End-to-End Testing (Priority: Medium)

1. **Create integration tests:**
   ```arc
   import ( math )

   func main() f64 {
       return math.sqrt(16.0)  // Should return 4.0
   }
   ```

2. **Test unused import warnings**
3. **Test unknown module errors**
4. **Test unknown member errors**

## Spec Compliance Checklist

| Spec Feature | Grammar | Analyzer | Compiler | Runtime |
|--------------|---------|----------|----------|---------|
| `import ( math )` | ✓ | ✓ | ✗ | ✗ |
| `import ( math as m )` | ✓ | ✓ | ✗ | ✗ |
| `import ( math.trig )` | ✓ | ✓ | ✗ | ✗ |
| Module member access | ✓ | ✓ | ✗ | ✗ |
| Duplicate import error | ✓ | ✓ | N/A | N/A |
| Unknown module error | ✓ | ✓ | N/A | N/A |
| Unused import warning | ✓ | ✓ | N/A | N/A |
| Ambiguous qualifier error | ✓ | ✓ | N/A | N/A |
| Alias conflict error | ✓ | ✓ | N/A | N/A |
| Import position error | ✓ | ✓ | N/A | N/A |
| Unknown member error | ✓ | ✓ | N/A | N/A |
| `math.sqrt(x)` | ✓ | ✓ | ✗ | ✗ |
| `time.now()` | ✓ | ✓ | ✗ | ✗ |
| `len(x)` builtin | N/A | ✓ | ✓ | ✓ |

## Running Tests

```bash
# Import tests only
cd arc/go
ginkgo -v ./analyzer/imports/...

# Full test suite
ginkgo -r

# Specific tests
ginkgo -r --focus "import"
```

## Architecture Notes

- Modules are `Symbol` objects with `Kind == symbol.KindModule`
- Module symbols have a `Resolver` field containing their members
- Member access resolves through: `scope.Resolve("math")` → `symbol.Resolver.Resolve("sqrt")`
- The type system validates argument types for stdlib calls
- Aliases work by registering the module under the alias name instead of original name
- When an alias is used, the original module name is not accessible

## Files Modified/Created

### Phase 1
- `arc/go/symbol/symbol.go` - Fixed duplicate DefaultValue field
- `arc/go/analyzer/imports/imports.go` - Fixed API mismatches
- `arc/go/analyzer/imports/imports_test.go` - Fixed type mismatches
- `arc/go/stdlib/stdlib.go` - **NEW** - Module definitions
- `arc/go/analyzer/analyzer.go` - Integrated imports.Analyze
- `arc/go/text/analyze.go` - Pass stdlib modules
- `arc/go/analyzer/expression/expression.go` - Module member access
- Multiple test files - Updated AnalyzeProgram calls

### Phase 2
- `arc/go/parser/ArcLexer.g4` - Added AS token
- `arc/go/parser/ArcParser.g4` - Added alias syntax
- `arc/go/parser/*.go` - Regenerated
- `arc/go/analyzer/imports/imports.go` - Alias handling
- `arc/go/analyzer/imports/imports_test.go` - Alias tests

## Future Considerations (Lower Priority)

- Selective imports: `import ( sqrt from math )` - spec says not supported
- User-defined modules: `module filters { ... }` - spec says not supported
- Hierarchical modules: `import ( math.trig )`
- LSP autocomplete for module members
- Additional stdlib modules (io, string, etc.)
