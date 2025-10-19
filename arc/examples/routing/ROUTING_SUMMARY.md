# Arc Routing: The Complete Picture

## TL;DR - Conditional Routing Changed Everything

**Initial question:** How to express fan-out (one node to many)?

**Critical insight:** Control systems need **conditional routing** (route differently
based on state/conditions).

**Conclusion:** Named output routing is **mandatory**, not optional.

---

## The Problem Space

### 1. Simple Fan-out (Original Question)

```
sensor → controller_1
sensor → controller_2
sensor → logger
```

**Solutions:**

- Multiple statements ✅ (works today)
- Brackets `[a, b, c]` 🔧 (nice to have)
- Explicit tee ✅ (works today)

### 2. Conditional Routing (Real Requirement)

```
IF sensor > 100:
    route to alarm
ELSE:
    route to logger
```

**ONLY solution:**

- Named output routing 🚨 (REQUIRED)

---

## Why Conditional Routing Matters

### Arc's Execution Model

- **Routing topology:** STATIC (graph structure fixed at compile time)
- **Data flow:** DYNAMIC (which edges carry data varies at runtime)

You **cannot** write:

```arc
if (pressure > 100) {
    sensor -> alarm{}  // ❌ Flow statements not in control flow!
}
```

### The Solution: Multi-Output Stages

```arc
stage demux{threshold f64} (value f32) {
    high f32    // Output 1
    low f32     // Output 2
} {
    if (value > f32(threshold)) {
        high = value
        low = 0.0
    } else {
        high = 0.0
        low = value
    }
}

// Named output routing (ESSENTIAL)
sensor -> demux{threshold: 100} -> {
    high -> alarm{},
    low -> logger{}
}
```

Both paths exist in the graph, but only one receives non-zero data at runtime.

---

## Common Conditional Patterns

### Pattern 1: State-Based Routing

```arc
stage state_router{} (value f32, state u8) {
    idle_out f32
    active_out f32
    error_out f32
} { /* ... */ }

sensor -> state_router{} -> {
    idle_out -> idle_handler{},
    active_out -> active_handler{},
    error_out -> error_handler{}
}
```

### Pattern 2: Threshold Filtering

```arc
stage range_splitter{low f64, high f64} (value f32) {
    below_range f32
    in_range f32
    above_range f32
} { /* ... */ }

sensor -> range_splitter{low: 50, high: 500} -> {
    below_range -> low_alarm{},
    in_range -> normal_control{},
    above_range -> high_alarm{}
}
```

### Pattern 3: Sensor Selection

```arc
stage redundant_selector{} (primary f32, backup f32, fault u8) f32 {
    return fault ? backup : primary
}
```

### Pattern 4: Mode-Dependent Processing

```arc
stage adaptive_logger{} (value f32, critical_mode u8) {
    high_rate_out f32
    low_rate_out f32
} { /* ... */ }

sensor -> adaptive_logger{} -> {
    high_rate_out -> logger{rate: 1khz},
    low_rate_out -> logger{rate: 10hz}
}
```

---

## Syntax Comparison for Conditionals

### Named Outputs (Winner)

```arc
sensor -> demux{threshold: 100} -> {
    high -> alarm{},
    low -> logger{}
}
```

✅ Clear semantic mapping ✅ Grouped under router ✅ Visual structure preserved ✅ Works
with visual editor

### Multiple Statements (Loses Structure)

```arc
sensor -> demux{} -> high -> alarm{}
sensor -> demux{} -> low -> logger{}
```

❌ Relationship unclear ❌ Repetitive ❌ Can't see it's a conditional

### Tee (Wrong Tool)

```arc
sensor -> tee{alarm{}, logger{}}
```

❌ Broadcasts to ALL (no decision) ❌ Would need filters after tee (verbose)

---

## Visual Graph Editor Implications

### Multi-Port Node Display

```
    ┌───────────┐
    │   demux   │
    │ threshold │
    │   100.0   │
    │           │
    │ •high     │────────> [alarm]
    │           │
    │ •low      │────────> [logger]
    └───────────┘
```

### Essential Features

1. **Output port labels** - Show `high`, `low`, etc.
2. **Port tooltips** - Display condition (e.g., "when value > threshold")
3. **Runtime highlighting** - Active port glows during execution
4. **Color coding** - Green=safe, yellow=warning, red=danger
5. **Condition display** - Show routing logic in properties panel

### Text ↔ Graph Round-Trip

Must preserve named output structure:

```
Text: sensor -> demux{} -> {high -> alarm, low -> logger}
 ↓
Graph: demux_1 node with two output ports
 ↓
Text: sensor -> demux{} -> {high -> alarm, low -> logger}  ✅

NOT:
Text: sensor -> demux -> high -> alarm; sensor -> demux -> low -> logger  ❌
```

---

## Implementation Requirements

### Grammar Changes (ArcParser.g4)

#### 1. Multi-Output Stage Declarations

```antlr
stageDeclaration
    : STAGE IDENTIFIER configBlock? LPAREN parameterList? RPAREN
      multiReturnType? block
    ;

multiReturnType
    : type                              // Single return (current)
    | LBRACE namedReturnList RBRACE     // Multiple named returns (NEW)
    ;

namedReturnList
    : namedReturn (COMMA namedReturn)*
    ;

namedReturn
    : IDENTIFIER type
    ;
```

Example:

```arc
stage demux{} (value f32) {
    high f32
    low f32
} { /* ... */ }
```

#### 2. Named Output Routing

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

Example:

```arc
sensor -> demux{} -> {
    high -> alarm{},
    low -> logger{}
}
```

### Analyzer Changes

1. **Symbol table:** Track output names per stage
2. **Type checking:** Verify output names exist in routing tables
3. **Edge creation:** Map named outputs to `Handle.param` in IR
4. **Cycle detection:** Still works (uses node keys, not ports)

### Compiler Changes

Minimal - IR already supports this via `Handle`:

```go
type Handle struct {
    Node  string  // "demux_1"
    Param string  // "high" or "low"
}
```

---

## Standard Library Conditional Stages

### Recommended Built-ins

```arc
// Boolean demux
stage demux_bool{} (value f32, condition u8) {
    when_true f32
    when_false f32
}

// Threshold splitter
stage demux_threshold{threshold f64} (value f32) {
    high f32
    low f32
}

// Three-way range splitter
stage demux_range{low f64, high f64} (value f32) {
    below_range f32
    in_range f32
    above_range f32
}

// N-way enum router
stage switch_u8{} (value f32, selector u8) {
    out_0 f32
    out_1 f32
    out_2 f32
    out_3 f32
}

// Gate (enable/disable)
stage gate{} (value f32, enable u8) f32

// Priority selector (choose between sources)
stage select{} (option_0 f32, option_1 f32, which u8) f32
```

---

## Migration Path

### Phase 1: Core Named Outputs (CRITICAL)

1. Update grammar for multi-output stage declarations
2. Update grammar for named output routing tables
3. Implement analyzer support
4. Test basic demux patterns

### Phase 2: Visual Editor (HIGH PRIORITY)

1. Multi-port node rendering
2. Port labels and tooltips
3. Edge routing from specific ports
4. Round-trip Text ↔ Graph preservation

### Phase 3: Standard Library (IMPORTANT)

1. Implement core conditional stages (demux, gate, select)
2. Document conditional routing patterns
3. Provide examples for common use cases

### Phase 4: Syntactic Sugar (NICE TO HAVE)

1. Bracket syntax for sub-fan-out: `{high -> [alarm{}, log{}]}`
2. Binding syntax: `stage{} as x; x.out -> target`
3. Auto-generation of demux stages from inline expressions

---

## Real-World Impact

### Before (Multiple Statements Only)

```arc
// Hard to see this is state-dependent routing
sensor -> router{} -> idle -> handler_a{}
sensor -> router{} -> active -> handler_b{}
sensor -> router{} -> error -> handler_c{}
```

### After (Named Outputs)

```arc
// Clear conditional structure
sensor -> state_router{} -> {
    idle_out -> handler_a{},
    active_out -> handler_b{},
    error_out -> handler_c{}
}
```

### Visual Editor Before

```
[sensor] → [router] → [handler_a]
            [router] → [handler_b]
            [router] → [handler_c]
```

Can't tell which path for which state!

### Visual Editor After

```
[sensor] → [state_router]
             ├─ idle_out   → [handler_a]
             ├─ active_out → [handler_b]
             └─ error_out  → [handler_c]
```

Crystal clear mapping!

---

## Conclusion

**Conditional routing is not a nice-to-have - it's fundamental to control systems.**

Named output routing must be a **first-class language feature**, not an afterthought. It
affects:

- Grammar design
- Type system (multi-return stages)
- Visual editor (multi-port nodes)
- Standard library (demux/gate/select stages)
- Example patterns
- Documentation

**Recommendation:** Implement named output routing immediately, before bracket syntax or
other syntactic sugar. It's the foundation for expressing real control logic in Arc.

---

## Critical Design Decision: "No Send" Semantics

**Question:** How do you choose NOT to send to a named output?

```arc
stage demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        // What about 'low'? How to say "don't send"?
    }
}
```

### Options Explored

| Approach               | Example              | Pros      | Cons                                    |
| ---------------------- | -------------------- | --------- | --------------------------------------- |
| **Sentinel values**    | `low = 0.0`          | Simple    | 0 might be valid data, wastes execution |
| **Undefined**          | Don't assign         | Natural   | Requires tracking                       |
| **Optional types**     | `low = none`         | Explicit  | Adds type complexity                    |
| **Void keyword**       | `low = void`         | Clear     | New syntax                              |
| **Reactive semantics** | Don't assign         | Efficient | Implicit behavior                       |
| **Return statement**   | `return high: value` | Explicit  | Only one output per execution           |
| **Enable flags**       | `low_enabled = 0`    | Explicit  | Verbose                                 |

### Recommended: Reactive Semantics

**Rule:** Outputs only "fire" when assigned. Unassigned outputs don't execute downstream
stages.

```arc
stage demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        // 'low' not assigned → logger doesn't run
    } else {
        low = value
        // 'high' not assigned → alarm doesn't run
    }
}

sensor -> demux{threshold: 100} -> {
    high -> alarm{},    // Runs only when high assigned
    low -> logger{}     // Runs only when low assigned
}
```

**Implementation:** Compiler generates dirty flags per output:

```c
struct demux_outputs {
    f32 high;
    bool high_dirty;
    f32 low;
    bool low_dirty;
};

// Clear flags at start of execution
out->high_dirty = false;
out->low_dirty = false;

// Set flag when assigned
if (condition) {
    out->high = value;
    out->high_dirty = true;  // Mark as "should fire"
}

// Only execute downstream if dirty
if (out->high_dirty) {
    alarm_execute(out->high);
}
```

**Benefits:**

- Natural reactive semantics
- Efficient (skip unnecessary execution)
- No sentinel value confusion
- Type-safe (no optionals needed)
- Matches Arc's dataflow model

**Static Analysis:**

```arc
stage bad{} (value f32) {
    out f32
} {
    if (value > 10.0) {
        out = value
    }
    // Warning: 'out' not assigned in all branches
}
```

See `routing_no_send_semantics.arc` for detailed exploration of all options.

---

## See Also

- `routing_conditional.arc` - All conditional patterns
- `routing_conditional_realistic.arc` - Rocket engine examples
- `routing_conditional_syntax_comparison.arc` - Syntax trade-offs
- `routing_conditional_graph.arc` - Visual editor design
- `routing_no_send_semantics.arc` - "Don't send" design options
- `README_ROUTING.md` - Complete routing guide
