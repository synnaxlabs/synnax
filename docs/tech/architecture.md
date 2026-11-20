# Architecture

## 0 Purpose

This page summarizes Synnax's high level architecture. It is intended to be a stepping
point for diving into Synnax's codebase and [technical RFCs](rfc).

## 1 Services

At the highest level, Synnax implements three core products: A time-series engine for
transporting and storing time-series data, a set of client libraries for communicating
programmatically with the time-series engine, and user interfaces for interacting
graphically with the time-series engine.

## 2 Time-series engine

The time-series engine is the core of the platform. It accepts writes, transports
streams, and serves reads of time-series data. It also serves auxiliary roles such as
user authentication, managing user interface configurations, and storing metadata. This
engine, commonly referred to as "Synnax" or "the Core", is written completely in Go, and
compiles to a single (~35 MB) binary. A Core runs on one machine.

### 2.0 Nodes

A Synnax **node** is a process that runs the Synnax binary. A Core runs one node, the
host. Each channel key holds the key of the node that leases the channel. A node can run
on a bare-metal machine, in a virtual machine, in a container, or on an edge device.

### 2.1 Layered architecture

The Core's architecture is **layered**. Each layer (there are four) extends the layer
below it to augment its functionality. The core principle behind layered design is that
layers below are _unaware_ of layers above i.e. function calls and type access can only
be made from layers above to layers below, and not vice versa. **We follow this
principle with extreme rigor**. The Core has four layers, which we'll cover in the
following sections. Each layer has additional guidelines for what types of code belong
in it.

### 2.2 Layer 1 - Storage

The storage layer is the foundation of the engine, and is responsible for reading and
writing [telemetry](telemetry.md) and corresponding metadata to and from-disk. It is,
with few exceptions, the _only_ layer that interacts with the disk.

The storage layer implements two types of storage engines: key-value and time-series. We
use the key-value engine to store metadata (channel definitions, users, etc.) and the
time-series engine to store telemetry.

#### 2.2.0 Pebble - Key-value engine

Synnax's [key-value](https://www.mongodb.com/databases/key-value-database) engine stores
binary key-value pairs. It is used to store metadata such as channel definitions, user
information, user interface configurations, etc. Anything that is not time-series data
and takes a relatively small amount of space is stored in the key-value engine.

Implementing a performant key-value database is hard, and is not Synnax's focus.
Luckily, there are many great open-source key-value databases available. Synnax uses the
Go-embedded [Pebble](https://github.com/cockroachdb/pebble) which is the backing store
for [CockroachDB](https://www.cockroachlabs.com/). On startup, Synnax opens the database
and lets Pebble handle the rest. We'll return to this topic in the
[distribution layer](#23-layer-2---distribution).

#### 2.2.1 Cesium - Time-series engine

Synnax's custom time-series engine, [Cesium](../../cesium) is responsible for reading
and writing _frames_ of telemetry to and from disk. A frame is a map of channel names to
arrays of time-ordered values. Cesium ensures that frames are valid, separates them into
component arrays, and finds the optimal location to store them on disk. To retrieve
frames as quickly as possible, Cesium builds several indexing structures to reduce the
amount of disk seeks required to locate a specific data set.

As its interface, Cesium exposes two essential types: `Writer` and `Iterator`. A
`Writer` is a transactional object that accepts frames and writes them to disk, only
committing them when explicitly told to do so. If multiple frames are written that have
overlapping time ranges, the `Writer` will fail to commit. This ensures that data
integrity is never corrupted by concurrent writes. An `Iterator` iterates over a set of
frames in time-ordered fashion, allowing the caller to process very large data sets
without loading them all into memory at once.

If you're interested in reading more about Cesium, check out the following RFCs:

- [RFC 0001 - Cesium - Channel segment storage engine](rfc/0001-cesium-segment-storage.md)
- [RFC 0008 - Columnar storage for irregular data](rfc/0008-cesium-columnar.md)
- [RFC 0010 - Telemetry frame specification](rfc/0010-frame-spec.md)

### 2.3 Layer 2 - Distribution

The distribution layer turns the storage layer into the data plane that the layers above
use. It holds the identity of the Core, assigns channel keys, and reads, writes, and
streams frames of telemetry.

#### 2.3.0 Gorp - Metadata storage

The distribution layer opens [Gorp](../../gorp) on the Pebble store. Gorp keeps typed
records, such as channels and users, in tables, and keeps indexes of them in memory. A
write is a transaction that commits to Pebble in one batch. The distribution layer also
reads the cluster key and the host key of the Core from the key-value store.

If you're interested in reading more about Gorp, check out the following RFCs:

- [RFC 0034 - Gorp in-memory indexes](rfc/0034-gorp-indexes.md)

#### 2.3.1 Framer - Time-series reads and writes

The framer extends the time-series engine of the storage layer. Like Cesium, it writes
frames through a `Writer` and reads them through an `Iterator`. It also streams live
frames through a `Streamer`, and carries the frames of free virtual channels, which the
storage layer does not store.

### 2.4 Layer 3 - Service

The service layer implements the majority of Synnax's core functionality. This includes
registering and authenticating users, managing channels, ranges, and other entities to
name a few. As with the storage layer, the service layer is **network unaware**, and
reads and writes telemetry through the distribution layer. While it plays a critical
role in the Synnax ecosystem, the service layer holds little architectural significance
and is not discussed in detail here.

### 2.5 Layer 4 - Interface

The highest layer of the Core is the interface layer. This layer takes the functionality
in the service layer and exposes a clean network API to clients. The interface layer
implements as little service specific functionality as possible, and is designed in such
a way that it can be easily extended to support new protocols. Currently, it only
supports HTTP (+ Websockets), but in the future it will support gRPC and other
protocols.
