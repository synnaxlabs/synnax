---
title: Synnax - Segment Distribution
layout: "@/layouts/MainLayout.astro"
---

# 3 - Delta - Segment Distribution

- **Feature Name**: Delta - Segment Distribution
- **Status**: Complete
- **Start Date**: 2022-06-04
- **Authors**: Emiliano Bonilla

# 0 - Summary

In this RFC I propose an architecture for exposing time-series segment storage as a
monolithic data space. This proposal focuses on serving simple, range-based queries in
an efficient manner while laying the foundations for data replication and transfer
across the cluster.

Defining a clear level of abstraction for the data space is challenging. The
distribution layer must maintain adequate low level control to support distributed
aggregation, but must also minimize complexity (in terms of locality and networking) for
the layers above.

This RFC lays out a domain-oriented, locality abstracted interface that allows callers
to read and write data to the cluster as if it was a single machine. This interface does
not require the user to be aware of the storage topology, but provides additional
context if the caller wants to perform network optimization themselves.

# 1 - Vocabulary

- **Sample** - An arbitrary byte array recorded at a specific point in time.
- **Channel** - A collection of samples across a time range. These samples typically
  come from a single source (sensor, software metric, event, etc.).
- **Segment** - A partitioned region of a channel's data.
- **Node** - A machine in the cluster.
- **Cluster** - A group of nodes that can communicate with each other.
- **Leaseholder** - A node that holds a lease on a particular piece of data. The
  leaseholder is the only node that can modify the data.
- **Gateway** - The node that receives a query, and is responsible for executing it on
  the cluster.
- **Data Warehouse (DWH)** - A system used for storing and reporting data for analysis
  and business intelligence purposes. Data warehouses typically involve long-running
  queries on much larger data sets than typical OLTP systems. Data warehouse queries
  fall into the OLAP category of workloads.
- **OTN** - Over the Network.

# 2 - Motivation

Separating storage and compute has become a popular technique for scaling data intensive
systems (see
[The Firebolt Cloud Data Warehouse Whitepaper](https://www.firebolt.io/resources/firebolt-cloud-data-warehouse-whitepaper)).

This decoupling is a double-edged sword. Processing engines and storage layers can de
developed and deployed independently, allowing the data warehouse to flexibly scale to
meet the needs of its users. However, processing engines must now retrieve data from
storage OTN, which is a costly operation that can cause problems when retrieving large
datasets.

The simplest way to solve this problem is by reducing the amount of data a processing
engine must retrieve OTN from the storage layer. This idea, while obvious, is
challenging to implement.

DWH queries typically perform aggregations on large spans of data, returning a small
value (such as an average, sum, or count) to the caller. To serve a count over one
billion rows, a warehouse would need to retrieve massive amounts of data from storage,
compute the count, and then return the value. To reduce network traffic, a DWH can
pre-compute a set of materialized indexes and aggregations (i.e. pre-calculate a
generalized count for common query patterns). A network trip that once required hundreds
of gigabytes of data may now require only a few hundred bytes. Pre-aggregation is
expensive, and the challenge comes in determining the most _useful_ aggregations for a
particular data set (constantly computing an un-queried average on billions of rows is a
massive waste of CPU time).

The above example is extreme, but still outlines the value of performing aggregations
closer to the data source in order to reduce the amount of information transferred OTN.

Delta falls into a category that blends the lines between a data warehouse and a
traditional OLTP database. On the one hand, aggregations are very common (i.e. maximum
value for a sensor over a particular time-range). On the other hand, it's typical for a
user to retrieve massive amounts of raw time-series data for advanced computing (such as
signal processing). The first pattern lends itself well to a decoupled architecture,
while the second benefits greatly from reducing the amount of network hops.

This RFC attempts to reconcile these two workloads by defining an architecture that
separates the components for storing data from those who perform computations on it.
Defining clear requirements and interfaces for the distribution layer is essential to
the success of this reconciliation. What are the algorithms in the distribution layer
responsible for? Should we provide rudimentary support for aggregations? Should we make
the caller aware of the underlying network topology to enable optimization? Or should we
make it a completely black box? The following sections reason about and propose an
architecture that answers these questions.

# 3 - Design

The proposed distribution layer (DL) architecture exposes cluster storage as a
monolithic data space that provides _optional_ locality context to caller. A user can
read and write data from the DL as a black box without any knowledge of the underlying
cluster topology, but can also ask for additional context to perform optimizations
within its own layer/domain.

This is a similar approach to CockroachDB's separation between their
[Distributed SQL](https://github.com/cockroachdb/cockroach/blob/master/docs/RFCS/20160421_distributed_sql.md)
and key-value layers. When executing a query, the SQL layer can turn a logical plan into
a physical plan that executes on a single machine, performing unaware reads and writes
from the distributed kv layer below. It can, however, also construct a physical plan
that moves aggregation logic to the SQL layers' of _ other_ machines in the cluster.
This distributed physical plan can perform aggregations on nodes where the data is
stored, and then return a much smaller result OTN back to the SQL layer of the
responsible node.

Delta's distribution layer plays a similar role to the key-value layer in CRDB. Its main
focus, however, will be to serve time-series segments instead of key-value pairs. Layers
above the DL will do the heavy lifting of generating and executing a physical plan for a
particular query. Parsing a physical plan that can be distributed across multiple nodes
is by no means an easy task. CockroachDB was already several years old before the
development team began to implement these optimizations. By providing topology
abstraction in the distribution layer, we enable a simple path forward to a Delta MVP
while laying the groundwork for distributed query processing.

## 1 - Principles

**Computation and Aggregation** - DL contains no computation or aggregation logic. Its
focus is completely on serving raw segments reads and writes efficiently.

**Network Awareness** - DL's interface does _not_ require the caller to be aware of data
locality or underlying network topology. The distribution layer provides optional
context to the caller if they want to implement optimizations themselves.

**Layer Boundary** - Services/domains that do _not_ require custom distribution logic do
not have any components within DL.

**Domain Oriented** - DL does not expose a single facade as its interface. Instead, it
composes a set of domain-separated services that rely on common distribution logic.

**Generic** - DL only supports rudimentary, low-level queries in a similar fashion to a
key-value store. It should not provide any support for specific data types or specialty
queries.

**Transport Abstraction** - DL is not partial to a particular network transport
implementation (GRPC, WS, HTTP, etc.). It's core logic does not interact with any
specific networking APIs.

## 2 - Storage Engine Integration

Delta's distribution layer directly interacts with two storage engines: Cesium and
Aspen. DL uses
[Aspen](https://github.com/synnaxlabs/synnax/blob/main/docs/tech/rfc/0002-220518-aspen-distributed-storage.md)
for querying cluster topology as well as storing distributed key-value data. It uses one
or more
[Cesium](https://github.com/synnaxlabs/delta/blob/main/docs/rfc/1-220517-cesium-segment-storage.md)
database(s) for reading and writing time-series data from disk. Because the distribution
layer uses multiple storage engines, there's a certain amount of overlap and data
reconciliation that must be performed in order to ensure that information stays
consistent (this is particularly relevant for [channels](#Channels)).

<p align="middle">
    <img
        src="img/0003-220604-segment-distribution/distributed-storage-high-level.png"
        width="80%"
    />
    <h5 align="middle">(Very) High Level Distributed Storage Architecture</h5>
</p>

### 1 - Aspen

Aspen implements two critical pieces of functionality that the distribution layer
depends on. The first is the ability to query the address of a node in the cluster:

```go
addr, err := aspenDB.Resolve(1)
```

This query returns the address of the node with an ID of `1`. The DL uses this to
determine the location of a channel's lease and its corresponding segment data.

The second piece of functionality is an eventually consistent distributed key-value
store. The DL uses aspen to propagate two important pieces of metadata across the
cluster:

1. Channels in the cluster (name, key, data rate, leaseholder node, etc.)
2. Segments for a channel (i.e. what ranges of a channel's data exist on which node).

By performing lookups on this metadata, the DL can serve segment data for any channel
from any node in the cluster.

### 2 - Cesium

Cesium is the main time-series storage engine for Delta. Each Cesium database occupies a
single directory on disk. The distribution layer interacts with Cesium via four APIs:

```go
// Create a new channel.
db.CreateChannel()
// Retrieve a channel.
db.RetrieveChannel()
// Write segments.
db.NewCreate().Exec(ctx)
// Read segments.
db.NewRetrieve().Iterate(ctx)
```

Besides these four interfaces, Delta treats Cesium as a black box.

## 3 - Channels

A channel is a collection of samples across a time-range. The data within a channel
typically arrives from a single source. This can be a physical sensor, software sensor,
metric, event, or any other entity that emits regular, consistent, and time-ordered
values. Channels have a few important fields:

1. Data Rate - The number of samples per second of data (Hz). This data rate is fixed,
   and cannot be changed without deleting and recreating a channel. All data written to
   the channel have the same data rate.
2. Name - A human-readable name for the channel. This name is not used for internal
   purposes.
3. Data Type - An alias for a channel's _density_ i.e. the number of bytes used by a
   single sample. A Float64 channel would have a density of 8 bytes. This data type is
   fixed, and cannot be changed without deleting and recreating a channel. All data
   written to the channel will have the same data type.
4. Key - A unique identifier for the channel across the entire cluster. This key is
   automatically assigned and cannot be changed. See [Keys](#Keys) for more information
   on how this value is selected.
5. Node ID - The ID of the node that owns the channel. This node is known as the
   leaseholder, and is the only node that can write _new_ channel data to disk. The
   leaseholder is typically kept in proximity (physically) to the source generating the
   channel's data (e.g. a sensor).

Channel data is partitioned into entities called _segments_. A segment is a reasonably
sized (roughly one byte to ten megabyte) sub range of a channel's data. When a client
wants to add data to a channel, they submit a set of one or more segments in a write
request. The size of these segments typically grows with the data rate.

It's important to note the only one client can write to a channel at a time. This is
accomplished by acquiring a lock within the cesium storage engine. This helps us to
solve a lot of complicated distributed systems problems (we don't need to implement SSI
and transaction retries, for example).

For more information on channel's and segments, see the
[Cesium RFC](https://github.com/synnaxlabs/delta/blob/main/docs/rfc/220517-cesium-segment-storage.md)
.

### 1 - Keys

A key is a byte array that uniquely identifies a channel across the entire cluster. This
key is composed of two parts.

1. The ID of the leaseholder node for the channel.
2. An auto-incrementing counter for the Cesium DB on the leaseholder node where channel
   data is written to.

Together, these two elements guarantee uniqueness. By keeping the node ID in the key, we
can also avoid needing to make a key-value lookup when resolving the location of the
channel's leaseholder.

## 4 - Segment Reads - Iteration

### 1 - Query Patterns

The distribution layer focuses on serving a single query type: sequential iteration over
large volumes of unprocessed channel data. This 'scan' style pattern serves as the basis
for adding aggregation and computation to layers above. To open a query, a client must
provide two pieces of information:

1. A set of channel keys.
2. A time range.

After opening the query, the distribution layer returns a `segment.Iterator` that
traverses the segments in the range. The caller can seek the iterator to different
positions in the range using the `SeekFirst`, `SeekLast`, `SeekLT` , `SeekGE`, and
`Seek` methods. Once in the correct position, the caller can get the next segment using
the `Next` method, or the previous segment using the `Prev` method.

They can also traverse fixed time spans using the `NextSpan` and `PrevSpan` methods;
these can return a partial segment, multiple segments, or no segments at all. These two
methods are particularly useful for controlling data flow across the cluster. By
altering the time span passed, the caller can control the amount of memory, cpu, and
network bandwidth used.

Internally, the `segment.Iterator` implementation has the following structure:

<p align="middle">
    <img
        src="img/0003-220604-segment-distribution/segment-iterator-gateway.png"
        width="60%"
    />
    <h5 align="middle">Segment Iterator - Gateway Node </h5>
</p>

<p align="middle">
    <img
        src="img/0003-220604-segment-distribution/segment-iterator-peer.png"
        width="40%"
    />
    <h5 align="middle">Segment Iterator - Peer Node </h5>
</p>

### 2 - Opening an Iterator

When a client makes a call to `iterator.New`, the distribution layer assembles the
iterator components in a multi-step process.

1. The DL validates the channel keys to ensure that they exist in the cluster.
2. The DL resolves the leaseholder node for each channel. These nodes are grouped into
   two broad categories: local and remote.
3. If necessary, the DL opens a local iterator on the gateway node for any local
   channels.
4. If necessary, the DL opens a streaming transport to each remote node for any channels
   with non-gateway leaseholders. It then sends an `Open` request containing the keys
   and time-range. The remote node acknowledges the response by opening a local iterator
   on its own data store.

If all of these steps complete successfully, the DL returns the iterator to the client
where it can begin processing requests.

### 3 - Execution Flow

Let's say the caller makes a call to the `First` method (retrieves the first segment in
the range). Let's also say we're reading channel data on nodes 3 (the gateway), 5,
and 7. The execution flow is as follows:

1. The emitter translates the method call into a transportable request, and emits the
   value to the broadcaster. The iterator then makes a call to the synchronizer that
   waits for all nodes (3,5,7) to acknowledge the request execution before returning to
   the caller. This all occurs synchronously within the `First` method body.
2. The broadcaster receives the request, and distributes it to the remote sender and
   local iterator.
3. The local iterator (Node 3) receives the request and executes it on the data store.
   The local iterator responds to the synchronizer with an acknowledgement that the
   request was executed successfully.
4. The remote sender receives the request and sends it to Nodes 5 and 7.
5. Nodes 5 and 7 receive the request and execute it on their data stores. They respond
   to the gateway with an acknowledgement that the request was executed successfully.
6. Receivers for nodes 5 and 7 forward the acknowledgements back to the synchronizer.
7. The synchronizer receives the acknowledgements and returns `true`from the `First`
   method. The caller is now free to make more method calls.
8. The local iterator (node 3) finishes reading the segment from disk and returns it to
   the client.
9. Nodes 5 and 7 finish reading the segment from disk and send it over the network to
   the gateway.
10. The gateway receives the segments from nodes 5 and 7 and returns them to the client.

Two distinct processes occur during a method call: acknowledgement and data transfer.
Acknowledgement, which coordinates iterator validity state across the cluster, is done
synchronously before the method returns. Acknowledgement requires no disk IO and only
small network payloads, making it efficient to do synchronously. Data transfer, on the
other hand, is IO and network intensive. This process is done concurrently; batches of
segments are returned to the caller via a channel. Transport and channel buffers are
used for flow control.

### 4 - Error Handling

Errors are communicated to the caller via iterator validity state during method calls.
If any seeking or traversal calls return `false`, the iterator has either reached the
edges of the range or has encountered an error. In either of these cases, the caller is
expected to make a call to the `Error` or `Close` methods, both of which return error
most recently accumulated by the iterator.

All errors are considered fatal i.e. any error encountered, whether on disk or over the
network will result in the complete shutdown of the iterator. The caller is expected to
open a new iterator to continue operations. This is mainly for simplicity, and future
improvements may include automatic retries and transient error handling.

### 5 - Closing an Iterator

There are two ways of closing an iterator: by cancelling the context provided to
`iterator.New` or calling the `Close` method. Canceling the context immediately aborts
operations and frees all resources. `Close`, on the other hand, shuts the iterator down
gracefully. The process is as follows:

1. The emitter emits a `Close` request to the broadcaster. The iterator then makes a
   call to the synchronizer that waits for all nodes to acknowledge the closure.
2. The emitter closes its output channel, which signals to the broadcaster and sender to
   shut down.
3. Once all nodes return their last segment, they close their network transports.
4. The Gateway node receivers detect the closures and close their output channels.
5. These closures are propagated to the filter, which closes its output channel,
   signaling to the caller that the final segment was returned.
6. `Close` waits for all components to exit before returning any accumulated errors.

By calling `Close`, the caller can ensure that they have received all data from the
iterator before moving on. The distribution layer will track all opened iterators to
ensure that they are closed before the distribution layer is shut down.

## 5 - Segment Writes

The distribution layer must optimize for recording large volumes of sensor data
contiguously. It's common for a single node to receive telemetry from thousands of
sensors. As a result, the DL designs its architecture around long-running writes to many
channels. To open a write query, the caller must provide a slice of channel keys that
they will write segments for. The distribution layer will then open a `segment.Writer`
that allows them to write segments to disk.

Internally, the `segment.Writer` implementation has the following structure:

<p align="middle">
    <img
        src="img/0003-220604-segment-distribution/segment-writer-gateway.png"
        width="60%"
    />
    <h5 align="middle">Segment Writer - Gateway Node </h5>
</p>

<p align="middle">
    <img
        src="img/0003-220604-segment-distribution/segment-writer-peer.png"
        width="40%"
    />
    <h5 align="middle">Segment Writer - Peer Node </h5>
</p>

### 1 - Opening a Writer

When a client opens a `Writer`, the distribution layer assembles the writer components
in a multi-step process similar to the opening of an iterator:

1. The DL validates the channel keys to ensure that they exist in the cluster.
2. The DL resolves the leaseholder node for each. These nodes are grouped into two broad
   categories: local and remote.
3. The DL acquires a **write lock** on each channel. This ensures that no other caller
   can write to the same channel while the writer is open. By acquiring a lock, we can
   avoid complex distributed concurrency issues.
4. If necessary, the DL opens a local writer on the gateway node for any local channels.
5. If necessary, the DL opens a streaming transport to each remote node for any channels
   with non-gateway leaseholders. It then sends an `Open` request containing the keys of
   the channels it will write to. The remote node acknowledges the response by opening a
   local writer on its own data store.

### 2 - Execution Flow

Writer execution is less complex than iteration. Let's say we're writing data to
channels `strain-01`, whose data resides on node 3 (the gateway) and `strain-02` , whose
data resides on node 5. Let's work through the following scenario:

1. The caller sends a segment for `strain-01` through the `Writer` input channel.
2. The remote-local switch determines whether the segment should be written on the
   gateway or a peer node. In this case, the segment is routed to a local writer on the
   gateway.
3. The local writer writes the segment to disk, and pipes any errors encountered through
   its output channel. The error is then returned to the caller via the `Writer` output
   channel.
4. The caller sends a segment for `strain-02` through the `Writer` input channel. The
   `remote-local` switch routes the segment to the remote switch sender.
5. The remote switch sender resolves the address of the peer that the segment should be
   sent to. It then sends the segment to the peer node.
6. The peer node receives the segment and writes it to disk. It then sends any errors
   encountered back over the network to the `Writer`.

### 3 - Closing a Writer

A `Writer` can be closed in two ways: by cancelling the context provided to `writer.New`
or by closing the writers input channel and calling `Close`. Cancelling the context
immediately aborts operations and frees all resources. Closing the input channel, on the
other hand, shuts the writer down gracefully. The code for closing a writer gracefully
resembles the following:

```go
package irrelevant

func main() {
    // Write a segment to disk.
    writer.Requests() <- writer.Request{Segments: MY_SEGMENTS}
    // Once we're done writing, close the input channel.
    close(writer.Requests())
    // Wait for the writer to return  all errors and close
    // its response channel.
    for res := range writer.Responses() {
        // Do something with the error here
        if res.Error != nil {
            log.Error(res.Error)
        }
    }
    // Ensures all writer resources have been freed.
    err := writer.Close()
}
```

When the caller closes the input channel `w.Requests()`, it triggers a waterfall of
channel closures that eventually reach the local writers on both the gateway and remote
nodes, signaling them to shut down. After returning all errors, the local writers close
their error output channels, resulting in another waterfall that eventually results in
the closure of the `Responses()` channel.
