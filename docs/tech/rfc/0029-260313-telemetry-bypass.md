# 29 - Telemetry Bypass

**Feature Name**: Telemetry Bypass <br /> **Status**: Draft <br /> **Start Date**:
2026-03-13 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

# 1 - Vocabulary

# 2 - Motivation

## 2.0 - The Core Round-Trip Problem

Arc is designed for real-time control scenarios where a program reads sensor data,
evaluates conditions, and writes commands to actuators in a tight loop. The current
execution pipeline places the Synnax Core directly in this loop. Every control cycle
requires data to traverse the following path:

```
Hardware → Driver (C++) → Network → Core (Go) → Arc WASM → Core → Network → Driver → Hardware
```

A single iteration of the control loop requires two network hops at minimum (sensor data
in, command out), plus all intermediate buffering and processing within the Core. When
the Driver and Core are co-located on the same machine, the network hops become loopback
traffic, but the data still passes through the full Core pipeline: the distribution
layer, the framer service, Cesium's control gate, and the Arc runtime's flush cycle.

This architecture exists for good reason. The Core is the authoritative source of truth
for channel data, authority validation, and telemetry distribution. Routing all data
through the Core means every sample is persisted, every client sees the same stream, and
authority conflicts are resolved in one place. For observability workloads and
low-frequency control (< 10 Hz), this design works well.

The problem emerges at higher control rates. A 1 kHz control loop has a 1 ms budget per
cycle. The Core round-trip consumes a significant and variable portion of that budget,
leaving insufficient headroom for the control logic itself. At 10 kHz, the budget drops
to 100 μs, making the round-trip untenable.

## 2.1 - Sources of Non-Determinism

The round-trip latency is not just high. It is unpredictable. Several components in the
pipeline introduce variable delays that make it impossible to guarantee bounded cycle
times.

### 2.1.0 - Go Garbage Collector

The Arc WASM runtime executes inside the Synnax Core, which is a Go process. Go's
garbage collector runs concurrently but still introduces stop-the-world pauses on the
order of hundreds of microseconds to low milliseconds. These pauses are invisible to the
Arc program and occur at unpredictable intervals. A control loop running at 1 kHz cannot
tolerate a 500 μs GC pause mid-cycle.

### 2.1.1 - Go Goroutine Scheduling

The Go runtime multiplexes goroutines onto OS threads using a cooperative/preemptive
scheduler. Goroutines yield at function calls and are preempted at ~10 ms intervals. The
Arc runtime's goroutine competes with every other goroutine in the Core process for CPU
time. Under load (multiple clients streaming, Cesium compaction, Aspen gossip), the
scheduler may delay the Arc goroutine by milliseconds.

### 2.1.2 - Network Transport

Even on loopback, TCP and WebSocket framing add variable latency. Nagle's algorithm
batches small writes, kernel socket buffers introduce queuing delays, and the Freighter
transport layer adds its own serialization overhead. Under network congestion (or when
the Core is on a remote machine), these delays grow by orders of magnitude.

### 2.1.3 - Flush Batching

The Arc runtime accumulates channel writes in a buffer and flushes them once per
scheduler cycle (`task.go:390-413`). Authority changes flush before data writes. Each
flush produces a `framer.WriterRequest` that enters the Core's write pipeline, where it
passes through the distribution layer, Cesium's control gate, and the storage engine.
This pipeline is optimized for throughput (batching many samples), not for latency
(minimizing time-to-hardware).

### 2.1.4 - Cumulative Effect

These sources of jitter are independent and compound. A single control cycle might
experience GC latency on the inbound path, scheduler delay during Arc execution, and
network buffering on the outbound path simultaneously. The result is a latency
distribution with a long tail that makes real-time guarantees impossible.

| Source                   | Typical Latency | Worst Case    |
| ------------------------ | --------------- | ------------- |
| Network (loopback)       | 50-200 μs       | 1-5 ms        |
| Go GC pause              | 100-500 μs      | 1-3 ms        |
| Goroutine scheduling     | 0-1 ms          | 10+ ms        |
| Flush + write pipeline   | 200-500 μs      | 2-5 ms        |
| **Total (single cycle)** | **~1 ms**       | **10-20+ ms** |

For context, the C++ Driver's internal acquisition loop (hardware read, transform, write
to Synnax) operates with < 100 μs determinism when the Core is not in the critical path.
The Driver is already built for real-time. The bottleneck is that the control
intelligence lives in the Core, not alongside the hardware.

## 2.2 - Use Cases

### 2.2.0 - High-Frequency PID Control

A pressure regulation loop running at 1 kHz reads a pressure transducer, computes a PID
output, and commands a proportional valve. The 1 ms cycle budget leaves no room for a
Core round-trip. The control law must execute locally on the Driver with direct access
to the sensor frame and direct output to the valve command.

### 2.2.1 - Safety Interlocks

An abort sequence monitors multiple sensors and must respond within a bounded time
window (e.g., close a valve within 5 ms of detecting an overpressure condition). The
non-deterministic tail latency of the Core pipeline makes it unsuitable for
safety-critical response times. The interlock logic must execute with worst-case
guarantees that only a local execution path can provide.

### 2.2.2 - Coordinated Multi-Actuator Control

A test stand sequence that coordinates fuel valve, oxidizer valve, and igniter timing
requires sub-millisecond synchronization between command outputs. When commands route
through the Core independently, relative timing between actuators depends on network and
scheduling jitter. Local execution ensures commands are computed and dispatched in the
same deterministic cycle.

### 2.2.3 - Closed-Loop Feedback with State Estimation

A Kalman filter or state estimator running at high frequency needs to ingest sensor
data, update its internal state, and produce corrected outputs every cycle. The
estimator's accuracy degrades with variable-latency feedback. Consistent cycle timing is
a correctness requirement, not just a performance preference.

## 2.3 - Control Authority in a Bypassed Pipeline

Moving the control loop off the Core introduces a fundamental tension with the authority
system. Today, every write from Arc passes through Cesium's control gate before reaching
hardware. The gate is the single point of authority validation: it holds the definitive
record of which subject controls each channel at each moment. A bypass removes the gate
from the critical path, which means the Driver must make control decisions without
synchronous access to the source of truth.

### 2.3.0 - How Authority Works Today

Cesium maintains a `Controller` per channel that tracks a set of `Gate` objects, one per
open writer. Each gate carries an `Authority` (`uint8`, 0-255) and a `Subject` (name +
key). When multiple gates contend for the same channel, the gate with the highest
authority becomes the current controller. Equal authorities are broken by opening order
(first writer wins). Every `Write()` and `Commit()` call checks authorization against
the current gate before proceeding.

Authority changes propagate through a virtual channel (`sy_node_1_control`). When a gate
opens, closes, or changes authority, Cesium encodes a `ControlUpdate` containing
`Transfer` objects (from-state, to-state) as JSON and writes it to this channel. The
update enters the relay and is delivered to any client streaming that channel. This is
the mechanism by which the Console shows real-time control indicators and by which any
client can observe authority changes cluster-wide.

### 2.3.1 - The Bypass Removes Synchronous Validation

In the current pipeline, the sequence of operations for a single Arc control cycle is:

1. Arc reads sensor data (from Core streamer)
2. Arc executes control logic (WASM)
3. Arc writes command (to Core writer)
4. Core validates authority (Cesium gate)
5. If authorized, command reaches hardware via Driver streamer
6. If unauthorized, command is silently dropped

Step 4 is synchronous. The command never reaches hardware without passing through the
gate. This guarantee disappears in a bypass. If Arc executes on the Driver and writes
directly to the hardware sink, there is no gate in the path. The command reaches the
actuator before the Core knows it happened.

### 2.3.2 - The Authority Propagation Window

The bypass can restore authority awareness by subscribing to the control state channel.
The Driver already uses streamers (the control pipeline's `Streamer` reads command
frames), so subscribing to `sy_node_1_control` requires no new infrastructure. When an
operator takes control in Console, the transfer propagates:

```
Console write → Cesium gate transfer → virtual channel → relay → Driver streamer
```

This path has bounded latency. The relay delivers updates within a single relay cycle
(typically sub-millisecond on loopback, low single-digit milliseconds across Cores).
However, there is a non-zero window between the moment the gate transfers on the Core
and the moment the Driver receives the notification. During this window, the Driver is
commanding hardware under an authority it no longer holds.

The size of this window depends on relay latency and network conditions. On a co-located
deployment, it is on the order of 1-5 ms. On a multi-Core deployment, it may reach 10-50
ms. The question is whether this window is acceptable for the control scenarios the
bypass is designed to serve.

### 2.3.3 - Comparison with the Current Pipeline

The current pipeline does not eliminate this window. It hides it. When an operator takes
control in Console, the gate transfers immediately on the Core. But the Arc runtime,
running on the Core, does not learn about the transfer until its next write returns
`Authorized: false`. If Arc is mid-cycle (executing WASM, buffering writes, waiting for
the flush), the operator's takeover is not reflected until the next flush completes and
the response propagates back. This is also a window, and its duration depends on cycle
time, flush latency, and pipeline depth.

In practice, the bypass's authority propagation window is comparable to the current
pipeline's. The difference is structural: the current pipeline guarantees that an
unauthorized write never reaches Cesium (and therefore never reaches hardware through
the normal streamer path), while the bypass must enforce this guarantee locally on the
Driver.

Note: The interaction between this propagation window and the Core-side group exclusion
mechanism (Section 4.3) creates a more severe problem than the window alone suggests.
See Section 8.0 for a detailed analysis of the dual-filter authority gap and Section 8.1
for the proposed solution.

### 2.3.4 - Per-Channel Authority Granularity

Authority operates per-channel, not per-subject. An Arc program writing to channels A,
B, and C might lose authority on A while retaining it on B and C. In the current
pipeline, Cesium handles this transparently: the stream writer excludes unauthorized
channels from the frame and relays only the authorized portion.

In a bypass, the Driver must replicate this behavior. When the Driver receives a control
transfer notification indicating it has lost authority on channel A, it must stop
commanding channel A while continuing to command B and C. The local authority mirror
must track per-channel state and the bypass sink must filter frames accordingly.

### 2.3.5 - Authority Changes Initiated by the Arc Program

Arc programs can change their own authority via `set_authority{}`. In the current
pipeline, these changes flow through the writer to Cesium, where the gate updates and
potentially triggers transfers. In a bypass, the Arc program's authority changes must
still reach the Core so that:

1. The gate updates (other clients see the change)
2. The control state channel reflects the new authority
3. Other writers contending for the same channels see the transfer

This means authority changes cannot be purely local. They must be forwarded to the Core
asynchronously, and the Driver must optimistically apply the change locally while
waiting for Core confirmation. If the Core rejects the change (e.g., another writer
already holds higher authority), the Driver must reconcile its local state with the
Core's response.

# 3 - Design Philosophy

## 3.0 - The Pipeline is the Integration Point

The Driver's pipeline infrastructure (`pipeline::Acquisition`, `pipeline::Control`)
already abstracts the boundary between hardware and network through factory interfaces
(`WriterFactory`, `StreamerFactory`). These factories are injected into the common task
layer, which every hardware integration uses. The bypass leverages this existing
abstraction by wrapping the factories rather than modifying the pipelines or hardware
integrations. No hardware-specific code changes.

## 3.1 - Always Present, Automatically Effective

The bypass is not a mode, a configuration flag, or an opt-in feature. It is
infrastructure that is always present in the Driver. When local routes exist (a local
consumer subscribes to channels that a local producer publishes), the bypass
automatically short-circuits data through the local path. When no local routes exist,
the overhead is effectively zero. The system gets faster when local routing is possible
without anyone asking for it.

## 3.2 - Authority Is the Filter, Not the Router

The bypass does not participate in the authority system. It does not open writers or
hold gates. Existing tasks continue to manage their own authority through the standard
Core path. The bypass observes authority state via the control state channel and uses it
to filter frames at route time. This replaces Cesium's synchronous gate validation with
local filtering that operates on the same authority information, propagated
asynchronously.

## 3.3 - Consumers Define Their Own Semantics

The bypass routes frames by channel key. How a consumer ingests those frames (latest
value, queued, batched, lossy, lossless) is the consumer's responsibility. The bypass
defines the routing contract, not the consumption contract.

# 4 - Detailed Design

## 4.0 - The Telemetry Bus

The telemetry bus is a process-wide frame router inside the Driver. It maintains a
routing table that maps channel keys to local subscribers. Producers publish frames
tagged with channel keys. The bus delivers each frame to every subscriber registered for
any key in that frame.

The bus is created at rack startup and is accessible to all tasks through the task
context. It has three roles:

1. **Routing**: Deliver frames from local producers to local consumers by channel key.
2. **Authority filtering**: Maintain a local mirror of per-channel authority state and
   filter frames that are not authorized to reach their destination.
3. **Discovery**: Build and maintain the routing table as tasks start and stop.

The bus is not a message queue. It does not buffer frames, enforce ordering across
producers, or guarantee delivery to slow consumers. Each subscriber provides its own
consumption interface. The bus calls that interface synchronously on the publisher's
thread or dispatches to the subscriber's thread depending on the subscriber's declared
semantics.

### 4.0.0 - Routing Table

The routing table is a map from channel key to a set of subscriber references. It is
built incrementally:

- When a task starts, its middleware registers channel keys with the bus (produced keys
  and consumed keys).
- When a task stops, its middleware unregisters those keys.
- The routing table changes only at task start/stop boundaries, not at frame time.

A middleware can query the routing table at registration time to determine whether any
local routes exist for its channel keys. If no routes exist, the middleware sets a
fast-path flag and performs no per-frame bus operations, making the overhead effectively
zero for non-bypassed tasks.

### 4.0.1 - Subscriptions

The bus uses queue-based subscriptions rather than virtual callbacks. A `Subscription`
is a shared object that holds an unbounded deque of frames, filtered to the channel keys
the subscriber declared interest in. The bus pushes frames into the subscription's
queue. The consumer pulls them out on its own thread, at its own pace.

```cpp
class Subscription {
    std::mutex mu;
    std::condition_variable cv;
    std::deque<x::telem::Frame> queue;
    std::vector<synnax::channel::Key> keys;
    std::unordered_set<synnax::channel::Key> key_set;

public:
    explicit Subscription(std::vector<synnax::channel::Key> keys);

    /// Non-blocking pop. Returns true if a frame was available.
    bool try_pop(x::telem::Frame &frame);

    /// Blocking pop. Returns false if closed with no remaining frames.
    bool pop(x::telem::Frame &frame);

    /// Filters frame to subscribed keys, applies alignment, and enqueues.
    /// Returns true if any channel matched and the frame was delivered.
    bool filter_and_push(
        const x::telem::Frame &frame,
        const std::vector<x::telem::Alignment> &alignments
    );
};
```

The subscription stores channel keys in both a vector (for iteration) and an
`unordered_set` (for O(1) containment checks in `filter_and_push`).

The bus tracks subscriptions via `weak_ptr`. Destroying a subscription automatically
expires its route entries during the next publish cycle. No explicit unsubscribe is
required, though an eager `unsubscribe()` method exists for immediate cleanup.

The publisher's thread and consumer's thread are fully decoupled. The push side appends
to the deque under a mutex (effectively instantaneous), so the bus never blocks on a
slow consumer. The consumer controls its own read cadence via `try_pop` (polling) or
`pop` (blocking with condition variable). The queue is unbounded, so a consumer that
falls behind accumulates memory rather than blocking the publisher.

## 4.1 - Factory Wrapping

The bypass integrates with existing pipelines through factory wrappers. The common task
layer (`common::ReadTask`, `common::WriteTask`) already accepts `WriterFactory` and
`StreamerFactory` as constructor parameters. The bypass provides wrapped versions of
these factories that add bus integration without modifying the pipeline or hardware
code.

### 4.1.0 - Acquisition Side (Read Tasks)

For read tasks, the acquisition pipeline reads from a hardware `Source` and writes to a
`Writer` provided by the `WriterFactory`. The bypass wraps the `WriterFactory` to
produce a `Writer` that publishes frames to the bus before forwarding to the Core. The
writer also holds a reference to the authority mirror (`control::States`) and filters
frames before bus publication, keeping only channels where the writer's subject holds
authority:

```cpp
class Writer final : public pipeline::Writer {
    std::unique_ptr<pipeline::Writer> server;
    std::shared_ptr<Bus> bus;
    std::shared_ptr<control::States> states;
    synnax::framer::WriterConfig cfg;

public:
    x::errors::Error write(const x::telem::Frame &fr) override {
        if (this->states->all_authorized(fr, this->cfg.subject))
            this->bus->publish(fr);
        else {
            auto filtered = this->states->filter(fr, this->cfg.subject);
            if (!filtered.empty()) this->bus->publish(filtered);
        }
        return this->server->write(fr);
    }

    x::errors::Error
    set_authority(const pipeline::Authorities &authorities) override {
        // Short-circuit: apply authority increases directly to the mirror
        // before forwarding to the Core (see Section 8.1).
        for (size_t i = 0; i < keys.size(); i++)
            this->states->apply_increase(this->cfg.subject, keys[i], auth);
        return this->server->set_authority(authorities);
    }
};
```

The `WriterFactory` wraps a Core `WriterFactory`, injects the rack's group identity into
the writer config for Core-side deduplication (Section 4.3), and registers the writer's
channels for alignment tracking:

```cpp
class WriterFactory final : public pipeline::WriterFactory {
    std::shared_ptr<pipeline::WriterFactory> server;
    std::shared_ptr<Bus> bus;
    std::shared_ptr<control::States> states;
    std::uint32_t group;

public:
    std::pair<std::unique_ptr<pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &config) override {
        auto cfg = config;
        if (this->group != 0 && cfg.subject.group == 0)
            cfg.subject.group = this->group;
        this->bus->register_channels(cfg.channels);
        auto [writer, err] = this->server->open_writer(cfg);
        if (err) return {nullptr, err};
        return {std::make_unique<Writer>(
            std::move(writer), this->bus, this->states, cfg
        ), x::errors::NIL};
    }
};
```

When the writer opens, it registers the channel keys with the bus for alignment tracking
and injects the group identity for Core-side deduplication. Every frame written by the
acquisition pipeline is published to the bus (for local consumers) and forwarded to the
Core (for persistence and relay to remote clients). Authority filtering happens at
publish time: the writer checks the local mirror and only publishes channels where its
subject holds authority, matching Cesium's behavior of stripping unauthorized channels
before relaying.

### 4.1.1 - Control Side (Write Tasks)

For write tasks, the control pipeline reads from a `Streamer` provided by the
`StreamerFactory` and writes to a hardware `Sink`. The bypass wraps the
`StreamerFactory` to produce a `Streamer` that merges frames from both the Core streamer
and the local bus subscription:

```cpp
class Streamer final : public pipeline::Streamer {
    std::unique_ptr<pipeline::Streamer> server;
    std::shared_ptr<Subscription> subscription;
    std::thread server_thread;

public:
    std::pair<x::telem::Frame, x::errors::Error> read() override {
        while (true) {
            // Prioritize local bus frames (non-blocking).
            x::telem::Frame local;
            while (this->subscription->try_pop(local))
                if (!local.empty()) return {std::move(local), x::errors::NIL};
            // Fall through to Core frames queued by background thread.
            // ... check server_frames deque ...
            // Block with 5ms timeout waiting for either source.
            this->notify_cv.wait_for(lock, std::chrono::milliseconds(5), ...);
        }
    }
};
```

The `Streamer` runs the Core read on a background thread so that both local bus frames
and Core frames are delivered without blocking each other. Local bus frames are
prioritized: the subscription is polled first on every `read()` call. Core frames are
checked only when no local frame is available.

No authority filtering happens in the `Streamer`. The bypass `Writer` is responsible for
filtering unauthorized channels before publishing to the bus (Section 4.1.0), matching
Cesium's behavior of stripping unauthorized channels before relaying.

The `StreamerFactory` wraps a Core `StreamerFactory` and injects the subject's group
into `ExcludeGroups` for Core-side deduplication (Section 4.3):

```cpp
class StreamerFactory final : public pipeline::StreamerFactory {
    std::shared_ptr<pipeline::StreamerFactory> server;
    std::shared_ptr<Bus> bus;
    x::control::Subject subject;

public:
    std::pair<std::unique_ptr<pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) override {
        if (this->subject.group != 0)
            config.exclude_groups.push_back(this->subject.group);
        auto [streamer, err] = this->server->open_streamer(config);
        if (err) return {nullptr, err};
        auto subscription = this->bus->subscribe(config.channels);
        return {std::make_unique<Streamer>(
            std::move(streamer), std::move(subscription)
        ), x::errors::NIL};
    }
};
```

When the streamer opens, it subscribes to the bus for the command channel keys and
injects its group into `ExcludeGroups` on the Core streamer. On each `read()`, it checks
the bus subscription first (non-blocking). If a locally routed frame is available, it is
returned immediately without waiting for the Core. If no local frame is available, the
streamer falls back to the Core path. Core-side group exclusion (Section 4.3) prevents
duplicate delivery of frames that were already routed via the bus. This is the short
circuit: locally routed commands skip the Core round-trip.

### 4.1.2 - Injection Point

The common task layer constructs factories in the `ReadTask` and `WriteTask`
constructors. Today, this creates `SynnaxWriterFactory` and `SynnaxStreamerFactory`
directly. With the bypass, helper functions (`make_writer_factory`,
`make_streamer_factory`) wrap the Core factories with bus-aware versions when a bus is
available:

```cpp
// WriterFactory wrapping (acquisition side)
auto server_factory = std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client);
auto factory = std::make_shared<bypass::pipeline::WriterFactory>(
    server_factory, ctx->bus, ctx->states, rack_key
);
```

```cpp
// StreamerFactory wrapping (control side)
auto server_factory = std::make_shared<pipeline::SynnaxStreamerFactory>(ctx->client);
auto factory = std::make_shared<bypass::pipeline::StreamerFactory>(
    server_factory, ctx->bus, subject
);
```

If `ctx->bus` is nullptr (no bus configured), the helpers fall back to the direct Core
factories. The pipeline classes, hardware `Source`/`Sink` implementations, and factory
interfaces remain untouched.

## 4.2 - Authority Mirror

The authority mirror (`control::States`) is a thread-safe data structure that maintains
a local copy of per-channel authority state. The control pipeline's `Streamer`
subscribes to the control state virtual channel (`sy_node_{N}_control`) as part of its
normal operation. When control updates arrive, the pipeline feeds them to the `States`
instance, which parses the JSON and updates its internal map.

### 4.2.0 - Data Structures

The mirror uses existing C++ types from `x/cpp/control/` that match the Core's JSON wire
format:

```cpp
template <typename ResourceKey>
struct State {
    x::control::Subject subject;   // {name, key}
    ResourceKey resource;           // channel key
    x::control::Authority authority; // uint8
};

template <typename ResourceKey>
struct Transfer {
    std::optional<State<ResourceKey>> from; // null on initial acquire
    std::optional<State<ResourceKey>> to;   // null on release
};
```

The mirror stores the current controlling state per channel:

```cpp
class States {
    mutable std::shared_mutex mu;
    std::unordered_map<
        synnax::channel::Key,
        x::control::State<synnax::channel::Key>
    > states;

public:
    /// Apply a control update (batch of transfers).
    void apply(const x::control::Update<synnax::channel::Key> &update);

    /// Parse and apply a JSON-encoded control update from a series.
    void apply(const x::telem::Series &series);

    /// Optimistically apply an authority increase (see Section 8.1).
    void apply_increase(
        const x::control::Subject &subject,
        synnax::channel::Key channel,
        x::control::Authority authority
    );

    /// Filter a frame, keeping only channels where subject holds authority
    /// or no authority state exists (uncontrolled).
    x::telem::Frame filter(const x::telem::Frame &frame,
                           const x::control::Subject &subject) const;
    x::telem::Frame filter(x::telem::Frame &&frame,
                           const x::control::Subject &subject) const;

    /// Check whether subject holds authority for all frame channels.
    bool all_authorized(const x::telem::Frame &frame,
                        const x::control::Subject &subject) const;
};
```

### 4.2.1 - Update Path

The `States` instance receives control updates through two paths:

1. **Relay path**: The control pipeline's streamer subscribes to `sy_node_{N}_control`.
   On first read, the distribution layer's `controlStateSender` injects the full current
   state automatically. On subsequent reads, incremental `Transfer` updates arrive as
   JSON-encoded string series. The pipeline feeds these to `States::apply(Series)`,
   which parses the JSON and updates the map.

2. **Short-circuit path**: When the bypass `Writer` calls `set_authority` with an
   authority strictly greater than the current holder's, it calls
   `States::apply_increase` directly, updating the mirror before the request reaches the
   Core (see Section 8.1).

Readers (`filter`, `all_authorized`) acquire a shared lock, so filtering does not block
on updates and updates do not block on filtering.

### 4.2.2 - Wire Format

Each frame on the control state channel contains a single JSON-encoded string series.
The JSON structure is:

```json
{
  "transfers": [
    {
      "from": {
        "resource": 123,
        "subject": { "key": "k", "name": "n" },
        "authority": 200
      },
      "to": {
        "resource": 123,
        "subject": { "key": "k2", "name": "n2" },
        "authority": 250
      }
    }
  ]
}
```

The mirror parses this using the existing `nlohmann/json` library already available in
the Driver's Bazel dependencies.

### 4.2.3 - Filtering

The bypass `Writer` calls `states->filter(frame, subject)` or
`states->all_authorized(frame, subject)` before publishing each frame to the bus
(Section 4.1.0). The filter iterates over the frame's channel keys. For each key, it
checks whether the given subject currently holds authority on that channel or whether no
authority state exists (uncontrolled channel). Channels where a different subject holds
authority are removed from the frame. If all channels are removed, no frame is published
to the bus.

The filter has two overloads: a `const`-reference version that shallow copies passing
series (used when the frame is still needed for the Core write), and a move version that
transfers ownership of passing series without copying (used when the frame is consumed).
The `all_authorized` fast path avoids any per-channel work when the writer is fully
authorized, which is the common case during normal operation.

This is the local equivalent of Cesium's gate check. The guarantee is the same: commands
only reach local subscribers for channels where the subject holds authority. The
difference is that the authority state may be up to one relay cycle stale (Section
2.3.2). Authority filtering happens at publish time in the `Writer`, not at read time in
the `Streamer`, matching Cesium's behavior of stripping unauthorized channels before
relaying frames.

## 4.3 - Deduplication via Core-Side Group Exclusion

When an Arc task writes a command frame, the data flows through two paths:

1. **Bus path (fast)**: Arc output queue -> bus -> `Subscription` -> `Sink`
2. **Core path (slow)**: Arc output queue -> acquisition pipeline -> Core writer ->
   Cesium relay -> write task's Core streamer -> `Sink`

Both paths terminate at the same write task. The fast path arrives first. The slow path
arrives later with the same data. Without deduplication, the Sink would execute the same
command twice.

### 4.3.0 - Group Identity

Deduplication is handled Core-side through a group exclusion mechanism. Each writer
carries a `ControlSubject.Group` field (a `uint32` identifying the writer's origin). The
bypass `WriterFactory` injects the Driver's rack key as the group identity for all
writers it creates. When Cesium relays a frame to the distribution layer, it tags the
frame with the writer's group:

```go
// cesium/writer_stream.go
w.relay.Inlet() <- relayResponse{
    frame: req.Frame.ExcludeKeys(excludeUnauthorized),
    group: w.ControlSubject.Group,
}
```

The group field propagates through the relay's `Response` type and into the protobuf
wire format (`RelayResponse.group`).

### 4.3.1 - Streamer Exclusion

The bypass `StreamerFactory` injects the subject's group into the Core streamer's
`ExcludeGroups` configuration before opening the connection:

```cpp
// driver/bypass/pipeline/streamer.h
if (this->subject.group != 0)
    config.exclude_groups.push_back(this->subject.group);
```

On the Core-side, the relay streamer checks every frame against the exclusion list
before delivery:

```go
// core/pkg/distribution/framer/relay/streamer.go
if r.Group != 0 && slices.Contains(s.cfg.ExcludeGroups, r.Group) {
    continue
}
```

Frames originating from the same rack are dropped before they ever reach the Driver's
Core streamer. The bus already delivered them locally. Frames from other groups (remote
Console, other Drivers) pass through normally.

### 4.3.2 - Why Core-Side Exclusion?

The Driver never sees the same frame twice, so it does not need sequence numbers,
timestamp tracking, or source tags. The Core relay already processes every frame, and
the group check is a single `slices.Contains` call on a small slice (typically one
element). The cost is negligible.

Client-side alternatives (sequence numbers, timestamp-based deduplication, source
tagging) would require tracking state in the `Streamer` and handling edge cases around
route appearance/disappearance mid-stream. Core-side exclusion avoids all of this.

## 4.4 - Route Discovery

The bus learns which channel keys are locally routable through factory registration.
When the `WriterFactory` opens a writer, it calls `bus->register_channels(keys)` to set
up alignment tracking for the writer's channel keys. When the `StreamerFactory` opens a
streamer, it calls `bus->subscribe(keys)` to create a subscription for the streamer's
channel keys. The bus delivers frames to every subscription whose key set overlaps with
the published frame.

### 4.4.0 - Registration Lifecycle

Registration follows task lifecycle:

1. Task starts. Common task layer constructs bus-aware factories.
2. Pipeline starts. `WriterFactory` opens a writer, registering channels for alignment
   tracking. `StreamerFactory` opens a streamer, creating a subscription.
3. Bus delivers frames from publishers to matching subscriptions by channel key overlap.
4. Task stops. Pipeline stops. The subscription's `shared_ptr` is destroyed, and the bus
   automatically expires the `weak_ptr` during the next publish cycle.

The bus does not maintain an explicit routing table. Instead, it iterates over all live
subscriptions on each publish, checking key overlap via each subscription's
`filter_and_push`. Subscriptions are tracked via `weak_ptr`, so destruction of the
owning `shared_ptr` (when the streamer closes) is sufficient to remove the route.

### 4.4.1 - No-Subscriber Fast Path

When no subscriptions exist, `publish()` acquires a shared lock, finds the subscriber
list empty, and returns. This costs 11 ns regardless of frame size (benchmarked in
Section 7.1). No alignment computation or frame copying occurs. The common case (tasks
with no local consumers) pays effectively zero per-frame cost.

## 4.5 - Data Flow

### 4.5.0 - Bypassed Control Loop

When a read task, an Arc runtime, and a write task are all running on the same Driver
with overlapping channel keys, the bypassed control loop operates as follows:

```
1. Hardware Source produces frame [channels: pressure, temperature]
2. Acquisition pipeline calls bypass Writer.write(frame)
3. Writer checks authority mirror: is subject authorized on all channels?
4. Writer publishes authorized channels to bus
5. Writer forwards full frame to Core writer (async, for persistence)
6. Bus delivers frame to Arc runtime's subscription
7. Arc runtime ingests frame, executes WASM, produces command frame [channels: valve]
8. Arc's acquisition pipeline calls bypass Writer.write(command_frame)
9. Writer checks authority, publishes authorized channels to bus
10. Writer forwards command_frame to Core writer (for persistence + authority gate)
11. Bus delivers command_frame to write task's bypass Streamer subscription
12. Streamer returns command_frame to control pipeline
13. Control pipeline calls Sink.write(command_frame)
14. Valve actuates
```

Steps 1-14 happen within the Driver process. The Core is not in the loop. Steps 5 and 10
send data to the Core asynchronously for persistence, relay to Console, and authority
management, but the control-critical path (steps 1 -> 7 -> 11 -> 14) is entirely local.
Authority filtering happens at publish time in the Writer (steps 3-4 and 9), not at read
time in the streamer.

### 4.5.1 - Non-Local Channels

When an Arc runtime writes to a channel whose hardware is on a different Driver, no bus
subscriber exists for that channel key. The bus `publish()` delivers to zero
subscriptions (11 ns constant cost). The frame flows through the Core path only: Core
writer -> distribution layer -> remote Driver's streamer -> remote `Sink`. Real-time
guarantees are naturally relaxed for these channels. No special handling is needed.

### 4.5.2 - Operator Takeover

When an operator takes control of a channel via Console with higher authority:

1. Console opens a writer with authority 250 on channel `valve`.
2. Cesium gate transfers control from Arc (authority 200) to operator.
3. Cesium writes `ControlTransfer` to `sy_node_1_control`.
4. `control::States` receives the transfer via the relay, updates its map.
5. On the next `Writer.write()` call, the authority filter removes `valve` from the
   frame before publishing to the bus.
6. Arc's commands to `valve` stop reaching hardware via the local path.
7. Operator's commands arrive via the Core streamer (the operator's group differs from
   the rack's group, so `ExcludeGroups` does not filter them).
8. When the operator releases control, the mirror updates, and Arc's commands resume
   reaching hardware via the bus.

# 5 - Implementation

## 5.0 - New Components

| Component                    | Location                  | Description                                          |
| ---------------------------- | ------------------------- | ---------------------------------------------------- |
| `Bus`                        | `driver/bypass/bypass.h`  | Process-wide frame router with alignment assignment  |
| `Subscription`               | `driver/bypass/bypass.h`  | Queue-based subscription with key filtering          |
| `bypass::pipeline::Writer`   | `driver/bypass/pipeline/` | Wraps `Writer`, publishes to bus with auth filtering |
| `bypass::pipeline::Streamer` | `driver/bypass/pipeline/` | Merges bus and Core frames via background thread     |
| `WriterFactory`              | `driver/bypass/pipeline/` | Wraps `WriterFactory`, injects group + alignment     |
| `StreamerFactory`            | `driver/bypass/pipeline/` | Wraps `StreamerFactory`, injects group exclusion     |
| `control::States`            | `driver/control/state.h`  | Local authority mirror with short-circuit updates    |
| `x::control::State`          | `x/cpp/control/control.h` | Control state type (matches JSON wire format)        |
| `x::control::Transfer`       | `x/cpp/control/control.h` | Control transfer type                                |
| `x::control::Update`         | `x/cpp/control/control.h` | Batch of transfers with JSON parsing                 |

## 5.1 - Modified Components

| Component                         | Change                                                         |
| --------------------------------- | -------------------------------------------------------------- |
| `task::Context`                   | Add `bus` and `states` members                                 |
| `common::ReadTask`                | Wrap `WriterFactory` with bypass `WriterFactory`               |
| `common::WriteTask`               | Wrap both factories with bypass versions                       |
| `rack`                            | Create `Bus` and `control::States` at startup, pass to context |
| `StreamerConfig` (Go, C++, proto) | Add `ExcludeGroups` field                                      |
| `WriterConfig` (Go, C++, proto)   | Add `ControlSubject.Group` field                               |
| `Cesium relay`                    | Tag relayed frames with writer's group                         |
| `relay::Streamer`                 | Filter frames by `ExcludeGroups` before delivery               |
| `x::telem::Series`                | `shared_ptr` data for copy-on-write, `shallow_copy()` method   |

## 5.2 - Unchanged Components

- `pipeline::Acquisition`, `pipeline::Control` (no changes)
- `pipeline::Source`, `pipeline::Sink`, `pipeline::Writer`, `pipeline::Streamer`
  interfaces (no changes)
- All hardware integrations: Modbus, NI, LabJack, OPC UA, HTTP, EtherCAT (no changes)
- All hardware `Source`/`Sink` implementations (no changes)
- Arc runtime (`arc/cpp/runtime/`) (no changes, no bus awareness)

# 6 - Testing Strategy

## 6.0 - Unit Tests

- **Bus routing**: Publish to bus, verify subscribers receive correct frames by key.
- **Authority mirror**: Feed mock control state updates, verify filter behavior.
- **Bypass Writer**: Verify frames reach both bus and Core writer, authority filtering
  at publish time works.
- **Bypass Streamer**: Verify local frames are preferred, Core frames are fallback,
  group exclusion prevents duplicate delivery.
- **Fast-path**: Verify zero overhead when no subscribers exist.

## 6.1 - Integration Tests

- **End-to-end bypass**: Hardware source -> bus -> Arc runtime -> bus -> hardware sink.
  Verify commands reach hardware without Core in the loop.
- **Operator takeover**: Verify authority transfer disables local path and enables Core
  path.
- **Mixed local/remote**: Arc runtime writing to both local and remote channels. Verify
  local channels bypass, remote channels go through Core.
- **Latency measurement**: Compare control loop latency with and without bypass.

# 7 - Performance Baseline

Benchmarks measured on Apple M4 Max (16 cores, 4096 KiB L2 per core), compiled with
`-c opt` (Clang with full optimizations). These numbers establish the baseline cost of
each operation in the bypass hot path before any optimization work begins.

All benchmark source code lives in `x/cpp/telem/frame_bench.cpp` and
`driver/bus/bus_bench.cpp`.

## 7.0 - Frame/Series Primitives

These isolate the cost of telem data operations with zero threading or bus overhead.

| Benchmark         | 8 B (1x1 `f64`) | 10 B (10x1 `u8`) | 40 kB (10x1000 `f32`) | 480 kB (30x4000 `f32`) |
| ----------------- | --------------- | ---------------- | --------------------- | ---------------------- |
| Frame `deep_copy` | 132 ns          | 476 ns           | 1,354 ns              | 12,791 ns              |
| Frame move        | 537 ns          | 654 ns           | 650 ns                | 1,013 ns               |
| Frame construct   | 134 ns          | 652 ns           | 1,538 ns              | 13,321 ns              |
| Frame iterate     | 1 ns            | 5 ns             | 5 ns                  | 13 ns                  |

| Benchmark          | 32 B   | 16 kB  | 64 kB    | 480 kB   |
| ------------------ | ------ | ------ | -------- | -------- |
| Series `deep_copy` | 36 ns  | 411 ns | 1,411 ns | 8,885 ns |
| Series move        | 509 ns | 513 ns | 1,008 ns | 1,015 ns |

Key observations:

- **`deep_copy` is `memcpy`-dominated.** Cost scales linearly with data size. At 480 kB
  (large acquisition frame), a single `deep_copy` costs ~13 µs.
- **`move` is constant time** regardless of data size (~1 µs, dominated by `unique_ptr`
  transfer and `PauseTiming` overhead in the benchmark harness).
- **Frame construction cost equals `deep_copy` cost.** Both allocate heap storage and
  `memcpy` data. There is no "free" way to build a frame.
- **Frame iteration is negligible.** 13 ns for 30 channels. The iteration overhead in
  `publish()` and `filter()` is not a bottleneck.

## 7.1 - Bus Component Operations

These measure individual bus operations in isolation.

| Benchmark                     | small_cmd (10 B) | medium (40 kB) | large_acq (480 kB) |
| ----------------------------- | ---------------- | -------------- | ------------------ |
| Bus publish (no subscribers)  | 11 ns            | 11 ns          | 11 ns              |
| Bus publish (one subscriber)  | 574 ns           | 1,535 ns       | 12,913 ns          |
| Subscription push + `try_pop` | 530 ns           | 1,539 ns       | 11,939 ns          |
| Authority filter (all pass)   | 767 ns           | 1,757 ns       | 13,401 ns          |
| Authority filter (half pass)  | 512 ns           | 981 ns         | 6,925 ns           |
| Authority filter (none pass)  | 49 ns            | 49 ns          | 121 ns             |

Subscriber scaling at `large_acq` (480 kB):

| Subscribers | Time      |
| ----------- | --------- |
| 1           | 12,984 ns |
| 2           | 25,603 ns |
| 5           | 67,854 ns |

Cross-thread subscription (`large_acq`): 3,424 ns CPU / 14,804 ns wall.

Key observations:

- **No subscribers publish is 11 ns constant.** The `routes.empty()` early exit under
  shared_lock is effectively free regardless of frame size.
- **Publish with subscribers is dominated by `deep_copy`.** 12.9 µs for `large_acq`
  matches the 12.8 µs frame `deep_copy` from the primitives benchmark. The
  `shared_mutex`, hash map lookup, and `unordered_set` dedup add less than 200 ns
  combined.
- **Subscriber scaling is linear in N.** Each additional subscriber adds one
  `deep_copy`. No surprise, but confirms no hidden overhead in the routing logic.
- **Authority filter cost equals `deep_copy` when all channels pass.** The
  `shared_lock` + hash map lookups add ~600 ns on top of the copy for small frames,
  negligible for large frames.
- **Authority filter with no passing channels is 49-121 ns.** This is just the hash map
  lookup cost with no copies. Confirms that copies dominate.
- **Cross-thread CV wake-up adds ~11 µs wall time** beyond the CPU cost. This is OS
  thread scheduling latency, not something we can optimize in the bus code.

## 7.2 - End-to-End Path

Full path: `bus::Writer::write` -> `Bus::publish` -> `Subscription` ->
``control::States`::filter`.

| Workload             | Time      | Throughput |
| -------------------- | --------- | ---------- |
| `small_cmd` (10 B)   | 1,834 ns  | 20.8 MiB/s |
| `medium` (40 kB)     | 8,009 ns  | 4.7 GiB/s  |
| `large_acq` (480 kB) | 72,093 ns | 6.3 GiB/s  |

## 7.3 - Cost Breakdown (`large_acq`, 480 kB)

The end-to-end time of ~72 µs breaks down as follows:

| Component                                 | Cost   | % of total |
| ----------------------------------------- | ------ | ---------- |
| `deep_copy` in `Bus::publish`             | ~13 µs | 18%        |
| `deep_copy` in mock Core writer           | ~13 µs | 18%        |
| `deep_copy` in `control::States::filter`  | ~13 µs | 18%        |
| Frame construction (make_frame in mock)   | ~13 µs | 18%        |
| Mutex + deque + hash map + dedup overhead | ~2 µs  | 3%         |
| Measurement overhead / other              | ~18 µs | 25%        |

The three `deep_copy` operations account for roughly 54% of the measured end-to-end
time. In production, the mock Core writer's `deep_copy` is replaced by Protobuf
serialization (comparable cost), so the production breakdown is similar.

## 7.4 - Optimization: Move-Based Authority Filter

The first optimization target was the authority filter's redundant `deep_copy()`. The
bus already deep-copies frames in `publish()` to give each subscriber its own copy. The
subscriber pops the frame by move (zero cost). The authority filter then deep copied
every passing series again to build a filtered frame. This second copy was unnecessary
since the subscriber already owns the frame exclusively.

The fix adds a move overload `filter(Frame&&, Subject)` that:

- **All pass (common case):** Returns the input frame by move. Zero copies.
- **Partial pass:** Builds a new frame, moving (not copying) passing series from the
  input.
- **None pass:** Returns empty. No copies.

`bus::Streamer::read()` now calls `filter(std::move(local), subject)` instead of
`filter(local, subject)`.

### Results (`large_acq`, 480 kB)

Authority filter comparison:

| Scenario  | Copy (ns) | Move (ns) | Speedup |
| --------- | --------- | --------- | ------- |
| All pass  | 13,411    | 984       | 13.6x   |
| Half pass | 6,945     | 1,572     | 4.4x    |
| None pass | 120       | 120\*     | 1x      |

\*Move benchmark shows ~1,117 ns due to `PauseTiming`/`ResumeTiming` harness overhead
(frame must be reconstructed each iteration since the move consumes it). The actual
filter logic for none-pass is ~120 ns in both cases.

End-to-end comparison:

| Workload             | Before (ns) | After (ns) | Improvement |
| -------------------- | ----------- | ---------- | ----------- |
| `small_cmd` (10 B)   | 1,834       | 993        | -46%        |
| `medium` (40 kB)     | 8,009       | 5,981      | -25%        |
| `large_acq` (480 kB) | 72,093      | 58,938     | -18%        |

The `large_acq` improvement of ~14 µs matches the predicted ~13 µs savings from
eliminating one `deep_copy`. The `small_cmd` improvement is proportionally larger
because the filter's per-channel overhead (hash map lookups, frame construction) was a
larger fraction of the total cost at small sizes.

### Updated Cost Breakdown (`large_acq`, 480 kB, after move filter)

| Component                                    | Cost      | % of total     |
| -------------------------------------------- | --------- | -------------- |
| `deep_copy` in `Bus::publish`                | ~13 µs    | 22%            |
| `deep_copy` in mock Core writer              | ~13 µs    | 22%            |
| ~~`deep_copy` in `control::States::filter`~~ | ~~13 µs~~ | **eliminated** |
| Move in `control::States::filter`            | ~1 µs     | 2%             |
| Frame construction (`make_frame` in mock)    | ~13 µs    | 22%            |
| Mutex + deque + hash map + dedup overhead    | ~2 µs     | 3%             |
| Measurement overhead / other                 | ~17 µs    | 29%            |

## 7.5 - Optimization: Copy-on-Write Series Data

The remaining `deep_copy` in `Bus::publish()` existed because the publisher still needs
the frame for `server->write(fr)` after publishing. A traditional `deep_copy` was
required to give subscribers independent data. However, after the transform chain runs,
frame data is never mutated again. The bus subscriber reads it. The Core writer reads
it. The authority filter reads it. The `Sink` reads it. All read-only.

The fix changes `Series::data_` from `unique_ptr<byte[]>` to `shared_ptr<byte[]>` and
adds a `shallow_copy()` method that shares the underlying data buffer via reference
counting instead of `memcpy`. A copy-on-write mechanism (`ensure_exclusive()`) is called
at the top of every mutation method. If `use_count() > 1`, it materializes a private
copy before mutating. Since mutations only happen in the transform chain (before the
frame reaches the bus), the copy-on-write check is always a no-op in the hot path (one
atomic load returning 1).

### Changes

- `Series::data_`: `unique_ptr<byte[]>` -> `shared_ptr<byte[]>` (mutable for
  copy-on-write)
- `Series::shallow_copy()`: shares data buffer, ~4 ns constant regardless of size
- `Series::ensure_exclusive()`: copy-on-write check before mutations (~1 ns atomic load)
- `Frame::shallow_copy()`: shallow-copies all series, ~230 ns for 30 channels
- `Bus::publish()`: uses `frame.shallow_copy()` instead of `frame.deep_copy()`
- All mutation methods (`set`, `write`, `write_casted`, `resize`, `fill_from`,
  `apply_numeric_op`) call `ensure_exclusive()` for correctness

### Results

Series copy comparison:

| Size   | `deep_copy` (ns) | `shallow_copy`s (ns) | Speedup |
| ------ | ---------------- | -------------------- | ------- |
| 32 B   | 55               | 3.8                  | 14x     |
| 16 kB  | 507              | 3.8                  | 133x    |
| 64 kB  | 1,367            | 3.8                  | 360x    |
| 480 kB | 8,366            | 3.8                  | 2,200x  |

Frame copy comparison:

| Workload             | `deep_copy` (ns) | `shallow_copy` (ns) | Speedup |
| -------------------- | ---------------- | ------------------- | ------- |
| `single_f64` (8 B)   | 135              | 87                  | 1.6x    |
| `small_cmd` (10 B)   | 651              | 127                 | 5x      |
| `medium` (40 kB)     | 1,480            | 127                 | 12x     |
| `large_acq` (480 kB) | 12,831           | 230                 | 56x     |

Bus publish with 1 subscriber:

| Workload             | `deep_copy` (ns) | `shallow_copy` (ns) | Speedup |
| -------------------- | ---------------- | ------------------- | ------- |
| `small_cmd` (10 B)   | 574              | 273                 | 2.1x    |
| `medium` (40 kB)     | 1,535            | 277                 | 5.5x    |
| `large_acq` (480 kB) | 12,913           | 473                 | **27x** |

End-to-end comparison (all optimizations combined):

| Workload             | Baseline (ns) | Final (ns) | Improvement |
| -------------------- | ------------- | ---------- | ----------- |
| `small_cmd` (10 B)   | 1,834         | 866        | **-53%**    |
| `medium` (40 kB)     | 8,009         | 5,379      | **-33%**    |
| `large_acq` (480 kB) | 72,093        | 48,168     | **-33%**    |

### Final Cost Breakdown (`large_acq`, 480 kB)

| Component                                 | Cost      | % of total     |
| ----------------------------------------- | --------- | -------------- |
| ~~`deep_copy` in `Bus::publish`~~         | ~~13 µs~~ | **eliminated** |
| `shallow_copy` in `Bus::publish`          | ~0.5 µs   | 1%             |
| `deep_copy` in mock Core writer           | ~13 µs    | 27%            |
| Move in `control::States`::filter         | ~1 µs     | 2%             |
| Frame construction (`make_frame` in mock) | ~13 µs    | 27%            |
| Mutex + deque + hash map + dedup overhead | ~2 µs     | 4%             |
| Measurement overhead / other              | ~19 µs    | 39%            |

In production, the mock Core writer's `deep_copy` is replaced by Protobuf serialization
(comparable cost). The bus-specific overhead is now ~1.5 µs total (`shallow_copy` + move
filter + routing), down from ~26 µs at baseline. The bus adds less than 0.2% overhead to
a 1 ms control loop budget.

## 7.6 - Remaining Bottlenecks

The bus copy path is effectively solved. Remaining costs in the end-to-end benchmark are
dominated by the mock Core writer (which `deep_copy`s for test recording) and frame
construction overhead in the benchmark harness. Neither is present in the production hot
path.

In production, the only remaining significant cost is Protobuf serialization for the
Core write (~13 µs for large frames). This is unavoidable since data must reach the Core
for persistence and relay. It runs on the acquisition pipeline thread and does not block
the local bus delivery path.

Non-targets (confirmed by data, not worth optimizing):

- **Mutex/lock overhead**: ~200 ns combined. Not a bottleneck.
- **Hash map routing**: Sub-microsecond. Not a bottleneck.
- **`unordered_set` dedup**: Negligible. Not a bottleneck.
- **CV wake-up latency**: ~11 µs wall, but this is OS scheduler cost. Not addressable in
  user-space code (and only matters for cross-thread latency, not throughput).

## 7.6 - Platform Considerations

These benchmarks were run on Apple M4 Max (ARM64) with large L2 caches (4096 KiB per
core). Expected differences on target platforms:

- **x86_64 Linux** (Core deployments): Similar or slightly better `memcpy` throughput
  due to AVX/AVX-512 on modern Xeon/EPYC. Relative rankings unchanged.
- **ARM64 NI Linux Real-Time** (cRIO): Significantly lower memory bandwidth and smaller
  caches. The 480 kB `large_acq` frame exceeds typical embedded L2 sizes, so `deep_copy`
  costs will be 2-5x higher. This makes the copy elimination optimizations even more
  impactful on the primary real-time target.
- **x86_64 Windows** (lab workstations): Comparable to Linux x86_64. CRT allocator may
  differ but the relative cost structure is the same.

The key insight is architecture-independent: **copies dominate, synchronization is
cheap.** This holds across all targets because memcpy scales with data size while
mutex/CV operations have fixed cost.

# 8 - Open Problems

## 8.0 - The Dual-Filter Authority Gap

### 8.0.0 - Problem Statement

The bus introduces two independent frame filters that, when combined, can create a
window where the wrong controller commands hardware or the correct controller receives
nothing.

**Filter 1 (Core-side, coarse):** The streamer's `ExcludeGroups` drops all frames from
the Driver's own group on the Core relay path. This prevents duplicate delivery of
frames that were already routed via the local bus. It operates per-group and has no
knowledge of per-channel authority.

**Filter 2 (client-side, fine):** The `control::States` filter on the bus streamer drops
local bus frames for channels where the consumer's subject does not hold authority. This
is the local replacement for Cesium's control gate. It is eventually consistent with a
staleness window of 1-5 ms on loopback.

These two filters are not coordinated. Together they can block all delivery paths during
an authority transition.

### 8.0.1 - The Scenario: Hotfire with Abort and Manual Override

A realistic test stand deployment has three controllers competing for the same actuator
channels (e.g., `main_fuel_valve`, `main_ox_valve`, `igniter`):

1. **Nominal Hotfire** (Arc task, local, authority=100): Runs the ignition sequence,
   opening valves in timed order, holding steady state, then executing a nominal
   shutdown.
2. **Abort Listener** (Arc task, local, authority=255): Monitors overpressure,
   temperature, and leak sensors. If any threshold is exceeded, slams all valves shut
   immediately. Must always win authority because it is the safety backstop.
3. **Manual Override** (Console schematic, remote, authority=200): Allows an operator to
   take direct control of any valve for pre-test checkout, manual safing, or overriding
   a stuck sequence.

All three controllers write to the same LabJack output task's command channels. The
Nominal Hotfire and Abort Listener run as Arc tasks on the same Driver rack. The Manual
Override runs in Console on a separate machine.

The authority hierarchy is: Abort (255) > Manual Override (200) > Nominal Hotfire (100).

#### Scenario A: Abort During Nominal Operation

The most safety-critical scenario. The hotfire sequence is running at 1 kHz. The abort
listener detects an overpressure condition and must close all valves within a bounded
time window (e.g., 5 ms).

```
T=0    Abort Listener detects overpressure
       Calls set_authority(valve channels, authority=255) [fire-and-forget]
       Writes valve_close commands to its acquisition pipeline
T=1    Core transfers authority: Hotfire (100) → Abort (255)
       Gate updated immediately. ControlUpdate queued to digest inlet.
T=2    Core sends response. Driver discards it (ack=false).
T=3    Abort's Writer.write() calls states->filter(frame, abort_subject)
       Mirror still says Hotfire holds authority
       Abort's subject != Hotfire's subject → FRAME NOT PUBLISHED TO BUS
T=4    Hotfire's Writer.write() calls states->filter(frame, hotfire_subject)
       Mirror says Hotfire holds authority → FRAME PUBLISHED TO BUS
       LabJack receives hotfire's valve_open command
       VALVE REMAINS OPEN DURING OVERPRESSURE
T=5    `control::States` receives update from relay, applies transfer
       Subsequent abort commands now pass the filter
       Valve finally closes
```

Between T=1 and T=5 (1-5 ms on loopback):

- The abort listener's commands are rejected by the stale authority filter.
- The hotfire sequence's commands continue reaching hardware.
- The valve stays open during an overpressure condition.
- The Core path cannot help because `ExcludeGroups` blocks all same-rack frames.

For a rocket engine test stand, 1-5 ms of continued fuel flow during an overpressure
event can mean the difference between a controlled shutdown and a catastrophic failure.

#### Scenario B: Operator Takeover During Nominal Operation

The operator grabs control from Console to manually safe the system.

```
T=0    Operator sets authority=200 on valve channels from Console
T=1    Core transfers authority: Hotfire (100) → Operator (200)
       ControlUpdate queued to digest inlet
T=2    Operator sends valve_close from Console schematic
       Console is REMOTE (different machine, different group)
       Operator's frames arrive at Core relay
       LabJack's Core streamer: ExcludeGroups=[rack_key]
       Operator's group != rack_key → NOT excluded → FRAME DELIVERED
T=3    Hotfire's Writer.write() checks mirror: Hotfire still holds authority
       Frame published to bus → LabJack receives it
       LabJack receives BOTH operator's close AND hotfire's open
```

Between T=1 and mirror update:

- The operator's commands arrive via the Core path (correctly, because the operator is
  remote and not subject to `ExcludeGroups`).
- But the hotfire's commands also arrive via the bus path (incorrectly, because the
  mirror is stale).
- The LabJack receives conflicting commands from two controllers simultaneously. The
  outcome depends on which command the control pipeline processes last in each cycle.

This is less dangerous than Scenario A (the operator's commands do get through), but the
conflicting commands create unpredictable actuator behavior during the transition. This
is a serious problem for a safety-critical system.

#### Scenario C: Abort While Operator Has Manual Control

The operator is manually controlling valves (authority=200). The abort listener detects
a hazard.

```
T=0    Abort Listener detects hazard
       Calls set_authority(authority=255) [fire-and-forget]
       Writes valve_close commands
T=1    Core transfers authority: Operator (200) → Abort (255)
T=2    Abort's Writer.write() checks mirror: Operator holds authority
       Abort's subject != Operator's subject → FRAME NOT PUBLISHED TO BUS
T=3    Operator's commands arrive via Core relay
       ExcludeGroups=[rack_key], Operator's group != rack_key → DELIVERED
       Operator (who may not be watching) continues commanding
T=4    Mirror updates. Abort's commands start reaching hardware.
```

Between T=1 and T=4:

- The abort listener's commands are rejected from both paths (bus filtered by stale
  mirror, Core blocked by `ExcludeGroups`).
- The operator's commands continue arriving because the operator is remote.
- The safety system is completely ineffective during the staleness window.

This is the worst combined failure: the highest-priority safety mechanism (abort) is
blocked while a lower-priority controller (operator) continues commanding hardware.

### 8.0.2 - Asymmetry Between Local and Remote Controllers

The scenarios above reveal a structural asymmetry in how the dual-filter system treats
local and remote controllers:

**Remote controllers (Console)** are unaffected by `ExcludeGroups` because they have a
different group. Their frames always reach the Core-side streamer. They can take
authority and their commands arrive immediately via the Core path.

**Local controllers (Arc tasks on the same rack)** are subject to both filters. Their
Core path is blocked by `ExcludeGroups` (same group), and their bus path depends on the
authority mirror being current. During transitions, local controllers can be completely
blocked from both paths.

This means authority transitions FROM a remote controller TO a local controller are the
worst case. The local controller gains authority on the Core but cannot deliver commands
through either path until the mirror catches up. The remote controller's commands may
continue arriving during the window.

Authority transitions between two local controllers also exhibit the gap: the old local
controller's commands pass the stale filter while the new local controller's commands
are rejected from both paths.

Authority transitions FROM a local controller TO a remote controller are less severe:
the remote controller's commands arrive via the Core path immediately. The local
controller's commands may leak through the bus for the duration of the window, creating
brief conflicting commands rather than a complete blockout.

### 8.0.3 - Severity

The staleness window is 1-5 ms on a co-located deployment and 10-50 ms on a multi-node
deployment. Its duration depends on:

- Digest writer goroutine scheduling (~microseconds)
- Relay fan-out (~microseconds)
- Network/loopback transport (50-200 µs loopback, 1-50 ms remote)
- JSON parsing and mirror lock acquisition (~microseconds)

The impact during this window ranges from annoying (brief conflicting commands during
operator takeover) to safety-critical (abort commands blocked while valves remain open).
The severity depends on which controller is gaining authority and whether it is local or
remote.

### 8.0.4 - Root Cause

The root cause is that authority is enforced in two places with different consistency
guarantees, and neither can compensate for the other's failure mode:

1. The Core-side group exclusion is always correct but too coarse. It blocks all
   same-rack frames regardless of per-channel authority, creating a hard dependency on
   the bus path being correct.
2. The client-side authority filter is per-channel but eventually consistent. During
   transitions, it makes decisions based on stale state.

The group exclusion was designed to prevent duplicate delivery. It assumes the bus path
is authoritative for same-rack frames. But the bus path relies on the authority mirror,
which lags the Core. The combination creates a window where the bus path is wrong and
the Core path is blocked.

The asymmetry between local and remote controllers (Section 8.0.2) is a direct
consequence of this design: `ExcludeGroups` only penalizes same-rack traffic, so only
local controllers suffer the dual-filter blockout.

### 8.0.5 - Comparison with Current Pipeline

In the current pipeline (without bypass), all three controllers write through the Core.
Cesium's control gate validates authority synchronously on every write. When the abort
listener acquires authority, the gate immediately rejects the hotfire sequence's writes
and accepts the abort's writes. The valve closes on the next write cycle. There is no
staleness window because the gate is the single, synchronous enforcement point.

The bypass replaces this synchronous gate with an asynchronous mirror. The tradeoff is
latency (the mirror avoids a Core round-trip per frame) for consistency (the mirror may
be stale during transitions). This tradeoff is acceptable for the data path (frames per
cycle) but creates a safety gap for authority transitions, particularly when the gaining
controller is local and the losing controller is local or remote.

## 8.1 - Proposed Solution: Short-Circuit Authority Increases Through the Mirror

### 8.1.0 - Key Observation

Authority increases are locally decidable. When a task calls `set_authority` with
authority strictly greater than the current holder's, the outcome is deterministic: the
caller wins. The Core will confirm this, but the Driver already has enough information
to predict the result. Equal authority preserves the current holder (Core uses
position-based tiebreak favoring the earlier gate), so the Driver must not
optimistically apply equal-authority updates.

Authority decreases are not locally decidable. When authority drops, the mirror cannot
determine who takes over next because that depends on what other gates exist on the
Core. Decreases must wait for the Core's relay update.

### 8.1.1 - The Fix

When the bypass Writer's `set_authority` is called, apply the authority change directly
to `control::States` for any channel where the new authority is strictly greater than
the current holder's. Then forward the request to the Core as before (fire-and-forget).
The state update happens in the same function call, before `set_authority` returns.

The timeline becomes:

```
T=0    Task B calls set_authority(authority=255)
       Writer.set_authority calls states->apply_increase for each channel
       States now reflects Task B as authority holder (255 > anything)
T=0+ε  set_authority forwards request to Core (fire-and-forget)
T=0+ε  Next Writer.write(): states->filter uses correct state
       Task B's commands are published to the bus immediately
T=1-5  Core relay arrives, states->apply() overwrites with same state (idempotent)
```

The cost is zero additional latency. The mirror update is a hash map write under a
mutex, completing in nanoseconds. No network round-trip is required. No protocol changes
are needed.

### 8.1.2 - Implementation

1. **`control::States::apply_increase`.** Takes a subject, channel key, and authority
   level. If the incoming authority is strictly greater than the current state for that
   channel (or no state exists), the mirror is updated under a write lock. Equal or
   lower authority is ignored, matching the Core's position-based tiebreak where the
   earlier gate wins ties.

2. **Bypass `Writer` holds `control::States` reference.** The `Writer` is constructed
   with a `shared_ptr<control::States>`, the `WriterConfig` (which carries the subject
   and channel keys). When `set_authority` is called with empty keys (meaning "all
   channels"), the Writer expands to the config's full channel list. For each channel,
   it calls `apply_increase` before forwarding the request to the Core.

3. **Bypass `WriterFactory` threads `control::States`.** The factory accepts a
   `shared_ptr<control::States>` and passes it through to each Writer it creates.

4. **Factory guard on bus availability.** If the bus is nullptr (no bypass configured),
   the helper functions fall back to the direct Core factories.

### 8.1.3 - How This Addresses Each Scenario

**Scenario A (Abort during hotfire):** The abort listener calls `set_authority` with
authority 255. The Writer calls `apply_increase(abort, channel, 255)` for each channel.
Since 255 > 200 (hotfire's authority), the mirror updates immediately. The abort's next
valve_close command is published to the bus and passes the authority filter. The
hotfire's commands are now filtered out. Zero latency between authority change and first
correct command.

**Scenario B (Operator takeover):** The operator is remote, so the authority change
originates from outside the Driver. The mirror updates via the relay path with 1-5 ms
staleness. During this window, the hotfire's commands may still pass through the bus.
This scenario is unchanged from the baseline and requires Core-initiated mirror
notifications to fully address (see Section 8.1.5).

**Scenario C (Abort while operator has control):** Same as Scenario A. The abort
listener is local and calls `set_authority` with authority 255. The mirror updates
immediately, the abort's commands pass through, and the operator's commands are filtered
on the bus path. The operator's commands still reach hardware via the Core path
(unaffected by `ExcludeGroups` since they originate from a different group), but the
abort's higher authority at the Core's control gate takes precedence.

### 8.1.4 - Correctness Properties

**Authority=255 (abort) is guaranteed correct locally.** No gate can outrank it, unless
another gate already exists holding authority 255. `apply_increase` computes
`255 > anything` which is always true. The mirror is correct before the next frame.

**Idempotent with relay.** When the Core's relay update arrives 1-5 ms later, `apply()`
overwrites with the same state. No conflict because the relay carries the authoritative
transfer which matches what was optimistically applied.

**Decreases are safe.** `apply_increase` is a no-op when incoming authority <= current.
The Core handles decreases and the relay corrects the mirror. No optimistic update is
made for authority drops.

**Equal authority preserves holder.** The `>=` comparison in `apply_increase` means
equal authority does not trigger an update. This matches the Core's behavior where the
earlier gate wins ties.

**Rare edge case: two tasks set authority=255 simultaneously.** Both think they won
locally. The relay corrects the loser within 1-5 ms. This is strictly better than the
baseline where both are stale for the full window.

### 8.1.5 - Remote Authority Changes and the Existing Control Channel

Scenario B reveals a gap: when a remote controller takes authority, local mirrors learn
about it through the relay path with 1-5 ms of staleness. During this window, the old
local controller's commands leak through the bus.

The Driver already subscribes to `sy_node_{N}_control` via the `control::States` class,
which parses JSON-encoded `ControlUpdate` frames from the relay and applies transfers to
its internal mirror. This is the same mechanism the Core uses to notify all clients of
authority changes, and it is the path through which the Driver learns about remote
authority transitions.

A dedicated push mechanism would use the same transport and experience the same latency.
The `sy_node_{N}_control` channel already delivers authority transfers with
relay-bounded latency (sub-millisecond on loopback, low single-digit milliseconds across
nodes). The 1-5 ms staleness for remote-initiated transitions comes from the relay path,
not from missing notification infrastructure.

The short-circuit `apply_increase` (Section 8.1.1) eliminates staleness for the critical
case: locally initiated authority increases (abort). Remote-initiated transitions are
less time-sensitive because the remote controller's commands arrive via the Core path
(unaffected by `ExcludeGroups`), so hardware receives the correct commands even while
the local mirror is stale. The only consequence is a brief window of conflicting
commands from the old local controller, which resolves when the mirror catches up.

### 8.1.6 - Alternative Considered: Synchronous Core Round-Trip

An earlier design made `set_authority` synchronous (`ack=true`), waiting for the Core's
response before returning. The response would carry the authority transfer, and the
Driver would apply it to the mirror directly. This adds one network round-trip per
authority change (~100-500 µs on loopback).

The short-circuit approach is preferred because it adds zero latency, requires no
protocol changes (no need to extend `FrameWriterResponse` with transfer details), and
handles the critical abort scenario with the same correctness guarantee. The synchronous
approach would additionally help with authority decreases (the Driver would learn
immediately who takes over), but this is not required for the safety-critical abort use
case where the abort always increases authority.
