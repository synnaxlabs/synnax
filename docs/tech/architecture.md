# Architecture

# 0 - Purpose

This page summarizes Synnax's high level architecture. It is intended to be a stepping
point for diving into Synnax's codebase and [technical rfc's](rfc).

# 1 - Services

At the highest level, Synnax implements three core products: A time-series engine for
transporting and storing time-series data, a set of client libraries for communicating
programmatically with the time-series engine, and user interfaces for interacting
graphically
with the time-series engine.

# 1 - Time-Series Engine

The time-series engine is the core of platform. It accepts stores writes, transports
streams, and serves reads of time-series data. It also serves auxiliary roles such as
user authentication, managing user interface configurations, and storing meta-data.
This engine, commonly referred to as "Synnax" or "Synnax Server", is written completely
in go, and compiles to a single (~35mb) binary. The server is **horizontally scalable**,
meaning that it can be run on multiple machines while still appearing as a single server
to clients accessing its data.

### 1.0 - Horizontal Scale

This collaboration between multiple computers forms an entity called a **cluster**.
The machines that make up a cluster are called **nodes**. Formally, a Synnax **node** is
a process running the Synnax binary. This process can be run on bare-metal machine, a
virtual machine, in a container, and on edge devices. Synnax employs various
synchronization algorithms to ensure that all nodes are in agreement about the state of
the cluster.

### 1.1 - Layered Architecture

Synnax Server's architecture is **layered**. Each layer (there are four) extends the
layer below it to augment its functionality. The core principle behind layered design
is that layers below are *unaware* of layers above i.e. function calls and type access
can only be made from layers above to layers below, and not vice versa. **We follow
this principle with extreme rigor**. Synnax Server has four layers, which we'll cover
in the following sections. Each layer has additional guidelines for what types of code
belong in it.

### 1.2 - Layer 1 - Storage

The storage layer is the foundation of the engine, and is responsible for reading
and writing [telemetry](telemetry.md) and corresponding meta-data to and from-disk.
It is, with few exceptions, the *only* layer that interacts with the disk.

The storage layer implements two types of storage engines: key-value and time-series.
We use the key-value engine to store meta-data (channel definitions, users, etc.) and
the
time-series engine to store telemetry.

#### 1.2.1 - Key-Value Engine

Synnax's [key-value](https://www.mongodb.com/databases/key-value-database) engine stores
binary key-value pairs. It is used to store meta-data such as channel definitions, user
information, user interface configurations, etc. Anything that is not time-series data
and takes a relatively small amount of space is stored in the key-value engine.

Implementing a performant key-value database is hard, and is not Synnax's focus.
Luckily, there are many great open-source key-value databases available. Synnax uses the
go-embedded [pebble](https://github.com/cockroachdb/pebble) which is the backing store for
[cockroachdb](https://www.cockroachlabs.com/). On startup, Synnax opens the database and
let's pebble handle the rest. We'll return to this topic in the [distribution layer](#12---layer-2---distribution).

#### 1.2.2 - Time-Series Engine



### 1.2 - Layer 2 - Distribution

### 1.3 - Layer 3 - Service

# 2 - Client Libraries

# 3 - User Interfaces

# 2 - Repository Organization
