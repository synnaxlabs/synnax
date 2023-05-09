# 12 - Live Telemetry

**Feature Name**: Live Telemetry <br />
**Start Date** 2023-05-01 <br />
**Authors**: Emiliano Bonilla <br />
**Status**: Draft <br />

# 0 - Summary

Live telemetry transportation lies at the foundation for any industrial control system.
Until now, Synnax's focus has been solely on distributed telemetry storage and
retrieval. While it has been used for plotting in real-time scenarios at rates of up to
1Hz, dedicated real-time infrastructure is necessary for supporting active control and
higher data rates.

In this RFC I propose an architecture for integrating live telemetry into the existing
Synnax ecosystem. I'll discuss modifications to several existing components, including
the core read and write pipeline, as well as establishing several new components to
support real-time needs.

# 1 - Vocabulary

# 2 - Motivation

# 3 - Philosophy

# 3.0 - Leveraging Properties of Telemetry

Building an efficient telemetry engine starts with having a clear understanding of the
core characteristics of telemetry as it relates to hardware systems. We can leverage
these properties to define constraints that allow us to build simpler, more efficient
software while still providing our users with sufficient flexibility to meet their
needs.

This approach is not novel, and, most notably, has played a major role in the simplicity
of Cesium's storage architecture. As we extend our storage and distribution systems,
we need to keep leveraging these fundamentals to our advantage.

I've decided these are so important that they deserve a page of their
own, [here](./telemetry-properties.md).

# 4 - Requirements

# 5 - Design

## 5.0 - Summary

The live telemetry design is built around extending the existing frame read and write
infrastructure to develop

## 5.1 - Changes to Cesium

### 5.1.0 - Self-Indexing Channels

### 5.1.1 - Variable Size Channels

### 5.1.2 - Compression

## 5.2 - Pacer

### 5.2.x - Write Durability

## 5.3 - Relay

Synnax employs a relay to efficiently stream telemetry to peers and clients. At the
storage level, this relay "taps off" of Pacer's write pipeline, essentially serving
as an observer to incoming live frames. The relay extends upwards into the distribution
layer, where both clients in the service layer and peer nodes can open streams of
telemetry.

### 5.3.0 - Storage Layer

As covered above, the storage level relay listens to incoming writes to pacer. The
simplest way to accomplish this is to have Pacer implements the `observe.Observable`
interface and notify arbitrary subscribers of changes via `Notify` and the `OnChange`
handlers. There are two challenges with this approach.

#### 5.3.0.0 - Tap-Off Staging

The stage in the write pipeline in which subscribers are notified is left under pacer's
control; pacer could notify the subscriber before writing the frame to cache, after
writing the frame to cache, or after flushing the frame to Cesium. Obviously this is a
minor issue as we control how pacer is implemented and can adjust as necessary, but
it'd be nice to shift this responsibility to a more flexible area of the codebase.

Carefully positioning the relay tap has important implications for latency and jitter,
and it's important to prioritize delivering live frames than persisting them. If we
have a lot of heavy code sitting on top of the tap, we risk increasing both latency
and jitter under high load.

#### 5.3.0.0 - Clogging the Write Pipeline

Depending on which handlers are bound, synchronously notifying subscribers on every
frame write could cause considerable performance regressions under high load. Forking a
new goroutine for every notification is also clearly not an option, as we'd have no way
of sustainably tracking the lifecycle of these routines, and could end up leaking many
of them.

A bad solution is to notify subscribers through a heavily buffered channel. A channel
would be needed for each subscriber, and it's still possible to halt the entire write
pipeline if the channel buffer fills completely.

Solving this problem depends heavily on the delivery guarantees we're trying to satisfy.
Namely, is it ok to drop frames or close hanging sockets to maintain a fixed capacity
buffer while making sure the write pipeline never gets clogged? Or should we allow for
infinite (within reason) buffering to give slow subscribers extensive leeway. The first
is, in principle, the less fault-tolerant and forgiving approach at the cost of
performance and missing subtle leaks. My intuition leads me towards using a well sized
buffer and performing non-blocking `select` operations to notify subscribers. In the
case that the buffer fills up, we can warn the user of a slow subscriber and that we're
dropping frames to accommodate.

### 5.3.1 - Distribution Layer

# 6 - Future Work

## 6.0 - Anti-Jitter

## 6.1 - Framing Flight Protocol

## 6.2 - Anti-Jitter

## 6.3 - Virtual Channels

Question here deals with lease holding. We generally don't want to tie general cluster
change capture across

What if we have live readers that relay through the key-value store? Do CDC that way.

We need to put channels into perspective -> They're just a way of passing messages
between nodes and to clients.
