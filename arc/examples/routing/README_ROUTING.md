# Arc Routing Examples

This directory contains example files exploring different approaches to expressing
routing (fan-out/fan-in) in Arc programs.

## Files Overview

### 1. `routing_multiple_statements.arc`

**Current Spec Compliant** ✅

Shows the simplest approach: one flow statement per edge.

```arc
sensor -> controller{}
sensor -> logger{}
sensor -> display{}
```

**Pros:**

- Already works with current parser
- Crystal clear what edges exist
- Easy to add/remove individual connections
- Direct 1:1 mapping to IR edges

**Cons:**

- Verbose for large fan-outs
- Fan-out pattern not visually grouped

---

### 2. `routing_brackets.arc`

**Proposed Extension** 🔧

Introduces bracket syntax for concise fan-out grouping.

```arc
sensor -> [controller{}, logger{}, display{}]
```

**Key Design Question:** Does this desugar to multiple edges, or create an implicit
`tee{}` node?

**Option A (Desugar):**

```arc
sensor -> [a, b]
// Becomes:
sensor -> a
sensor -> b
```

**Option B (Implicit Tee):**

```arc
sensor -> [a, b]
// Becomes:
sensor -> tee{a, b}
```

**Status:** Would require grammar changes to support.

---

### 3. `routing_named_outputs.arc`

**Partially Supported** ⚠️

For stages with multiple return values, route specific outputs to specific targets.

```arc
stage analyzer{} (input f32) {
    mean f32
    peak f32
} {
    // ... logic ...
}

sensor -> analyzer{} -> {
    mean -> logger{},
    peak -> alarm{}
}
```

**Note:** The `select.arc` example already uses a variant of this:

```arc
ox_pt_1 > 10 -> select{} -> [
    if_true -> ox_pt_1_high,
    if_false -> ox_pt_1_low
]
```

**Status:** Syntax exists but needs formal grammar support for multi-output stages.

---

### 4. `routing_explicit_tee.arc`

**Current Spec Compliant** ✅

Uses the standard library `tee{}` stage (spec line 1112).

```arc
sensor -> tee{controller{}, logger{}, display{}}
```

**Pros:**

- Already in the spec
- Tee appears as node in graph (visible, configurable)
- Explicit about broadcast semantics

**Cons:**

- More verbose than brackets
- Adds extra node to graph
- Implementation detail in user code

---

### 5. `routing_comprehensive.arc`

**Mixed Approaches** 📋

A realistic rocket engine test stand example combining multiple patterns:

- Multiple statements for critical paths
- Bracket syntax for grouped monitoring
- Named outputs for multi-value stages
- Explicit tee for broadcast patterns

Shows how different approaches can coexist in the same program.

---

### 6. `routing_edge_cases.arc`

**Design Exploration** 🤔

Explores ambiguities and edge cases:

- Bracket parsing vs config braces
- Multi-level nesting
- Fan-in semantics `[a, b] -> c`
- Mixed fan-in/fan-out
- Empty/single-element brackets
- Precedence rules

Highlights why explicit approaches (multiple statements, explicit tee) avoid these
issues.

---

### 7. `routing_ir_comparison.arc`

**IR Analysis** 📊

Side-by-side comparison showing how each text syntax translates to IR:

```
Multiple Statements:    4 nodes, 3 edges
Brackets (desugar):     4 nodes, 3 edges  ← Same IR!
Brackets (implicit):    5 nodes, 4 edges  ← Extra tee node
Explicit Tee:           5 nodes, 4 edges
Named Outputs:          4 nodes, 3 edges (multi-port node)
```

Includes visual diagrams showing graph editor appearance.

---

## Implementation Recommendations

### Phase 1: Core Features (Already Working)

1. ✅ Multiple statements - `sensor -> a; sensor -> b`
2. ✅ Explicit tee - `sensor -> tee{a, b}`

### Phase 2: Multi-Output Support (Partial)

3. ⚠️ Named output routing - `stage -> {out_a -> x, out_b -> y}`
   - Requires formal multi-return syntax
   - Grammar updates needed

### Phase 3: Syntactic Sugar (Optional)

4. 🔧 Bracket syntax - `sensor -> [a, b, c]`
   - Purely desugars to multiple statements
   - No implicit nodes
   - Simplifies common patterns

### Not Recommended

- ❌ Implicit tee creation (hidden nodes confuse visual editor)
- ❌ Bracket fan-in without explicit merge/all semantics
- ❌ Complex nested routing without clear rules

---

## Graph ↔ Text Considerations

### Text → Graph (Easy)

All approaches can be converted to graph edges:

```
sensor -> [a, b] → Create 2 edges in graph
```

### Graph → Text (Harder)

When converting back, should we:

**Option 1: Always expand to multiple statements**

```
Graph: sensor→a, sensor→b
Text:  sensor -> a
       sensor -> b
```

**Option 2: Detect fan-out patterns and use brackets**

```
Graph: sensor→a, sensor→b, sensor→c
Text:  sensor -> [a, b, c]
```

**Option 3: Preserve original syntax in metadata**

```json
{
  "edges": [...],
  "metadata": {
    "original_syntax": "brackets"
  }
}
```

---

## Current Arc Grammar

The current `ArcParser.g4` supports:

```
flowStatement
    : flowNode (ARROW flowNode)+ SEMICOLON?
    ;

flowNode
    : channelIdentifier
    | stageInvocation
    | expression
    ;
```

This allows:

- ✅ `a -> b -> c`
- ✅ `sensor -> stage{config: 1}`
- ✅ `expr > 10 -> alarm{}`

This **does NOT** allow:

- ❌ `a -> [b, c]` (brackets)
- ❌ `a -> {out: b}` (named routing)
- ❌ `[a, b] -> c` (fan-in)

---

## Conditional Routing (Critical Insight!)

After exploring conditional routing patterns, **named output routing becomes
ESSENTIAL**, not optional:

### Why Conditional Routing Changes Everything

**Key insight:** Arc's routing topology is STATIC (graph fixed at compile time), but
data flow is DYNAMIC (which edges carry data varies at runtime).

You can't write:

```arc
if (condition) {
    sensor -> path_a{}
} else {
    sensor -> path_b{}
}
```

Instead, you use **stages with multiple outputs** that make routing decisions:

```arc
stage demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        low = 0.0
    } else {
        high = 0.0
        low = value
    }
}

sensor -> demux{threshold: 100} -> {
    high -> alarm{},
    low -> logger{}
}
```

### Conditional Routing Examples

See the new example files:

- `routing_conditional.arc` - All conditional patterns
- `routing_conditional_realistic.arc` - Rocket engine state-dependent routing
- `routing_conditional_syntax_comparison.arc` - Why named outputs win
- `routing_conditional_graph.arc` - Visual editor implications

### The Verdict

**Named output routing is mandatory** for Arc because:

1. **Control systems need conditional routing** - Route data differently based on state,
   sensor values, operating modes
2. **Visual graph editor requires it** - Multi-port nodes show which output connects
   where
3. **Semantic clarity** - Output names match conditions (`high`/`low`,
   `safe`/`warning`/`danger`)
4. **Round-trip preservation** - Graph ↔ Text conversions maintain structure

**Multiple statements fail** for conditionals:

```arc
// Loses structure - hard to see this is a demux
sensor -> demux{} -> high -> alarm{}
sensor -> demux{} -> low -> logger{}
```

**Tee is wrong tool** for conditionals:

```arc
// Tee broadcasts to ALL, doesn't decide
sensor -> tee{alarm{}, logger{}}  // Both get ALL data!
```

---

## Updated Recommendations

### Phase 1: Critical (Must Have)

1. ✅ Multiple statements - Simple cases
2. ✅ Explicit tee - Broadcast patterns
3. 🚨 **Named output routing** - **REQUIRED for conditional logic**
   - Multi-output stage declarations
   - Output routing table syntax `stage{} -> {out1 -> x, out2 -> y}`
   - Essential for control systems

### Phase 2: Nice to Have

4. 🔧 Bracket syntax - Syntactic sugar for fan-out
   - Only after named outputs work
   - Useful: `demux{} -> {high -> [alarm{}, log{}]}`

---

## Next Steps

1. **Prioritize named output syntax** - It's not optional for control systems
2. **Update grammar** - Add multi-output stage declarations and routing tables
3. **Design visual editor** - Multi-port nodes with labeled outputs
4. **Test conditional patterns** - State machines, mode switching, fault handling
5. **Document patterns** - Standard library demux/switch/gate stages

---

## Questions for Discussion

1. ~~Should brackets create nodes or just desugar?~~ → Named outputs first, brackets
   later
2. ~~How important is concise syntax vs. explicitness?~~ → Explicitness wins for
   conditionals
3. Should multi-output stages support variadic outputs? (e.g., N-way switch)
4. How should visual editor display conditional routing at runtime?
5. What standard library conditional stages should we provide?
