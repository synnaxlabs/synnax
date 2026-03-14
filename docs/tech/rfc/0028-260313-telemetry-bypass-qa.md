# 28 - Telemetry Bypass - Architectural Q&A

This document records the architectural decisions made during the design of the
telemetry bypass. Each entry captures a question, the decision, and the reasoning.

## Q1 - Unified Task vs. Local Bus

**Question**: Should the bypass be a single unified task that owns both the hardware
Source and Sink directly, or a local bus between existing tasks?

**Decision**: Local bus. The bypass is middleware that sits inside the existing pipeline
infrastructure, between the Source/Sink and the network-facing Writer/Streamer. It
intercepts frames at the pipeline level without touching Source/Sink implementations.
Hardware integrations do not need to change.

## Q2 - Arc Runtime Relationship to the Bus

**Question**: Should the Arc runtime be aware of the bus, or should it be a plain
consumer/producer with no knowledge of the routing layer?

**Decision**: The Arc runtime has no awareness of the bus. Its interface does not
change. It reads frames from an inlet and writes frames to an outlet. The bus is wired
to the other end of those inlets/outlets. The Arc runtime is a pure data transform:
frames in, frames out.

## Q3 - Server Forwarding Path

**Question**: Does the acquisition middleware forward to the server directly (bus is a
parallel tap), or does it publish exclusively to the bus with a separate server
forwarder as a bus participant?

**Decision**: Open. Two options with clear tradeoffs:

- **Option A**: Middleware forwards to server directly, bus is a parallel tap. Server
  path is unchanged, simple, safe. Middleware has two output paths.
- **Option B**: Middleware publishes exclusively to bus, server forwarder is a bus
  participant. Cleaner separation, forwarder can be independently configured (rate,
  batching, backpressure). Server path depends on bus health.

## Q4 - Bus Consumer Semantics

**Question**: Should the bus enforce a specific synchronization model (ring buffer,
queue, etc.) for consumers?

**Decision**: No. The bus defines the routing contract: frames go in tagged with channel
keys, frames come out to subscribers of those keys. How a subscriber consumes is the
subscriber's responsibility. Each subscriber wraps its own consumption strategy behind
the subscription interface. The Arc runtime might use a latest-value slot. A server
forwarder might use a buffered queue. The bus is agnostic.

## Q5 - Authority Awareness

**Question**: Should the middleware be authority-aware, or should it forward everything
and let the Sink figure it out?

**Decision**: The bus is authority-aware. This is the key piece of the bus. It
subscribes to the control state channel, maintains a local mirror of per-channel
authority, and filters frames at route time. Unauthorized channels are filtered before
they reach the Sink. This replaces Cesium's gate as the enforcement point on the local
path.

## Q6 - Bypass Authority Registration

**Question**: Does the bypass need to register its own authority with the server, or do
existing tasks handle this?

**Decision**: The bypass does not open any writers. Existing tasks still open their
writers and streamers to the server. Authority registration, gate management, and
control state all flow through the normal server path. The bus is a filter that observes
authority state and makes local routing decisions. It does not participate in the
authority system.

## Q7 - Which Arc Runtime Executes

**Question**: Does the Arc runtime execute on the server (Go/wazero) with the bus only
short-circuiting data, or on the driver (C++/Wasmtime) for fully local execution?

**Decision**: The C++ Arc runtime on the driver. The Go runtime on the server is a
separate deployment target for non-bypass Arc tasks. They do not execute simultaneously.
The driver-side runtime connects to the bus as a consumer/producer and opens a
server-side writer via the C++ client purely to hold a gate for authority.

## Q8 - Dual-Path Writes and Deduplication

**Question**: Arc writes flow through both the bus (local) and the server writer
(persistence). When both paths deliver the same command to the same local Sink, how is
deduplication handled?

**Decision**: Deduplication is needed for channels local to this driver where both the
bus and the server path terminate at the same Sink. The exact mechanism (control
middleware vs. bus-level) is an open design question. For channels on remote drivers,
only the server path can reach them, so no deduplication is needed.

## Q9 - Server Forwarding Rate

**Question**: Should the server forwarding path carry the full sample rate, or can it
decouple from the acquisition rate?

**Decision**: Out of scope. The acquisition pipeline already writes at full rate.
Nothing about the bypass changes that concern. This is a separate, independent problem.

## Q10 - Channel Key Discovery

**Question**: How does the bus learn which channel keys are locally routable?

**Decision**: Dynamic discovery. Middlewares register their channel keys when tasks
start and unregister when tasks stop. The bus builds its routing table automatically
from these registrations. The exact discovery interface is an open design question.

## Q11 - Middleware Injection Point

**Question**: Where does the middleware get injected: the pipeline layer or the common
task layer?

**Decision**: The common task layer (`driver/common/read_task.h`,
`driver/common/write_task.h`). Every hardware integration (Modbus, LabJack, NI, OPC UA,
HTTP, EtherCAT) uses this common infrastructure. Injecting at this layer covers all six
integrations without touching hardware-specific code. The only task that does not use
common infrastructure is Arc itself, which is the consumer on the bus, not a hardware
integration.

## Q12 - Bypass as Opt-In Configuration

**Question**: Is the bypass opt-in per task, configured at the Arc level, or configured
at a higher level?

**Decision**: None of the above. The bypass is not opt-in. It is an internal,
intelligent mechanism. The bus is always present. The middleware is always in the
pipeline. Local routing happens automatically when local routes exist. When no local
routes exist, frames pass through to the server as they always have. No configuration
required.

## Q13 - Zero-Cost When No Local Routes

**Question**: Is the middleware overhead acceptable as a universal tax on every task?

**Decision**: The routing table is built at task startup, not at frame time. When no
local routes exist for a middleware's channel keys, the middleware knows this at
registration time and fast-paths to a direct passthrough. Per-frame cost in the no-route
case is effectively zero. The routing table only changes when tasks start or stop.

## Q14 - Simultaneous Server and Driver Arc Execution

**Question**: Do the Go and C++ Arc runtimes execute the same program simultaneously?

**Decision**: No. There is one deployment target per Arc task. When the task targets the
driver-side C++ runtime, the server-side Go runtime does not execute it. This is a
deployment concern, not a bypass architecture concern.

## Q15 - Bus Failure and Recovery

**Question**: Does bus failure need its own recovery mechanism?

**Decision**: Out of scope for MVP.

## Q16 - Arc Runtime STL Completeness

**Question**: Does the system need capability negotiation for missing C++ STL modules?

**Decision**: Out of scope. The Arc runtime's STL completeness is an independent
implementation detail unrelated to the bypass architecture.
