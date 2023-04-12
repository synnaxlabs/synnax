# 11 - Alamos Instrumentation

**Feature Name**: Alamos Instrumentation <br />
**Start Date**: 2023-04-01 <br />
**Authors**: emilbon99 <br />
**Status**: Draft <br />

# 0 - Summary

As we move towards the Beta release of Synnax, the core architectural components
will begin to solidify, and, for the more stable components, we will shift focus
from building functional to stable and performant software. To improve these qualities,
we will need to profile the execution state of the cluster.

In this RFC I propose a high-level plan for implementing distributed instrumentation
across all of Synnax's components. This plan outlines methods for handling traces, metrics, and
logs from the storage engine to the user interface.

# 1 - Vocabulary

**Alamos** - The Synnax instrumentation library, and the core package for implementing
this RFC. <br />
**Instrumentation** - The process of collecting and reporting data about the execution
state of a Synnax cluster. This includes traces, metrics, and logs. <br />
**Trace** - A record of the execution of a request through the Synnax cluster. <br />
**Metric** - A measurement of a specific aspect of the execution of a request. <br />
**Log** - A record of a specific event that occurred during the execution of a request. <br />

# 2 - Motivation

Synnax is in stable Alpha (v0.4.2 as of this writing), and we're beginning to plan the road
to Beta. This involves shifting our mindset from building a technical proof-of-concept to
creating a stable, performant, and maintainable product. These qualities are not emergent.

Improving the performance and stability of Synnax starts with having a clear, on-demand
picture of the execution state of the entire cluster. Without proper observability, we
cannot know where to focus our efforts.

# 3 - Philosophy

### 3.0 - Instrumentation Classes

When approaching the high-level design of instrumentation, I found it useful to think
about a Synnax cluster as a supply chain structured as a grid. The rows represent areas
of horizontal integration, and specifically layers of the Synnax stack. The bottom begins
with the storage layer, and the top ends with the user interface. The columns represent
areas of vertical integration, or the 'pathways' a particular feature takes through the
stack. For example, frame reads and writes strike different areas of each layer than an
ontology traversal would.

It's important to understand the execution of the cluster from the perspective of a vertical
pathway, a horizontal layer, or the intersection of the two.

At a high level, both horizontal and vertical instrumentation strictly observes architectural
boundaries, meaning that instrumentation in a layer below cannot be directly dependent on
the instrumentation in a layer above, and should attempt to expose very deep interfaces between
layers.

It's relevant to note that approaching a design problem from this perspective is not new. The
design of the Signal package in [RFC 0004](https://github.com/synnaxlabs/synnax/blob/main/docs/rfc/0004-220623-signal-gr.md)
also discusses issues between application and request-scoped goroutine management, especially in
regard to error handling.

### 3.0.0 - Horizontal Instrumentation

Horizontal instrumentation collects and reports data about different layers (and vertical partitions
of those layers) of the Synnax architecture. Its role is to build a picture of how a layer behaves
over time. For example, we need to measure the growth of the cesium's range pointer cache over time,
or track the network load on the distribution layer.

On the tracing side, Synnax starts a number of goroutines on server startup. The goroutines cannot
be tied to a specific request, but knowledge of their state is essential in debugging several
classes of issues, most notably behavioral integrity on server startup, shutdown, or failure.

### 3.0.1 - Vertical Instrumentation

Vertical instrumentation collects and reports data about the execution of a specific feature. A perfect
example tracks the lifecycle of a frame writer, measuring goroutine creation and destruction, mean
frame sizes, and performance metrics such as serialization/deserialization and storage throughput.

# 4 - Requirements

Alamos must handle three types of instrumentation: logs, metrics, and traces. These are widely
referred to as the three pillars of observability. I'm omitting an argument for why these types
exist and why each is important; this should be obvious to the reader, and, if it isn't, there
is extensive literature on the subject.

Instead, I'm focused on describing the specific requirements for each. These requirements are not
organized by type, but rather by grouping of related requirements. For example, all three types
need some means of persistence; the persistence requirements for each are grouped together.

## 4.1 - Distribution

(Obviously) Synnax is a distributed system, and, perhaps the most challenging requirement for an
instrumentation system is to provide an aggregated view of the execution state for several machines.

### 4.1.0 - Instrumentation Must Support Clients

Many features exist above the server-side waterline. Our Python and Typescript libraries, Synnax CLI,
and User Interfaces all play an essential role in delivering a quality user experience. When
designing a distributed instrumentation system, we must keep in mind how we tie together the
execution state of the server and the client.

### 4.1.1 - Instrumentation Must Distribute Across Nodes

Requests and cluster synchronization tasks typically take place across several nodes. To effectively improve
these processes, we must not only understand execution state within several nodes, but also
how they interact with each other. Supporting distributed traces and metrics is essential.

## 4.2 - Meta-Data

### 4.2.0 - Categorization

Collecting telemetry is only useful if we can correlate it with meta-data about the cluster's
configuration. If we don't know critical information about the cluster, such as the version
of the software, we place ourselves at a significant disadvantage when it comes to debugging
issues and improving performance.

The two classes of meta-data we need to collect closely parallel the classes of instrumentation
introduced in [Section 3](#3-philosophy). The first class covers layer specific configuration,
such as the storage engine's maximum cache size. The second class covers request specific meta-data,
such as the requesting user or the protocol used.

### 4.2.1 - Vertical Meta-Data - Tracing

Vertical meta-data is bound to a specific request, and should be viewable at the level of an individual
trace or an aggregated view of all traces.

### 4.2.2 - Horizontal Meta-Data

Horizontal meta-data is bound to a specific layer, and describes that layer's configuration i.e. the
protocols supported, storage directories, maximum cache sizes etc.

## 4.3 - Filtering

As with any instrumentation system, we should be able to filter the instrumentation data we collect
depending on the environment we're running in. In a development environment, we focus on collecting
data for debugging and correctness purposes. In a benchmarking environment, we collect critical performance
metrics and traces.

### 4.3.0 - Levels

#### 4.3.0.1 - Logs

<table>
<tr>
<th>Level</th>
<th>Meaning</th>
</tr>
<tr>
<td>debug</td>
<td>Debugging information</td>
</tr>
<tr>
<td>info</td>
<td>Informational messages</td>
</tr>
<tr>
<td>warn</td>
<td>Warnings</td>
</tr>
<tr>
<td>error</td>
<td>Errors</td>
</tr>
<tr>
<td>fatal</td>
<td>Fatal errors</td>
</tr>
<tr>
<td>panic</td>
<td>Panic errors</td>
</tr>
</table>

Log-filtering levels are as follows:

- `debug` - Debugging information
- `info` - Informational messages
- `warn` - Warnings
- `error` - Errors
- `fatal` - Fatal errors
- `panic` - Panic errors

### 4.3.1 - Tracing

Tracing

## 4.4 - Analytics

## 4.5 - Persistence

## 4.6 - Development

## 4.7 - Production

# 5 - Design

## 5.1 - The Instrumentation Type

Experiments (`alamost.Experiment`) form the structural framework for instrumenting Synnax. Their organization mimics that
of context propagation through the parts of the Synnax stack (I'll recommend transferring them through a context in the next section).
form a tree-like structure, where a single background experiment forms the root node, and experiments gradually
become more specific as they are passed to different areas of the application.

All logging, tracing, and metric collection is tied to a specific experiment that is identified by a unique key. The key is
similar to a file path, and contains a record of the keys of its parents.

## 5.2 - Propagation

The most challenging part of designing this system is figuring out how to inject instrumentation into virtually every part of the
code base in a clear and effective manner.

The previous version of `alamos` was only focused on instrumenting a layer horizontally, and, as a result, metrics and loggers were
dependency injected into a particular service. For example, the storage engine has the following section of its config (pseudo-code):

```go
package irrelivant

import "github.com/synnaxlabs/alamos"

type Config struct {
    // Experiment is the old version of alamos.Instrumentation
    Instrumentation *alamos.Instrumentation
    // ... rest of config
}
```

This pattern is effective, and, most importantly, it's extremely clear. Introducing Y instrumentation makes the dependency injection
process much more complicated. Take, for example, the `Set` operation on the current key-value interface:

```go
package irrelivant

type Writer interface {
    Set(key []byte, value []byte) error
}
```

The most naive way to instrument this interface is to add a new parameter to the function:

```go
package irrelivant

type Writer interface {
    Set(key []byte, value []byte, instrumentation *alamos.Instrumentation) error
}
```

This pattern is extremely clear, but it's obviously unsustainable. Requiring every request-scoped function to take an instrumentation
parameter increases interface footprint dramatically. The alternative and more go-like approach is to use a context instead.

### 5.2.1 - Context Propagation

Contexts in go are used for two purposes: cancellation and request-scoped data. The former is used in virtually every area
of the Synnax code base. The latter is controversial within the go community. On the one hand, it's useful to implicitly attach dynamic
data to a request. On the other hand, implicit code makes it difficult to reason about execution. A prime example is passing a database
transaction through a context. This reduces interface footprint, but also makes it difficult to understand which parts of the code base are using the
transaction and which are not. As a result, we've generally avoided using `context.WithValue` in the Synnax code base. It's much more
challenging to abuse if it's not available.

Instrumentation poses a particularly powerful use case for context propagation. It's so widely accessed and documented that its implicit nature
will stay apparent to the reader. Our key-value interface will now look like this:

```go
package irrelivant

type Writer interface {
    Set(ctx context.Context, key []byte, value []byte) error
}
```

Inside the implementation of `Set`, we can retrieve the instrumentation from the context:

```go
package irrelivant

func (w *writer) Set(ctx context.Context, key []byte, value []byte) error {
    instrumentation := alamos.FromContext(ctx)
    // ... rest of function
}
```

#### 5.2.1.0 - Performance

Aside from what was discussed above, the remaining concern lies in performance in the hot-path of a request. I chose the key-value interface
to illustrate this concern. Even if we have some sort of no-op instrumentation that is returned when no instrumentation exists on the context, we still need
to perform that check. This overhead is negligible for a single request, but can add up to cause performance regressions when executing a large number of
operations. This concern is minor, and is mostly something to monitor as we move forward.

### 5.2.3 - Context Propagation for Horizontal Instrumentation

## 5.3 - Application Critical Metrics

## 5.3 - Distribution

# Working Notes

When working with request scoped goroutines, we typically have to types of contexts were passing. We have the initialization context
and the signal context. Need to think about making this simple.

# for request instrumentation (vertical)

Experiment -> (Trace -> Span)

We need to provide reasonable justifications for passing instrumentation through a context. Especially
contexts that live throughout the lifetime of the application.

Perhaps metrics are kept separate from logs and traces?

The question is where to keep operation critical metrics.

One thing is instrumentation for the purpose of debugging, and another is instrumentation for the
purpose of making operational decisions.

Metrics -> Capturing changes
