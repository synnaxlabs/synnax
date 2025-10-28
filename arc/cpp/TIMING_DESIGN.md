# Arc Runtime Timing and Interval Execution Design

## Overview

This document describes the design for interval-based scheduling and hybrid reactive/interval execution in the Arc runtime. Intervals are **not yet implemented** in either Go or C++ but are planned as a standard library feature.

---

## Interval Functions: Function Instances, Not Node Metadata

### Key Concept

**Intervals are functions** that get instantiated into nodes with concrete configuration:

- **Function Definition**: `interval` (stdlib function template)
  - Config parameter: `period: TimeSpan`
  - Output parameter: `output: U8` (tick signal)

- **Node Instance**: `interval_0` (concrete instantiation)
  - Type: `"interval"` (references the function)
  - Config values: `period: 100000000` (100ms in nanoseconds)

### IR Structure

**Function Definition** (in `IR.functions`):
```json
{
  "key": "interval",
  "config": {
    "keys": ["period"],
    "values": {
      "period": {"kind": "TimeSpan"}
    }
  },
  "outputs": {
    "keys": ["output"],
    "values": {
      "output": {"kind": "U8"}
    }
  }
}
```

**Node Instances** (in `IR.nodes`):
```json
[
  {
    "key": "interval_0",
    "type": "interval",
    "config_values": {
      "period": 100000000
    },
    "channels": {
      "write": {"output": 1}
    }
  },
  {
    "key": "interval_1",
    "type": "interval",
    "config_values": {
      "period": 250000000
    },
    "channels": {
      "write": {"output": 2}
    }
  }
]
```

**Strata Placement**:
```json
{
  "strata": [
    ["interval_0", "interval_1"],
    ["downstream_processor"]
  ]
}
```

**Critical**: Interval nodes **must be in stratum 0** because:
- They don't depend on upstream data
- They need to check time on every cycle
- Stratum 0 always executes

---

## Hybrid Execution Model

### Two Execution Modes

The Arc runtime supports **two independent trigger mechanisms** that can coexist:

#### 1. Reactive (Data-Driven)
- Triggered by channel data arrival via `input_queue`
- `State::process_input_queue()` detects new data
- Executes nodes when inputs change
- Similar to current core server behavior

#### 2. Interval (Time-Driven)
- Triggered by time wheel ticks
- `interval{}` nodes check if their period elapsed
- Emit tick signals to downstream nodes
- Enables periodic execution without data dependency

### Trigger Logic

**Either trigger causes full graph execution**:
```cpp
bool should_execute = has_new_channel_data() || time_wheel.has_ready_intervals();

if (should_execute) {
    scheduler->next(has_new_data, check_intervals);
}
```

### Optimization Strategy

**Smart detection** - only create infrastructure when needed:

```
No interval{} blocks in IR → no time wheel, pure reactive
No reactive channel reads  → no queue watching, pure interval
Both present               → execute on either trigger
```

**Example:**
```arc
// Pure reactive - no time wheel needed
func process{} (input f64) {output f64} {
    output = input * 2
}

// Pure interval - no reactive queue watching
func ticker{} () {} {
    interval{100ms} -> log{}
}

// Hybrid - both mechanisms active
func pid_controller{} (setpoint f64) {control f64} {
    interval{10ms} -> compute{setpoint} -> control
}
```

---

## Time Wheel Design

### GCD-Based Tick Calculation

**Problem**: Multiple intervals with different periods need efficient scheduling.

**Solution**: Calculate Greatest Common Divisor (GCD) of all periods as base tick rate.

**Algorithm**:
```cpp
uint64_t calculate_base_period(
    const std::vector<uint64_t>& periods,
    uint64_t min_period_ns = 10'000'000  // 10ms minimum
) {
    if (periods.empty()) return min_period_ns;

    // Calculate GCD of all periods
    uint64_t gcd = periods[0];
    for (size_t i = 1; i < periods.size(); i++) {
        gcd = std::gcd(gcd, periods[i]);
    }

    // Clamp to minimum (prevent sub-millisecond ticks)
    return std::max(gcd, min_period_ns);
}
```

**Example**:
```
Intervals: [100ms, 250ms, 1s]
GCD: 50ms (base tick period)
Scheduler runs at 50ms rate:
  - 100ms interval fires every 2 ticks
  - 250ms interval fires every 5 ticks
  - 1s interval fires every 20 ticks
```

### Time Wheel Structure

**RT-Safe Fixed-Size Design**:
```cpp
class TimeWheel {
    uint64_t base_period_ns_;     // GCD result
    uint64_t current_tick_ = 0;   // Global tick counter

    struct IntervalEntry {
        std::string node_id;      // Which node to execute
        uint64_t period_ns;       // Original period from config
        uint64_t period_ticks;    // period_ns / base_period_ns
        uint64_t last_tick = 0;   // Last tick this node fired
    };

    // Pre-allocated (RT-safe)
    static constexpr size_t MAX_INTERVALS = 256;
    std::array<IntervalEntry, MAX_INTERVALS> entries_;
    size_t num_entries_ = 0;

public:
    void register_interval(std::string node_id, uint64_t period_ns);

    std::vector<std::string> check_ready_nodes();

    void advance_tick();
};
```

### Tick Alignment Check

**Efficient O(1) per interval**:
```cpp
std::vector<std::string> TimeWheel::check_ready_nodes() {
    std::vector<std::string> ready;
    for (size_t i = 0; i < num_entries_; i++) {
        auto& entry = entries_[i];
        uint64_t elapsed_ticks = current_tick_ - entry.last_tick;

        if (elapsed_ticks >= entry.period_ticks) {
            ready.push_back(entry.node_id);
            entry.last_tick = current_tick_;  // Reset
        }
    }
    return ready;
}
```

**Complexity**: O(num_intervals) per tick, constant time per interval.

---

## Scheduler Integration

### Modified Execution Flow

**Current (Pure Reactive)**:
```cpp
xerrors::Error Scheduler::next() {
    state_->process_input_queue();           // Step 1: Ingest data
    execute_stratum_0();                     // Step 2: Always execute
    execute_higher_strata_if_changed();      // Step 3: Reactive
    changed_.clear();                        // Step 4: Reset
}
```

**Future (Hybrid)**:
```cpp
xerrors::Error Scheduler::next(bool has_new_data, bool check_intervals) {
    // 1. Process data if available
    if (has_new_data) {
        state_->process_input_queue();
    }

    // 2. Check which interval nodes are ready
    std::unordered_set<std::string> ready_intervals;
    if (check_intervals && time_wheel_) {
        auto ready = time_wheel_->check_ready_nodes();
        ready_intervals.insert(ready.begin(), ready.end());
        time_wheel_->advance_tick();
    }

    // 3. Execute stratum 0 with filtering
    for (const auto& node_id : strata_[0]) {
        bool is_interval = interval_nodes_.count(node_id) > 0;
        bool should_execute = false;

        if (is_interval) {
            // Interval nodes: only if tick ready
            should_execute = ready_intervals.count(node_id) > 0;
        } else {
            // Reactive nodes: only if data available
            should_execute = has_new_data && has_input_data(node_id);
        }

        if (should_execute) {
            execute(node_id);
            mark_downstream_changed(node_id);
        }
    }

    // 4. Execute higher strata (unchanged - reactive only)
    for (size_t i = 1; i < strata_.size(); i++) {
        for (const auto& node_id : strata_[i]) {
            if (changed_.count(node_id) > 0) {
                execute(node_id);
            }
        }
    }

    // 5. Clear changed set
    changed_.clear();

    return xerrors::NIL;
}
```

### Key Logic: Stratum 0 Filtering

**Reactive nodes** (normal functions):
- Execute only when `has_new_data && refresh_inputs()` succeeds
- Driven by channel updates from I/O thread

**Interval nodes** (`type == "interval"`):
- Execute only when `ready_intervals` contains their node_id
- Driven by time wheel tick alignment

**Higher strata** (stratum > 0):
- Execute only when marked as "changed" by upstream
- Works for both reactive and interval-triggered execution

---

## Interval Node Implementation

### Stdlib Function Behavior

**What interval nodes do**:
```cpp
class IntervalNode : public Node {
    std::string id_;
    State* state_;
    ChannelKey output_ch_;  // Output channel for tick signal

public:
    xerrors::Error execute() override {
        // Simply emit a tick (u8 value of 1)
        uint8_t tick = 1;

        // Write to output with current timestamp
        auto err = state_->write_channel(output_ch_, tick);

        // Downstream nodes receive this via their refresh_inputs()
        return err;
    }
};
```

**TimeWheel decides WHEN** to execute, node just emits tick.

### Downstream Consumption

**Example Arc program**:
```arc
func controller{} (setpoint f64) {control f64} {
    interval{10ms} -> compute{setpoint} -> control
}
```

**IR representation**:
```
interval_0 (type=interval, period=10ms) → compute_0 (type=compute)
```

**Execution flow**:
1. TimeWheel: "interval_0 ready at tick 10"
2. Scheduler: Execute interval_0.execute()
3. interval_0: Writes tick to channel 1
4. Scheduler: Marks compute_0 as changed
5. compute_0.refresh_inputs(): Reads from interval_0.output
6. compute_0: Processes tick + setpoint → produces control output

---

## Module Loader Changes

### Interval Detection

**During module loading**:
```cpp
std::pair<AssembledRuntime, xerrors::Error>
ModuleLoader::load(const Module& module) {
    // ... existing initialization ...

    // NEW: Scan for interval nodes
    std::vector<uint64_t> interval_periods;
    std::map<std::string, uint64_t> interval_configs;

    for (const auto& node : module.ir.nodes) {
        if (node.type == "interval") {
            // Extract period from config_values
            auto period_it = node.config_values.find("period");
            if (period_it != node.config_values.end()) {
                uint64_t period_ns = period_it->second.get<uint64_t>();
                interval_periods.push_back(period_ns);
                interval_configs[node.key] = period_ns;
            }
        }
    }

    // NEW: Create time wheel if intervals exist
    std::unique_ptr<TimeWheel> time_wheel;
    if (!interval_periods.empty()) {
        uint64_t base_period = TimeWheel::calculate_base_period(
            interval_periods,
            10'000'000  // 10ms minimum
        );
        time_wheel = std::make_unique<TimeWheel>(base_period);

        for (const auto& [node_id, period] : interval_configs) {
            time_wheel->register_interval(node_id, period);
        }
    }

    // ... create scheduler ...
    if (time_wheel) {
        scheduler->set_time_wheel(std::move(time_wheel));
    }

    // ... continue node registration ...
}
```

### Node Creation Dispatch

**Create different node types based on function**:
```cpp
for (const auto& ir_node : module.ir.nodes) {
    if (ir_node.type == "interval") {
        // Create IntervalNode
        auto output_ch = extract_write_channel(ir_node);
        auto interval_node = std::make_unique<IntervalNode>(
            ir_node.key, state.get(), output_ch
        );
        scheduler->register_node(ir_node.key, std::move(interval_node), stratum);
        scheduler->mark_as_interval_node(ir_node.key);

    } else {
        // Create WASMNode (existing code)
        auto edges = state->incoming_edges(ir_node.key);
        auto outputs = build_output_handles(ir_node);
        auto node_state = std::make_unique<NodeState>(state, ir_node.key, edges, outputs);
        auto wasm_node = std::make_unique<WASMNode>(...);
        scheduler->register_node(ir_node.key, std::move(wasm_node), stratum);
    }
}
```

---

## Driver Integration Pattern (Future)

### RT Thread Loop

**The driver will call scheduler in a loop**:
```cpp
// driver/arc/read_task.cpp (future)
void ArcTask::run_rt_thread() {
    while (!should_stop()) {
        bool has_data = !runtime->input_queue->empty();
        bool should_tick = time_wheel_exists && time_elapsed();

        if (has_data || should_tick) {
            runtime->scheduler->next(has_data, should_tick);
        }

        // Sleep until next event
        wait_for_next_event();
    }
}
```

### Timing Strategy Options

**Option 1: Periodic Tick (Simple)**
```cpp
void run_rt_thread() {
    uint64_t tick = 0;
    const uint64_t base_period_ns = time_wheel->base_period_ns();

    while (!should_stop()) {
        // Process any available data
        bool has_data = !runtime->input_queue->empty();

        // Always tick at base period
        runtime->scheduler->next(has_data, true);

        // Sleep for base period
        nanosleep(base_period_ns);
        tick++;
    }
}
```

**Pros**: Simple, deterministic
**Cons**: Wastes cycles if no intervals

**Option 2: Event-Driven (Optimal)**
```cpp
void run_rt_thread() {
    while (!should_stop()) {
        bool has_data = !runtime->input_queue->empty();
        bool should_tick = false;

        if (time_wheel) {
            auto now = TimeStamp::now();
            should_tick = (now - last_tick) >= base_period;
        }

        if (has_data || should_tick) {
            runtime->scheduler->next(has_data, should_tick);
            if (should_tick) last_tick = now;
        }

        // Sleep until: min(next_tick, data_arrival)
        uint64_t sleep_ns = calculate_sleep_time();
        nanosleep(sleep_ns);
    }
}
```

**Pros**: Efficient, no wasted cycles
**Cons**: More complex timing logic

**Option 3: Condition Variable (Hybrid)**
```cpp
void run_rt_thread() {
    std::condition_variable cv;

    while (!should_stop()) {
        bool has_data = !runtime->input_queue->empty();
        bool should_tick = time_wheel && check_tick_ready();

        if (has_data || should_tick) {
            runtime->scheduler->next(has_data, should_tick);
        } else {
            // Wait for: data arrival OR timeout
            cv.wait_for(lock, base_period);
        }
    }
}
```

**Pros**: Responsive to both triggers
**Cons**: Requires thread synchronization (may impact RT-safety)

---

## Time Wheel Implementation Details

### Structure

```cpp
class TimeWheel {
    uint64_t base_period_ns_;     // GCD of all intervals
    uint64_t current_tick_ = 0;   // Global tick counter

    struct IntervalEntry {
        std::string node_id;      // Node to execute
        uint64_t period_ns;       // Original period (e.g., 100ms)
        uint64_t period_ticks;    // Normalized (e.g., 2 for 100ms / 50ms)
        uint64_t last_tick = 0;   // Last tick this fired
    };

    static constexpr size_t MAX_INTERVALS = 256;
    std::array<IntervalEntry, MAX_INTERVALS> entries_;
    size_t num_entries_ = 0;

public:
    TimeWheel(uint64_t base_period_ns) : base_period_ns_(base_period_ns) {}

    void register_interval(std::string node_id, uint64_t period_ns) {
        if (num_entries_ >= MAX_INTERVALS) return;  // Full

        entries_[num_entries_] = IntervalEntry{
            .node_id = std::move(node_id),
            .period_ns = period_ns,
            .period_ticks = period_ns / base_period_ns_,
            .last_tick = 0
        };
        num_entries_++;
    }

    std::vector<std::string> check_ready_nodes() {
        std::vector<std::string> ready;

        for (size_t i = 0; i < num_entries_; i++) {
            auto& entry = entries_[i];
            uint64_t elapsed = current_tick_ - entry.last_tick;

            if (elapsed >= entry.period_ticks) {
                ready.push_back(entry.node_id);
                entry.last_tick = current_tick_;
            }
        }

        return ready;
    }

    void advance_tick() {
        current_tick_++;
    }

    uint64_t base_period_ns() const { return base_period_ns_; }
    uint64_t current_tick() const { return current_tick_; }
};
```

### RT-Safety Characteristics

**Pre-allocated**:
- Fixed-size array of intervals (MAX_INTERVALS = 256)
- No dynamic allocation during `check_ready_nodes()`
- Result vector may allocate, but bounded by MAX_INTERVALS

**Deterministic**:
- O(num_intervals) per tick
- Each check is O(1) (arithmetic comparison)
- No hash lookups, no tree traversals

**Performance**:
- 256 intervals @ 1kHz = ~256 comparisons per ms = negligible
- Tick increment is atomic (single write)

---

## Minimum Period Clamping

### Rationale

**Problem**: If intervals are [1ms, 3ms, 7ms], GCD = 1ms → 1kHz tick rate

**Solutions**:

**Hybrid Approach** (Recommended):
```cpp
uint64_t base = calculate_gcd(periods);  // e.g., 1ms
uint64_t clamped = std::max(base, MIN_PERIOD);  // e.g., max(1ms, 10ms) = 10ms

if (clamped > base) {
    // Round intervals up to nearest multiple
    for (auto& period : periods) {
        period = ceil_to_multiple(period, clamped);
    }
}
```

**Example**:
```
Input: [1ms, 3ms, 7ms]
GCD: 1ms
Minimum: 10ms
Result: Round to [10ms, 10ms, 10ms] or warn user
```

**Configurable minimum**:
```cpp
static constexpr uint64_t DEFAULT_MIN_PERIOD = 10'000'000;  // 10ms
static constexpr uint64_t AGGRESSIVE_MIN = 1'000'000;        // 1ms
```

---

## Execution Examples

### Example 1: Pure Interval

**Arc Program**:
```arc
func logger{} () {} {
    interval{1s} -> log_status{}
}
```

**Execution**:
```
Tick 0:   interval_0 fires → writes to channel → downstream processes
Tick 1-9: No execution (scheduler next() returns immediately)
Tick 10:  interval_0 fires again
...
```

**No reactive queue watching** - optimization skips `process_input_queue()`.

### Example 2: Pure Reactive

**Arc Program**:
```arc
func processor{} (input f64) {output f64} {
    output = input * 2
}
```

**Execution**:
```
Data arrives → scheduler->next(true, false)
               ↓
           process_input_queue()
               ↓
           execute reactive nodes
               ↓
           propagate changes
```

**No time wheel created** - `time_wheel_ == nullptr`.

### Example 3: Hybrid

**Arc Program**:
```arc
func pid_controller{} (setpoint f64) {control f64} {
    interval{10ms} -> pid{setpoint, sensor} -> control
}

func sensor_reader{} () {sensor f64} {
    // Reads from external channel reactively
    sensor = channel_read_f64(sensor_ch)
}
```

**Execution**:
```
Every 10ms tick:
  1. Interval fires → writes tick
  2. If sensor data arrived → process_input_queue()
  3. PID node has TWO inputs (interval tick + sensor data)
  4. refresh_inputs() aligns both to trigger timestamp
  5. PID executes with aligned inputs
  6. Control output written

Between ticks (if sensor updates):
  1. process_input_queue() → sensor data available
  2. Interval not ready → skip interval_0
  3. Downstream may or may not execute (depends on refresh_inputs)
```

---

## Data Flow: Interval → Downstream

**Scenario**: `interval{100ms} -> compute{}`

**Step-by-step**:
```
1. TimeWheel: current_tick=10, period_ticks=2
   → elapsed = 10 - 8 = 2 >= 2 → READY

2. Scheduler: Execute interval_0
   → interval_0.execute()
   → write_channel(output_ch, 1)  // Tick signal

3. RT Thread: Queue scalar to I/O thread
   → output_queue.push({output_ch, 1, now()})

4. I/O Thread: Consumes output
   → Writes to Synnax channel
   → Synnax broadcasts to subscribers
   → Subscribers write back to Arc via input_queue

5. Next cycle: compute_0.refresh_inputs()
   → Reads from interval_0 output (via edge)
   → Aligns timestamp
   → Executes computation

6. Scheduler: Marks downstream changed
   → Higher strata execute if connected
```

**Key insight**: Interval nodes produce **timestamped data** just like any other node. Temporal alignment handles the rest.

---

## Performance Characteristics

### Time Wheel Overhead

**Per-tick cost**:
```
num_intervals × (1 subtraction + 1 comparison + 1 branch)
```

**Example**: 100 intervals @ 10kHz (100μs period)
- 100 × 3 ops = ~300 ops
- Modern CPU: ~0.3 cycles per op = 90 cycles
- @ 3GHz: 90 / 3000 = **0.03μs per tick**

**Negligible overhead** even at high rates.

### Scheduler Overhead

**Added cost per cycle**:
```
1. Check time_wheel existence (1 branch)
2. check_ready_nodes() if exists (~0.03μs for 100 intervals)
3. advance_tick() (1 increment)
4. Stratum 0 filtering (1 hash lookup per node)
```

**Total**: ~**0.1-0.5μs** additional latency per cycle for 100 intervals.

**Acceptable** for 1kHz-10kHz target rates (1ms - 100μs budgets).

### Memory Overhead

**Per interval**:
```cpp
sizeof(IntervalEntry) = sizeof(string) + 3×uint64_t + padding
                      ≈ 32 + 24 = 56 bytes
```

**256 intervals**: 56 × 256 = **14KB** (negligible)

---

## Open Design Questions

### 1. Interval Offset/Phase

**Should intervals support phase offsets?**
```json
{
  "type": "interval",
  "config_values": {
    "period": 100000000,
    "offset": 25000000  // Start 25ms after t=0
  }
}
```

**Use case**: Stagger multiple intervals to balance load.

**Implementation**: `if ((current_tick - offset_ticks) % period_ticks == 0)`

### 2. Drift Correction

**Issue**: Accumulated timing error over long runs.

**Solution options**:
- Absolute timestamps: `last_tick_time` instead of `last_tick`
- Periodic resync with monotonic clock
- Drift tolerance parameter

### 3. Overrun Handling

**What if execution takes longer than base period?**
```
Tick 0:  Interval fires, processing takes 15ms
Tick 1:  (Should be at 10ms) Already late!
```

**Options**:
- **Skip**: Don't fire again until caught up
- **Burst**: Fire multiple times to catch up
- **Error**: Report overrun and stop

**Go pattern**: TBD (not implemented yet)

### 4. Dynamic Reconfiguration

**Can intervals change period at runtime?**

**Likely answer**: No (requires recompiling IR)
- Simpler implementation
- Predictable behavior
- Matches Arc's static compilation model

---

## Summary

### What Makes Intervals Work

1. **Interval nodes** are stdlib function instances with `period` config
2. **Time wheel** calculates GCD-based tick rate from all periods
3. **Scheduler** filters stratum 0 execution by node type:
   - Reactive: Execute if `has_data`
   - Interval: Execute if `ready_intervals` contains node_id
4. **Interval nodes** emit timestamped ticks to output channels/edges
5. **Downstream nodes** consume ticks via normal `refresh_inputs()` mechanism

### Key Design Principles

- **Orthogonal triggers**: Data and time are independent execution drivers
- **Unified scheduling**: Both use same stratified reactive model
- **Temporal alignment**: Works seamlessly with interval-generated data
- **Zero special cases**: Interval outputs look like any other node output
- **RT-safe**: Time wheel uses fixed-size structures, bounded operations
- **Optimized**: Only created when needed (no intervals → no overhead)

### Implementation Roadmap

1. ✅ **State architecture** - Complete (matches Go)
2. ⏱️ **Time wheel** - Design complete, needs implementation
3. ⏱️ **Interval node** - Simple implementation (emit tick)
4. ⏱️ **Scheduler filtering** - Modify stratum 0 execution
5. ⏱️ **Module loader** - Detect intervals, build time wheel
6. ⏱️ **Driver integration** - RT thread loop with timing
7. ⏱️ **Testing** - Validate GCD, alignment, hybrid execution
8. ⏱️ **Performance** - Benchmark overhead, tune parameters

The foundation is now **correct and ready** for interval implementation.
