# 11 - Alamos Instrumentation

**Feature Name**: Alamos Instrumentation <br />
**Start Date**: 2023-04-01 <br />
**Authors**: emilbon99 <br />
**Status**: Draft <br />

# 0 - Summary

As we move towards the Beta release of Synnax, the core architectural components
will begin to solidify, and, for the more stable components, we will shift focus
from building functional to stable and performant software. To improve these qualities,
we will need to profile the execution state of the cluster from various view points.
Instrumentation also allows our users to monitor their own Synnax deployments, an
essential feature for operations-critical software.

In this RFC I propose a high-level plan for implementing distributed instrumentation
across all of Synnax's components as well as providing a means for exporting and
accessing
this data.

# 1 - Vocabulary

**Alamos** - The Synnax instrumentation library, and the core package for implementing
this RFC. <br />
**Instrumentation** - The process of collecting and reporting data about the execution
state of a Synnax cluster. This includes traces, metrics, and logs. <br />
**Trace** - A record of the execution of a request through the Synnax cluster. <br />
**Metric** - A measurement of a specific aspect of the execution of a request. <br />
**Log** - A record of a specific event that occurred during the execution of a
request. <br />

# 2 - Motivation

Synnax is in stable Alpha (v0.4.2 as of this writing), and we're beginning to plan the
road
to Beta. This involves shifting our mindset from building a technical proof-of-concept
to
creating a stable, performant, and maintainable product. These qualities are not
emergent.

Improving these characteristics starts with having a clear, on-demand picture of the
execution
state of the entire cluster. Proper observability allows us to find critical bugs in
development,
systematically improve performance through iterative optimization and benchmarking, and
monitor
the health of our users' deployments.

# 3 - Philosophy

### 3.0 - Instrumentation Classes

When approaching the high-level design of instrumentation, I found it useful to think
about a Synnax cluster as a supply chain structured as a grid. The rows represent areas
of horizontal integration, and specifically layers of the Synnax stack. The bottom
begins
with the storage layer, and the top ends with the user interface. The columns represent
areas of vertical integration, or the 'pathways' a request takes through the stack. For
example,
frame reads and writes strike different areas of each layer than an ontology traversal
would.

It's important to understand the execution of the cluster from both the vertical and
horizontal
perspectives.

It's relevant to note that approaching a design problem from this perspective is not
new. The
design of the Signal package
in [RFC 0004](https://github.com/synnaxlabs/synnax/blob/main/docs/rfc/0004-220623-signal-gr.md)
also discusses issues between application and request-scoped goroutine management,
especially in
regard to error handling.

### 3.0.1 - X Instrumentation

Horizontal, or`X`, instrumentation collects and reports data about different layers (and
partitions
of those layers) of Synnax. Its role is to build a picture of how a layer behaves over
time. For
example, we need to measure the growth of the cesium's range pointer cache over time,
track the
network load on the distribution layer, or track the balance of leased KV operations on
different
nodes over time.

The tracing side of X instrumentation tracks goroutines and processes that aren't
associated with a
particular request.

### 3.0.2 - Y Instrumentation

Y instrumentation collects and reports data about the execution of a specific request. A
perfect
example tracks the lifecycle of a frame writer, measuring goroutine creation and
destruction, mean
frame sizes, and performance metrics such as serialization/deserialization and storage
throughput.

### 3.0.3 - General Characteristics

Both X and T instrumentation strictly observe architectural boundaries, meaning that
instrumentation
in a layer below cannot be directly dependent on the instrumentation in a layer above,
and should
not expose shallow interfaces to layers above.

# 4 - Requirements

Alamos must handle three types of instrumentation: logs, metrics, and traces. These are
widely
referred to as the three pillars of observability. I'm omitting an argument for why
these types
exist and why each is important; this should be obvious to you, and, if it isn't, there
is extensive literature on the subject.

Instead, I'm focused on describing the specific requirements for each. These are not
organized by pillar,
but rather by type of requirement. For example, all three pillars need some means of
persistence; this is
all covered within a single section.

## 4.1 - Distribution

(Obviously) Synnax is a distributed system, and, perhaps the most challenging
requirement for an
instrumentation system is to provide an aggregated view of the execution state for
several machines.

### 4.1.0 - Instrumentation Must Support Clients

Many features exist above the server-side waterline. Our Python and Typescript
libraries, Synnax CLI,
and User Interfaces play a role in delivering a quality user experience. When designing
a distributed
instrumentation system, we must keep in mind how we tie together the execution state of
both the server
and client.

### 4.1.1 - Instrumentation Must Distribute Across Nodes

Requests and cluster synchronization tasks regularly span several nodes. To understand
these processes,
we must not only understand execution within a node, but also how execution crosses node
boundaries.
Supporting distributed tracing is essential.

## 4.2 - Meta-Data

### 4.2.0 - Categorization

Collecting telemetry is only useful if we can correlate it with meta-data about the
cluster's
configuration. If we don't know critical information about the cluster, such as the
version
of the software, we place ourselves at a significant disadvantage when it comes to
debugging
issues and improving performance.

### 4.2.1 - Y Meta-Data - Tracing

Y meta-data is bound to a specific request, and should be viewable at the level of an
individual
trace or an aggregated view of all traces. This includes protocols, user id's, etc.

### 4.2.2 - X Meta-Data

X meta-data is bound to a specific layer, and describes that layer's configuration i.e.
the
protocols supported, storage directories, maximum cache sizes etc.

## 4.3 - Filtering

As with any instrumentation system, we should be able to filter the data we collect
depending on
the environment we're running in. In a development environment, we focus on collecting
data for
debugging and correctness purposes. In a benchmarking environment, we collect critical
performance
metrics and traces.

### 4.3.0 - Log Levels

The Alamos log levels mirror those
of [zap](https://pkg.go.dev/go.uber.org/zap#pkg-constants).

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

## 4.4 - Analytics

## 4.5 - Persistence

## 4.6 - Development

## 4.7 - Production

# 5 - Design

## 5.0 - The Instrumentation Type

The instrumentation type (`alamos.Instrumentation`) forms the structural core of Alamos'
tooling. It serves as an aggregation
of several other types, and is aimed for use as a 'bag of tools' for injecting
instrumentation into a given service.

```go
package irrelivant

type Instrumentation struct {
    T *Tracer
    L *Logger
    R *Reporter
}
```

The type's fields are intentionally terse and it's methods are kept unique in order to
avoid potential conflicts when embedding
it into a struct.

## 5.1 - Dependency Injection in the X Direction

The instrumentation type should be embedded as the first field in a service's
configuration, or passed as an option to a service's
constructor. This allows the service to access the instrumentation tools as if they were
part of the configuration itself.

```go
package irrelivant

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
possible. Part of this effort involves allowing for zero-value instrumentation to
operate correctly

## 5.2 - Propagation

The most challenging part of designing this system is figuring out how to inject
instrumentation into virtually every part of the code base in a clear and effective
manner.

The previous version of `alamos` was only focused on instrumenting a layer xly, and,
as a result, metrics and loggers were dependency injected into a particular service.
For example, the storage engine has the following section of its config (pseudo-code):

```go
package irrelivant

import "github.com/synnaxlabs/alamos"

type Config struct {
    // Experiment is the old version of alamos.Instrumentation
    Instrumentation *alamos.Instrumentation
    // ... rest of config
}
```

This pattern is effective, and, most importantly, it's extremely clear. Introducing
Y instrumentation makes the dependency injection process much more complicated. Take,
for example, the `Set` operation on the current key-value interface:

```go
package irrelivant

type Writer interface {
    Set(key []byte, value []byte) error
}
```

The most naive way to instrument this interface is to add a new parameter to the
function:

```go
package irrelivant

type Writer interface {
    Set(key []byte, value []byte, instrumentation *Instrumentation) error
}
```

This pattern is clear, but also unsustainable. Requiring every request-scoped function
to take an instrumentation parameter increases interface footprint dramatically. The
alternative and more go-like approach is to use a context instead.

### 5.2.1 - Context Propagation in the Y Direction

Contexts in go are used for two purposes: cancellation and request-scoped data.
The former is used in virtually every area of the Synnax code base. The latter is
controversial within the go community. On the one hand, it's useful to implicitly
attach dynamic data to a request. On the other hand, implicit code makes it difficult
to reason about execution. A prime example is passing a database transaction through
a context. This reduces interface footprint, but also makes it difficult to understand
which parts of the code base are using the transaction and which are not. As a result,
we've generally avoided using `context.WithValue` in the Synnax code base. It's much
more
challenging to abuse if it's not available.

Instrumentation poses a particularly powerful use case for context propagation. It's
so widely accessed and documented that its implicit nature will stay apparent to the
reader.
Our key-value interface will now look like this:

```go
package irrelivant

type Writer interface {
    Set(ctx context.Context, key []byte, value []byte) error
}
```

Inside the implementation of `Set`, we can retrieve the instrumentation from the
context:

```go
package irrelivant

func (w *writer) Set(ctx context.Context, key []byte, value []byte) error {
    instrumentation := alamos.FromContext(ctx)
    // ... rest of function
}
```

#### 5.2.1.0 - Performance

Aside from what was discussed above, the remaining concern lies in performance in the
hot-path of a request. I chose the key-value interface
to illustrate this concern. Even if we have some sort of no-op instrumentation that is
returned when no instrumentation exists on the context, we still need
to perform that check. This overhead is negligible for a single request, but can add up
to cause performance regressions when executing a large number of
operations. This concern is minor, and is mostly something to monitor as we move
forward.

### 5.2.3 - Context Propagation for X Instrumentation

## 5.3 - Application Critical Metrics

## 5.3 - Distribution

# Working Notes

When working with request scoped goroutines, we typically have to types of contexts were
passing. We have the initialization context and the signal context. Need to think about
making this simple.

# for request instrumentation (y)

Experiment -> (Trace -> Span)

We need to provide reasonable justifications for passing instrumentation through a
context. Especially contexts that live throughout the lifetime of the application.

Perhaps metrics are kept separate from logs and traces?

The question is where to keep operation critical metrics.

One thing is instrumentation for the purpose of debugging, and another is
instrumentation for the purpose of making operational decisions.

Metrics -> Capturing changes
