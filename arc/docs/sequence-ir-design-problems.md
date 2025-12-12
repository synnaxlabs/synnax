# Arc Sequence IR Design Problems

## Context

Arc is a domain-specific language for hardware automation and control systems. The core execution model is a **dataflow graph** with nodes (computations) and edges (data connections).

Arc has two edge operators:
- `->` (reactive/continuous): Data flows continuously while source is active
- `=>` (transition/one-shot): Fires once when condition becomes true

Sequences are state machines where **stages** group nodes that should be active together. Transitioning between stages changes which nodes are active.

## Current Implementation Problems

### 1. Over-Engineered Special Structures

The current implementation created special IR types that don't fit the dataflow model:

```go
type Sequence struct {
    Key    string
    Stages []string
    Entry  string
}

type Stage struct {
    Key         string
    Sequence    string
    Flows       []string      // Why separate from edges?
    Transitions []Transition  // Why not just edges?
    Imperative  string
    IsTerminal  bool          // Graph topology tells you this
}

type Transition struct {
    Condition  string
    TargetKind TransitionKind
    Target     string
    Priority   int
}
```

**Problem**: This creates a parallel structure to the node/edge graph instead of using the graph itself.

### 2. Top-Level Transitions Treated Specially

Code like `start_cmd => main` was handled as a special `TopLevelTransition` type instead of just being an edge from a channel node to sequence entry nodes.

### 3. IsTerminal Flag is Redundant

A stage is terminal if it has no outgoing transition edges. The graph topology already encodes this - no need for a flag.

### 4. Flows vs Transitions Distinction at Wrong Level

The current design separates `Flows []string` and `Transitions []Transition` in the Stage struct. But `->` and `=>` should both just be edges with different semantics. The distinction belongs on the Edge type, not in separate collections.

### 5. Imperative Blocks Overcomplicated

An imperative block like:
```arc
{
    // code
} => match {
    result1 => stage_a,
    result2 => stage_b
}
```

Should just be:
- An anonymous function node (the block)
- Edges from that node to different targets based on output value

Instead, it was treated as a special `Imperative` field.

### 6. Sequences Are Just Activation Controllers

The key insight: **sequences don't need special IR structures**. They just determine which nodes are active at any given time. The runtime activates/deactivates node groups based on stage transitions.

## What Sequences Actually Do

1. **Group nodes into stages** - "when stage X is active, these nodes run"
2. **Control activation** - transitioning from stage A to B deactivates A's nodes, activates B's nodes
3. **Define entry points** - which stage starts when the sequence is triggered

## Example: What the Graph Should Look Like

```arc
start_cmd => main

sequence main {
    stage pressurize {
        interval{100ms} -> log{"checking pressure"}
        pressure > 100 => vent
    }
    stage vent {
        valve_cmd <- 1
        wait{5s} => next
    }
    stage complete {
    }
}
```

This should become nodes and edges:
- `start_cmd` channel node
- `interval{100ms}` timer node (in stage `main.pressurize`)
- `log{"checking pressure"}` node (in stage `main.pressurize`)
- `pressure > 100` expression node (in stage `main.pressurize`)
- `valve_cmd <- 1` write node (in stage `main.vent`)
- `wait{5s}` timer node (in stage `main.vent`)

Edges:
- `start_cmd` → `main.pressurize` entry (transition edge, activates pressurize)
- `interval` → `log` (reactive edge)
- `pressure > 100` → `main.vent` entry (transition edge, activates vent)
- `wait{5s}` → `main.complete` entry (transition edge, activates complete)

## Open Questions

1. **How to represent stage membership?**
   - Option A: Nodes carry `Stage` and `Sequence` fields
   - Option B: Separate `StageNodes map[string][]string` on IR
   - Option C: Stages are just node groupings computed from edge topology

2. **How to represent edge semantics?**
   - Option A: `EdgeKind` enum (Reactive vs Transition)
   - Option B: Separate edge types
   - Option C: Edge metadata/properties

3. **How to handle `next` keyword?**
   - Resolved at analysis time to concrete stage target?
   - Or preserved and resolved at IR build time?

4. **How to represent stage activation?**
   - Is this purely a runtime concept?
   - Or does the IR need activation metadata?

5. **How to handle match blocks?**
   - Multiple edges from same source with different conditions?
   - Or a routing node that selects target?

---

## Prompt for Design Evaluation

The following prompt should be used in a new Claude session to evaluate different modeling approaches:

---

**Prompt:**

I'm designing the intermediate representation (IR) for Arc, a domain-specific language for hardware automation. Arc compiles to a dataflow graph that executes reactively.

**Core Language Constructs:**

1. **Reactive flows** (`->`): Continuous dataflow, e.g., `sensor -> filter{} -> output`
2. **Transitions** (`=>`): One-shot state changes, e.g., `condition => next_stage`
3. **Sequences**: State machines with ordered stages
4. **Stages**: Groups of nodes that are active together

**Example Program:**
```arc
// Top-level entry point
start_cmd => main

sequence main {
    stage pressurize {
        // Reactive flow: runs continuously while stage is active
        interval{100ms} -> log{"checking pressure"}

        // Transition: fires once when condition is true
        pressure > 100 => vent
    }

    stage vent {
        // One-shot action when stage activates
        valve_cmd <- 1

        // Timer-based transition
        wait{5s} => next
    }

    stage complete {
        // Terminal stage (no outgoing transitions)
    }
}

sequence abort {
    stage safed {
        all_valves <- 0
    }
}
```

**Key Semantics:**
- `->` edges flow data continuously while source node is active
- `=>` edges fire once when condition becomes true, potentially changing active stage
- `next` keyword resolves to the next stage in definition order
- Stages can transition to other stages in same sequence or to other sequences
- Imperative blocks `{ code } => match { ... }` are anonymous functions that route to different targets

**Current IR Structure:**
```go
type IR struct {
    Functions Functions  // Function definitions
    Nodes     Nodes      // Instantiated nodes in the graph
    Edges     Edges      // Dataflow connections
    Strata    Strata     // Execution ordering
}

type Node struct {
    Key    string
    Type   string         // Function type
    Config map[string]any
}

type Edge struct {
    Source Handle  // {Node, Param}
    Target Handle  // {Node, Param}
}
```

**Design Challenge:**

How should sequences and stages be represented in the IR? I see several options:

**Option A: Minimal - Just Tag Nodes**
- Add `Stage` and `Sequence` fields to Node
- Add `Kind` (Reactive/Transition) to Edge
- Runtime uses these tags to manage activation

**Option B: Explicit Stage Groups**
- Keep nodes simple
- Add `Stages map[string][]NodeKey` to IR
- Edges have `Kind` for semantics

**Option C: Activation Graph**
- Model stage activation as separate graph layer
- Stage nodes with activation edges between them
- Computation nodes belong to stage nodes

**Option D: Unified - Stages as Special Nodes**
- Stages become nodes in the graph
- "Activation" is a special edge type
- Stage nodes route to their child computation nodes

Please evaluate these options (and propose others) considering:
1. Simplicity of representation
2. Ease of IR construction from AST
3. Runtime execution model clarity
4. Optimization opportunities
5. Debuggability and tooling support

Provide a recommended approach with rationale and concrete Go struct definitions.
