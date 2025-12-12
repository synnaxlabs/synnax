# Arc Sequence IR Refactoring Plan

## Problem Summary

The current Arc IR has parallel structures for sequences that don't fit the dataflow model:
- `Sequence`, `Stage`, `Transition`, `TopLevelTransition` types duplicate the node/edge graph
- `Stage.Flows` and `Stage.Transitions` are separate collections instead of edge types
- `IsTerminal` flag is redundant (derivable from topology)
- Imperative blocks stored as function key instead of being proper graph nodes

## Design Option Evaluation

### Option A: Tag Nodes with Stage/Sequence Fields
```go
type Node struct {
    Key      string
    Type     string
    Config   map[string]any
    Stage    string  // NEW
    Sequence string  // NEW
}
```
**Pros:** Minimal change, direct stage lookup per node
**Cons:** Denormalized (stage/sequence info duplicated on every node), harder to query "all nodes in stage X"

### Option B: Stage Groups with Edge Kind (Recommended)
```go
type EdgeKind int
const (
    EdgeReactive   EdgeKind = iota  // -> continuous flow
    EdgeTransition                   // => one-shot transition
)

type Edge struct {
    Source Handle
    Target Handle
    Kind   EdgeKind  // NEW
}

type IR struct {
    // ... existing ...
    StageNodes map[string][]string  // NEW: stage key -> node keys
}
```
**Pros:**
- Clean separation: edges describe semantics, StageNodes groups membership
- Single source of truth for edge semantics
- Easy to query nodes per stage
- Minimal changes to existing Node type

**Cons:** Two places to look (edges for flow, StageNodes for grouping)

### Option C: Separate Activation Graph
Two parallel graphs: computation graph + activation graph
**Pros:** Clean separation of concerns
**Cons:** Over-engineered, harder to reason about, runtime must coordinate two graphs

### Option D: Stages as Special Nodes
```go
type Node struct {
    Key    string
    Type   string  // includes "stage" type
    Config map[string]any
}
// Stage activation becomes edges between stage nodes
```
**Pros:** Fully unified model
**Cons:** Stages are conceptually different from computation nodes, muddies the abstraction

## Recommendation: Option B

Option B best balances simplicity with correctness:
1. Edge semantics belong on edges (`Kind` field)
2. Stage membership is a grouping concern, not a node property
3. Removes redundant `IsTerminal` (check for outgoing transition edges)
4. Removes special `TopLevelTransition` type (just edges with transition kind)
5. Imperative blocks become regular nodes with multiple outgoing transition edges

---

## Implementation Plan

### Phase 1: IR Type Changes (`arc/go/ir/`)

#### 1.1 Add EdgeKind to Edge type
**File:** `arc/go/ir/edge.go`
```go
type EdgeKind int

const (
    EdgeReactive   EdgeKind = iota  // -> operator
    EdgeTransition                   // => operator
)

type Edge struct {
    Source Handle  `json:"source"`
    Target Handle  `json:"target"`
    Kind   EdgeKind `json:"kind"`
}
```

#### 1.2 Simplify Sequence/Stage types
**File:** `arc/go/ir/sequence.go`

Remove:
- `Stage.Flows []string` - use edges instead
- `Stage.Transitions []Transition` - use edges instead
- `Stage.Imperative string` - becomes a regular node
- `Stage.IsTerminal bool` - derive from edge topology
- `Transition` type - edges with EdgeTransition kind
- `TopLevelTransition` type - edges with EdgeTransition kind
- `TopLevelTransitions` type

Keep (simplified):
```go
type Sequence struct {
    Key    string   `json:"key"`
    Stages []string `json:"stages"`  // Ordered for 'next' resolution
    Entry  string   `json:"entry"`
}

type Stage struct {
    Key      string `json:"key"`
    Sequence string `json:"sequence"`
}
```

#### 1.3 Add StageNodes to IR
**File:** `arc/go/ir/ir.go`
```go
type IR struct {
    Functions  Functions
    Sequences  Sequences
    Stages     Stages
    Nodes      Nodes
    Edges      Edges
    Strata     Strata
    Symbols    *symbol.Scope
    TypeMap    map[antlr.ParserRuleContext]types.Type
    StageNodes map[string][]string  // NEW: stage key -> node keys
}
```

Remove from IR:
- `EntryPoints TopLevelTransitions`

### Phase 2: Analyzer Updates (`arc/go/analyzer/`)

#### 2.1 Update edge creation to include Kind
**File:** `arc/go/analyzer/analyzer.go` (and flow analysis)

When creating edges:
- `->` operator creates `Edge{..., Kind: EdgeReactive}`
- `=>` operator creates `Edge{..., Kind: EdgeTransition}`

#### 2.2 Update sequence analyzer
**File:** `arc/go/analyzer/sequence/sequence.go`

Changes:
- Remove `buildTransition()` - create edges directly
- Remove `TopLevelTransition` handling - create edges
- Track which nodes belong to which stage during analysis
- Build `StageNodes` map during IR construction

#### 2.3 Handle imperative blocks as nodes
- Imperative block `{ code } => match { a => X, b => Y }` becomes:
  - Anonymous function node (the block)
  - Multiple outgoing `EdgeTransition` edges with conditions

#### 2.4 Derive terminal status
- A stage is terminal if it has no outgoing `EdgeTransition` edges
- Query at runtime, not stored

---

## Files to Modify

| File | Change |
|------|--------|
| `arc/go/ir/edge.go` | Add `EdgeKind` enum and `Kind` field |
| `arc/go/ir/sequence.go` | Simplify `Stage`, remove `Transition`, `TopLevelTransition` |
| `arc/go/ir/ir.go` | Add `StageNodes`, remove `EntryPoints` |
| `arc/go/analyzer/sequence/sequence.go` | Update to produce new IR format |
| `arc/go/analyzer/analyzer.go` | Update edge creation |
| C++ runtime | Parse new IR format, implement activation model |

## Open Questions

1. **Match block conditions**: How to represent `match { a => X, b => Y }` edges?
   - Option: Edge has optional `Condition` value to match against source output

2. **Transition priority**: When multiple transitions could fire, which wins?
   - Proposal: Source order (first transition in source wins)

3. **Cross-sequence transitions**: `stage_a => other_sequence`
   - Creates edge targeting entry stage of other sequence

---

## Deep Analysis: Runtime Execution Model with Option B

### Current Runtime Architecture

The C++ runtime (`/arc/cpp/runtime/`) has these key components:

1. **Tick-based scheduler** (`scheduler/scheduler.h`) - Executes nodes in strata order
2. **Change propagation** - Only executes nodes whose inputs changed
3. **State management** (`state/state.h`) - Data flow between nodes
4. **WASM execution** (`wasm/node.h`) - Calls compiled user functions
5. **Event loop** (`loop/loop.h`) - Platform-specific timing

### Current Execution Flow (per tick)

```cpp
// From runtime.h:74-98
void run() {
    this->loop->wait(this->breaker);           // 1. Wait for timing interval
    while (this->inputs->try_pop(frame))       // 2. Ingest new data
        this->state->ingest(frame);

    this->scheduler->next(elapsed);            // 3. Execute scheduler tick

    if (auto writes = this->state->flush_writes(); !writes.empty())
        this->outputs->push(std::move(out_frame));  // 4. Output results

    this->state->clear_reads();                // 5. Cleanup
}
```

### Scheduler Change Propagation (current)

```cpp
// From scheduler.h:59-73
void next(const telem::TimeSpan elapsed) {
    bool first = true;
    for (const auto& stratum: this->strata.strata) {
        for (const auto& node_key: stratum)
            if (first || this->changed.contains(node_key)) {
                this->current_state->node->next(this->ctx);
            }
        first = false;
    }
    this->changed.clear();
}
```

**Key insight**: Stratum 0 (sources) always executes; other strata only execute if inputs changed.

---

### Option B Runtime Design

#### 1. New Runtime State

```cpp
class Runtime {
    // Existing...
    std::unique_ptr<scheduler::Scheduler> scheduler;

    // NEW for stages
    std::string active_sequence;                      // Current sequence
    std::string active_stage;                         // Current stage in sequence
    std::unordered_map<std::string, std::vector<std::string>> stage_nodes;  // stage -> nodes
    std::unordered_set<std::string> transition_edges; // Edges with kind=TRANSITION

    // Transition state tracking (for one-shot semantics)
    struct TransitionState {
        bool was_true_last_tick;
        bool has_fired_this_stage;
    };
    std::unordered_map<std::string, TransitionState> transition_states;
};
```

#### 2. Modified Scheduler with Stage Filtering

```cpp
void Scheduler::next(telem::TimeSpan elapsed) {
    this->ctx.elapsed = elapsed;
    bool first = true;

    for (const auto& stratum: this->strata.strata) {
        for (const auto& node_key: stratum) {
            // NEW: Filter by active stage
            if (!is_node_in_active_stage(node_key)) continue;

            if (first || this->changed.contains(node_key)) {
                this->current_state = &this->nodes[node_key];
                this->current_state->node->next(this->ctx);
            }
        }
        first = false;
    }
    this->changed.clear();
}

bool is_node_in_active_stage(const std::string& node_key) {
    // Nodes with no stage are always active (top-level)
    auto it = node_to_stage.find(node_key);
    if (it == node_to_stage.end()) return true;
    return it->second == active_stage;
}
```

#### 3. Transition Evaluation (after scheduler tick)

```cpp
void Runtime::evaluate_transitions() {
    // Get transition edges from active stage
    auto& transitions = get_stage_transition_edges(active_stage);

    // Evaluate in priority order (source order)
    for (const auto& edge : transitions) {
        if (should_fire_transition(edge)) {
            execute_transition(edge);
            break;  // First-wins semantics
        }
    }
}

bool should_fire_transition(const Edge& edge) {
    // Get condition node output
    auto* condition_output = state->get_node_output(edge.source.node, edge.source.param);
    bool is_true_now = is_truthy(condition_output);

    auto& ts = transition_states[edge_key(edge)];

    // One-shot: fire on rising edge, only once per stage entry
    bool rising_edge = is_true_now && !ts.was_true_last_tick;
    bool not_fired_yet = !ts.has_fired_this_stage;

    ts.was_true_last_tick = is_true_now;

    if (rising_edge && not_fired_yet) {
        ts.has_fired_this_stage = true;
        return true;
    }
    return false;
}
```

#### 4. Stage Transition Execution

```cpp
void Runtime::execute_transition(const Edge& edge) {
    std::string target = edge.target.node;  // Target stage or sequence

    if (is_stage_in_current_sequence(target)) {
        // Intra-sequence transition
        transition_to_stage(target);
    } else if (is_sequence(target)) {
        // Cross-sequence transition
        transition_to_sequence(target);
    }
}

void Runtime::transition_to_stage(const std::string& new_stage) {
    // 1. Deactivate current stage nodes (they'll stop executing)
    // No explicit cleanup needed - scheduler just won't run them

    // 2. Reset timers in new stage
    for (const auto& node_key : stage_nodes[new_stage]) {
        if (auto* timer = get_timer_node(node_key)) {
            timer->reset();  // Reset last_fired to allow immediate trigger
        }
    }

    // 3. Reset transition states for new stage
    for (auto& [edge_key, ts] : transition_states) {
        if (edge_in_stage(edge_key, new_stage)) {
            ts.has_fired_this_stage = false;
            ts.was_true_last_tick = false;
        }
    }

    // 4. Activate new stage
    active_stage = new_stage;
}

void Runtime::transition_to_sequence(const std::string& new_sequence) {
    // 1. Clear all stateful variables for current sequence
    for (auto& [key, val] : bindings->state_vars) {
        if (belongs_to_sequence(key, active_sequence)) {
            val.clear();
        }
    }

    // 2. Reset all timers in current sequence
    // (handled by scheduler filtering - they just stop)

    // 3. Switch to new sequence's entry stage
    active_sequence = new_sequence;
    active_stage = sequences[new_sequence].entry;

    // 4. Reset all transition states for new sequence
    reset_transition_states_for_sequence(new_sequence);
}
```

#### 5. Modified Change Propagation

```cpp
void Scheduler::mark_changed(const std::string& param) {
    for (const auto& edge : current_state->output_edges) {
        if (edge.source.param != param) continue;

        // NEW: Only propagate to active stage nodes
        if (edge.kind == EdgeKind::REACTIVE) {
            if (is_node_in_active_stage(edge.target.node)) {
                this->changed.insert(edge.target.node);
            }
        }
        // Transition edges are handled separately in evaluate_transitions()
    }
}
```

#### 6. Complete Tick Cycle with Stages

```cpp
void Runtime::tick() {
    // 1. Wait for timing interval
    loop->wait(breaker);

    // 2. Ingest external data
    telem::Frame frame;
    while (inputs->try_pop(frame)) {
        state->ingest(frame);
    }

    // 3. Execute reactive flows (only active stage nodes)
    scheduler->next(elapsed);

    // 4. Evaluate transitions (NEW)
    evaluate_transitions();

    // 5. Output results
    if (auto writes = state->flush_writes(); !writes.empty()) {
        outputs->push(std::move(writes));
    }

    // 6. Cleanup
    state->clear_reads();
}
```

---

### Edge Cases and Semantics

#### Terminal Stages
A stage with no outgoing `EdgeTransition` edges is terminal. Detection:
```cpp
bool is_terminal_stage(const std::string& stage) {
    for (const auto& edge : edges) {
        if (edge.kind == EdgeTransition &&
            node_to_stage[edge.source.node] == stage) {
            return false;
        }
    }
    return true;
}
```

#### Top-Level Entry Points
`start_cmd => main` becomes an edge with `kind=EdgeTransition` where:
- Source: `start_cmd` channel node
- Target: Entry stage of `main` sequence

The runtime checks top-level transition edges before sequence is active:
```cpp
void Runtime::check_entry_points() {
    if (active_sequence.empty()) {
        for (const auto& edge : entry_point_edges) {
            if (should_fire_transition(edge)) {
                transition_to_sequence(get_sequence_from_stage(edge.target.node));
                break;
            }
        }
    }
}
```

#### Imperative Blocks with Match
```arc
{
    // code
} => match {
    result1 => stage_a,
    result2 => stage_b
}
```

Becomes:
- Anonymous function node (executes the block)
- Multiple `EdgeTransition` edges, each with a condition comparing output to match value
- First matching edge fires

```cpp
// During transition evaluation
for (const auto& edge : match_edges_from_imperative_block) {
    auto output = get_node_output(imperative_node, "result");
    if (output == edge.match_value && should_fire_transition(edge)) {
        execute_transition(edge);
        break;
    }
}
```

#### `wait{}` and `interval{}` Timers

**wait{5s}**: One-shot timer, fires once after duration
```cpp
class Wait : public node::Node {
    telem::TimeSpan duration;
    telem::TimeSpan started_at = -1;
    bool fired = false;

    xerrors::Error next(node::Context& ctx) override {
        if (started_at < 0) started_at = ctx.elapsed;

        if (!fired && ctx.elapsed - started_at >= duration) {
            fired = true;
            ctx.mark_changed("out");
            // Output "true"
        }
        return xerrors::NIL;
    }

    void reset() override {
        started_at = -1;
        fired = false;
    }
};
```

**interval{100ms}**: Periodic timer, fires repeatedly
```cpp
class Interval : public node::Node {
    telem::TimeSpan interval;
    telem::TimeSpan last_fired = -1;

    xerrors::Error next(node::Context& ctx) override {
        if (ctx.elapsed - last_fired >= interval) {
            last_fired = ctx.elapsed;
            ctx.mark_changed("out");
        }
        return xerrors::NIL;
    }

    void reset() override {
        last_fired = -1;  // Allows immediate first fire
    }
};
```

---

### Performance Analysis

#### Memory Overhead
| Structure | Size (typical) |
|-----------|----------------|
| `active_stage` | 40 bytes |
| `stage_nodes` map | 1-2 KB |
| `transition_edges` set | 500 bytes |
| `transition_states` map | 1-2 KB |
| **Total** | **~5 KB per runtime** |

#### CPU Overhead per Tick
| Operation | Cost |
|-----------|------|
| Stage membership check | O(1) hash lookup per node |
| Transition evaluation | O(T) where T = transitions in stage (~5-20) |
| Change propagation filter | O(1) additional check per edge |
| **Total** | **< 5% overhead** |

#### Optimization: Pre-compute Stage Data
```cpp
void Runtime::initialize_stage_data() {
    // Build stage -> nodes mapping
    for (const auto& [node_key, node] : nodes) {
        if (!node.stage.empty()) {
            stage_nodes[node.stage].push_back(node_key);
        }
    }

    // Build stage -> transition edges mapping
    for (const auto& edge : edges) {
        if (edge.kind == EdgeTransition) {
            auto stage = node_to_stage[edge.source.node];
            stage_transitions[stage].push_back(&edge);
        }
    }

    // Sort transitions by priority
    for (auto& [stage, transitions] : stage_transitions) {
        std::sort(transitions.begin(), transitions.end(),
            [](auto* a, auto* b) { return a->priority < b->priority; });
    }
}
```

---

### IR Changes for C++ Runtime

#### Updated Protobuf Schema
```protobuf
message PBIR {
    repeated PBFunction functions = 1;
    repeated PBNode nodes = 2;
    repeated PBEdge edges = 3;
    repeated PBStratum strata = 4;
    // NEW
    repeated PBSequence sequences = 5;
    repeated PBStage stages = 6;
    map<string, PBNodeList> stage_nodes = 7;
}

message PBEdge {
    PBHandle source = 1;
    PBHandle target = 2;
    EdgeKind kind = 3;  // NEW
}

enum EdgeKind {
    EDGE_REACTIVE = 0;
    EDGE_TRANSITION = 1;
}

message PBSequence {
    string key = 1;
    repeated string stages = 2;
    string entry = 3;
}

message PBStage {
    string key = 1;
    string sequence = 2;
}
```

#### Updated C++ IR Struct
```cpp
// arc/cpp/runtime/ir/ir.h
struct Edge {
    Handle source;
    Handle target;
    EdgeKind kind;  // NEW
};

enum class EdgeKind {
    Reactive = 0,
    Transition = 1
};

struct Sequence {
    std::string key;
    std::vector<std::string> stages;
    std::string entry;
};

struct Stage {
    std::string key;
    std::string sequence;
};

struct IR {
    std::vector<Function> functions;
    std::vector<Node> nodes;
    std::vector<Edge> edges;
    Strata strata;
    // NEW
    std::vector<Sequence> sequences;
    std::vector<Stage> stages;
    std::unordered_map<std::string, std::vector<std::string>> stage_nodes;
};
```

---

### Implementation Phases

#### Phase 1: IR Types (Go + Protobuf)
- Add `EdgeKind` to `Edge` type
- Simplify `Stage` (remove `Flows`, `Transitions`, `IsTerminal`)
- Add `StageNodes` map to IR
- Update protobuf schema
- **Estimated: 100 lines changed**

#### Phase 2: Analyzer Updates (Go)
- Create edges with `Kind` field during flow analysis
- Build `StageNodes` map during sequence analysis
- Remove separate `Transition` building
- **Estimated: 200 lines changed**

#### Phase 3: C++ IR Parsing
- Update IR structs to match protobuf
- Update JSON parsing for `EdgeKind`
- Add stage/sequence loading
- **Estimated: 150 lines changed**

#### Phase 4: Scheduler Modification
- Add stage filtering to node execution
- Modify change propagation for stage awareness
- **Estimated: 50 lines changed**

#### Phase 5: Transition Evaluation
- Add `evaluate_transitions()` to tick cycle
- Implement one-shot semantics with state tracking
- **Estimated: 100 lines changed**

#### Phase 6: Stage Lifecycle
- Timer reset on stage entry
- Transition state reset
- Cross-sequence cleanup
- **Estimated: 80 lines changed**

**Total: ~680 lines across Go and C++**

---

## Testing Strategy

1. Unit tests for new IR types (serialization/deserialization)
2. Update existing analyzer tests to verify new edge format
3. Add tests for edge kind propagation
4. Integration test: compile example program, verify IR structure
5. Runtime tests for one-shot transition semantics
6. Timer reset behavior on stage re-entry
