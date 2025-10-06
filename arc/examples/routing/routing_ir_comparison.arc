// Routing Approaches: Text → IR Mapping
// Shows how each syntax translates to graph IR

// ============================================================================
// APPROACH 1: Multiple Statements
// ============================================================================

// Text:
sensor -> controller{};
sensor -> logger{};
sensor -> display{};

// IR Edges:
// [
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "controller_1", "param": "input"}},
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "logger_1", "param": "input"}},
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "display_1", "param": "input"}}
// ]
//
// IR Nodes:
// [
//   {"key": "sensor", "type": "channel"},
//   {"key": "controller_1", "type": "controller"},
//   {"key": "logger_1", "type": "logger"},
//   {"key": "display_1", "type": "display"}
// ]
//
// Total: 4 nodes, 3 edges

// ============================================================================
// APPROACH 2A: Bracket Syntax (Desugar to Multiple Edges)
// ============================================================================

// Text:
sensor -> [controller{}, logger{}, display{}];

// IR Edges (desugared):
// [
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "controller_1", "param": "input"}},
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "logger_1", "param": "input"}},
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "display_1", "param": "input"}}
// ]
//
// IR Nodes:
// [
//   {"key": "sensor", "type": "channel"},
//   {"key": "controller_1", "type": "controller"},
//   {"key": "logger_1", "type": "logger"},
//   {"key": "display_1", "type": "display"}
// ]
//
// Total: 4 nodes, 3 edges
// SAME AS APPROACH 1 - Pure syntax sugar!

// ============================================================================
// APPROACH 2B: Bracket Syntax (Create Implicit Tee Node)
// ============================================================================

// Text:
sensor -> [controller{}, logger{}, display{}];

// IR Edges (with implicit tee):
// [
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "tee_0", "param": "input"}},
//   {"source": {"node": "tee_0", "param": "output"},
//    "target": {"node": "controller_1", "param": "input"}},
//   {"source": {"node": "tee_0", "param": "output"},
//    "target": {"node": "logger_1", "param": "input"}},
//   {"source": {"node": "tee_0", "param": "output"},
//    "target": {"node": "display_1", "param": "input"}}
// ]
//
// IR Nodes:
// [
//   {"key": "sensor", "type": "channel"},
//   {"key": "tee_0", "type": "tee"},        // Extra node!
//   {"key": "controller_1", "type": "controller"},
//   {"key": "logger_1", "type": "logger"},
//   {"key": "display_1", "type": "display"}
// ]
//
// Total: 5 nodes, 4 edges
// DIFFERENT - Creates intermediate node

// ============================================================================
// APPROACH 3: Named Output Routing
// ============================================================================

stage analyzer{} (input f32) {
    mean f32
    peak f32
} {
    // ... logic ...
}

// Text:
sensor -> analyzer{} -> {;
    mean -> logger{},;
    peak -> alarm{};
}

// IR Edges:
// [
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "analyzer_1", "param": "input"}},
//   {"source": {"node": "analyzer_1", "param": "mean"},
//    "target": {"node": "logger_1", "param": "input"}},
//   {"source": {"node": "analyzer_1", "param": "peak"},
//    "target": {"node": "alarm_1", "param": "input"}}
// ]
//
// IR Nodes:
// [
//   {"key": "sensor", "type": "channel"},
//   {"key": "analyzer_1", "type": "analyzer"},
//   {"key": "logger_1", "type": "logger"},
//   {"key": "alarm_1", "type": "alarm"}
// ]
//
// Total: 4 nodes, 3 edges
// Uses Handle.param to distinguish outputs

// ============================================================================
// APPROACH 4: Explicit Tee
// ============================================================================

// Text:
sensor -> tee{controller{}, logger{}, display{}};

// IR Edges:
// [
//   {"source": {"node": "sensor", "param": "output"},
//    "target": {"node": "tee_1", "param": "input"}},
//   {"source": {"node": "tee_1", "param": "output"},
//    "target": {"node": "controller_1", "param": "input"}},
//   {"source": {"node": "tee_1", "param": "output"},
//    "target": {"node": "logger_1", "param": "input"}},
//   {"source": {"node": "tee_1", "param": "output"},
//    "target": {"node": "display_1", "param": "input"}}
// ]
//
// IR Nodes:
// [
//   {"key": "sensor", "type": "channel"},
//   {"key": "tee_1", "type": "tee", "config": {...}},
//   {"key": "controller_1", "type": "controller"},
//   {"key": "logger_1", "type": "logger"},
//   {"key": "display_1", "type": "display"}
// ]
//
// Total: 5 nodes, 4 edges
// Tee is explicit node in graph

// ============================================================================
// Visual Comparison in Graph Editor
// ============================================================================

// Approach 1 & 2A (Multiple edges, no tee):
//
//     ┌────────┐
//     │ sensor │
//     └───┬────┘
//         │
//    ┌────┼────┬────┐
//    │    │    │    │
//    v    v    v    v
// [ctrl][log][disp]
//
// 4 boxes on canvas

// Approach 2B & 4 (With tee node):
//
//     ┌────────┐
//     │ sensor │
//     └───┬────┘
//         │
//         v
//     ┌────────┐
//     │  tee   │
//     └───┬────┘
//         │
//    ┌────┼────┬────┐
//    │    │    │    │
//    v    v    v    v
// [ctrl][log][disp]
//
// 5 boxes on canvas (tee is visible)

// Approach 3 (Named outputs):
//
//     ┌────────┐
//     │ sensor │
//     └───┬────┘
//         │
//         v
//     ┌──────────┐
//     │ analyzer │
//     │  •mean   │──────> [logger]
//     │  •peak   │──────> [alarm]
//     └──────────┘
//
// Node has multiple output ports

// ============================================================================
// Complexity Analysis
// ============================================================================

// For N target stages:

// Multiple Statements:
//   - Text lines: N
//   - IR nodes: N + 1 (source + targets)
//   - IR edges: N
//   - Graph boxes: N + 1

// Brackets (desugar):
//   - Text lines: 1 (+ N children)
//   - IR nodes: N + 1
//   - IR edges: N
//   - Graph boxes: N + 1
//   - SAME runtime structure as multiple statements

// Brackets (implicit tee):
//   - Text lines: 1 (+ N children)
//   - IR nodes: N + 2 (adds tee)
//   - IR edges: N + 1
//   - Graph boxes: N + 2

// Explicit Tee:
//   - Text lines: 1 (+ N children)
//   - IR nodes: N + 2 (explicit tee)
//   - IR edges: N + 1
//   - Graph boxes: N + 2
//   - SAME runtime structure as implicit tee

// Named Outputs:
//   - Text lines: 1 (+ N outputs)
//   - IR nodes: N + 1
//   - IR edges: N
//   - Graph boxes: N + 1 (multi-port node)
//   - Requires stage to define outputs

// ============================================================================
// Recommendation for Implementation
// ============================================================================

// START WITH:
// 1. Multiple statements (simplest, unambiguous)
// 2. Named output routing (necessary for multi-output stages)
// 3. Explicit tee{} (already in spec)

// CONSIDER LATER:
// 4. Bracket syntax as pure desugar (syntactic sugar only)
//    sensor -> [a, b, c]  =>  sensor->a; sensor->b; sensor->c;

// AVOID:
// - Implicit tee creation (confusing, hidden nodes)
// - Bracket fan-in (ambiguous merge/all semantics)
// - Complex nesting without clear rules
