# Arc Routing Documentation Index

**Quick navigation for all routing-related documentation and examples.**

---

## 📋 Start Here

### [RECOMMENDATIONS.md](./RECOMMENDATIONS.md) ⭐ **READ THIS FIRST**

**Final design recommendations with implementation roadmap.**

What to implement now, what to defer, grammar changes needed, 10-week implementation
plan.

**TL;DR:** Implement named output routing with reactive semantics immediately. It's
essential for control systems, not optional.

---

## 📊 Summary Documents

### [ROUTING_SUMMARY.md](./ROUTING_SUMMARY.md)

**Complete analysis of routing patterns and design decisions.**

Covers:

- Simple fan-out vs conditional routing
- Why named outputs are mandatory
- "Don't send" semantics (7 options analyzed)
- Visual editor implications
- Grammar changes needed

### [README_ROUTING.md](./README_ROUTING.md)

**Comprehensive guide to all routing approaches.**

Covers:

- Overview of each routing pattern
- Files overview with status (✅ works, 🔧 proposed, ❌ not recommended)
- Current grammar capabilities
- Phase 1-4 implementation recommendations

---

## 🎯 Example Files by Topic

### Basic Routing (Original Question)

**How to route one source to multiple targets?**

| File                                                                 | Description             | Status           |
| -------------------------------------------------------------------- | ----------------------- | ---------------- |
| [routing_multiple_statements.arc](./routing_multiple_statements.arc) | One statement per edge  | ✅ Works today   |
| [routing_explicit_tee.arc](./routing_explicit_tee.arc)               | Using `tee{}` stage     | ✅ Works today   |
| [routing_brackets.arc](./routing_brackets.arc)                       | Proposed bracket syntax | 🔧 Needs grammar |
| [routing_ir_comparison.arc](./routing_ir_comparison.arc)             | Text → IR mapping       | 📊 Analysis      |
| [routing_edge_cases.arc](./routing_edge_cases.arc)                   | Parsing ambiguities     | 🤔 Design        |

---

### Conditional Routing (Critical Discovery)

**How to route differently based on state/conditions?**

| File                                                                                     | Description              | Key Insight                      |
| ---------------------------------------------------------------------------------------- | ------------------------ | -------------------------------- |
| [routing_conditional.arc](./routing_conditional.arc)                                     | All conditional patterns | ⭐ Demux, gate, switch, selector |
| [routing_conditional_realistic.arc](./routing_conditional_realistic.arc)                 | Rocket engine examples   | 🚀 Real-world patterns           |
| [routing_conditional_syntax_comparison.arc](./routing_conditional_syntax_comparison.arc) | Why named outputs win    | 📊 Comparison matrix             |
| [routing_conditional_graph.arc](./routing_conditional_graph.arc)                         | Visual editor design     | 🎨 Multi-port nodes              |
| [routing_named_outputs.arc](./routing_named_outputs.arc)                                 | Named output syntax      | 🚨 Required feature              |

---

### "Don't Send" Semantics (Design Decision)

**How to not send to an output?**

| File                                                             | Description        | Recommendation               |
| ---------------------------------------------------------------- | ------------------ | ---------------------------- |
| [routing_no_send_semantics.arc](./routing_no_send_semantics.arc) | 7 options explored | ✅ Reactive with dirty flags |

**Options analyzed:**

1. Sentinel values (0 = no data) ❌
2. Undefined (don't assign) ✅ **Recommended**
3. Optional types (f32?) ❌
4. Void keyword (void) 🔧
5. Reactive semantics (dirty flags) ✅ **Recommended**
6. Return statement (return high: value) ❌
7. Enable flags (high_enabled) ❌

---

### Complete Examples

| File                                                           | Description              | Use For                  |
| -------------------------------------------------------------- | ------------------------ | ------------------------ |
| [routing_complete_example.arc](./routing_complete_example.arc) | End-to-end rocket engine | Reference implementation |
| [routing_comprehensive.arc](./routing_comprehensive.arc)       | Mixed patterns           | See all approaches       |

---

## 🗺️ Reading Paths

### Path 1: "I just want to know what to do"

1. **[RECOMMENDATIONS.md](./RECOMMENDATIONS.md)** - Implementation roadmap
2. **[routing_complete_example.arc](./routing_complete_example.arc)** - See it in action

### Path 2: "I want to understand the design"

1. **[ROUTING_SUMMARY.md](./ROUTING_SUMMARY.md)** - Complete analysis
2. **[routing_conditional_syntax_comparison.arc](./routing_conditional_syntax_comparison.arc)** -
   Why this approach
3. **[routing_no_send_semantics.arc](./routing_no_send_semantics.arc)** - Design
   decisions

### Path 3: "I'm implementing the grammar"

1. **[RECOMMENDATIONS.md](./RECOMMENDATIONS.md)** - Grammar changes needed
2. **[routing_named_outputs.arc](./routing_named_outputs.arc)** - Syntax examples
3. **[README_ROUTING.md](./README_ROUTING.md)** - Current grammar state

### Path 4: "I'm building the visual editor"

1. **[routing_conditional_graph.arc](./routing_conditional_graph.arc)** - Multi-port
   node design
2. **[ROUTING_SUMMARY.md](./ROUTING_SUMMARY.md)** - Graph representation section
3. **[routing_complete_example.arc](./routing_complete_example.arc)** - Visual examples

### Path 5: "I want to see all options"

1. **[README_ROUTING.md](./README_ROUTING.md)** - All approaches overview
2. **[routing_comprehensive.arc](./routing_comprehensive.arc)** - Mixed patterns
3. **[routing_edge_cases.arc](./routing_edge_cases.arc)** - What could go wrong

---

## 🔑 Key Concepts

### Static Topology, Dynamic Flow

- **Routing topology:** Fixed at compile time (graph structure)
- **Data flow:** Dynamic at runtime (which edges carry data)

You can't write:

```arc
if (condition) {
    sensor -> alarm{}  // ❌ Flow statements not in control flow
}
```

Instead:

```arc
sensor -> demux{} -> {
    high -> alarm{},  // Both exist in graph
    low -> logger{}   // Only one fires at runtime
}
```

### Reactive Semantics

**Rule:** Outputs only "fire" when assigned.

```arc
stage demux{} (value f32) {
    high f32
    low f32
} {
    if (value > 100.0) {
        high = value  // Assigned → downstream executes
        // low not assigned → downstream skips
    }
}
```

### Named Outputs Are Mandatory

Not for fan-out (that's easy) - for **conditional routing** (that's essential).

Control systems need to route data differently based on:

- Engine state (idle, prestart, running, shutdown, fault)
- Sensor thresholds (below, in-range, above)
- Operating modes (manual, auto, emergency)
- Fault conditions (primary, backup, degraded)

**This requires named output routing.**

---

## 📈 Implementation Status

### ✅ Works Today

- Multiple flow statements
- Explicit tee stage
- Single-output stages
- Simple fan-out

### 🚨 Must Implement (Phase 1)

- Named output stage declarations
- Routing table syntax
- Reactive semantics (dirty flags)
- Static analysis for unassigned outputs

### 🔧 Nice to Have (Phase 2+)

- Bracket syntax (pure desugar)
- Visual editor multi-port nodes
- Standard library conditional stages
- Runtime visualization

### ❌ Not Recommended

- Bracket fan-in (ambiguous)
- Optional types for routing
- Implicit tee creation
- Sentinel value patterns

---

## 📝 Files by Status

### Works Today ✅

- `routing_multiple_statements.arc`
- `routing_explicit_tee.arc`

### Needs Grammar 🚨

- `routing_named_outputs.arc`
- `routing_conditional*.arc`
- `routing_complete_example.arc`

### Proposed Syntax 🔧

- `routing_brackets.arc`

### Analysis & Design 📊

- `routing_ir_comparison.arc`
- `routing_edge_cases.arc`
- `routing_no_send_semantics.arc`

### Documentation 📚

- `RECOMMENDATIONS.md`
- `ROUTING_SUMMARY.md`
- `README_ROUTING.md`
- `ROUTING_INDEX.md` (this file)

---

## 🎓 Learning Resources

### Concepts

- **Fan-out:** One source, multiple targets
- **Fan-in:** Multiple sources, one target
- **Demux:** Route data to different outputs based on condition
- **Gate:** Enable/disable a path
- **Tee:** Broadcast to all outputs (always)
- **Merge:** Combine multiple sources (any fires → output fires)
- **All:** Wait for all sources before firing

### Patterns

- **State-dependent routing:** Route based on system state
- **Threshold routing:** Route based on sensor values
- **Redundancy management:** Switch between primary/backup
- **Adaptive processing:** Different handling based on mode
- **Conditional logging:** High/low rate based on criticality

### Anti-Patterns

- Using tee when you need demux
- Sentinel values (0 = no data)
- Bracket fan-in without explicit merge
- Routing inside control flow (can't do this!)

---

## 💡 Quick Reference

### Syntax (Proposed)

**Multi-output stage:**

```arc
stage my_stage{} (input f32) {
    output_a f32
    output_b f32
} {
    // Logic
}
```

**Named routing:**

```arc
source -> my_stage{} -> {
    output_a -> target_a{},
    output_b -> target_b{}
}
```

**Reactive semantics:**

```arc
if (condition) {
    output_a = value  // Fires
    // output_b not assigned, doesn't fire
}
```

### Grammar Changes Needed

1. Multi-return types in stage declarations
2. Routing table syntax in flow statements
3. No other changes (IR already supports this)

### Compiler Changes Needed

1. Generate dirty flags per output
2. Clear flags at stage start
3. Set flag on assignment
4. Check flag before downstream execution

---

## ❓ FAQ

**Q: Why not just use multiple statements?** A: They work for simple fan-out but lose
semantic structure for conditionals. You can't see that outputs are mutually exclusive.

**Q: Why not use tee for everything?** A: Tee broadcasts to ALL targets. Can't express
"route to A OR B based on condition."

**Q: Why reactive semantics instead of optional types?** A: Simpler, more efficient,
matches dataflow model, no type system complexity.

**Q: Can I still use multiple statements?** A: Yes! They work for simple cases and will
continue to work.

**Q: When should I use brackets?** A: For sub-fan-out within routing tables:
`{high -> [alarm{}, log{}]}`

**Q: How do I route from multiple sources to one target?** A: Use explicit `merge{}`,
`all{}`, or `any{}` stages.

---

## 📞 Getting Help

**Found issues?** See edge cases in `routing_edge_cases.arc`

**Need examples?** See `routing_complete_example.arc`

**Want to discuss?** All design options explored in:

- `ROUTING_SUMMARY.md`
- `routing_no_send_semantics.arc`

**Ready to implement?** See `RECOMMENDATIONS.md`

---

**Last updated:** 2025-10-06 **Total files:** 15 (12 examples + 3 docs + 1 index)
