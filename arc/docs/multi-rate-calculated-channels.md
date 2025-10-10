# Multi-Rate Calculated Channels

**Status:** Design Proposal
**Author:** Arc Team
**Date:** 2025-10-10

## Problem Statement

Arc calculated channels currently require all input channels to share the same index (same timestamps). This constraint works well for synchronized data but prevents many real-world calculations involving channels sampled at different rates.

### Motivating Examples

**Example 1: Temperature Correction**
- Temperature standard channel @ 10 Hz
- Pressure sensor @ 10 kHz
- Need: Calculate temperature-corrected pressure at 10 kHz

**Example 2: Statistical Windowing**
- High-frequency vibration sensor @ 100 kHz
- Need: Calculate RMS vibration over 1-second windows @ 1 Hz

**Example 3: Control Loop**
- Setpoint changes @ 1 Hz (operator commands)
- Sensor feedback @ 1 kHz
- Need: Calculate error signal at 1 kHz for control

## Current Limitations

From `core/pkg/distribution/channel/create_test.go:196-216`:
```go
// Validation enforces: all required channels must share the same index
It("Should return an error if all required channels do not share the same index", ...)
```

This prevents:
- Cross-rate calculations
- Temporal interpolation
- Statistical reductions
- Resampling operations

## Proposed Solution: Multi-Rate Strategies

We propose introducing **temporal alignment strategies** that define how mismatched sample rates are reconciled.

### Strategy 1: Sample-and-Hold (Upsample)

**Semantics:** Hold the most recent value from slower channels when faster channels update.

**Output Rate:** Matches the **fastest** input channel.

**Use Case:** Real-time control where only past data is available (causal).

**Example:**
```
Input A @ 2 Hz: [t=0s: 100, t=0.5s: 110, t=1.0s: 120]
Input B @ 5 Hz: [t=0s: 10, t=0.2s: 20, t=0.4s: 30, t=0.6s: 40, t=0.8s: 50]

Output C = A + B @ 5 Hz:
  t=0.0s: 100 + 10 = 110  (A[0] held)
  t=0.2s: 100 + 20 = 120  (A[0] still held)
  t=0.4s: 100 + 30 = 130  (A[0] still held)
  t=0.6s: 110 + 40 = 150  (A[1] now held)
  t=0.8s: 110 + 50 = 160  (A[1] still held)
```

**Implementation:**
- Each slow channel maintains "last seen value"
- Fast channel triggers evaluation
- Slow channel values are held constant between updates

**Arc Stage Behavior:**
```arc
stage temp_corrected_pressure{
    temp <-chan f64      // 10 Hz
    pressure <-chan f64  // 10 kHz
} f64 {
    // When pressure updates (10 kHz), use most recent temp (10 Hz)
    return pressure * temp_correction_factor(temp)
}
```

---

### Strategy 2: Linear Interpolation (Upsample)

**Semantics:** Interpolate slower channels between samples when faster channels update.

**Output Rate:** Matches the **fastest** input channel.

**Use Case:** Smooth calculated values, scientific calculations requiring continuity.

**Example:**
```
Input A @ 2 Hz: [t=0s: 100, t=0.5s: 110]
Input B @ 5 Hz: [t=0s: 10, t=0.2s: 20, t=0.4s: 30]

Output C = A + B @ 5 Hz:
  t=0.0s: 100 + 10 = 110
  t=0.2s: 104 + 20 = 124  (A interpolated: 100 + 0.4*(110-100) = 104)
  t=0.4s: 108 + 30 = 138  (A interpolated: 100 + 0.8*(110-100) = 108)
```

**Implementation:**
- Requires buffering at least 2 samples from slow channels
- Interpolation factor = `(t_current - t_prev) / (t_next - t_prev)`
- Handles edge cases (before first sample, after last sample)

**Arc Extension Required:**
```arc
// Requires syntax for specifying interpolation strategy
stage smooth_calculation{
    slow <-chan f64 @interpolate(linear)
    fast <-chan f64
} f64 {
    return slow * fast
}
```

---

### Strategy 3: Windowed Reduction (Downsample)

**Semantics:** Aggregate fast channel samples into windows aligned with slow channel.

**Output Rate:** Matches the **slowest** input channel (or explicitly specified window rate).

**Use Case:** Statistical analysis, data compression, periodic reporting.

**Common Reduction Functions:**
- `mean(series)` - arithmetic average
- `sum(series)` - total
- `min(series)`, `max(series)` - extrema
- `stddev(series)` - standard deviation
- `rms(series)` - root mean square
- `count(series)` - number of samples

**Example:**
```
Input A @ 1 Hz:  [t=0s: 100, t=1s: 110, t=2s: 120]
Input B @ 10 Hz: [t=0.0s: 1, t=0.1s: 2, ..., t=0.9s: 10,
                  t=1.0s: 11, t=1.1s: 12, ..., t=1.9s: 20]

Output C = A * mean(B over 1s windows) @ 1 Hz:
  t=1s: 100 * mean(1,2,...,10) = 100 * 5.5 = 550
  t=2s: 110 * mean(11,12,...,20) = 110 * 15.5 = 1705
```

**Implementation Challenges:**
- **Window alignment:** Do windows start at slow channel samples or are they independent?
- **Boundary handling:** What happens to partial windows at start/end?
- **Memory management:** Need to buffer fast samples for reduction
- **Timing:** When does the reduction emit? At window close or immediately?

**Arc Syntax Options:**

**Option A: Built-in reduction functions**
```arc
stage vibration_rms{
    vibration <-chan f64  // 100 kHz
    window_trigger <-chan u8  // 1 Hz
} f64 {
    // Collect vibration samples over 1s, output RMS when trigger arrives
    return rms(vibration)
}
```

**Option B: Explicit windowing**
```arc
stage periodic_stats{
    data <-chan f64      // 10 kHz
    period timespan = 1s
} {
    mean_value f64
    max_value f64
} {
    window := buffer(data, period)
    mean_value = mean(window)
    max_value = max(window)
}
```

---

### Strategy 4: Nearest Neighbor (Bidirectional)

**Semantics:** For each output timestamp, use the nearest sample from mismatched channels.

**Output Rate:** Can be **any** of the input rates or independent.

**Use Case:** Event-driven calculations, sparse data alignment.

**Example:**
```
Input A @ irregular: [t=0s: 100, t=0.7s: 110, t=1.5s: 120]
Input B @ 1 Hz:      [t=0s: 10, t=1.0s: 20, t=2.0s: 30]

Output C = A + B @ 1 Hz:
  t=0s: 100 + 10 = 110  (A[0] closest to t=0)
  t=1s: 110 + 20 = 130  (A[1] closest to t=1, distance=0.3s)
  t=2s: 120 + 30 = 150  (A[2] closest to t=2, distance=0.5s)
```

**Implementation:**
- Search forward/backward to find closest timestamp
- May require tolerance threshold for "too far"
- Can be expensive for large datasets

---

## Alignment and Series Considerations

### Alignment Propagation

**Question:** What alignment should the output series have?

**Option 1: Inherit from Primary Input**
```go
// If strategy is upsample (fast rate), use fast channel's alignment
output.Alignment = fastChannel.Alignment

// If strategy is downsample (slow rate), use slow channel's alignment
output.Alignment = slowChannel.Alignment
```

**Option 2: Create New Alignment**
```go
// Calculated channels get a new alignment sequence
// Useful when output rate doesn't match any input
output.Alignment = NewAlignment(calculatedDomainIdx, sampleIdx)
```

### Series Length Handling

**Challenge:** Input series may have different lengths within the same time range.

**Example:**
```
TimeRange: [0s, 1s]
Channel A @ 10 Hz:  10 samples
Channel B @ 100 Hz: 100 samples
```

**For Upsample (Strategy 1/2):**
- Output series length = fast channel length (100 samples)
- Slow channel values are repeated/interpolated

**For Downsample (Strategy 3):**
- Output series length = slow channel length (10 samples)
- Fast channel values are aggregated

---

## Reactive Model Implications

### When Does Calculation Execute?

Arc stages execute when inputs change. For multi-rate:

**Upsample Strategies:**
- Execute when **fast** channel updates
- Slow channel provides "context" (held or interpolated value)

**Downsample Strategies:**
- Execute when **window completes** (slow channel update or time-based)
- Fast channel accumulates into buffer

### Buffering Requirements

**Sample-and-Hold:** Minimal - just last value from slow channels

**Interpolation:** 2 samples from slow channels (prev, next)

**Windowed Reduction:** Full window of fast channel samples
- Memory concern: 10 kHz over 1s = 10,000 samples
- Need eviction policy or max buffer size

### Stateful Variables

Reductions require state across stage invocations:

```arc
stage running_average{
    input <-chan f64
    window_size i64 = 100
} f64 {
    buffer $= series f64[]  // Stateful buffer

    buffer = buffer.append(input)
    if buffer.len() > window_size {
        buffer = buffer.slice(buffer.len() - window_size, buffer.len())
    }

    return mean(buffer)
}
```

**Implementation Question:** How do we handle series in stateful variables (`$=`)?

---

## Index Resolution

### Current Constraint

Calculated channels currently **cannot have an index** - they're implicitly indexed by their dependencies.

From validation: calculated channels must depend on channels with the same index.

### Multi-Rate Indexing

**Option 1: Output Adopts Fast Channel Index**
```
Temp @ 10 Hz with index = time_slow
Pressure @ 10 kHz with index = time_fast

Calculated = temp_corrected_pressure
  Index = time_fast (inherits from pressure)
  Rate = 10 kHz
```

**Option 2: Output Creates New Index**
```
Calculated = downsampled_stats
  Creates new index channel at output rate
  Index = calculated_time @ 1 Hz
```

**Option 3: Output is Unindexed (Virtual)**
```
Calculated channels remain virtual with no persistent index
Only available for real-time streaming, not historical queries
```

---

## Implementation Phases

### Phase 1: Sample-and-Hold (Minimum Viable)

**Goal:** Enable basic cross-rate calculations with zero-order hold.

**Changes Required:**
1. Relax validation: allow calculated channels with different indices
2. Implement state tracking for "last seen value" per slow input
3. Define output index as fastest input's index
4. Update Arc runtime to handle held values

**Arc Syntax:**
```arc
// No new syntax required - inferred from channel rates
stage temp_corrected{
    temp <-chan f64      // Slow
    pressure <-chan f64  // Fast
} f64 {
    return pressure / temp
}
```

**Testing:**
- 2 inputs at different rates
- Verify output matches fast rate
- Verify slow values are held constant

---

### Phase 2: Windowed Reductions

**Goal:** Enable statistical calculations over time windows.

**Changes Required:**
1. Add buffer management to Arc runtime
2. Implement reduction functions: `mean()`, `sum()`, `max()`, `min()`, `rms()`
3. Define window semantics (time-based vs count-based)
4. Handle series aggregation

**Arc Syntax:**
```arc
// Option A: Time-based windows
stage periodic_rms{
    vibration <-chan f64
} f64 {
    window := last(vibration, 1s)  // Last 1 second of data
    return rms(window)
}

// Option B: Sample-based windows
stage moving_average{
    data <-chan f64
} f64 {
    window := last(data, 1000)  // Last 1000 samples
    return mean(window)
}
```

**Implementation Considerations:**
- **Window trigger:** Emit on slow channel update or periodic timer?
- **Buffer size limits:** What if window is too large?
- **Partial windows:** Output partial results or wait for full window?

---

### Phase 3: Interpolation

**Goal:** Enable smooth calculated values via linear/spline interpolation.

**Changes Required:**
1. Buffer 2+ samples from slow channels
2. Implement interpolation algorithms (linear, cubic)
3. Handle edge cases (extrapolation at boundaries)
4. Define interpolation strategy per input

**Arc Syntax:**
```arc
// Annotation-based approach
stage smooth_calculation{
    temp <-chan f64 @interpolate(linear)
    pressure <-chan f64
} f64 {
    return pressure * temp
}
```

---

### Phase 4: Advanced Strategies

**Goal:** Support nearest-neighbor, custom resampling, multi-output reductions.

**Potential Features:**
- Custom reduction functions (user-defined)
- Multi-output reductions (e.g., `stats() -> {mean, stddev, min, max}`)
- Configurable window alignment (sliding vs tumbling)
- Timestamp offset/alignment controls

---

## Open Questions

### 1. Execution Trigger Semantics

**Question:** For downsampling, when exactly does the stage execute?

**Option A: Slow Channel Trigger**
- Stage executes when slow channel updates
- Fast channel samples are aggregated since last slow update
- **Pro:** Simple, predictable
- **Con:** What if slow channel never updates?

**Option B: Time-Based Trigger**
- Stage executes on fixed intervals (independent of inputs)
- Aggregates whatever data is available
- **Pro:** Guaranteed output rate
- **Con:** Requires scheduler/timer mechanism in Arc runtime

**Option C: Fast Channel Trigger with Buffering**
- Stage executes on every fast channel update
- Maintains rolling window buffer
- Outputs reduction over current window
- **Pro:** Low latency, incremental updates
- **Con:** Output rate same as fast rate (wasteful?)

### 2. Alignment for Multi-Input Reductions

**Question:** When reducing `rms(channel_1, channel_2, channel_3)` where all are at 10 kHz, what alignment does the output use?

**Options:**
- First channel's alignment
- Synthesized alignment (new domain)
- All channels must have identical alignments (current behavior)

### 3. Historical Queries on Calculated Channels

**Question:** Should multi-rate calculated channels be queryable historically?

**Challenge:** If calculation depends on 10 kHz data reduced to 1 Hz, the historical query needs to re-run the reduction.

**Options:**
- **Materialize:** Store calculated results as regular channel (expensive)
- **Recompute:** Run calculation on-the-fly during query (slow)
- **Stream-only:** Calculated channels only available in real-time (limitation)

### 4. Error Propagation

**Question:** What happens when input channels have gaps or different time ranges?

**Example:**
```
Channel A: [0s - 10s]
Channel B: [5s - 15s]
Calculated C = A + B
```

**Options:**
- Only output where both inputs exist (intersection)
- Fill gaps with last known value
- Mark outputs as "uncertain" during gaps
- Error/warning when inputs don't overlap

### 5. Performance and Memory Limits

**Question:** How do we prevent unbounded memory growth for large windows?

**Scenarios:**
- 100 kHz data with 10-second windows = 1M samples buffered
- Long-running calculations with slow triggers

**Mitigations:**
- Configurable max buffer size
- Eviction policies (FIFO, time-based)
- Warn/error when buffer limit reached
- Incremental reduction algorithms (running mean vs full buffer mean)

---

## Examples and Use Cases

### Example 1: Temperature Compensation

```arc
// Background: Pressure sensors drift with temperature
// Temperature sensor: 10 Hz
// Pressure sensor: 10 kHz
// Need: Real-time temperature-compensated pressure

stage temp_compensated_pressure{
    temp <-chan f64       // 10 Hz - ambient temperature
    pressure <-chan f64   // 10 kHz - raw pressure

    // Calibration coefficients
    coeff_a f64 = 0.01
    coeff_b f64 = 1.5
} f64 {
    // Sample-and-hold: temp is held constant at 10 Hz rate
    // Calculation executes at 10 kHz (fast channel rate)
    correction := coeff_a * temp + coeff_b
    return pressure * correction
}

// Usage:
temp_sensor -> stage_instance -> corrected_pressure_output
pressure_sensor -> stage_instance
```

**Strategy:** Sample-and-Hold (Phase 1)
**Output Rate:** 10 kHz (matches pressure)
**Index:** Inherits from pressure sensor

---

### Example 2: Vibration Monitoring

```arc
// Background: Monitor machine health via vibration
// Accelerometer: 100 kHz
// Need: 1-second RMS windows for trending

stage vibration_rms{
    accel_x <-chan f64   // 100 kHz
    accel_y <-chan f64   // 100 kHz
    accel_z <-chan f64   // 100 kHz

    window_period timespan = 1s
} f64 {
    // Collect 1 second of data (100k samples per axis)
    window_x := last(accel_x, window_period)
    window_y := last(accel_y, window_period)
    window_z := last(accel_z, window_period)

    // Calculate RMS for each axis
    rms_x := rms(window_x)
    rms_y := rms(window_y)
    rms_z := rms(window_z)

    // Total vibration magnitude
    return sqrt(rms_x^2 + rms_y^2 + rms_z^2)
}

// Usage with time-based trigger:
accel_x_sensor -> stage_instance
accel_y_sensor -> stage_instance
accel_z_sensor -> stage_instance
interval{period: 1s} -> stage_instance -> rms_output @ 1Hz
```

**Strategy:** Windowed Reduction (Phase 2)
**Output Rate:** 1 Hz
**Buffer Size:** 300k samples (100k per axis)

---

### Example 3: Control Error Signal

```arc
// Background: PID control loop
// Setpoint: 1 Hz (operator adjustments)
// Sensor: 1 kHz (feedback)
// Need: Error signal at 1 kHz for responsive control

stage control_error{
    setpoint <-chan f64   // 1 Hz - target value
    feedback <-chan f64   // 1 kHz - measured value
} f64 {
    // Sample-and-hold: setpoint held at 1 Hz
    // Executes at 1 kHz when feedback updates
    return setpoint - feedback
}

// Usage in control loop:
setpoint_channel -> control_error -> pid_controller -> actuator
sensor_channel -> control_error
```

**Strategy:** Sample-and-Hold (Phase 1)
**Output Rate:** 1 kHz (matches sensor)
**Use Case:** Real-time control requires causal calculation

---

### Example 4: Moving Average Filter

```arc
// Background: Reduce noise in sensor readings
// Noisy sensor: 10 kHz
// Need: Smoothed output via 100-sample moving average

stage moving_average_filter{
    input <-chan f64
    window_size i64 = 100
} f64 {
    buffer $= series f64[]

    // Append new sample
    buffer = append(buffer, input)

    // Keep only last window_size samples
    if len(buffer) > window_size {
        buffer = buffer[len(buffer) - window_size:]
    }

    return mean(buffer)
}
```

**Strategy:** Windowed Reduction (Phase 2)
**Output Rate:** Same as input (10 kHz)
**Stateful:** Requires persistent buffer

---

## Comparison with Other Systems

### InfluxDB Downsampling
```sql
SELECT MEAN(value)
INTO "average_1h"
FROM "sensor_data"
GROUP BY time(1h)
```
- Uses explicit GROUP BY for windowing
- Clear aggregation function
- Separate output series

### Prometheus Recording Rules
```yaml
- record: job:api_requests:rate5m
  expr: rate(api_requests_total[5m])
```
- Time-based windows via `[5m]` range selector
- Pre-computed at fixed intervals
- Materialized results

### Pandas Resample
```python
df.resample('1H').mean()
```
- Rich set of aggregation functions
- Interpolation methods: ffill, bfill, linear
- Explicit output frequency

**Arc's Opportunity:**
- Real-time reactive execution (not just batch)
- Type-safe calculations at compile time
- Hardware-oriented (sensor fusion, control)
- Distributed execution across cluster

---

## Recommendations

### Prioritize Phase 1 (Sample-and-Hold)

**Rationale:**
- Solves 80% of real-world use cases
- Minimal complexity (no buffering)
- Natural fit for control applications
- Fast implementation path

**Deliverables:**
1. Relax calculated channel index validation
2. Add "last value" state tracking in runtime
3. Update documentation with examples
4. Test suite with multi-rate scenarios

### Design for Phase 2 (Reductions) Now

**Rationale:**
- Reductions require architectural decisions (buffering, eviction)
- Influences Phase 1 design choices
- Syntax should support future extensions

**Key Decisions Needed:**
- Window trigger semantics
- Buffer size limits
- Reduction function API
- Series handling in stateful variables

### Defer Phase 3 (Interpolation) Until Needed

**Rationale:**
- Complex implementation
- Edge cases (extrapolation)
- Unclear demand for smooth interpolation in hardware monitoring
- Can be added without breaking changes later

---

## Next Steps

1. **Review and Refine:** Discuss this document with team, gather feedback
2. **Prototype Phase 1:** Implement sample-and-hold in experimental branch
3. **User Research:** Interview users for priority use cases
4. **Specify Reduction API:** Define syntax and semantics for Phase 2
5. **Update Arc Spec:** Formalize multi-rate behavior in `arc/spec.md`
6. **Integration Tests:** Create test suite for mixed-rate calculations

---

## Appendix: Related Work

### In Synnax Codebase

- **Alignment system:** `x/go/telem/alignment.go` - Already supports multi-domain data
- **Series operations:** `x/go/telem/series.go` - Elementwise operations on fixed-length series
- **Channel constraints:** `core/pkg/distribution/channel/` - Current validation logic
- **Arc runtime:** `arc/runtime/` - Stage execution and data flow

### External References

- LabVIEW Resample Signal VI
- MATLAB `resample()` function
- NumPy `numpy.interp()` for interpolation
- Pandas `DataFrame.resample()` for time series
- Apache Kafka Streams windowing