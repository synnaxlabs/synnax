# 11 - Alamos Instrumentation

- **Feature Name**: Alamos Instrumentation
- **Start Date**: 2023-04-01
- **Authors**: Emiliano Bonilla
- **Status**: Complete

# 0 - Summary

As we move towards the Beta release of Synnax, the core architectural components will
begin to solidify, and, for the more stable areas of the codebase, we will shift focus
from building functional to stable and performant software. To improve these qualities,
we'll need to profile the execution state of the cluster from various perspectives. This
includes improving the logging infrastructure, and updating the `alamos` metrics package
to include tracing. Instrumentation also enables our users to monitor their own Synnax
deployments, an essential feature for operations-critical software. In this RFC I
propose a high-level plan for implementing distributed instrumentation across all of
Synnax's components.

# 1 - Vocabulary

- **Alamos** - The Synnax instrumentation library, and the core package for implementing
  this RFC.
- **Instrumentation** - The process of collecting and reporting data about the execution
  state of a Synnax cluster. This includes traces, metrics, and logs.
- **Trace** - A record of the execution of a request through the Synnax cluster.
- **Metric** - A measurement of a specific aspect of the execution of a request.
- **Log** - A record of a specific event that occurred during the execution of a
  request.

# 2 - Motivation

Synnax is in stable Alpha (v0.4.2 as of this writing), and we're beginning to plan the
road to Beta. This involves shifting our mindset from building a technical
proof-of-concept to creating a stable, performant, and maintainable product. These
qualities are not emergent.

Improving these characteristics starts with having a clear, on-demand picture of the
execution state of the entire cluster. Proper observability allows us to find critical
bugs in development, systematically improve performance through iterative optimization
and benchmarking, and monitor the health of our users' deployments.

# 3 - Philosophy

## 3.0 - Instrumentation Classes

When approaching the high-level design of instrumentation, I found it useful to think
about a Synnax cluster as a supply chain structured as a grid. The rows represent areas
of horizontal integration, and specifically layers of the Synnax stack. The bottom
begins with the storage layer, and the top ends with the user interface. The columns
represent areas of vertical integration, or the 'pathways' a request takes through the
stack. For example, frame reads and writes strike different areas of each layer than an
ontology traversal would.

It's important to understand the execution of the cluster from both the vertical and
horizontal perspectives. For the sake of brevity, I've chosen to contract these
directions into `X` and `Y`.

It's relevant to note that approaching a design problem from this perspective is not
new. The design of the Signal package in
[RFC 0004](https://github.com/synnaxlabs/synnax/blob/main/docs/rfc/0004-220623-signal-gr.md)
also discusses issues between application and request-scoped goroutine management,
especially in regard to error handling.

## 3.1 - X Instrumentation

Horizontal, or`X`, instrumentation collects and reports data about different layers (and
partitions of those layers) of the Synnax architecture. Its role is to build a picture
of how a layer behaves over time. For example, we need to measure the growth of the
cesium's range pointer cache over time, track the network load on the distribution
layer, or track the balance of leased KV operations on different nodes over time.

The tracing side of X instrumentation tracks goroutines and processes that aren't
associated with a particular request.

## 3.2 - Y Instrumentation

Y instrumentation collects and reports data about the execution of a specific request. A
perfect example tracks the lifecycle of a frame writer, measuring goroutine creation and
destruction, mean frame sizes, and performance metrics such as serialization and storage
throughput.

Tracing is a quintessential example of Y instrumentation. It tracks the execution of a
request through the Synnax cluster, and is essential for debugging various classes of
issues.

## 3.3 - General Characteristics

Both X and Y instrumentation strictly observe architectural boundaries, meaning that
instrumentation in a layer below cannot be directly dependent on the instrumentation in
a layer above, and should not expose shallow interfaces to layers above.

## 4 - Requirements

Alamos must handle three types of instrumentation: logs, metrics, and traces. These are
widely referred to as the three pillars of observability. I'm omitting an argument for
why these types exist and why each is important; this should be obvious to you, and, if
it isn't, there is extensive literature on the subject.

Instead, I'm focused on describing the specific requirements for each. These are not
organized by pillar, but rather by type of requirement. For example, all three pillars
need some means of persistence; this is all covered within a single section.

## 4.1 - Distribution

(Obviously) Synnax is a distributed system, and, perhaps the most challenging
requirement for an instrumentation system is to provide an aggregated view of the
execution state for several machines.

### 4.1.0 - Instrumentation Must Support Clients

Many features exist above the server-side waterline. Our Python and TypeScript
libraries, Synnax CLI, and User Interfaces play a role in delivering a quality user
experience. When designing a distributed instrumentation system, we must keep in mind
how we tie together the execution state of both the server and client.

### 4.1.1 - Instrumentation Must Distribute Across Nodes

Requests and cluster synchronization tasks regularly span several nodes. To understand
these processes, we must not only understand execution within a node, but also how
execution crosses node boundaries. Supporting distributed tracing is essential.

## 4.2 - Metadata

### 4.2.0 - Categorization

Collecting telemetry is only useful if we can correlate it with metadata about the
cluster's configuration. If we don't know critical information about the cluster, such
as the version of the software, we place ourselves at a significant disadvantage when it
comes to debugging issues and improving performance.

### 4.2.1 - Y Metadata - Tracing

Y metadata is bound to a specific request, and should be viewable at the level of an
individual trace or an aggregated view of all traces. This includes protocols, user
id's, etc.

### 4.2.2 - X Metadata

X metadata is bound to a specific layer, and describes that layer's configuration i.e.
the protocols supported, storage directories, maximum cache sizes etc.

## 4.3 - Filtering

As with any instrumentation system, we should be able to filter the data we collect
depending on the environment we're running in. In a development environment, we focus on
collecting data for debugging and correctness purposes. In a benchmarking environment,
we collect critical performance metrics and traces.

### 4.3.0 - Log Levels

The Alamos log levels mirror those of
[zap](https://pkg.go.dev/go.uber.org/zap#pkg-constants).

<table>
<tr>
<th>Level</th>
<th>Meaning</th>
</tr>
<tr>
<td><code>debug</code></td>
<td>Debugging information for development</td>
</tr>
<tr>
<td><code>info</code></td>
<td>Informational messages that should be included in production logging
</td>
</tr>
<tr>
<td><code>warn</code></td>
<td>Warnings that indicate non-critical issues
</td>
</tr>
<tr>
<td><code>error</code></td>
<td>Errors that indicate the failure of a request or process</td>
</tr>
<tr>
<td><code>fatal</code></td>
<td>Fatal errors that cause the entire node to exit immediately
to prevent data corruption
</td>
</tr>
<tr>
<td><code>dpanic</code></td>
<td>Situations that should panic in development, but are recoverable in production</td>
</tr>
</table>

### 4.3.1 - Trace, Metric, and Report Environments

Unlike logs, traces, metrics, and reports aren't typically bound to a specific level.
Instead, they should be filtered based on the environment they're collected in.

<table>
<tr>
<th>Level</th>
<th>Meaning</th>
</tr>
<tr>
<td><code>debug</code></td>
<td>Debug traces used for evaluating program correctness during development</td>
</tr>
<tr>
<td><code>prod</code></td>
<td>Production traces</td>
</tr>
<td><code>bench</code></td>
<td>Traces specifically used for tracking program performance during benchmarking</td>
</table>

## 4.5 - Persistence

### 4.5.0 - Log Persistence

### 4.5.1 - Trace Persistence

## 4.6 - Development

Instrumentation is as critical in development as it is in production. Along with the
debugger, traces and logs are the primary means of understanding the correctness of our
algorithms. We need to find a way to provide our developers with a means of collecting
and viewing instrumentation data in a development environment. Ideally this does not
involve a cloud hosted server, as aggregating data from all developers would cost a lot
of money and be difficult to navigate. Instead, our developers should be able to run a
self-hosted tracing tool to maintain control over their environment.

## 4.7 - Production

Instrumentation in production allows us to view and collect telemetry from the
deployments of our users. Ideally we'd have a cloud hosted solution that enables us to
detect and debug issues in near-real time.

### 4.7.0 - Privacy and Security

Synnax is designed to operate in scenarios are operations critical and closely
controlled by regulations such as ITAR or EAR. We're also holding confidential data and
personally identifiable information (PII). When collecting telemetry, we **must** ensure
that no sensitive or controlled data is collected or transmitted to our servers.

Telemetry must also be completely opt-in. This has been a hot-topic of debate in the
software world recently, and it's essential for any open source project to specifically
ask for consent before collecting and reporting any telemetry.

# 5 - Design

## 5.0 - The Instrumentation Type

The instrumentation type (`alamos.Instrumentation`) forms the structural core of Alamos'
tooling. It serves as an aggregation of several other types, and is aimed for use as a
'bag of tools' for injecting instrumentation into a given service.

```go
package irrelevant

type Instrumentation struct {
    T *Tracer
    L *Logger
    R *Reporter
}
```

The type's fields are intentionally terse and it's methods are kept unique in order to
avoid potential conflicts when embedding it into a struct.

## 5.1 - Propagation in the X Direction - Embedded Dependency Injection

The instrumentation type should be embedded as the first field in a service's
configuration, or passed as an option to a service's constructor. This allows the
service to access the instrumentation tools as if they were part of the configuration
itself.

```go
package irrelevant

import "github.com/synnaxlabs/alamos"

type MyServiceConfig struct {
    alamos.Instrumentation
}
cfg := MyServiceConfig{}

// The logger is now directly accessible from the configuration.
cfg.L.Debug("Hello World")

```

## 5.2 - Zero Value Operation

Instrumentation operates in the background, and it's footprint should be as small as
possible. Part of this effort involves allowing for zero-value instrumentation to be
valid as no-op instrumentation i.e. the following code is valid:

```go
package irrelevant

ins := alamos.Instrumentation{}

// no-op without panicking
cfg.L.Debug("Hello World")
```

## 5.2 - Trace Propagation in the Y Direction - Contexts

Perhaps the most challenging part of designing this system is finding a sustainable away
to pass instrumentation between services and layers.

The previous version of `alamos` was only focused on instrumenting a layer xly, and, as
a result, metrics and loggers were dependency injected into a particular service. For
example, the storage engine has the following section of its config (pseudo-code):

```go
package irrelevant

import "github.com/synnaxlabs/alamos"

type Config struct {
    // Experiment is the old version of alamos.Instrumentation
    Instrumentation *alamos.Instrumentation
    // ... rest of config
}
```

This pattern is effective, and, most importantly, it's extremely clear. Unfortunately,
this pattern doesn't allow for tracing the path of a request through the system. Let's
say we wanted to add tracing to the `Set` operation on the current key-value interface:

```go
package irrelevant

type Writer interface {
    Set(key []byte, value []byte) error
}
```

The most naive way to instrument this interface is to simply pass the entire
instrumentation struct to the function.

```go
package irrelevant

type Writer interface {
    Set(key []byte, value []byte, instrumentation *Instrumentation) error
}
```

This pattern is clear, but also unsustainable. Requiring every request-scoped function
to take an instrumentation parameter increases interface footprint dramatically, and
chains the service to a specific instrumentation implementation. Remember, we're trying
to keep the instrumentation footprint small. The alternative is to use contexts.

Contexts in go are used for two purposes: cancellation and request-scoped data. The
former is used in virtually every area of the Synnax code base. The latter is
controversial within the go community. On the one hand, it's useful to implicitly attach
dynamic data to a request. On the other hand, implicit code makes it difficult to reason
about execution. A prime example is passing a database transaction through a context.
This reduces interface footprint, but also makes it difficult to understand which parts
of the code base are using the transaction and which are not. As a result, we've
generally avoided using `context.WithValue` in the Synnax codebase. It's a pattern for
more difficult to abuse if it's not available.

Tracing poses a particularly powerful use case for context propagation. Tracing is
required by almost every area of the codebase. The more an implicit pattern is used the
more explicit it becomes. This reduces the consequences of passing instrumentation
through a context, as it's usage will remain clear to the reader. We can modify our
key-value interface to look like this:

```go
package irrelevant

type Writer interface {
    Set(ctx context.Context, key []byte, value []byte) error
}
```

Inside the implementation we can pull the instrumentation from the context and use it
for whatever purpose we need.

```go
package irrelevant

type writerConfig struct {
    alamos.Instrumentation
}

type writer struct {
    writerConfig
}

func (w *writer) Set(ctx context.Context, key []byte, value []byte) error {
    instrumentation := alamos.FromContext(ctx)
    // ... rest of function
    ctx, span := instrumentation.T.Trace(ctx, "Set")
    defer span.End()
}
```

This pattern allows us to effectively move a trace through the call stack, but it
creates a separation between X and Y instrumentation. We're now passing Y
instrumentation down through the call stack, so where should we use X instrumentation?
The solution is to only propagate traces through the callstack, and rely on the
instrumentation stored in the service configuration to extend those traces. Our
implementation now looks like this:

```go
package irrelevant

type writerConfig struct {
    alamos.Instrumentation
}

type writer struct {
    writerConfig
}

func (w *writer) Set(ctx context.Context, key []byte, value []byte) error {
    ctx, span := w.T.Trace(ctx, "Set")
    defer span.End()
    // ... rest of function
}
```

This pattern allows us to develop an integrated view of X and Y instrumentation. We can
bind traces to both request verticals and layer bound-services. The most notable method
for doing is adding the instrumentation key to the name of the span. So, instead of
`Set` our span would be named `writer.Set`. This allows us to evaluate the performance
of a specific service over time, giving valuable insight into where our performance
improvement efforts should be focused.

## 5.3 - Application Critical Metrics

You may have noticed metrics have been largely left out of this RFC. Metrics add
additional layers of complexity over logs and traces. There are cases where we'll want
to leverage metrics to make operational decisions in real-time. The question is, should
Alamos be the system responsible for collecting and storing these metrics, or should we
use a more explicit method such as the one used by CockroachDB's
[pebble](https://pkg.go.dev/github.com/cockroachdb/pebble#Metrics), which involves
explicitly declaring an interface or struct that a service exposes publicly.

As stated previously, Alamos is intended to maintain a minimal footprint, and is
designed to fill the traditional 'observability' role of simply collecting information
without affecting the application's behavior. Using it to enable operational decisions
is a slippery slope. I'm worried about gradually extending its footprint to the point
where we indefinitely build application-wide dependencies on it.

The alternative is to use a more explicit method of exposing metrics. For example, we
eventually need to collect information on latency between nodes. Piggybacking on top of
Aspen's gossip system is an ideal way to do this. We can use lightweight, regular
payloads to accumulate latency information. We can use this latency information to
throttle the priority of less important services in operations critical scenarios. This
process could occur in different parts of the code base that are unrelated to aspen. As
a result, Aspen would expose this metric through its top level interface, such as a
struct, interface, or observable. The issue here is that we'd need to explicitly define
and implement infrastructure for exposing metrics.

Alamos would not be out of the picture entirely. It would still be considered a consumer
of these explicit metrics, and would be responsible for exporting them to uptrace. This
decision plays well with many of the other architectural choices we've made, such as the
ontology.

## 5.4 - Distribution

Propagating metrics across services and client/server boundaries is easier than I
originally assumed. We only need to attach a trace id to the metadata of each OTN
message. At a high level, this involves attaching a `Propagate` and `Depropagate` method
to the`alamos.Tracer` type, as follows:

```go
package alamos

type Tracer interface {
    // ... rest of interface
    Propagate(ctx context.Context, carrier Carrier) error
    Depropagate(ctx context.Context, carrier Carrier) error
}
```

`Carrier` is a simple string key-value setter and getter that a transport must implement
in order to propagate headers. This is trivial.

# 6 - Future Work

## 6.0 - Analytics

As I started working on this RFC, I began to think about the relationship between
software observability and analytics, such as tracking the number of active users and
what features are the most popular. This is obviously relevant for improving the
product, and defining the actual requirements for what we'd like to collect is the
subject of a different document.

The (slightly) more relevant question for this RFC is whether analytics should
(eventually) be included in the scope of Alamos. Observability and analytics
instrumentation share enough similarities that it's worth seriously considering
integrating them as part of the alamos interface.

This is a topic that has few implications on the design of the system, and should be
addressed in a future RFC.
