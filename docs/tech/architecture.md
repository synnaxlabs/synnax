# Architecture

# 0 - Purpose

This page summarizes Synnax's high level architecture. It is intended to be a stepping
point for diving into Synnax's codebase and [technical rfc's](rfc).

# 1 - Services

At the highest level, Synnax implements three core products: A time-series engine for
transporting and storing time-series data, a set of client libraries for communicating
programmatically with the time-series engine, and user interfaces for interacting
graphically with the time-series engine.

# 1 - Time-Series Engine

The time-series engine is the core of platform. It accepts stores writes, transports
streams, and serves reads of time-series data. It also serves auxiliary roles such as
user authentication, managing user interface configurations, and storing metadata. This
engine, commonly referred to as "Synnax" or "Synnax Server", is written completely in
go, and compiles to a single (~35mb) binary. The server is **horizontally scalable**,
meaning that it can be run on multiple machines while still appearing as a single server
to clients accessing its data.

### 1.0 - Horizontal Scale

This collaboration between multiple computers forms an entity called a **cluster**. The
machines that make up a cluster are called **nodes**. Formally, a Synnax **node** is a
process running the Synnax binary. This process can be run on bare-metal machine, a
virtual machine, in a container, and on edge devices. Synnax employs various
synchronization algorithms to ensure that all nodes are in agreement about the state of
the cluster.

### 1.1 - Layered Architecture

Synnax Server's architecture is **layered**. Each layer (there are four) extends the
layer below it to augment its functionality. The core principle behind layered design is
that layers below are _unaware_ of layers above i.e. function calls and type access can
only be made from layers above to layers below, and not vice versa. **We follow this
principle with extreme rigor**. Synnax Server has four layers, which we'll cover in the
following sections. Each layer has additional guidelines for what types of code belong
in it.

### 1.2 - Layer 1 - Storage

The storage layer is the foundation of the engine, and is responsible for reading and
writing [telemetry](telemetry.md) and corresponding metadata to and from-disk. It is,
with few exceptions, the _only_ layer that interacts with the disk.

The storage layer implements two types of storage engines: key-value and time-series. We
use the key-value engine to store metadata (channel definitions, users, etc.) and the
time-series engine to store telemetry.

#### 1.2.1 - Pebble - Key-Value Engine

Synnax's [key-value](https://www.mongodb.com/databases/key-value-database) engine stores
binary key-value pairs. It is used to store metadata such as channel definitions, user
information, user interface configurations, etc. Anything that is not time-series data
and takes a relatively small amount of space is stored in the key-value engine.

Implementing a performant key-value database is hard, and is not Synnax's focus.
Luckily, there are many great open-source key-value databases available. Synnax uses the
go-embedded [pebble](https://github.com/cockroachdb/pebble) which is the backing store
for [cockroachdb](https://www.cockroachlabs.com/). On startup, Synnax opens the database
and lets pebble handle the rest. We'll return to this topic in the
[distribution layer](#12---layer-2---distribution).

#### 1.2.2 - Cesium - Time-Series Engine

Synnax's custom time-series engine, [cesium](../../cesium) is responsible for reading
and writing _frames_ of telemetry to and from disk. A frame is a map of channel names to
arrays of time-ordered values. Cesium ensures that frames are valid, separates them into
component arrays, and finds the optimal location to store them on disk. To retrieve
frames as quickly as possible, cesium builds several indexing structures to reduce the
amount of disk seeks required to locate a specific data set.

As its interface, cesium exposes two essential types: `Writer` and `Iterator`. A
`Writer` is a transactional object that accepts frames and writes them to disk, only
committing them when explicitly told to do so. If multiple frames are written that have
overlapping time ranges, the `Writer` will fail to commit. This ensures that data
integrity is never corrupted by concurrent writes. An `Iterator` iterates over a set of
frames in time-ordered fashion, allowing the caller to process very large data sets
without loading them all into memory at once.

If you're interested in reading more about cesium, check out the following RFC's:

- [001 - Cesium Segment Storage](rfc/0001-220517-cesium-segment-storage.md)
- [008 - Cesium Columnar Storage](rfc/0008-221012-cesium-columnar.md)
- [0010 - Frame Specifications](rfc/0010-230104-frame-spec.md)

### 1.3 - Layer 2 - Distribution

The distribution layer composes the storage layers of several nodes to expose a
distributed, monolithic data space to layers above. A monolithic data space allows a
caller to access any piece of data in the cluster, without needing to know where it's
located. This is a powerful abstraction that allows us to build a horizontally scalable
system whose high level services are unaware of the underlying network complexity.

The distribution layer exposes a similar interface to the storage layer, providing both
an eventually consistent, cluster-wide metadata store and a time-series engine. It also
exposes information about the cluster, such as the number of nodes, the health of each
node, and the addresses of reachable nodes.

#### 1.3.1 - Aspen - Cluster Membership, State Synchronization and Distributed Key-Value Storage

Aspen is backbone of the distribution layer. It serves two primary roles: managing
cluster membership and providing a distributed key-value store.

At its core, Aspen implements a distributed pledging algorithm that determines whether a
new Synnax process is allowed to join the cluster. Once a node is allowed to join, a
distributed counter assigns a unique `uint16` key to the node. To keep cluster
membership synchronized, Aspen implements an
[SI gossip protocol](https://medium.com/dsp-labs/knowing-dsp-in-3-minutes-network-gossip-protocol-27a8ff7af3ff)
that propagates cluster state at a 1Hz interval. Various parameters are passed through
gossip, the most relevant of which are node health and reachable node addresses over a
network.

On-top of the cluster membership and gossip protocols, Aspen uses an additional **SIR**
gossip protocol to synchronize key-value operations. These operations are propagated at
a configurable (default 1Hz) interval, meaning that Aspen's key-value store is
**eventually** consistent. Each node in the cluster maintains a local copy of the
key-value database, making Aspen heavily read-optimized. This suits our use-case well,
as the key-value database is used primarily for metadata storage that is read much more
often than it is written. Aspen uses [pebble](#121---pebble---key-value-engine) as the
underlying key-value store.

If you're interested in reading more about Aspen, check out the following RFC's:

- [0002 - Aspen Distributed Storage](rfc/0002-220518-aspen-distributed-storage.md)

#### 1.3.2 - Framer - Distributed Time-Series Storage

Using Aspen's cluster membership and host-resolution capabilities, the distribution
layer extends the time-series engine on each node to provide distributed, strongly
consistent time-series storage. Just like the storage layer time-series engine, the
distribution layer writes frames through a `Writer` and reads them through an
`Iterator`. The difference is that the distribution layer intelligently resolves the
address of the nodes that own particular pieces of data, and routes requests to them
accordingly.

If you're interested in reading more about distributed frame reads and writes, check out
the following RFC's:

- [0003 - Segment Distribution](rfc/0003-220604-segment-distribution.md)

### 1.3 - Layer 3 - Service

The service layer implements the majority of Synnax's core functionality. This includes
registering and authenticating users, managing channels, ranges, and other entities to
name a few. As with the storage layer, the service layer is **network unaware**, and
relies on the distribution layer to handle any distributed operations. While it plays a
critical role in the Synnax ecosystem, the service layer holds little architectural
significance and is not discussed in detail here.

### 1.4 - Layer 4 - Interface

The highest layer of Synnax server is the interface layer. This layer takes the
functionality in the service layer and exposes a clean network API to clients. The
interface layer implements as little service specific functionality as possible, and is
designed in such a way that it can be easily extended to support new protocols.
Currently, it only supports HTTP (+ Websockets), but in the future it will support gRPC
and other protocols.
