# 11 - Alamos Instrumentation

**Feature Name**: Alamos Instrumentation <br />
**Start Date**: 2023-04-01 <br />
**Authors**: emilbon99 <br />

# 0 - Summary

As we move towards the Beta release of Synnax, the core architectural components
will begin to solidify, and, for the more stable components, we will shift focus
from building something that works to building something stable and performant. To
improve these qualities, we will need to measure and profile the execution
of the code.

In this RFC I propose a high-level plan for implementing distributed instrumentation
across the Synnax verticals. This plan outlines method for handling traces, metrics, and
logs from the storage engine to the user interface.

# 1 - Vocabulary

**Alamos** - The Synnax instrumentation library, and the core package for implementing
this RFC.
**Instrumentation** - The process of collecting and reporting data about the execution
state of a Synnax cluster. This includes traces, metrics, and logs.
**Trace** - A record of the execution of a request through the Synnax cluster.
**Metric** - A measurement of a specific aspect of the execution of a request.
**Log** - A record of a specific event that occurred during the execution of a request.

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

At a high level, both horizontal and vertical instrumentation strictly observe architectural
boundaries, meaning that instrumentation in a layer below cannot be directly dependent on
the instrumentation in a layer above, and should attempt to expose very deep interfaces between
layers.

It's relevant to note that approaching a design problem from this perspective is not new.  The
design of the Signal package in [RFC 0014](https://github.com/synnaxlabs/synnax/blob/main/docs/rfc/0004-220623-signal-gr.md)
also discusses issues between application and request-scoped goroutine management, especially in
regard to error handling.

### 3.0.0 - Horizontal Instrumentation

Horizontal instrumentation collects and reports data about different layers (and vertical partitions
of those layers) of the Synnax architecture. Its role is to build a picture of how a layer behaves
over time. For example, we need to measure the growth of the cesium's range pointer cache over time,
or track the network load on the distribution layer.

On the tracing side, Synnax starts a number of goroutines on server startup. The goroutines cannot
be tied to a specific request, but intimate knowledge of their state is essential in debugging several
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
require some means of persistence; the persistence requirements for each are grouped together.

## 4.1 - Distribution

(Obviously) Synnax is a distributed system, and, perhaps the most challenging requirement for an
instrumentation system is to provide an aggregated view of the execution state for several machines.

### 4.1.0 - Instrumentation Must Support Clients

A considerable amount of the features Synnax delivers exist above the server-side waterline. Our Python
and Typescript libraries, Synnax CLI, and User Interfaces all play an essential role in delivering a quality
user experience. When designing a distributed instrumentation system, we must keep in mind how we tie
together the execution state of the server and the client.

### 4.1.1 - Instrumentation Must Distribute Across Nodes

Requests and cluster synchronization tasks typically take place across several nodes. To effectively improve
these processes, we must be able to not only understand the execution state within several nodes, but also
how they interact with each other. Supporting distributed traces and metrics is essential.

## 4.2 - Levels

### 4.2.0 - Log Levels

### 4.2.1 - Trace Levels

### 4.2.2 - Metric Levels

## 4.3 - Sensitive Data

## 4.4 - Analytics

## 4.5 - Persistence

# 5 - Design

## 5.1 - Experiments

Experiments (`alamost.Experiment`) form the structural framework for instrumenting Synnax. Their organization mimics that
of context propagation through the parts of the Synnax stack (I'll recommend transferring them through a context in the next section).
form a tree-like structure, where a single background experiment forms the root node, and experiments gradually
become more specific as they are passed to different areas of the application.

All logging, tracing, and metric collection is tied to a specific experiment that is identified by a unique key. The key is
similar to a file path, and contains a record of the keys of its parents.

#  Working Notes

When working with request scoped goroutines, we typically have to types of contexts were passing. We have the initialization context
and the signal context. Need to think about making this simple.

# for request instrumentation (vertical)

Experiment -> (Trace -> Span)
