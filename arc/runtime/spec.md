# Arc Runtime Specification

This document specifies the runtime behavior and execution model for Arc programs.

## Table of Contents

1. [Overview](#1-overview)
2. [Core Concepts](#2-core-concepts)
3. [Frame-Based Execution Model](#3-frame-based-execution-model)
4. [Channel State Management](#4-channel-state-management)
5. [Index Channels and Alignment](#5-index-channels-and-alignment)
6. [Stratified Execution](#6-stratified-execution)
7. [Node Activation](#7-node-activation)
8. [WASM Integration](#8-wasm-integration)
9. [Host Functions](#9-host-functions)
10. [State Persistence](#10-state-persistence)
11. [Multi-Output Handling](#11-multi-output-handling)
12. [Error Handling](#12-error-handling)
13. [Performance Considerations](#13-performance-considerations)

---

## 1. Overview

### 1.1 Purpose

[To be filled]

### 1.2 Design Goals

The Arc runtime is designed around six core goals that define its architecture and trade-offs:

#### 1.2.1 Deterministic, Glitch-Free Execution

**Safety-critical requirement**: The runtime must produce identical outputs for identical inputs, with no intermediate glitches during propagation.

A **glitch** is a transient incorrect value that appears when inputs update at different times. For example, if stage C reads from both A and B, and A updates before B, C must not produce an output using new-A with stale-B. Such glitches are correctness violations that can cause unsafe behavior in control systems.

The runtime prevents glitches through **stratified execution**: all stages at stratum N complete before any stage at stratum N+1 begins. This ensures single-pass propagation where each stage sees a consistent snapshot of its inputs.

Determinism enables:
- **Reproducible testing**: Record input frames, replay exactly
- **Distributed consensus**: Multiple runtimes produce identical results
- **Safety validation**: Prove system behavior under specific conditions

#### 1.2.2 Zero-Copy Frame Processing

**Fundamental architectural constraint**: The runtime must process `telem.Frame` structures without copying binary data.

Telemetry data arrives as `telem.Series` backed by binary buffers (`[]byte`). Copying multi-kilobyte buffers per channel per frame would dominate overhead. Instead:
- Frames are passed by reference
- Series data remains in original buffers
- WASM stages read directly from frame memory via host functions
- Output writes create new series without copying input data

Zero-copy is possible because:
1. Series are immutable within a frame
2. WASM isolation prevents unsafe memory access
3. Host functions control all data access

This constraint shapes the entire execution model, including how channels are read and how multi-output is handled.

#### 1.2.3 Frame-Based Batch Execution

**Goal**: Process batches of samples (frames) rather than individual samples.

Frames align with three system boundaries:
1. **I/O**: Hardware produces batches via DMA buffers, UDP packets, or storage reads
2. **Computation**: Amortizes WASM call overhead and cache misses across multiple samples
3. **Network**: Remote frames arrive as complete units

The runtime's responsibility is to **minimize processing latency** once a frame arrives. Buffering decisions (e.g., "wait for 1ms of data") are the caller's responsibility. This keeps the runtime simple: `Next(frame)` processes immediately.

Trade-off: Stages see only the **first element** of each input series (MVP policy). Full series processing is future work.

#### 1.2.4 WASM Isolation for Portability and Safety

**Goal**: Execute user-defined stages in sandboxed WebAssembly for memory safety and cross-platform portability.

WASM provides:
- **Memory safety**: Stages cannot corrupt runtime state or access arbitrary memory
- **Portability**: Compiled stages run on any architecture (x86, ARM, embedded RISC-V)
- **Determinism**: No platform-specific floating-point or undefined behavior

Trade-offs accepted:
- **10-30% performance overhead** vs native code (acceptable for safety)
- **No direct system calls** (stages must use host functions)
- **Limited debugging** compared to native code

Hot-reloading is explicitly a non-goal. Stages are compiled once and execute until the runtime terminates.

#### 1.2.5 Multi-Index Channel Support

**Goal**: Allow stages to operate on channels from different temporal indexes without requiring manual alignment.

In Synnax, channels are grouped by **index channels** (timestamp arrays). Channels sharing an index have coherent alignment; channels from different indexes do not.

The runtime must handle stages that mix inputs from:
- High-rate sensors (1kHz, index A)
- Low-rate sensors (10Hz, index B)
- Computed values (index-independent outputs)

**MVP policy**: Non-blocking reads return the first element of each series, ignoring temporal relationships. This trades correctness (potentially misaligned samples) for simplicity.

**Future work**: Time-based alignment via interpolation, or index-aware type system.

Rationale: Many stages (filters, unit conversions, simple arithmetic) don't require temporal precision. For those that do, we can add explicit alignment later without breaking existing programs.

#### 1.2.6 High-Throughput Telemetry Processing

**Goal**: Process hundreds of channels at rates from Hz to multi-kHz with minimal overhead.

Performance metric: **frames per second** the runtime can process. Target scale: 100+ channels updating at 1kHz, processed in <1ms per frame.

Optimization strategy:
1. Zero-copy (see 1.2.2) eliminates dominant overhead
2. Frame batching amortizes per-activation costs
3. Stratified execution minimizes wasted computation
4. Future: SIMD, parallel stratum execution

Minor performance optimizations are deferred. The architecture must support high throughput, but micro-optimizations come after correctness is proven.

### 1.3 Non-Goals

[To be filled]

---

## 2. Core Concepts

### 2.1 Runtime

The **Runtime** is a stateful execution engine that processes telemetry frames through a compiled Arc program.

**Structure**: A Runtime owns:
- **Graph**: Compiled IR containing stages, nodes, edges, and strata
- **Channel State**: Current series, queues, and subscriber lists for each channel
- **WASM Instances**: Module instances for each node
- **Node Activation State**: Tracking first-activation and input readiness per node
- **Stateful Variables**: Persistent storage for `$=` declarations across frames

**Lifecycle**:
- Created once per Arc program via `NewRuntime(ir.IR)`
- Lives for the duration of the program execution
- Multiple independent Runtimes can exist in the same process (different Arc programs)
- Not thread-safe: caller must serialize `Next()` calls

**API**:
```go
type Runtime struct {
    // Internal state (not exported)
}

func NewRuntime(program ir.IR) (*Runtime, error)
func (r *Runtime) Next(frame telem.Frame[uint32]) error
```

**Invariants**:
- Graph is immutable after construction (no dynamic modification)
- Channel definitions are fixed (new channels cannot be added)
- State persists across `Next()` calls but not across Runtime instances

---

### 2.2 Frame

A **Frame** is a sparse map of channel IDs to telemetry series, representing a batch of samples that arrived together.

**Structure**: `telem.Frame[uint32]` where keys are channel IDs (uint32).

**Properties**:
- **Sparse**: Contains only channels with new data (not all channels)
- **Variable Length**: Series within the frame can have different lengths
- **Type-Erased**: Series data is stored as binary `[]byte` with metadata
- **Zero-Copy**: Frames are passed by reference; runtime never copies series data

**Constraints**:
- Keys (channel IDs) may appear multiple times (e.g., multiple series for same channel)
- Series may be empty (zero-length data)
- No maximum frame size (limited only by memory)
- Frames can be empty (zero keys) - results in no node activations

**Ownership**:
- Runtime does **not** take ownership of the frame
- Caller must ensure frame data remains valid during `Next()` call
- After `Next()` returns, caller may reuse or modify the frame
- Same frame cannot be passed to multiple Runtimes concurrently (data race)

**Invariants Assumed** (caller responsibility):
- Series `DataType` matches channel definitions in the graph
- Series `TimeRange` represents actual temporal bounds of data
- `Alignment` values are valid for the channel's index

---

### 2.3 Series

A **Series** is a strongly-typed array of telemetry samples backed by a binary buffer.

**Structure**: `telem.Series` from `x/go/telem`

```go
type Series struct {
    TimeRange TimeRange  // [Start, End) temporal bounds (inclusive, exclusive)
    DataType  DataType   // float64, int32, timestamp, etc.
    Data      []byte     // Raw binary buffer
    Alignment Alignment  // Position in index channel's domain
}
```

**Immutability**: Series are **immutable** within a frame. The runtime:
- Never modifies `Series.Data`
- Never changes `TimeRange`, `DataType`, or `Alignment`
- Reads data via `telem.ValueAt[T](series, index)` or direct `[]byte` access

**Data Access**:
- **First-element policy (MVP)**: Runtime reads only `ValueAt(series, 0)`
- Future: May read multiple elements for time-based alignment
- Type safety: Caller ensures `DataType` matches Arc type (`ir.F64` ↔ `telem.Float64T`)

**Variable Length**:
- Series in the same frame have independent lengths
- Runtime does not validate length relationships
- First-element policy makes length irrelevant for MVP

---

### 2.4 Channel

A **Channel** is a uniquely-identified stream of telemetry data with an associated type.

**Channel Definition** (compile-time):
```go
type ChannelDef struct {
    ID      uint32    // Unique channel identifier
    Type    ir.Type   // Arc type (ir.F64, ir.Chan{ValueType: ir.I32}, etc.)
    IndexID string    // Index channel key (future use for alignment)
}
```

**Channel State** (runtime):
```go
type ChannelState struct {
    ID            uint32
    Type          ir.Type

    // Current series - what non-blocking reads see
    CurrentSeries telem.Series

    // FIFO queue for blocking reads
    Queue         []telem.Series

    // Metadata
    LastFrame     int64      // Frame number when last updated
    HasData       bool       // Ever received data?

    // Subscribers (nodes that read this channel)
    Subscribers   []string   // Node keys
}
```

**External vs Internal Channels**:
- **External (input)**: Appear in incoming frames, represent hardware/data sources
- **Internal (computed)**: Created by stage outputs, written by runtime
- Runtime treats both identically (no special handling)

**Virtual Channels**: All channels must be **pre-declared** in the graph. Stages cannot dynamically create new channel IDs. Output channels are defined when edges are created during graph construction.

---

### 2.5 Node

A **Node** is a specific invocation of a stage with concrete configuration values.

**Relationship**:
- **Stage**: Reusable template (like a function definition) - `stage filter { alpha f64 } ...`
- **Node**: Specific instance with config (like a function call) - `filter{alpha=0.95}`

**Node Structure** (`ir.Node`):
```go
type Node struct {
    Key      string            // Unique identifier (e.g., "filter_1", "filter_2")
    Type     string            // Stage key (which template to use)
    Config   map[string]any    // Concrete config values
    Channels Channels          // Read/write channel ID sets
}
```

**Node Identity**: Nodes are uniquely identified by their `Key` (string). Keys are generated during graph construction (typically auto-generated but can be user-provided).

**Node State** (runtime):
```go
type NodeState struct {
    Key            string

    // Activation tracking
    FirstActivated bool               // Has first activation completed?
    RequiredInputs set.Set[uint32]    // Channel IDs needed for first activation
    EverReceived   set.Set[uint32]    // Which channels have ever had data

    // WASM
    Instance       wazero.Module      // WASM module instance

    // Stateful variables (keyed by variable name)
    State          map[string]any
}
```

**Configuration**:
- Config values are **immutable** after node creation
- Config params are passed to WASM as function arguments (before runtime params)
- Missing config values cause compilation errors (not runtime errors)

---

### 2.6 Stratum

A **Stratum** is an execution level assigned to each node based on its dependencies.

**Definition**: If node B depends on node A (A → B), then `stratum(B) = stratum(A) + 1`.

**Purpose**: Stratification enables **glitch-free single-pass execution**:
- All nodes at stratum N complete before any node at stratum N+1 begins
- Each node sees a consistent snapshot of its inputs (no partial updates)
- Guarantees deterministic execution order

**Properties**:
- **No cycles**: Graph must be acyclic (stratification would fail)
- **No ordering within stratum**: Nodes at same stratum can execute in any order (or in parallel)
- **All N complete before N+1**: Strict barrier between strata

**Computation**:
- Performed during graph analysis (compile-time)
- Algorithm: Iterative deepening via topological sort
- See `stratifier` package for implementation

**Special Cases**:
- **Source nodes** (no inputs): Always stratum 0
- **Sink nodes** (no outputs): Stratum depends on inputs (not necessarily max stratum)
- **Stateful nodes**: State does **not** create implicit edges to self
  - State changes do not trigger re-execution
  - State persists across frames but does not affect stratification
  - Example: `stage integrator` with `sum $= 0.0` is stratum N (based on inputs), not N+1

**Storage** (`ir.Strata`):
```go
type Strata struct {
    Nodes map[string]int  // nodeKey → stratum level
    Max   int             // Highest stratum in graph
}
```

**Runtime Usage**:
```go
// Execute frame by processing strata in order
for stratum := 0; stratum <= ir.Strata.Max; stratum++ {
    for _, nodeKey := range affectedNodes {
        if ir.Strata.Get(nodeKey) == stratum {
            executeNode(nodeKey)
        }
    }
}
```

---

## 3. Frame-Based Execution Model

### 3.1 Overview

The runtime processes telemetry in discrete batches called **frames**. Each frame triggers a complete execution cycle: ingestion → stratified execution → garbage collection.

**Key Principle**: All samples in a series are processed before moving to the next stratum. This ensures deterministic, glitch-free propagation while maintaining high throughput.

---

### 3.2 Edges vs Channels

The runtime distinguishes between **edges** (node-to-node connections) and **channels** (external I/O).

**Edges**: Connect node outputs directly to downstream node inputs within the graph
```arc
filter{} -> controller{}  // Edge: filter's output → controller's input
```
- Data flows **within the frame** (stratum to stratum)
- No persistence across frames
- No water marks needed (consumed immediately)
- Buffered in memory between strata

**Channels**: External data sources (sensors) and sinks (actuators)
```arc
sensor_channel -> filter{}   // Channel input
controller{} -> actuator_channel  // Channel output
```
- Data flows **into/out of the runtime** (external system boundary)
- Persists across frames (SeriesQueue)
- Requires water marks (track consumption position)
- Subject to garbage collection

**Critical Distinction**: Water marks only apply to **channels**, not edges.

---

### 3.3 High Water Mark Model (Channels Only)

The runtime tracks **per-node, per-channel** consumption positions using `telem.Alignment` as high water marks.

**Purpose**: Each node independently tracks where it left off reading from **external channels**. This allows:
- Multiple nodes to consume the same channel at different rates
- Nodes to "catch up" with buffered channel data
- Real-time operation (GC old series when all nodes have consumed them)

**Structure**:
```go
type NodeState struct {
    Key              string
    ChannelWaterMarks  map[uint32]telem.Alignment  // channelID → last consumed alignment
    // ...
}
```

**Semantics**:
- Water mark represents the **last alignment consumed** from this **channel**
- On next read, node processes samples with `alignment > waterMark`
- Water marks persist across frames
- **Not used for edge inputs** (edges consumed immediately within frame)

---

### 3.4 Channel State: Series Queue + Latest Sample

Channels maintain a **queue of unconsumed series** plus the **latest sample value** for snapshot reads.

```go
type ChannelState struct {
    ID           uint32
    Type         ir.Type

    // Series queue: retained until all subscribers consume
    SeriesQueue  []telem.Series

    // Latest sample: always available for reads (never GC'd)
    LatestSample interface{}  // Typed value (float64, int32, etc.)

    // Subscribers (node keys that read this channel)
    Subscribers  []string
}
```

**Series Queue**:
- New series are **appended** on frame arrival
- Series are **removed** (GC'd) when all subscribers' water marks exceed the series' upper bound
- Enables nodes to process at different rates without losing data

**Latest Sample**:
- Updated whenever any series is processed (last sample becomes latest)
- Used for **snapshot reads** when channel doesn't update in a frame
- Persists indefinitely (never GC'd)

**Example**:
```
Frame 1: sensor → Series{Alignment: (0,0), Data: [1.0, 2.0, 3.0, 4.0]}
  - SeriesQueue = [Series(0,0)]
  - LatestSample = 4.0
  - Node A processes all 4 samples, water mark → (0,3)
  - Node B processes 2 samples, water mark → (0,1)
  - GC: Node B hasn't finished, keep Series(0,0)

Frame 2: sensor → Series{Alignment: (0,4), Data: [5.0, 6.0]}
  - SeriesQueue = [Series(0,0), Series(0,4)]  (append, don't replace)
  - LatestSample = 6.0
  - Node A: water mark (0,3), processes (0,4), (0,5) → water mark (0,5)
  - Node B: water mark (0,1), processes (0,2), (0,3) from Series(0,0),
            then (0,4), (0,5) from Series(0,4) → water mark (0,5)
  - GC: Both nodes finished Series(0,0), remove it
  - SeriesQueue = [Series(0,4)]

Frame 3: pressure updates, sensor doesn't
  - SeriesQueue = [Series(0,4)]  (no new data)
  - LatestSample = 6.0  (still 6.0!)
  - Node C reads sensor → gets 6.0 (snapshot read using LatestSample)
```

---

### 3.5 Edge Buffers

Edges buffer output values between strata for immediate consumption (no persistence).

```go
type EdgeBuffer struct {
    SourceNode   string         // Node key that produced these values
    SourceOutput string         // Which output name
    TargetNode   string         // Node key that will consume
    TargetParam  string         // Which parameter name
    Values       []interface{}  // Sample values for this frame
}
```

**Lifecycle**:
```
Stratum N execution:
  - Node produces outputs → write to EdgeBuffers

Stratum N+1 execution:
  - Node reads from EdgeBuffers as inputs
  - EdgeBuffers cleared after all downstream nodes consume

Frame complete:
  - All EdgeBuffers discarded (no persistence across frames)
```

**Example**:
```arc
sensor -> filter{} -> controller{}
```

```
Frame: sensor → [1, 2, 3, 4]

Stratum 0 (filter):
  - Reads [1, 2, 3, 4] from sensor channel (uses water mark)
  - Produces [0.9, 1.8, 2.7, 3.6]
  - Writes to EdgeBuffer{source: filter, output: "output", target: controller, values: [0.9, 1.8, 2.7, 3.6]}

Stratum 1 (controller):
  - Reads [0.9, 1.8, 2.7, 3.6] from EdgeBuffer (no water mark!)
  - Executes 4 times
  - Produces outputs

End of frame:
  - EdgeBuffer cleared (does not persist to next frame)
```

---

### 3.6 Frame Processing Pipeline

Each call to `Runtime.Next(frame)` executes these steps:

#### Step 1: Frame Ingestion

```go
func (r *Runtime) Next(frame telem.Frame[uint32]) error {
    // 1. Update channel state with new series
    affectedNodes := r.ingestFrame(frame)

    // 2. Execute affected nodes in stratified order
    err := r.executeStrata(affectedNodes)

    // 3. Garbage collect consumed series
    r.gcSeries()

    return err
}
```

**Frame Ingestion**:
```go
func (r *Runtime) ingestFrame(frame telem.Frame[uint32]) set.Set[string] {
    affectedNodes := set.New[string]()

    for channelID, series := range frame.Entries() {
        state := r.channelState[channelID]

        // Append to series queue
        state.SeriesQueue = append(state.SeriesQueue, series)

        // Update latest sample (last sample in series)
        if series.Len() > 0 {
            state.LatestSample = telem.ValueAt(series, -1)  // Last element
        }

        // Mark all subscribers as affected
        for _, nodeKey := range state.Subscribers {
            affectedNodes.Add(nodeKey)
        }
    }

    return affectedNodes
}
```

**Key Points**:
- New series are **appended**, not replaced (queue semantics)
- LatestSample is updated to the **last sample** in the new series
- All nodes that subscribe to updated channels become "affected"

---

#### Step 2: Stratified Execution

Execute affected nodes stratum-by-stratum, processing **all samples** at each stratum before moving to the next.

```go
func (r *Runtime) executeStrata(affectedNodes set.Set[string]) error {
    for stratum := 0; stratum <= r.graph.Strata.Max; stratum++ {
        // Get all nodes at this stratum
        stratumNodes := []string{}
        for nodeKey := range affectedNodes {
            if r.graph.Strata.Get(nodeKey) == stratum {
                stratumNodes = append(stratumNodes, nodeKey)
            }
        }

        // Execute each node (processes all samples)
        for _, nodeKey := range stratumNodes {
            if err := r.executeNode(nodeKey); err != nil {
                return err
            }
        }
    }
    return nil
}
```

**Batching Per Stratum**:
- All nodes at stratum N complete **all their sample processing** before stratum N+1 begins
- Within a stratum, nodes can execute in any order (or in parallel)
- This preserves glitch-free guarantees while processing multiple samples

---

#### Step 3: Node Execution (Sample Loop)

Each node processes **all unconsumed samples** from its input channels:

```go
func (r *Runtime) executeNode(nodeKey string) error {
    node := r.graph.GetNode(nodeKey)
    nodeState := r.nodeState[nodeKey]

    // 1. Check first-activation (need all inputs?)
    if !r.canActivate(nodeKey) {
        return nil  // Not ready yet
    }

    // 2. Collect samples to process from all inputs
    samples := r.collectSamples(node, nodeState)

    // 3. Execute once per sample
    outputSamples := []Sample{}
    for _, sample := range samples {
        output, err := r.executeWASM(node, nodeState, sample)
        if err != nil {
            return err
        }
        outputSamples = append(outputSamples, output)
    }

    // 4. Batch outputs into series and write to channels
    return r.writeOutputs(node, outputSamples)
}
```

---

### 3.7 Node Instantiation and Multi-Input

**Single Input** (each `->` creates a separate node instance):
```arc
sensor_a -> filter{}  // Creates filter_1 (reads sensor_a)
sensor_b -> filter{}  // Creates filter_2 (reads sensor_b) - different node!
```

**Multi-Input via Routing Table** (creates ONE node with multiple inputs):
```arc
{
    setpoint: setpoint_channel,
    measured: filter{}
} -> controller{}  // Creates controller_1 with 2 inputs
```

Only routing table syntax creates multi-input nodes that require alignment.

---

### 3.8 Sample Collection (Multi-Input Alignment)

When a node created via **routing table** has multiple inputs with **different lengths**, align samples using **max length + LatestSample repeat**:

**Applies to**:
- Channel inputs (different series lengths)
- Edge inputs (different output lengths from upstream nodes)
- Mixed channel + edge inputs

```go
func (r *Runtime) collectSamples(node *ir.Node, nodeState *NodeState) []map[string]interface{} {
    inputStreams := make(map[string][]Sample)  // paramName → samples
    maxLength := 0

    // 1. Collect samples from CHANNEL inputs (use water marks)
    for paramName, channelID := range node.ChannelInputs() {
        channelState := r.channelState[channelID]
        waterMark := nodeState.ChannelWaterMarks[channelID]

        samples := []Sample{}
        for _, series := range channelState.SeriesQueue {
            // Process samples where alignment > waterMark
            for i := int64(0); i < series.Len(); i++ {
                alignment := series.Alignment.AddSamples(uint32(i))
                if alignment > waterMark {
                    value := telem.ValueAt(series, int(i))
                    samples = append(samples, Sample{
                        Alignment: alignment,
                        Value:     value,
                    })
                }
            }
        }

        inputStreams[paramName] = samples
        if len(samples) > maxLength {
            maxLength = len(samples)
        }
    }

    // 2. Collect samples from EDGE inputs (no water marks)
    for paramName, edgeBuffer := range node.EdgeInputs() {
        samples := []Sample{}
        for _, value := range edgeBuffer.Values {
            samples = append(samples, Sample{Value: value})
        }

        inputStreams[paramName] = samples
        if len(samples) > maxLength {
            maxLength = len(samples)
        }
    }

    // 3. Align to max length (repeat LatestSample for short inputs)
    alignedSamples := []map[string]interface{}{}
    for i := 0; i < maxLength; i++ {
        sample := make(map[string]interface{})
        for paramName, samples := range inputStreams {
            if i < len(samples) {
                sample[paramName] = samples[i].Value
            } else {
                // Use LatestSample when input runs out
                // For channel inputs: use channel's LatestSample
                // For edge inputs: use last value in edge buffer
                sample[paramName] = r.getLatestSample(node, paramName)
            }
        }
        alignedSamples = append(alignedSamples, sample)
    }

    return alignedSamples
}
```

**Example (Channel + Edge inputs)**:
```arc
{
    setpoint: setpoint_channel,  // Channel input
    measured: filter{}            // Edge input
} -> controller{}
```

```
Frame: setpoint_channel → [10, 20, 30] (3 samples)
Stratum 0: filter produces [5, 6, 7, 8] (4 samples) → EdgeBuffer

Stratum 1: controller collects samples
  Channel input (setpoint): [10, 20, 30]  (length 3)
  Edge input (measured): [5, 6, 7, 8]     (length 4)

Aligned samples (max length = 4):
  Sample 0: {setpoint: 10, measured: 5}
  Sample 1: {setpoint: 20, measured: 6}
  Sample 2: {setpoint: 30, measured: 7}
  Sample 3: {setpoint: 30, measured: 8}  (repeat LatestSample for setpoint)

Controller executes 4 times.
```

---

### 3.9 Stateful Variables Within Frame

Stateful variables (`$=`) **accumulate across sample executions** within the same frame:

```arc
stage integrator {} (input f64) f64 {
    sum $= 0.0
    sum = sum + input
    return sum
}
```

```
Frame: input → [1.0, 2.0, 3.0]

Execution 1:
  - Load state: sum = 0.0 (from previous frame)
  - Compute: sum = 0.0 + 1.0 = 1.0
  - Store state: sum = 1.0
  - Output: 1.0

Execution 2:
  - Load state: sum = 1.0 (from execution 1!)
  - Compute: sum = 1.0 + 2.0 = 3.0
  - Store state: sum = 3.0
  - Output: 3.0

Execution 3:
  - Load state: sum = 3.0 (from execution 2!)
  - Compute: sum = 3.0 + 3.0 = 6.0
  - Store state: sum = 6.0
  - Output: 6.0

State at end of frame: sum = 6.0 (persists to next frame)
Output series: [1.0, 3.0, 6.0]
```

**Implementation**: State is loaded before the first execution, then cached in memory. Subsequent executions within the frame see the accumulated state. State is persisted after the last execution.

---

### 3.10 Output Handling

Outputs are written to **edge buffers** (for downstream nodes) or **channels** (for external sinks).

#### Named Output Routing

Multi-output stages with routing create separate edges:

```arc
alarm_detector{} -> {
    low_alarm: low_handler{},
    high_alarm: high_handler{},
    normal: logger{}
}
```

Creates three edges:
- `alarm_detector.low_alarm` → `low_handler`
- `alarm_detector.high_alarm` → `high_handler`
- `alarm_detector.normal` → `logger`

**Sparse Output Semantics**: If `alarm_detector` processes 4 samples but only sets:
- `low_alarm` twice (samples 0, 2)
- `high_alarm` once (sample 1)
- `normal` zero times

Then:
- `low_handler` receives **2 samples** in its edge buffer
- `high_handler` receives **1 sample** in its edge buffer
- `logger` receives **0 samples** (doesn't execute this frame)

---

### 3.11 Output to Channels

When a node writes to a channel (external output), create a series and append to the channel's queue:

```go
func (r *Runtime) writeOutputs(node *ir.Node, outputSamples []Sample) error {
    for outputName, channelID := range node.OutputChannels() {
        // Extract values for this output
        values := []interface{}{}
        for _, sample := range outputSamples {
            values = append(values, sample.Outputs[outputName])
        }

        // Create series
        series := telem.NewSeries(
            node.OutputType(outputName),
            values,
            // TimeRange and Alignment derived from input
        )

        // Write to channel (appends to SeriesQueue)
        r.writeChannel(channelID, series)
    }
    return nil
}
```

**Efficiency**: Creating a single series with N samples is much cheaper than N individual writes.

---

### 3.12 Garbage Collection

After stratified execution completes, remove series that all subscribers have consumed:

```go
func (r *Runtime) gcSeries() {
    for _, channelState := range r.channelState {
        // Find minimum water mark across all subscribers
        minWaterMark := telem.MaxAlignment
        for _, nodeKey := range channelState.Subscribers {
            nodeState := r.nodeState[nodeKey]
            waterMark := nodeState.InputWaterMarks[channelState.ID]
            if waterMark < minWaterMark {
                minWaterMark = waterMark
            }
        }

        // Remove series fully consumed by all subscribers
        newQueue := []telem.Series{}
        for _, series := range channelState.SeriesQueue {
            if series.AlignmentBounds().Upper > minWaterMark {
                newQueue = append(newQueue, series)  // Keep it
            }
            // else: discard (all subscribers past this series)
        }
        channelState.SeriesQueue = newQueue
    }
}
```

**Key**: LatestSample is **never** GC'd, ensuring snapshot reads always work.

---

### 3.13 Sparse Frame Handling

Frames are **sparse** - only updated channels appear:

```
Frame 1: {sensor_a, sensor_b, sensor_c}  → 3 channels updated
Frame 2: {sensor_a}                       → 1 channel updated
Frame 3: {}                               → 0 channels (empty frame)
```

**Empty Frame Behavior**:
- No channels updated → `affectedNodes = ∅`
- No nodes execute → `Next()` returns immediately
- State and water marks unchanged
- LatestSample values remain available for future reads

**Partial Updates**:
- Only nodes subscribing to updated channels execute
- Other nodes remain idle (don't re-execute with old data)

---

### 3.14 Complete Execution Example

Putting it all together:

```arc
// Graph
sensor_channel -> filter{alpha=0.9} -> {
    high: high_handler{},
    low: low_handler{}
} -> actuator_channel
```

**Frame arrives**: `sensor_channel → Series{Alignment: (0,0), Data: [10, 20, 30, 40]}`

**Ingestion**:
- Append series to `sensor_channel.SeriesQueue`
- Update `sensor_channel.LatestSample = 40`
- Mark `filter` as affected

**Stratum 0 (filter)**:
- Collect samples from `sensor_channel`:
  - Water mark = 0 (or previous value)
  - Process samples with alignment > water mark
  - Collected: [10, 20, 30, 40] (4 samples)
- Execute 4 times:
  - Sample 0: input=10 → output=9.0, classify as `low`
  - Sample 1: input=20 → output=18.0, classify as `high`
  - Sample 2: input=30 → output=27.0, classify as `high`
  - Sample 3: input=40 → output=36.0, classify as `high`
- Write to edge buffers:
  - `EdgeBuffer{source: filter, output: low, target: low_handler, values: [9.0]}`
  - `EdgeBuffer{source: filter, output: high, target: high_handler, values: [18.0, 27.0, 36.0]}`
- Update water mark: `filter.ChannelWaterMarks[sensor_channel] = (0,3)`

**Stratum 1 (handlers)**:
- `low_handler`:
  - Reads edge buffer: [9.0] (1 sample)
  - Executes 1 time
  - Writes to `actuator_channel`
- `high_handler`:
  - Reads edge buffer: [18.0, 27.0, 36.0] (3 samples)
  - Executes 3 times
  - Writes to `actuator_channel`

**Garbage Collection**:
- `sensor_channel.SeriesQueue`: Check if all subscribers consumed series(0,0)
  - `filter` water mark = (0,3) ≥ series upper bound (0,4) ✓
  - Remove series from queue
- Edge buffers cleared (not persisted)

**Result**: `actuator_channel` receives 4 total samples from handlers

---

## 4. Channel State Management

### 4.1 Channel State Structure

Each channel maintains runtime state that persists across frames, tracking unconsumed series and the latest observed value.

**Full Structure**:
```go
type ChannelState struct {
    // Identity (immutable)
    ID           uint32      // Channel identifier (matches IR definition)
    Type         ir.Type     // Arc type (ir.F64, ir.Chan{ValueType: ir.I32}, etc.)
    IndexID      string      // Index channel key (for alignment, future use)

    // Data queue (mutable)
    SeriesQueue  []telem.Series   // Unconsumed series, oldest first
    LatestSample interface{}      // Last observed value (typed: float64, int32, etc.)

    // Metadata (mutable)
    HasData      bool        // Has this channel ever received data?
    LastUpdated  int64       // Frame number when last series was added

    // Dependency tracking (immutable after initialization)
    Subscribers  []string    // Node keys that read from this channel
    Writers      []string    // Node keys that write to this channel (output nodes)
}
```

**Initialization** (during `NewRuntime`):
- `SeriesQueue` = empty slice
- `LatestSample` = zero value for type (0.0 for float, 0 for int, etc.)
- `HasData` = false
- `Subscribers` = populated from graph analysis (nodes with channel as input)
- `Writers` = populated from graph analysis (nodes with channel as output)

**Lifecycle**:
- Created once per channel when Runtime is constructed
- State persists for Runtime lifetime
- SeriesQueue grows (frame ingestion) and shrinks (GC)
- LatestSample updated on every ingestion, never reset

---

### 4.2 Series Queue Semantics

The **SeriesQueue** is a FIFO buffer of unconsumed series that allows subscribers to process at different rates.

#### 4.2.1 Append on Ingestion

When a frame arrives with data for this channel, **append** the new series:

```go
func (r *Runtime) ingestFrame(frame telem.Frame[uint32]) set.Set[string] {
    for channelID, series := range frame.Entries() {
        state := r.channelState[channelID]

        // Append (don't replace!)
        state.SeriesQueue = append(state.SeriesQueue, series)

        // Update metadata
        state.HasData = true
        state.LastUpdated = r.currentFrameNumber

        // Extract last sample
        if series.Len() > 0 {
            state.LatestSample = telem.ValueAt(series, series.Len()-1)
        }
    }
}
```

**Key Property**: Never replace - always append. This ensures nodes can independently consume at their own pace.

#### 4.2.2 Consumption via Water Marks

Each subscriber node tracks its position using `ChannelWaterMarks[channelID]`. When collecting samples:

```go
func (r *Runtime) collectSamplesFromChannel(channelID uint32, nodeState *NodeState) []Sample {
    channelState := r.channelState[channelID]
    waterMark := nodeState.ChannelWaterMarks[channelID]  // Last consumed alignment

    samples := []Sample{}
    for _, series := range channelState.SeriesQueue {
        // Process samples with alignment > waterMark
        for i := int64(0); i < series.Len(); i++ {
            alignment := series.Alignment.AddSamples(uint32(i))
            if alignment > waterMark {
                samples = append(samples, Sample{
                    Alignment: alignment,
                    Value:     telem.ValueAt(series, int(i)),
                })
            }
        }
    }

    // Update water mark to last consumed
    if len(samples) > 0 {
        nodeState.ChannelWaterMarks[channelID] = samples[len(samples)-1].Alignment
    }

    return samples
}
```

**Guarantees**:
- Each sample is processed **exactly once** per subscriber
- Different subscribers can be at different positions in the queue
- Alignment ordering ensures deterministic consumption

#### 4.2.3 Garbage Collection

After each frame execution completes, remove series that **all subscribers** have consumed:

```go
func (r *Runtime) gcSeries() {
    for channelID, channelState := range r.channelState {
        if len(channelState.Subscribers) == 0 {
            continue  // No subscribers, keep all (or clear all - policy decision)
        }

        // Find minimum water mark across all subscribers
        minWaterMark := telem.MaxAlignment
        for _, nodeKey := range channelState.Subscribers {
            waterMark := r.nodeState[nodeKey].ChannelWaterMarks[channelID]
            if waterMark < minWaterMark {
                minWaterMark = waterMark
            }
        }

        // Keep only series with samples beyond minWaterMark
        newQueue := []telem.Series{}
        for _, series := range channelState.SeriesQueue {
            // Check if series has unconsumed samples
            if series.Alignment.AddSamples(uint32(series.Len()-1)) > minWaterMark {
                newQueue = append(newQueue, series)
            }
            // else: all samples consumed by all subscribers, discard
        }

        channelState.SeriesQueue = newQueue
    }
}
```

**Policy**:
- Conservative: Keep series until **all** subscribers finish it
- Prevents data loss for slow subscribers
- Bounded memory only if subscribers keep up

#### 4.2.4 LatestSample Retention

The **LatestSample** field provides snapshot reads for nodes that execute when a channel doesn't update:

```go
// Example: controller needs setpoint value even if setpoint_channel didn't update this frame
func (r *Runtime) getChannelValue(channelID uint32, nodeState *NodeState) interface{} {
    channelState := r.channelState[channelID]

    // Try to get unconsumed samples first
    samples := r.collectSamplesFromChannel(channelID, nodeState)
    if len(samples) > 0 {
        return samples[0].Value  // Use first unconsumed sample
    }

    // Fall back to LatestSample (snapshot read)
    if channelState.HasData {
        return channelState.LatestSample
    }

    // Channel never received data - use zero value
    return r.zeroValueForType(channelState.Type)
}
```

**Properties**:
- Updated on every frame ingestion (last sample in new series)
- **Never GC'd** (persists indefinitely)
- Enables consistent reads for multi-input alignment
- Solves "stale data" problem for low-rate channels

**Example**:
```
Frame 1: sensor → [10, 20, 30]
  LatestSample = 30

Frame 2-10: sensor doesn't update (other channels do)
  LatestSample = 30 (still!)

Frame 11: controller reads sensor (no new data in queue)
  Returns: 30 (from LatestSample)
```

---

### 4.3 Blocking Queue

**Decision**: Blocking reads (`<-channel` syntax) are **not implemented** in the MVP runtime.

**Rationale**:
1. Frame-based execution makes blocking semantics unclear (what does "wait for data" mean when frame already arrived?)
2. Adds significant complexity to stratified execution (nodes can't block mid-stratum)
3. Not required for primary use cases (telemetry processing, control loops)

**Future Consideration**: If blocking reads become necessary:
- Option A: Implement via separate blocking queue per channel (FIFO, separate from SeriesQueue)
- Option B: Use async/await patterns with frame suspension (requires runtime state machine)
- Option C: Restrict blocking reads to specific node types (e.g., "async stages" in separate stratum)

**Current Behavior**: All channel reads are **non-blocking** and use the LatestSample fallback mechanism.

**Grammar Note**: The parser still accepts `<-` syntax, but the analyzer/compiler can reject it or treat it as non-blocking.

---

### 4.4 Subscriber Tracking

Subscriber tracking enables:
1. **Dependency analysis**: Which nodes activate when a channel updates?
2. **Garbage collection**: When is a series safe to discard?
3. **Optimization**: Skip work if no subscribers exist

#### 4.4.1 Subscriber Discovery

Subscribers are identified during **graph construction** by analyzing node inputs:

```go
func (r *Runtime) discoverSubscribers() {
    for _, node := range r.graph.Nodes {
        for paramName, channelID := range node.ChannelInputs() {
            // This node reads from channelID
            channelState := r.channelState[channelID]
            channelState.Subscribers = append(channelState.Subscribers, node.Key)
        }
    }
}
```

**Storage**:
```go
type ChannelState struct {
    Subscribers []string  // Node keys (e.g., ["filter_1", "logger_2"])
}
```

**Immutability**: Subscriber lists are **fixed** after initialization (graph structure doesn't change at runtime).

#### 4.4.2 Activation Propagation

When a frame updates a channel, activate all subscribers:

```go
func (r *Runtime) ingestFrame(frame telem.Frame[uint32]) set.Set[string] {
    affectedNodes := set.New[string]()

    for channelID, series := range frame.Entries() {
        state := r.channelState[channelID]

        // ... update SeriesQueue, LatestSample ...

        // Mark all subscribers as affected
        for _, nodeKey := range state.Subscribers {
            affectedNodes.Add(nodeKey)
        }
    }

    return affectedNodes
}
```

**Transitive Activation**: If node A's output is an edge to node B, and A is affected, B should also be affected. This happens naturally through stratified execution - A produces output, B sees it in next stratum.

#### 4.4.3 Writer Tracking

Writers are discovered similarly:

```go
func (r *Runtime) discoverWriters() {
    for _, node := range r.graph.Nodes {
        for outputName, channelID := range node.OutputChannels() {
            channelState := r.channelState[channelID]
            channelState.Writers = append(channelState.Writers, node.Key)
        }
    }
}
```

**Use Cases**:
- **Validation**: Detect multiple writers to same channel (potential conflict)
- **Debugging**: Trace data provenance ("who wrote this value?")
- **Optimization**: Skip GC for channels with no writers (external input only)

**Policy on Multiple Writers**:
- Currently **allowed** - series from different nodes are merged in SeriesQueue (order depends on stratum execution)
- Future: May add "exclusive writer" constraint for certain channels

#### 4.4.4 No-Subscriber Channels

If a channel has **zero subscribers**, it's an **output-only** (sink) channel:

```arc
controller{} -> actuator_channel  // actuator_channel has no subscribers
```

**Behavior**:
- Series are still appended to SeriesQueue
- LatestSample still updated
- **GC Policy Decision**:
  - Option A: Immediately discard after ingestion (no one will read)
  - Option B: Keep 1 series for debugging/observability
  - Option C: Keep all (user may query externally)

**Recommendation**: Option B (keep latest series only) balances memory and observability.

---

**Summary**: Channel state management uses a **queue + snapshot** model where:
- SeriesQueue buffers unconsumed data for independent subscriber rates
- LatestSample provides consistent fallback for multi-input alignment
- Water marks enable precise per-subscriber consumption tracking
- GC removes series when all subscribers have processed them

---

## 5. Index Channels and Alignment

### 5.1 Index Channel Concept

[To be filled]

### 5.2 Alignment Encoding

[To be filled]

### 5.3 Cross-Index Operations

[To be filled]

### 5.4 Alignment Policies

#### 5.4.1 First-Element Semantics (Default)

[To be filled]

#### 5.4.2 Time-Based Alignment (Future)

[To be filled]

#### 5.4.3 Index-Aware Stages (Future)

[To be filled]

---

## 6. Stratified Execution

### 6.1 Stratification Algorithm

[To be filled]

### 6.2 Execution Order

[To be filled]

### 6.3 Within-Stratum Ordering

[To be filled]

### 6.4 Cross-Frame Propagation

[To be filled]

---

## 7. Node Activation

### 7.1 Activation Conditions

[To be filled]

### 7.2 First Activation (All Inputs)

[To be filled]

### 7.3 Subsequent Activations (Any Input)

[To be filled]

### 7.4 Activation State Tracking

[To be filled]

---

## 8. WASM Integration

### 8.1 Module Management

[To be filled]

### 8.2 Instance Creation

[To be filled]

### 8.3 Function Invocation

[To be filled]

### 8.4 Memory Layout

[To be filled]

---

## 9. Host Functions

### 9.1 Channel Operations

#### 9.1.1 Non-Blocking Read

[To be filled]

#### 9.1.2 Blocking Read

[To be filled]

#### 9.1.3 Channel Write

[To be filled]

### 9.2 State Operations

#### 9.2.1 State Load

[To be filled]

#### 9.2.2 State Store

[To be filled]

### 9.3 Built-in Functions

#### 9.3.1 now()

[To be filled]

#### 9.3.2 len()

[To be filled]

---

## 10. State Persistence

### 10.1 Stateful Variables

[To be filled]

### 10.2 State Storage

[To be filled]

### 10.3 State Lifecycle

[To be filled]

---

## 11. Multi-Output Handling

### 11.1 Single Output

[To be filled]

### 11.2 Named Outputs

[To be filled]

### 11.3 Dirty Flag Protocol

[To be filled]

### 11.4 Memory Layout

[To be filled]

---

## 12. Error Handling

### 12.1 Parse Errors

[To be filled]

### 12.2 Type Errors

[To be filled]

### 12.3 Runtime Errors

[To be filled]

### 12.4 WASM Traps

[To be filled]

---

## 13. Performance Considerations

### 13.1 Memory Management

[To be filled]

### 13.2 Zero-Copy Optimization

[To be filled]

### 13.3 Batch Processing

[To be filled]

### 13.4 Caching Strategy

[To be filled]

---

## Appendix A: Examples

### A.1 Simple Pipeline

[To be filled]

### A.2 Multi-Input Stage

[To be filled]

### A.3 Stateful Stage

[To be filled]

### A.4 Multi-Output Routing

[To be filled]

---

## Appendix B: Implementation Notes

### B.1 Go Implementation

[To be filled]

### B.2 TypeScript/JavaScript Implementation

[To be filled]

### B.3 Testing Strategy

[To be filled]
