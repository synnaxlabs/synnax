# Arc Channels Field Population - Deep Research & Implementation Plan

## Problem Statement

The `Channels` field in `ir.Node` is not being properly populated during analysis and IR generation. This field tracks which Synnax channels each node reads from and writes to, which is critical for:
1. **Reactive dependency tracking** - Knowing which nodes to mark dirty when channel data arrives
2. **Optimization** - Understanding data flow for scheduling and resource allocation
3. **Validation** - Ensuring channel access patterns are correct

## Current State Analysis

### Data Structure (ir/ir.go:30-34)
```go
type Channels struct {
    Read  set.Set[uint32] `json:"read"`   // Channel keys this node reads from
    Write set.Set[uint32] `json:"write"`  // Channel keys this node writes to
}
```

### Where Channels Are Partially Tracked

#### 1. **text/text.go (Text-based flows)** ✅ WORKS PARTIALLY
- `analyzeChannel()` (line 146-167): Tracks channel reads when a channel is used as a flow source
- `extractConfigValues()` (line 189-206): Tracks config parameters that are channels
- **Problem**: Only tracks channels in config and flow sources, NOT channels used in function bodies

#### 2. **graph/graph.go (Graph-based flows)** ❌ INCOMPLETE
- Line 198: Has a commented-out assertion `//Expect(firstNode.Channels.Read).To(HaveLen(1))`
- **Problem**: Never populates Channels field at all in graph analysis

#### 3. **analyzer/analyzer.go** ❌ PLACEHOLDER
- Line 111-114: Has comment indicating channel tracking "will be handled at IR compilation stage"
- **Problem**: Never implemented - just a placeholder OnResolve hook

### Channel Usage Points in Arc Code

Channels can be used in multiple ways:

1. **Config parameters** (✅ tracked in text.go)
   ```arc
   stage controller{
       sensor chan f32
   } () { ... }
   ```

2. **Flow sources** (✅ tracked in text.go)
   ```arc
   temperature_sensor -> stage{}
   ```

3. **Flow targets** (❌ NOT tracked)
   ```arc
   stage{} -> output_channel
   ```

4. **Channel reads in function body** (❌ NOT tracked)
   ```arc
   func process() {
       val := <-my_channel  // blocking read
       current := my_channel  // non-blocking read
   }
   ```

5. **Channel writes in function body** (❌ NOT tracked)
   ```arc
   func process() {
       42 -> output_channel
       output_channel <- 99
   }
   ```

6. **Channel references in expressions** (❌ NOT tracked)
   ```arc
   func process() {
       avg := (sensor1 + sensor2) / 2
   }
   ```

### Why This Matters

From `runtime/state/state.go:57-72`:
```go
func (s *State) Ingest(fr telem.Frame[uint32], markDirty func(nodeKey string)) {
    for rawI, key := range fr.RawKeys() {
        s.channel.reads[key] = fr.RawSeriesAt(rawI)
        // This is where we need Channels.Read!
        for _, nodeKey := range s.cfg.ReactiveDeps[key] {
            markDirty(nodeKey)
        }
    }
}
```

The `ReactiveDeps map[uint32][]string` needs to be built from `Channels.Read` - it maps channel keys to nodes that read from them.

## Root Cause Analysis

The problem stems from **two different compilation paths**:

### Path 1: Text-based (text/text.go)
- Parse Arc source → AST
- Analyze declarations
- Build IR with flow statements converted to nodes/edges
- Channel tracking happens during `analyzeChannel` and `extractConfigValues`
- **Missing**: Tracking channels in function bodies

### Path 2: Graph-based (graph/graph.go)
- Receive pre-built graph (from Console's visual editor)
- Functions already parsed and in symbol table
- Nodes and edges provided directly
- **Missing**: ALL channel tracking

The analyzer was designed to be the common point for both paths, but channel tracking was deferred with the assumption it would happen "at IR compilation stage" (analyzer.go:114) - **this never happened**.

## Implementation Challenges

### Challenge 1: Two-Pass Problem
Channels are resolved during analysis (when symbols are available), but IR is built after analysis. We need channel information in the IR, but channel tracking requires:
- Symbol resolution (to map channel names → channel keys)
- AST traversal (to find all channel operations)
- Type information (to distinguish channels from variables)

### Challenge 2: Function Bodies Are Opaque in Graph Mode
In graph mode, functions are already compiled. The `Function.Body` contains:
```go
type Body struct {
    Raw string
    AST antlr.ParserRuleContext
}
```

If `Raw == ""`, we don't have source to analyze.

### Challenge 3: Channel Keys vs Channel Names
- Channels are referred to by **name** in Arc source
- The `Channels` struct uses **uint32 keys** (from Synnax)
- Resolution happens via `symbol.Resolver.Resolve(ctx, name)` which returns `sym.ID`
- The ID is only available during analysis, not during IR generation

### Challenge 4: Stateful Variables Complicate Tracking
Stateful variables (`$=`) can hold channel references:
```arc
stage foo{} () {
    ch $= sensor_channel
    val := <-ch  // This is a channel read, but 'ch' is a variable
}
```

Current type system can track this (`ch` has type `chan T`), but AST walker needs to follow variable types.

## Proposed Solution

### High-Level Approach

**During Analysis Phase** (when we have symbol resolution):
1. After type checking each function/stage body, perform a **channel tracking pass**
2. Walk the AST to find all channel operations
3. Resolve channel names to channel keys using the symbol table
4. Store channel tracking info in the **symbol table** (not IR yet)
5. Transfer from symbol table to IR during IR construction

### Detailed Implementation Plan

#### Phase 1: Extend Symbol Table with Channel Tracking

**File**: `arc/symbol/symbol.go`

Add channel tracking to the `Scope` structure:
```go
type Scope struct {
    Symbol
    GlobalResolver Resolver
    Parent         *Scope
    Children       []*Scope
    Counter        *int
    OnResolve      func(ctx context.Context, s *Scope) error

    // NEW: Channel tracking for this scope (function/stage)
    ChannelsRead   set.Set[uint32]  // Channel keys read in this scope
    ChannelsWrite  set.Set[uint32]  // Channel keys written in this scope
}
```

#### Phase 2: Create Channel Tracker

**File**: `arc/analyzer/channels/tracker.go` (NEW)

```go
package channels

import (
    "context"
    "github.com/antlr4-go/antlr/v4"
    acontext "github.com/synnaxlabs/arc/analyzer/context"
    "github.com/synnaxlabs/arc/parser"
    "github.com/synnaxlabs/arc/symbol"
    "github.com/synnaxlabs/arc/types"
    "github.com/synnaxlabs/x/set"
)

// TrackChannels walks an AST and populates channel read/write sets
// in the provided scope.
func TrackChannels(
    ctx context.Context,
    scope *symbol.Scope,
    ast antlr.ParserRuleContext,
) error {
    tracker := &channelTracker{
        ctx:   ctx,
        scope: scope,
        reads: make(set.Set[uint32]),
        writes: make(set.Set[uint32]),
    }

    // Walk the AST
    if block, ok := ast.(parser.IBlockContext); ok {
        tracker.walkBlock(block)
    } else if expr, ok := ast.(parser.IExpressionContext); ok {
        tracker.walkExpression(expr)
    }

    // Store results in scope
    scope.ChannelsRead = tracker.reads
    scope.ChannelsWrite = tracker.writes

    return tracker.err
}

type channelTracker struct {
    ctx    context.Context
    scope  *symbol.Scope
    reads  set.Set[uint32]
    writes set.Set[uint32]
    err    error
}

func (t *channelTracker) walkBlock(block parser.IBlockContext) {
    for _, stmt := range block.AllStatement() {
        t.walkStatement(stmt)
    }
}

func (t *channelTracker) walkStatement(stmt parser.IStatementContext) {
    switch {
    case stmt.ChannelOperation() != nil:
        t.walkChannelOperation(stmt.ChannelOperation())
    case stmt.Assignment() != nil:
        t.walkAssignment(stmt.Assignment())
    case stmt.Expression() != nil:
        t.walkExpression(stmt.Expression())
    case stmt.IfStatement() != nil:
        t.walkIfStatement(stmt.IfStatement())
    case stmt.VariableDeclaration() != nil:
        t.walkVariableDeclaration(stmt.VariableDeclaration())
    }
}

func (t *channelTracker) walkChannelOperation(op parser.IChannelOperationContext) {
    if read := op.ChannelRead(); read != nil {
        t.walkChannelRead(read)
    }
    if write := op.ChannelWrite(); write != nil {
        t.walkChannelWrite(write)
    }
}

func (t *channelTracker) walkChannelRead(read parser.IChannelReadContext) {
    // val := <-my_channel  or  val := my_channel
    var channelName string
    if blocking := read.BlockingRead(); blocking != nil {
        ids := blocking.AllIDENTIFIER()
        if len(ids) == 2 {
            channelName = ids[1].GetText()
        }
    } else if nonBlocking := read.NonBlockingRead(); nonBlocking != nil {
        ids := nonBlocking.AllIDENTIFIER()
        if len(ids) == 2 {
            channelName = ids[1].GetText()
        }
    }

    if channelName != "" {
        if key := t.resolveChannelKey(channelName); key != 0 {
            t.reads.Add(key)
        }
    }
}

func (t *channelTracker) walkChannelWrite(write parser.IChannelWriteContext) {
    // value -> my_channel  or  my_channel <- value
    channelName := write.IDENTIFIER().GetText()
    if key := t.resolveChannelKey(channelName); key != 0 {
        t.writes.Add(key)
    }

    // Also check the expression being written for channel reads
    if expr := write.Expression(); expr != nil {
        t.walkExpression(expr)
    }
}

func (t *channelTracker) walkExpression(expr parser.IExpressionContext) {
    // Walk expression tree looking for channel references
    // Channels used in expressions are READS
    // Example: (sensor1 + sensor2) / 2

    identifiers := t.findIdentifiers(expr)
    for _, id := range identifiers {
        if key := t.resolveChannelKey(id); key != 0 {
            t.reads.Add(key)
        }
    }
}

func (t *channelTracker) walkIfStatement(ifStmt parser.IIfStatementContext) {
    // Track condition expression
    if expr := ifStmt.Expression(); expr != nil {
        t.walkExpression(expr)
    }

    // Track then block
    if block := ifStmt.Block(); block != nil {
        t.walkBlock(block)
    }

    // Track else if clauses
    for _, elseIf := range ifStmt.AllElseIfClause() {
        if expr := elseIf.Expression(); expr != nil {
            t.walkExpression(expr)
        }
        if block := elseIf.Block(); block != nil {
            t.walkBlock(block)
        }
    }

    // Track else clause
    if elseClause := ifStmt.ElseClause(); elseClause != nil {
        if block := elseClause.Block(); block != nil {
            t.walkBlock(block)
        }
    }
}

func (t *channelTracker) walkVariableDeclaration(decl parser.IVariableDeclarationContext) {
    // Check initialization expression for channel reads
    var expr parser.IExpressionContext
    if local := decl.LocalVariable(); local != nil {
        expr = local.Expression()
    } else if stateful := decl.StatefulVariable(); stateful != nil {
        expr = stateful.Expression()
    }

    if expr != nil {
        t.walkExpression(expr)
    }
}

func (t *channelTracker) walkAssignment(assign parser.IAssignmentContext) {
    // lhs = rhs
    // Check RHS for channel reads
    if expr := assign.Expression(); expr != nil {
        t.walkExpression(expr)
    }
}

// Helper to resolve channel name to channel key
func (t *channelTracker) resolveChannelKey(name string) uint32 {
    sym, err := t.scope.Resolve(t.ctx, name)
    if err != nil {
        return 0
    }

    // Check if this is a channel
    if sym.Kind != symbol.KindChannel && sym.Type.Kind != types.KindChan {
        return 0
    }

    // sym.ID is the channel key (set during symbol resolution)
    return uint32(sym.ID)
}

// Helper to find all identifiers in an expression tree
func (t *channelTracker) findIdentifiers(expr parser.IExpressionContext) []string {
    // Traverse expression AST and collect all IDENTIFIER tokens
    identifiers := []string{}

    // Recursively walk the expression tree
    t.collectIdentifiersFromExpression(expr, &identifiers)

    return identifiers
}

func (t *channelTracker) collectIdentifiersFromExpression(expr parser.IExpressionContext, ids *[]string) {
    if expr == nil {
        return
    }

    // Walk through expression levels
    for _, logicalOr := range expr.LogicalOrExpression().AllLogicalAndExpression() {
        for _, equality := range logicalOr.AllEqualityExpression() {
            for _, relational := range equality.AllRelationalExpression() {
                for _, additive := range relational.AllAdditiveExpression() {
                    for _, mult := range additive.AllMultiplicativeExpression() {
                        for _, power := range mult.AllPowerExpression() {
                            t.collectIdentifiersFromUnary(power.UnaryExpression(), ids)
                        }
                    }
                }
            }
        }
    }
}

func (t *channelTracker) collectIdentifiersFromUnary(unary parser.IUnaryExpressionContext, ids *[]string) {
    if unary == nil {
        return
    }

    if postfix := unary.PostfixExpression(); postfix != nil {
        t.collectIdentifiersFromPostfix(postfix, ids)
    }

    // Handle blocking read: <-channel
    if blockRead := unary.BlockingReadExpr(); blockRead != nil {
        if id := blockRead.IDENTIFIER(); id != nil {
            *ids = append(*ids, id.GetText())
        }
    }
}

func (t *channelTracker) collectIdentifiersFromPostfix(postfix parser.IPostfixExpressionContext, ids *[]string) {
    if postfix == nil {
        return
    }

    if primary := postfix.PrimaryExpression(); primary != nil {
        if id := primary.IDENTIFIER(); id != nil {
            *ids = append(*ids, id.GetText())
        }
    }
}
```

#### Phase 3: Integrate into Analyzer

**File**: `arc/analyzer/analyzer.go`

Replace the placeholder OnResolve with actual channel tracking:

```go
func analyzeFunctionDeclaration(ctx acontext.Context[parser.IFunctionDeclarationContext]) bool {
    name := ctx.AST.IDENTIFIER().GetText()
    fn, err := ctx.Scope.Resolve(ctx, name)
    // ... existing code ...

    if block := ctx.AST.Block(); block != nil {
        // Analyze the block first
        if !statement.AnalyzeBlock(acontext.Child(ctx, block).WithScope(fn)) {
            return false
        }

        // NEW: Track channels after analysis (when all symbols are resolved)
        if err := channels.TrackChannels(ctx, fn, block); err != nil {
            ctx.Diagnostics.AddError(err, block)
            return false
        }

        // ... rest of validation ...
    }
    return true
}
```

#### Phase 4: Transfer from Symbol Table to IR

**File**: `arc/text/text.go`

Update the IR construction to copy channel info from symbol table:

```go
func Analyze(
    ctx_ context.Context,
    t Text,
    resolver symbol.Resolver,
) (ir.IR, analyzer.Diagnostics) {
    // ... existing analysis code ...

    // func 2: Iterate through the root scope children to assemble functions
    for _, c := range i.Symbols.Children {
        if c.Kind == symbol.KindFunction {
            fnDecl, ok := c.AST.(parser.IFunctionDeclarationContext)
            var bodyAst antlr.ParserRuleContext = fnDecl
            if ok {
                bodyAst = fnDecl.Block()
            }

            // NEW: Build channels from symbol scope
            channels := ir.NewChannels()
            if c.ChannelsRead != nil {
                channels.Read = c.ChannelsRead
            }
            if c.ChannelsWrite != nil {
                channels.Write = c.ChannelsWrite
            }

            i.Functions = append(i.Functions, ir.Function{
                Key:      c.Name,
                Body:     ir.Body{Raw: "", AST: bodyAst},
                Config:   *c.Type.Config,
                Inputs:   *c.Type.Inputs,
                Outputs:  *c.Type.Outputs,
                Channels: channels,  // NEW: Add channels to function
            })
        }
    }

    // ... existing node/edge building code ...

    // When building nodes, copy channels from function + add config channels
    for i, n := range nodes {
        // Find the function definition
        fn, _ := i.Functions.Find(n.Type)

        // Start with function's channels
        nodeChannels := ir.OverrideChannels(fn.Channels)

        // Add channels from config (already done in extractConfigValues)
        // This merges config channels with body channels
        if existingChannels, ok := n.Channels.Read != nil {
            for ch := range n.Channels.Read {
                nodeChannels.Read.Add(ch)
            }
        }

        n.Channels = nodeChannels
        nodes[i] = n
    }
}
```

**File**: `arc/ir/function.go`

Update Function struct to include Channels:

```go
type Function struct {
    Key      string       `json:"key"`
    Body     Body         `json:"body"`
    Config   types.Params `json:"config"`
    Inputs   types.Params `json:"inputs"`
    Outputs  types.Params `json:"outputs"`
    Channels Channels     `json:"channels"`  // NEW
}
```

#### Phase 5: Handle Graph Mode

**File**: `arc/graph/graph.go`

Update graph analysis to populate channels:

```go
func Analyze(
    ctx_ context.Context,
    g Graph,
    resolver symbol.Resolver,
) (ir.IR, analyzer.Diagnostics) {
    // ... existing function registration code ...

    // Step 2: Analyze Function Bodies AND track channels
    for _, fn := range g.Functions {
        funcScope, err := ctx.Scope.GetChildByParserRule(fn.Body.AST)
        // ... existing code ...

        // NEW: Track channels if we have source
        if fn.Body.Raw != "" {
            blockCtx, ok := fn.Body.AST.(parser.IBlockContext)
            if !ok {
                ctx.Diagnostics.AddError(errors.New("function body must be a block"), fn.Body.AST)
                return ir.IR{}, *ctx.Diagnostics
            }

            // Analyze block
            if !analyzer.AnalyzeBlock(acontext.Child(ctx, blockCtx).WithScope(funcScope)) {
                return ir.IR{}, *ctx.Diagnostics
            }

            // NEW: Track channels
            if err := channels.TrackChannels(ctx_, funcScope, blockCtx); err != nil {
                ctx.Diagnostics.AddError(err, fn.Body.AST)
                return ir.IR{}, *ctx.Diagnostics
            }
        }
    }

    // ... rest of analysis ...

    // Step 6: Build IR Nodes AND populate channels from function + config
    nodes := make(ir.Nodes, len(g.Nodes))
    for i, n := range g.Nodes {
        substituted := ctx.Constraints.ApplySubstitutions(freshTypes[n.Key])

        // NEW: Get function's channel tracking
        fnSym, err := ctx.Scope.Resolve(ctx, n.Type)
        if err != nil {
            ctx.Diagnostics.AddError(err, nil)
            return ir.IR{}, *ctx.Diagnostics
        }

        nodeChannels := ir.NewChannels()

        // Copy channels from function definition
        if fnSym.ChannelsRead != nil {
            for ch := range fnSym.ChannelsRead {
                nodeChannels.Read.Add(ch)
            }
        }
        if fnSym.ChannelsWrite != nil {
            for ch := range fnSym.ChannelsWrite {
                nodeChannels.Write.Add(ch)
            }
        }

        // Add channels from config values
        for key, configType := range substituted.Config.Iter() {
            if configType.Kind == types.KindChan {
                configValue, ok := n.ConfigValues[key]
                if !ok {
                    continue
                }
                var k uint32
                if err := zyn.Uint32().Coerce().Parse(configValue, &k); err == nil {
                    nodeChannels.Read.Add(k)
                }
            }
        }

        nodes[i] = ir.Node{
            Key:          n.Key,
            Type:         n.Type,
            ConfigValues: n.ConfigValues,
            Channels:     nodeChannels,  // NEW: populated
            Config:       *substituted.Config,
            Inputs:       *substituted.Inputs,
            Outputs:      *substituted.Outputs,
        }
    }

    // ... rest of code ...
}
```

#### Phase 6: Build ReactiveDeps for Runtime

**File**: `arc/runtime/state/state.go` or new builder file

```go
// BuildReactiveDeps creates the map from channel keys to node keys
func BuildReactiveDeps(nodes ir.Nodes) map[uint32][]string {
    deps := make(map[uint32][]string)

    for _, node := range nodes {
        for channelKey := range node.Channels.Read {
            deps[channelKey] = append(deps[channelKey], node.Key)
        }
    }

    return deps
}
```

## Testing Strategy

### Unit Tests

1. **Test channel tracker** (`arc/analyzer/channels/tracker_test.go`)
   - Test tracking reads in expressions
   - Test tracking writes in statements
   - Test tracking in config parameters
   - Test tracking in if/else blocks
   - Test tracking with stateful variables

2. **Test symbol table integration**
   - Verify ChannelsRead/Write are populated during analysis
   - Test with nested scopes

3. **Test IR transfer**
   - Verify channels copied from symbol table to IR.Function
   - Verify channels copied to IR.Node with config merging

### Integration Tests

1. **Text mode**
   - Parse Arc source with channel operations
   - Verify Channels field populated in IR

2. **Graph mode**
   - Build graph with functions containing channel ops
   - Verify Channels field populated in IR

3. **Runtime**
   - Build ReactiveDeps from IR
   - Verify correct node marking when channels fire

## Migration Path

1. **Phase 1 (Current PR)**: Implement core channel tracking
   - Add ChannelsRead/Write to symbol.Scope
   - Implement basic channel tracker
   - Wire into analyzer for text mode
   - Tests

2. **Phase 2**: Graph mode support
   - Update graph.Analyze to track channels
   - Tests

3. **Phase 3**: Runtime integration
   - BuildReactiveDeps helper
   - Update scheduler to use ReactiveDeps
   - Integration tests

## Open Questions

1. **How to handle channel references through variables?**
   ```arc
   stage foo{} () {
       ch $= sensor_channel
       val := <-ch
   }
   ```
   Answer: Track variable types during analysis, resolve transitively.

2. **How to handle unresolved channels?**
   Answer: Error during analysis (already happens with symbol resolution).

3. **Should we track channel usage in nested blocks differently?**
   Answer: No, union all reads/writes in function scope.

4. **What about channels passed as function arguments?**
   Answer: Not supported yet - functions can't take channels as runtime parameters, only config.

## Success Criteria

- [ ] All unit tests pass
- [ ] Integration test: Text mode flow with channel ops populates Channels field
- [ ] Integration test: Graph mode with channel ops populates Channels field
- [ ] Runtime can build ReactiveDeps from IR.Nodes
- [ ] Existing tests continue to pass
- [ ] No performance degradation (channel tracking is O(n) in AST size)
