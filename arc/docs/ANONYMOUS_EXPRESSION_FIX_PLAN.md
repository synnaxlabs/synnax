# Anonymous Expression Compilation Fix Plan

## Executive Summary

Anonymous expressions in Arc flow statements (e.g., `sensor > 100 -> alarm{}`) are currently analyzed but **not compiled to WASM**. This document provides a comprehensive plan to fix this critical gap in the compilation pipeline.

## Problem Statement

### Current Behavior

When an expression appears in a flow statement:
```arc
ox_pt_1 > 100 -> alarm{}
```

The **analyzer** creates a synthetic function (named `__expr_0`, `__expr_1`, etc.) with:
- Type: `types.Function` with no inputs, single output of the expression result type
- Body: The expression AST node
- Scope: Registered in the root symbol table

However, these synthetic functions are **never converted to `ir.Function`** entries, so:
- They don't appear in the IR's `Functions` list
- The compiler never sees them
- No WASM code is generated
- Runtime execution fails

### Test Evidence

`compiler/compiler_test.go:199-207` expects:
```go
output := MustSucceed(compileWithHostImports(`ox_pt_1 > 10 -> print{message = "dog"}`, resolver))
mod := MustSucceed(r.Instantiate(ctx, output.WASM))
readAndDouble := mod.ExportedFunction("__expr_0")  // ❌ FAILS - function not exported
```

## Root Cause Analysis

### 1. Analyzer Creates Wrong Type

**File**: `analyzer/flow/expression.go:22-51`

```go
func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) bool {
    exprType := atypes.InferFromExpression(ctx)
    if exprType.Kind == types.KindChan {
        exprType = *exprType.ValueType
    }
    t := types.EmptyFunction()  // ❌ Creates types.Function, not ir.Function
    t.Outputs.Put(ir.DefaultOutputParam, exprType)
    stageScope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
        Name: "",
        Kind: symbol.KindFunction,
        Type: t,      // ❌ Wrong type - should be ir.Function
        AST:  ctx.AST,
    })
    // ...
    return expression.Analyze(ctx.WithScope(blockScope))
}
```

**Issue**: The `Type` field contains `types.Function` (type information) instead of `ir.Function` (IR representation with body).

### 2. Text Analyzer Misses Synthetic Functions

**File**: `text/text.go:56-63`

```go
// func 2: Iterate through the root scope children to assemble functions and stages.
for _, c := range i.Symbols.Children {
    switch c.Kind {
    case symbol.KindFunction:
        i.Stages = append(i.Stages, c.Type.(ir.Stage))   // ❌ Type assertion fails
    case symbol.KindFunction:  // ❌ Duplicate case!
        i.Functions = append(i.Functions, c.Type.(ir.Function))  // ❌ Panic on assertion
    }
}
```

**Issues**:
1. Duplicate `case symbol.KindFunction` (second is unreachable)
2. Type assertions expect `ir.Stage`/`ir.Function` but get `types.Type`
3. Synthetic expression functions are never added to `i.Functions`

### 3. Missing Body Conversion

Even if synthetic functions were added to IR, they lack proper body representation:
- Analyzer stores expression AST in `Symbol.AST`
- IR needs `ir.Function.Body` with `types.Body{AST: expr}`
- Compiler expects `Body.AST` to be either `IBlockContext` or `IExpressionContext`

### 4. Channel Dependency Tracking

**File**: `text/text.go:243-261`

```go
func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) (ir.Node, ir.Handle, bool) {
    // ...
    n := ir.Node{
        Key:      sym.Name,
        Type:     sym.Name,
        Channels: ir.OverrideChannels(stageType.Channels),  // ❌ stageType.Channels is empty!
    }
    // ...
}
```

**Issue**: Expressions that reference channels (e.g., `sensor > 100`) need their channel dependencies tracked in `Channels.Read`, but this information is never extracted from the expression AST.

## Proposed Solution

### Phase 1: Fix Analyzer to Create Proper IR Functions

**File**: `analyzer/flow/expression.go`

#### Change 1: Create IR Function Instead of Type Function

```go
func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) bool {
    exprType := atypes.InferFromExpression(ctx)
    if exprType.Kind == types.KindChan {
        exprType = *exprType.ValueType
    }

    // ✅ NEW: Create IR function with proper Body
    fn := ir.Function{
        Key:     "",  // Will be set by AutoName
        Body:    types.Body{
            Raw: "", // Can be empty for synthetic functions
            AST: ctx.AST,  // Store the expression AST
        },
        Config:  types.Params{},
        Inputs:  types.Params{},
        Outputs: types.Params{},
    }
    fn.Outputs.Put(ir.DefaultOutputParam, exprType)

    // ✅ NEW: Extract channel dependencies from expression
    channels := extractChannelsFromExpression(ctx)
    fn.Channels = channels

    // Store as Symbol with ir.Function
    stageScope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
        Name: "",
        Kind: symbol.KindFunction,
        Type: fn,  // ✅ Now ir.Function
        AST:  ctx.AST,
    })
    if err != nil {
        ctx.Diagnostics.AddError(err, ctx.AST)
        return false
    }
    stageScope = stageScope.AutoName("__expr_")

    // Create block scope for expression analysis
    blockScope, err := stageScope.Add(ctx, symbol.Symbol{
        Name: "",
        Kind: symbol.KindBlock,
        AST:  ctx.AST,
    })
    if err != nil {
        ctx.Diagnostics.AddError(err, ctx.AST)
        return false
    }
    return expression.Analyze(ctx.WithScope(blockScope))
}
```

#### Change 2: Add Channel Extraction Helper

```go
// extractChannelsFromExpression walks the expression AST and collects channel references
func extractChannelsFromExpression(ctx acontext.Context[parser.IExpressionContext]) ir.Channels {
    channels := ir.NewChannels()
    visitor := &channelVisitor{
        ctx:      ctx,
        channels: &channels,
    }
    visitor.visit(ctx.AST)
    return channels
}

type channelVisitor struct {
    ctx      acontext.Context[parser.IExpressionContext]
    channels *ir.Channels
}

func (v *channelVisitor) visit(node antlr.Tree) {
    if node == nil {
        return
    }

    // Check if this is an identifier
    if idNode, ok := node.(antlr.TerminalNode); ok {
        if idNode.GetSymbol().GetTokenType() == parser.ArcLexerIDENTIFIER {
            name := idNode.GetText()
            // Resolve and check if it's a channel
            if sym, err := v.ctx.Scope.Resolve(v.ctx, name); err == nil {
                if sym.Kind == symbol.KindChannel {
                    v.channels.Read.Add(uint32(sym.ID))
                }
            }
        }
    }

    // Recursively visit children
    if parent, ok := node.(antlr.ParserRuleContext); ok {
        for i := 0; i < parent.GetChildCount(); i++ {
            v.visit(parent.GetChild(i))
        }
    }
}
```

### Phase 2: Fix Text Analyzer to Collect Functions

**File**: `text/text.go:56-63`

```go
// func 2: Iterate through the root scope children to assemble functions and stages.
for _, c := range i.Symbols.Children {
    if c.Kind == symbol.KindFunction {
        // ✅ Check if Type is ir.Function or types.Type
        if fn, ok := c.Type.(ir.Function); ok {
            i.Functions = append(i.Functions, fn)
        } else if typeFn, ok := c.Type.(types.Type); ok && typeFn.Kind == types.KindFunction {
            // ✅ Legacy path: convert types.Function to ir.Function
            // This handles functions declared in the text that haven't been converted yet
            fn := ir.Function{
                Key:     c.Name,
                Body:    types.Body{AST: c.AST},
                Config:  *typeFn.Config,
                Inputs:  *typeFn.Inputs,
                Outputs: *typeFn.Outputs,
            }
            i.Functions = append(i.Functions, fn)
        }
    }
}
```

### Phase 3: Ensure Compiler Handles Expression Bodies

**File**: `compiler/compiler.go:106-116`

The compiler already handles both blocks and expressions:

```go
if blockCtx, ok := body.(parser.IBlockContext); ok {
    if err = statement.CompileBlock(ccontext.Child(ctx, blockCtx)); err != nil {
        return errors.Wrapf(err, "failed to compile function '%s' body", ctx.Scope.Name)
    }
} else if exprCtx, ok := body.(parser.IExpressionContext); ok {
    if err = compileExpression(ccontext.Child(ctx, exprCtx)); err != nil {
        return errors.Wrapf(err, "failed to compile expression '%s'", ctx.Scope.Name)
    }
}
```

**Verification needed**: Ensure `compileExpression` properly returns the result (for implicit `return` behavior).

#### Potential Issue: Missing Return

Expression functions should implicitly return the expression result:

```go
// In compiler/compiler.go:123-126
func compileExpression(ctx ccontext.Context[parser.IExpressionContext]) error {
    _, err := expression.Compile(ctx)
    return err  // ❌ Result is on stack but not explicitly returned!
}
```

**Fix**:
```go
func compileExpression(ctx ccontext.Context[parser.IExpressionContext]) error {
    resultType, err := expression.Compile(ctx)
    if err != nil {
        return err
    }
    // Expression result is already on stack - it becomes the implicit return
    // No explicit return instruction needed for expression bodies
    return nil
}
```

### Phase 4: Fix Node Creation for Expressions

**File**: `text/text.go:243-261`

```go
func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) (ir.Node, ir.Handle, bool) {
    sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
    if err != nil {
        ctx.Diagnostics.AddError(err, ctx.AST)
        return ir.Node{}, ir.Handle{}, false
    }

    // ✅ Get ir.Function from symbol
    fn, ok := sym.Type.(ir.Function)
    if !ok {
        ctx.Diagnostics.AddError(
            errors.New("expression did not produce ir.Function"),
            ctx.AST,
        )
        return ir.Node{}, ir.Handle{}, false
    }

    // ✅ Create node with proper channels
    n := ir.Node{
        Key:          sym.Name,
        Type:         sym.Name,
        ConfigValues: make(map[string]any),
        Channels:     fn.Channels,  // ✅ Channels from function
        Config:       fn.Config,
        Inputs:       fn.Inputs,
        Outputs:      fn.Outputs,
    }
    h := ir.Handle{Node: sym.Name, Param: ir.DefaultOutputParam}
    return n, h, true
}
```

## Implementation Checklist

### Step 1: Analyzer Changes ✅
- [ ] Modify `analyzer/flow/expression.go:analyzeExpression` to create `ir.Function` instead of `types.Function`
- [ ] Add `extractChannelsFromExpression` helper function
- [ ] Add `channelVisitor` struct for AST walking
- [ ] Update tests in `analyzer/flow/expression_test.go` to verify `ir.Function` type

### Step 2: Text Analyzer Changes ✅
- [ ] Fix duplicate `case` in `text/text.go:56-63`
- [ ] Add type assertion handling for both `ir.Function` and `types.Type`
- [ ] Update `analyzeExpression` in `text/text.go:243-261` to use `ir.Function`
- [ ] Update tests to verify functions are collected

### Step 3: Compiler Verification ✅
- [ ] Verify `compiler/compiler.go:compileExpression` properly handles expression return
- [ ] Add test for expression compilation in `compiler/compiler_test.go`
- [ ] Ensure expression result stays on stack as implicit return

### Step 4: Integration Testing ✅
- [ ] Run existing test: `compiler/compiler_test.go:162` (Flow Expression test)
- [ ] Verify `__expr_0` function is exported in WASM
- [ ] Test execution returns correct result (1 for `ox_pt_1 > 10` when `ox_pt_1 = 32`)
- [ ] Add tests for multiple expressions in flow
- [ ] Add tests for complex expressions (arithmetic, logical, nested)

### Step 5: Channel Dependency Testing ✅
- [ ] Test expression with single channel reference
- [ ] Test expression with multiple channel references
- [ ] Test expression with nested channel operations
- [ ] Verify `ir.Channels.Read` contains all referenced channels

## Testing Strategy

### Unit Tests

**File**: `analyzer/flow/expression_test.go`

```go
It("should create ir.Function for expression", func() {
    ast := MustSucceed(parser.Parse(`ox_pt_1 > 100 -> alarm{}`))
    ctx := context.CreateRoot(bCtx, ast, testResolver)
    Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

    // Verify synthetic function exists
    synthFunc := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_0"))
    Expect(synthFunc.Kind).To(Equal(symbol.KindFunction))

    // Verify it's ir.Function type
    fn, ok := synthFunc.Type.(ir.Function)
    Expect(ok).To(BeTrue())
    Expect(fn.Key).To(Equal("__expr_0"))
    Expect(fn.Outputs.Count()).To(Equal(1))

    // Verify channel dependency tracking
    Expect(fn.Channels.Read).To(HaveLen(1))
    Expect(fn.Channels.Read.Contains(uint32(12))).To(BeTrue()) // ox_pt_1 ID
})
```

### Integration Tests

**File**: `compiler/compiler_test.go`

Already exists at line 162, needs to pass:

```go
It("Should correctly compile and execute a flow expression", func() {
    output := MustSucceed(compileWithHostImports(
        `ox_pt_1 > 10 -> print{message = "dog"}`,
        resolver,
    ))

    mod := MustSucceed(r.Instantiate(ctx, output.WASM))
    exprFunc := mod.ExportedFunction("__expr_0")
    Expect(exprFunc).ToNot(BeNil())  // ✅ Should pass after fix

    results := MustSucceed(exprFunc.Call(ctx))
    Expect(results).To(HaveLen(1))
    Expect(results[0]).To(Equal(uint64(1)))  // 32 > 10 = true (1)
})
```

### End-to-End Tests

```go
It("Should compile complex expression flow", func() {
    output := MustSucceed(compileWithHostImports(`
        (ox_pt_1 + ox_pt_2) / 2 > 100 -> alarm{}
        ox_pt_1 * 1.8 + 32 -> display{}
        pressure > 50 && temp < 100 -> warning{}
    `, resolver))

    mod := MustSucceed(r.Instantiate(ctx, output.WASM))

    // Verify all expression functions are exported
    Expect(mod.ExportedFunction("__expr_0")).ToNot(BeNil())
    Expect(mod.ExportedFunction("__expr_1")).ToNot(BeNil())
    Expect(mod.ExportedFunction("__expr_2")).ToNot(BeNil())
})
```

## Edge Cases to Handle

### 1. Expression with No Channel References
```arc
42 > 10 -> alarm{}  // Constant expression
```
- Should compile to function returning constant
- `Channels.Read` should be empty

### 2. Expression with Type Coercion
```arc
f32(temp_sensor) * 1.8 + 32.0 -> display{}
```
- Type inference must work correctly
- Cast operations must compile properly

### 3. Nested Expressions
```arc
((ox_pt_1 + ox_pt_2) / 2) > 100 -> alarm{}
```
- Parentheses should be handled correctly
- Channel extraction must traverse nested structure

### 4. Logical Expression with Multiple Channels
```arc
pressure > 100 && temp > 50 || emergency -> shutdown{}
```
- All three channels must be in `Channels.Read`
- Short-circuit evaluation must work correctly

### 5. Expression in Routing Table
```arc
demux{} -> {
    high: value * 2 -> display{}
}
```
- Currently not supported (expressions as routing targets)
- May need separate handling

## Migration Notes

### Breaking Changes

None - this is a bug fix, not a breaking change.

### Backward Compatibility

- Existing text-based functions still work (types.Function path)
- Graph-based functions unaffected
- Only fixes previously broken anonymous expressions

### Performance Considerations

- Channel extraction adds one extra AST walk per expression
- Negligible impact - expressions are small
- Could optimize with caching if needed

## Alternative Approaches Considered

### Alternative 1: Compile Expressions Inline

**Idea**: Don't create separate functions; inline expression compilation into the calling node.

**Rejected because**:
- Breaks stratification model (expressions need separate execution)
- No way to track expression as reactive node
- Would require major runtime changes

### Alternative 2: Convert Expressions to Blocks at Parse Time

**Idea**: Transform `sensor > 100` into synthetic block `{ return sensor > 100 }` during parsing.

**Rejected because**:
- Loses expression semantics
- Harder to optimize
- More complex AST manipulation

### Alternative 3: Create Expression Node Type

**Idea**: Add `ir.Expression` type separate from `ir.Function`.

**Rejected because**:
- Compiler already handles expression bodies
- Unnecessary duplication
- Would complicate IR structure

## Success Criteria

1. ✅ Test `compiler/compiler_test.go:162` passes
2. ✅ `__expr_N` functions appear in compiled WASM exports
3. ✅ Expression functions execute correctly with channel dependencies
4. ✅ All analyzer and flow tests pass
5. ✅ Integration tests demonstrate end-to-end compilation
6. ✅ Complex expressions (arithmetic, logical, nested) compile correctly
7. ✅ Channel dependency tracking works for all expression types

## Timeline Estimate

- **Phase 1** (Analyzer): 4-6 hours
  - Modify `analyzeExpression`: 1 hour
  - Implement channel extraction: 2-3 hours
  - Unit tests: 1-2 hours

- **Phase 2** (Text Analyzer): 2-3 hours
  - Fix function collection: 1 hour
  - Update `analyzeExpression`: 30 min
  - Tests: 30-90 min

- **Phase 3** (Compiler Verification): 1-2 hours
  - Verify expression compilation: 30 min
  - Test and debug: 30-90 min

- **Phase 4** (Integration Testing): 2-4 hours
  - Run existing tests: 30 min
  - Add new tests: 1-2 hours
  - Debug edge cases: 30-90 min

**Total**: 9-15 hours

## References

### Key Files

1. `analyzer/flow/expression.go` - Creates synthetic functions
2. `text/text.go` - Converts symbols to IR
3. `compiler/compiler.go` - Compiles IR to WASM
4. `compiler/expression/compiler.go` - Expression compilation
5. `ir/ir.go`, `ir/function.go` - IR data structures

### Related Issues

- Anonymous configuration values (line 255-259 in `analyzer/flow/flow.go`)
- TODO at line 467 in `analyzer/flow/flow.go` (input routing type checking)
- Expression compilation TODOs in `compiler/expression/`

### Relevant Tests

- `analyzer/flow/expression_test.go` - Expression analysis tests
- `analyzer/flow/flow_test.go:359` - Expression in flow test
- `compiler/compiler_test.go:162` - Flow expression compilation test

---

**Document Version**: 1.0
**Last Updated**: 2025-10-15
**Author**: Analysis of Arc codebase