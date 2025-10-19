# Arc Routing: Final Recommendations

**Status:** Design Recommendation **Date:** 2025-10-06 **Context:** Routing syntax for
fan-out, fan-in, and conditional logic

---

## Executive Summary

After exploring routing patterns for Arc, the **critical insight** is that conditional
routing (not simple fan-out) drives the design. Control systems need to route data
differently based on state, sensor values, and operating conditions.

**Core Recommendation:** Implement **named output routing with reactive semantics** as
the foundation. Multiple statements and explicit tee already work. Bracket syntax is
optional syntactic sugar for later.

---

## What to Implement Now

### 1. Named Output Stage Declarations ⭐ **CRITICAL**

**Add multi-return syntax to stage declarations:**

```arc
stage my_router{} (input f32) {
    output_a f32
    output_b f32
    output_c f32
} {
    // Stage body
}
```

**Grammar changes required:**

```antlr
stageDeclaration
    : STAGE IDENTIFIER configBlock? LPAREN parameterList? RPAREN
      returnType? block
    ;

returnType
    : type                              // Single return (existing)
    | LBRACE namedReturnList RBRACE     // Multiple named returns (NEW)
    ;

namedReturnList
    : namedReturn (COMMA namedReturn)*
    ;

namedReturn
    : IDENTIFIER type
    ;
```

**Why critical:**

- Required for expressing conditional routing
- Control systems need state-dependent data flow
- Visual editor needs multi-port nodes
- Foundation for all other routing patterns

---

### 2. Named Output Routing Tables ⭐ **CRITICAL**

**Add routing table syntax for connecting specific outputs:**

```arc
sensor -> my_router{} -> {
    output_a -> target_a{},
    output_b -> target_b{},
    output_c -> target_c{}
}
```

**Grammar changes required:**

```antlr
flowStatement
    : flowNode (ARROW flowNode)+ routingTable? SEMICOLON?
    ;

routingTable
    : LBRACE routingEntry (COMMA routingEntry)* RBRACE
    ;

routingEntry
    : IDENTIFIER ARROW flowNode
    ;
```

**Why critical:**

- Maps output names to specific targets
- Clear semantic meaning (high → alarm, low → logger)
- Preserves structure in Text ↔ Graph round-trips
- Natural for visual editing

---

### 3. Reactive "Don't Send" Semantics ⭐ **CRITICAL**

**Rule:** Outputs only "fire" when assigned. Unassigned outputs don't trigger downstream
execution.

```arc
stage demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        // 'low' not assigned → logger doesn't execute
    } else {
        low = value
        // 'high' not assigned → alarm doesn't execute
    }
}
```

**Compiler implementation:**

1. Generate dirty flags per output:

   ```c
   struct {
       f32 high;
       bool high_dirty;
       f32 low;
       bool low_dirty;
   }
   ```

2. Clear flags at start of stage execution:

   ```c
   out->high_dirty = false;
   out->low_dirty = false;
   ```

3. Set flag on assignment:

   ```c
   out->high = value;
   out->high_dirty = true;
   ```

4. Check flag before downstream execution:
   ```c
   if (out->high_dirty) {
       alarm_execute(out->high);
   }
   ```

**Static analysis rules:**

- **Warning** if output is never assigned in any branch
- **Warning** if output might not be assigned in some execution path
- **Error** if stage declares output but never assigns it anywhere

**Why critical:**

- No sentinel value confusion (0 might be valid data)
- Efficient (skip unnecessary downstream execution)
- Type-safe (no optional types needed)
- Matches Arc's reactive dataflow model

---

### 4. IR Handle Support ✅ **ALREADY EXISTS**

The IR already supports named outputs via `Handle`:

```go
type Handle struct {
    Node  string  // "router_1"
    Param string  // "output_a", "output_b", etc.
}

type Edge struct {
    Source Handle
    Target Handle
}
```

**No changes needed** - just use `Handle.Param` to distinguish outputs.

---

## What Already Works

### Multiple Flow Statements ✅

```arc
sensor -> controller{}
sensor -> logger{}
sensor -> display{}
```

**Keep this** - it's simple, explicit, and works for basic fan-out.

### Explicit Tee Stage ✅

```arc
sensor -> tee{controller{}, logger{}, display{}}
```

**Keep this** - useful for broadcast patterns where all targets always receive data.

---

## What to Defer

### Bracket Syntax 🔧 **PHASE 2**

```arc
sensor -> [controller{}, logger{}, display{}]
```

**Decision:** Defer until after named outputs are implemented.

**Rationale:**

- Not critical for control logic
- Purely syntactic sugar (can desugar to multiple statements)
- Named outputs solve the hard problem (conditionals)
- Can be added later without breaking changes

**If implemented later:**

- Should desugar to multiple edges (NO implicit tee node)
- Useful for: `router{} -> {high -> [alarm{}, log{}]}`

### Fan-In Bracket Syntax ❌ **DON'T DO**

```arc
[sensor_a, sensor_b, sensor_c] -> processor{}
```

**Decision:** Don't implement.

**Rationale:**

- Ambiguous semantics (merge? all? any?)
- Users should explicitly use `merge{}`, `all{}`, or `any{}` stages
- Explicit is better than implicit for dataflow

### Optional Types ❌ **DON'T NEED**

```arc
stage demux{} (value f32) {
    high f32?  // Optional type
    low f32?
}
```

**Decision:** Don't implement optional types for routing.

**Rationale:**

- Reactive semantics (dirty flags) solve the problem
- No need for additional type complexity
- Optional types may be useful elsewhere, but not for routing

---

## Implementation Roadmap

### Phase 1: Named Outputs (Sprint 1-2)

**Week 1: Grammar & Parser**

- [ ] Update `ArcParser.g4` with multi-return syntax
- [ ] Update `ArcParser.g4` with routing table syntax
- [ ] Regenerate parser with ANTLR
- [ ] Add parser tests

**Week 2: Analyzer**

- [ ] Extend symbol table to track output names per stage
- [ ] Add type checking for routing tables (verify output names exist)
- [ ] Implement static analysis for unassigned outputs
- [ ] Add analyzer tests

**Week 3: IR & Compiler**

- [ ] Ensure `Handle.Param` is used for named outputs
- [ ] Generate dirty flags per output in WASM
- [ ] Implement conditional downstream execution
- [ ] Add compiler tests

**Week 4: Integration & Testing**

- [ ] End-to-end test: demux stage with conditional routing
- [ ] Test stratification with conditional edges
- [ ] Test cycle detection with named outputs
- [ ] Update spec.md with named output semantics

### Phase 2: Visual Editor (Sprint 3-4)

**Week 5-6: Graph Rendering**

- [ ] Multi-port node rendering
- [ ] Output port labels
- [ ] Edge routing from specific ports
- [ ] Port tooltips (name, type, condition)

**Week 7: Runtime Visualization**

- [ ] Active port highlighting during execution
- [ ] Color-coded outputs (safe/warning/danger)
- [ ] Data flow animation
- [ ] Execution statistics per path

**Week 8: Text ↔ Graph**

- [ ] Parse named routing tables to multi-port edges
- [ ] Generate named routing tables from multi-port nodes
- [ ] Preserve routing structure in round-trips
- [ ] Test bidirectional conversion

### Phase 3: Standard Library (Sprint 5)

**Week 9: Core Conditional Stages**

- [ ] `demux_bool{} (value, condition) {when_true, when_false}`
- [ ] `demux_threshold{threshold} (value) {high, low}`
- [ ] `demux_range{low, high} (value) {below, in, above}`
- [ ] `gate{} (value, enable) → value`
- [ ] `select{} (option_0, option_1, which) → value`

**Week 10: Documentation & Examples**

- [ ] Document conditional routing patterns
- [ ] Rocket engine control examples
- [ ] State machine examples
- [ ] Sensor fusion examples
- [ ] Migration guide for existing code

### Phase 4: Optional Syntactic Sugar (Sprint 6+)

**Future (if desired):**

- [ ] Bracket syntax as pure desugar
- [ ] Binding syntax: `stage{} as x; x.out -> target`
- [ ] Inline expression routing

---

## Design Decisions Summary

| Feature                       | Decision         | Rationale                         |
| ----------------------------- | ---------------- | --------------------------------- |
| **Named output declarations** | ✅ Implement now | Required for conditional routing  |
| **Routing tables**            | ✅ Implement now | Clear output → target mapping     |
| **Reactive semantics**        | ✅ Implement now | Efficient, no sentinel values     |
| **Multiple statements**       | ✅ Keep          | Simple, explicit, works           |
| **Explicit tee**              | ✅ Keep          | Useful for broadcast              |
| **Bracket fan-out**           | 🔧 Phase 2       | Nice sugar, not critical          |
| **Bracket fan-in**            | ❌ Don't do      | Ambiguous, use explicit merge/all |
| **Optional types**            | ❌ Don't need    | Reactive semantics sufficient     |

---

## Critical Success Factors

### 1. Get the Semantics Right

The "don't send" semantics are fundamental. Reactive approach (dirty flags) is:

- Efficient (skip execution)
- Natural (assignment = fire)
- Type-safe (no optionals)
- Explicit (static analysis catches errors)

### 2. Visual Editor Integration

Named outputs aren't just text syntax - they're essential for visual editing:

- Multi-port nodes show routing decisions
- Port labels provide semantic clarity
- Runtime highlighting shows active paths
- Text ↔ Graph round-trips preserve structure

### 3. Standard Library

Provide common conditional stages out of the box:

- `demux_*` for threshold/range/boolean routing
- `gate{}` for enable/disable
- `select{}` for choosing between options

Users shouldn't have to write these from scratch.

### 4. Static Analysis

Catch routing mistakes at compile time:

- Output never assigned → warning
- Output might not be assigned → warning
- Stage declares output but never uses → error

Prevent runtime surprises.

---

## Migration Strategy

### For Existing Code

**Current pattern (works today):**

```arc
sensor -> tee{alarm{}, logger{}}
```

**Still works after update** - no breaking changes.

### For New Conditional Code

**Before (workaround):**

```arc
sensor -> alarm{}  // Always executes
sensor -> logger{} // Always executes
```

**After (proper):**

```arc
sensor -> demux{threshold: 100} -> {
    high -> alarm{},   // Only when > 100
    low -> logger{}    // Only when <= 100
}
```

### Documentation

Update spec.md with:

- Multi-return stage syntax
- Routing table syntax
- Reactive semantics rules
- Standard library conditional stages
- Visual editor screenshots
- Common patterns and anti-patterns

---

## Risks & Mitigations

### Risk 1: Complexity

**Concern:** Named outputs add language complexity.

**Mitigation:**

- Simple cases still work (single return, multiple statements)
- Complexity is essential, not accidental (control systems need conditionals)
- Good standard library reduces boilerplate

### Risk 2: Visual Editor Dependency

**Concern:** Named outputs are hard to use in text-only mode.

**Mitigation:**

- Text syntax is clear: `stage{} -> {out1 -> x, out2 -> y}`
- Multiple statements still work for simple cases
- LSP hover can show output names/types
- Good examples and documentation

### Risk 3: Performance

**Concern:** Dirty flags add overhead.

**Mitigation:**

- One bool per output (minimal memory)
- Simple check before execution (fast)
- Eliminates unnecessary downstream execution (net win)
- Critical for real-time systems

### Risk 4: Static Analysis False Positives

**Concern:** Warnings about unassigned outputs might be noisy.

**Mitigation:**

- Make warnings configurable
- Provide explicit "no-op" keyword if needed: `out = void`
- Good error messages with suggestions
- Only warn for likely mistakes

---

## Success Metrics

### Must Have (Phase 1)

- [ ] Can declare multi-output stages
- [ ] Can route named outputs to specific targets
- [ ] Unassigned outputs don't trigger execution
- [ ] Static analysis catches unassigned outputs
- [ ] Compiles to WASM with dirty flags

### Should Have (Phase 2)

- [ ] Visual editor shows multi-port nodes
- [ ] Runtime visualization highlights active paths
- [ ] Text ↔ Graph round-trip preserves structure
- [ ] Standard library conditional stages

### Nice to Have (Phase 3+)

- [ ] Bracket syntax for sub-fan-out
- [ ] Performance profiling per path
- [ ] Auto-generation of common patterns

---

## Conclusion

**Named output routing is the foundation for real control logic in Arc.**

It's not about fan-out (that's easy) - it's about **conditional routing** based on
state, sensors, and operating conditions. This is fundamental to control systems.

Implement named outputs first, before any syntactic sugar. Get the semantics right
(reactive with dirty flags). Provide good standard library stages. The visual editor
will make it shine.

**Bottom line:** This isn't a nice-to-have feature - it's **essential architecture** for
Arc to be useful in real control systems.

---

## Appendix: Example Use Cases

### Use Case 1: Engine State-Dependent Logging

```arc
chamber_temp -> adaptive_logger{} -> {
    high_rate_out -> critical_logger{rate: 10khz},  // During ignition
    low_rate_out -> normal_logger{rate: 10hz}       // During idle
}
```

### Use Case 2: Safety Threshold Routing

```arc
pressure -> range_classifier{low: 50, high: 500} -> {
    below_range -> low_pressure_alarm{},
    in_range -> normal_controller{},
    above_range -> emergency_shutdown{}
}
```

### Use Case 3: Redundant Sensor Selection

```arc
sensor_selector{} -> {
    selected -> controller{},
    fault_detected -> backup_mode{}
}
```

### Use Case 4: State Machine Routing

```arc
state_router{} -> {
    idle_out -> idle_handler{},
    prestart_out -> prestart_sequence{},
    running_out -> running_controller{},
    shutdown_out -> shutdown_sequence{},
    fault_out -> emergency_response{}
}
```

All of these **require named output routing**. Multiple statements or tee stages can't
express these patterns cleanly.

---

**For questions or discussion, see:**

- `ROUTING_SUMMARY.md` - Complete analysis
- `routing_conditional*.arc` - Pattern examples
- `routing_no_send_semantics.arc` - Design options
- `README_ROUTING.md` - Comprehensive guide
