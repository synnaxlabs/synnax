---
sidebar_position: 2
---
# Concepts

Synnax is a very complicated system. We've done our best to keep things simple, but there are a lot of moving parts. This 
page details the high level concepts and entities that make up the telemetry engine.

### Architecture 

#### Node 

A node is a single instance of Synnax that is running as part of a cluster. At its core, a node is a single
executable, and can be deployed on a container in the cloud, on a virtual machine, or on an edge device such as a 
Raspberry Pi.

#### Cluster

A cluster is a collection of nodes that communicate with one another to coordinate data storage, retrieval
and operations. A cluster may consist of one or more nodes.

### Telemetry 

#### Sample

A sample is a simple `(timestamp, value)` pair. Synnax stores timestamps as int64 values that represent
a unix epoch in nanoseconds. Timestamps currently store no timezone information, and it is up to the user to ensure
that timestamps are consistent with the timezone of the data they are storing. Internally, Synnax stores values as
arbitrary byte arrays, although it provides a number of built-in encoders and decoders for common types such as `float64`
and `int64`.

#### Channel 

A channel is a collection of telemetry samples across a time-range. The data within a channel typically 
arrives from a single source. This can be a physical sensor, metric, event, or other entity that emits regular, consistent,
and time-order values. A channel can also be used for storing derived data, such as a moving average or signal
processing result. Channels have a few important properties:

1. Data Rate - The number of samples per second (Hz) that are stored in a channel. This data rate is fixed, and cannot
be changed without deleting and recreating a channel. All data written to a channel must have the same data rate.
2. Name - A human-readable name for the channel that isn't required. It's a best practice to make sure this name is unique.
3. Data Type - A pre-defined data type that describes the type of sample stored in the channel. Common examples are `float64`,
`int64`, `bool`, etc. It's also possible to define custom data types. See the [Data Types](#data-types) section for more
information.
4. Key - A unique identifier for the channel that is used across the entire cluster. This key is automatically assigned 
and uniquely identifies the channel across the entire cluster.
5. Node ID - This is the ID of the node that holds the lease on the channel. This node is known as the leaseholder, and
is the only node that can write *new* channel data to disk. The leaseholder is typically kept in proximity (physically) 
to the source generating the channel's data (e.g. a sensor).

#### Data Types

Synnax supports a number of built-in data types:

- `float64`
- `float32`
- `int64`
- `int32`
- `int16`
- `int8`
- `uint64`
- `uint32`
- `uint16`
- `uint8`
- `bool`

It's also possible to define completely custom data types. There are two requirements for a data type:

1. It must be encodeable as a byte array.
2. It must have a fixed density i.e. the number of bytes per value must be constant.