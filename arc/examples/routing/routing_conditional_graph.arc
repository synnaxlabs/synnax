// Conditional Routing: Graph Representation
// How conditional routing appears in visual graph editor

// ============================================================================
// Graph Nodes: Multi-Port vs Single-Port
// ============================================================================

stage demux{
    threshold f64
} (value f32) {
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

// ============================================================================
// Text Representation
// ============================================================================

sensor -> demux{threshold: 100.0} -> {;
    high -> alarm{},;
    low -> logger{};
}

// ============================================================================
// Graph Representation Option A: Multi-Port Node
// ============================================================================

// Visual:
//
//    ┌────────┐
//    │ sensor │
//    └───┬────┘
//        │
//        v
//    ┌───────────┐
//    │   demux   │
//    │ threshold │
//    │   100.0   │
//    │           │
//    │ •high     │────────> ┌───────┐
//    │           │          │ alarm │
//    │ •low      │──┐       └───────┘
//    └───────────┘  │
//                   │
//                   └──────> ┌────────┐
//                            │ logger │
//                            └────────┘

// IR:
// {
//   "nodes": [
//     {"key": "sensor", "type": "channel"},
//     {"key": "demux_1", "type": "demux", "config": {"threshold": 100.0}},
//     {"key": "alarm_1", "type": "alarm"},
//     {"key": "logger_1", "type": "logger"}
//   ],
//   "edges": [
//     {"source": {"node": "sensor", "param": "output"},
//      "target": {"node": "demux_1", "param": "value"}},
//     {"source": {"node": "demux_1", "param": "high"},
//      "target": {"node": "alarm_1", "param": "input"}},
//     {"source": {"node": "demux_1", "param": "low"},
//      "target": {"node": "logger_1", "param": "input"}}
//   ]
// }

// ============================================================================
// Graph Representation Option B: Single Output with Labels
// ============================================================================

// Visual:
//
//    ┌────────┐
//    │ sensor │
//    └───┬────┘
//        │
//        v
//    ┌───────────┐
//    │   demux   │
//    │ threshold │
//    │   100.0   │
//    └─────┬─────┘
//          │
//     ┌────┴─────┐
//     │          │
//  [high]     [low]
//     │          │
//     v          v
// ┌───────┐  ┌────────┐
// │ alarm │  │ logger │
// └───────┘  └────────┘

// Edge labels show which output port

// ============================================================================
// Complex Multi-Stage Conditional Graph
// ============================================================================

stage state_router{} (value f32, state u8) {
    idle_out f32
    active_out f32
    error_out f32
} {
    // ... routing logic ...
}

stage safety_check{} (value f32) {
    safe_out f32
    warning_out f32
    danger_out f32
} {
    // ... safety logic ...
}

// Text:
sensor -> state_router{} -> {;
    idle_out -> logger{},;
    active_out -> safety_check{} -> {;
        safe_out -> normal_control{},;
        warning_out -> cautious_control{},;
        danger_out -> emergency_shutdown{};
    },
    error_out -> fault_handler{};
}

system_state -> state_router{};

// Graph (Multi-Port Nodes):
//
//    ┌────────┐
//    │ sensor │
//    └───┬────┘
//        │
//        v
//    ┌──────────────┐      ┌──────────────┐
//    │ state_router │      │ system_state │
//    │              │<─────┤              │
//    │ •idle_out    │──────> [logger]
//    │ •active_out  │──┐
//    │ •error_out   │──┼───> [fault_handler]
//    └──────────────┘  │
//                      │
//                      v
//                 ┌─────────────┐
//                 │safety_check │
//                 │ •safe_out   │──> [normal_control]
//                 │ •warning_out│──> [cautious_control]
//                 │ •danger_out │──> [emergency_shutdown]
//                 └─────────────┘

// ============================================================================
// Interactive Graph Features for Conditional Routing
// ============================================================================

// Feature 1: Color-coded outputs
//   - Green port: safe/normal path
//   - Yellow port: warning/caution path
//   - Red port: danger/error path

// Feature 2: Hover shows condition
//   Hovering over "high" port shows: "when value > threshold"

// Feature 3: Runtime highlighting
//   During execution, active output ports glow
//   Shows which path is currently receiving data

// Feature 4: Output port tooltips
//   Each port shows:
//   - Name: "high"
//   - Type: f32
//   - Condition: value > threshold
//   - Current state: ACTIVE/INACTIVE

// Feature 5: Collapsible sub-graphs
//   Can collapse entire conditional branch:
//
//    ┌──────────────┐
//    │ state_router │
//    │ •idle_out    │──> [...]
//    │ •active_out  │──> [3 stages ▼]
//    │ •error_out   │──> [...]
//    └──────────────┘
//
//   Click ▼ to expand the active_out branch

// ============================================================================
// Graph Editor UX for Conditional Routing
// ============================================================================

// Creating conditional routing:
//
// 1. User drags demux{} stage onto canvas
// 2. Graph shows stage with multiple output ports
// 3. User connects sensor to demux input
// 4. User drags from demux.high port to alarm
// 5. User drags from demux.low port to logger
// 6. Text representation updates automatically

// Editing conditional routing:
//
// 1. User clicks demux node
// 2. Properties panel shows:
//    - Config: threshold = 100.0
//    - Outputs: high (f32), low (f32)
//    - Condition logic (read-only, from stage definition)
//
// 3. User can:
//    - Change threshold value
//    - Add/remove output connections
//    - Cannot change output names (defined in stage)

// ============================================================================
// Text ↔ Graph Round-Trip
// ============================================================================

// Text → Graph (Forward):
// 1. Parse: sensor -> demux{threshold: 100} -> {high -> alarm, low -> logger};
// 2. Create nodes: [sensor, demux_1, alarm_1, logger_1]
// 3. Create edges: [sensor→demux_1.value, demux_1.high→alarm_1, demux_1.low→logger_1]
// 4. Layout: Auto-position nodes or use saved positions

// Graph → Text (Reverse):
// 1. Detect multi-output node with multiple edges from different ports
// 2. Group edges by source port
// 3. Generate: stage{} -> {port1 -> target1, port2 -> target2};
// 4. For single outputs, omit named routing syntax

// Graph → Text preservation:
// Original:
sensor -> demux{threshold: 100} -> {;
    high -> alarm{},;
    low -> logger{};
}

// After round-trip, should preserve structure
// NOT convert to:
sensor -> demux{threshold: 100} -> high -> alarm{};
sensor -> demux{threshold: 100} -> low -> logger{};

// ============================================================================
// Recommendation: Named Output Syntax Required
// ============================================================================

// For conditional routing to work well in visual editor:
// - Stages MUST be able to declare multiple named outputs
// - Graph MUST show output ports visually
// - Text syntax MUST support named output routing
// - Round-trip MUST preserve named routing structure

// Without named outputs:
// - Can't visually show which output is which
// - Can't map condition to output port
// - Graph becomes ambiguous
// - Round-trip loses semantic information

// Therefore: Named output routing is ESSENTIAL for visual editor
