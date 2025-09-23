# 2 - Aspen - Gossip Based Key-Value Store

- **Feature Name** - Aspen, a Gossip Based Peer to Peer Key-Value Store
- **Status** - Complete
- **Start Date** - 2022-05-18
- **Authors** - Emiliano Bonilla

# 0 - Summary

In this RFC I propose an architecture for a gossip based network that can meet Delta's
distributed storage and cluster membership requirements. Gossip based dissemination is
an efficient method for sharing cluster wide state in an eventually consistent fashion.
Delta requires a relatively small distributed store that should ideally be available
even on loss of connection to the rest of the cluster. A Gossip based network lays the
foundations for dynamic cluster membership, failure detection, and the eventual
construction of a strongly consistent store.

This proposal focuses on extreme simplicity to achieve a minimum viable implementation.
It aims to provide only functionality that contributes towards meeting the requirements
laid out in the Synnax Specification.

# 1 - Vocabulary

- **Node** - A machine in the cluster.
- **Cluster** - A group of nodes that can communicate with each other.
- **Redline** - A threshold for a particular channel value. Most often represents a
  safety limit that triggers an action.
- **Initiator** - A node that initiates gossip with another node.
- **Peer** - A node that engages in gossip with an initiating node.

# 2 - Motivation

This RFC is driven by the lack of a distributed key-value store that meets Delta's
needs. The ACID demands of OLTP databases typically require that they fail rather than
risk data loss. This is an ideal property where data integrity is an absolute
requirement (such as financial transactions), but can be disastrous in hardware control
systems.

Consider a set of redlines that execute when a node loses connection. Upon losing
communication with the rest of the cluster, an ACID compliant distributed database would
stop serving reads and writes in order to preserve data integrity. This could hinder a
nodes' ability to shut down the system safely. In extreme scenarios, such as launch
control systems, this can result in the loss of a vehicle or even a life.

Delta requires a distributed data store capable of servicing queries even when the rest
of the cluster is unreachable.

# 3 - Design

Aspen's design consists of two gossip layers:

1. Layer 1 - Uses a Susceptible-Infected (SI) model to spread cluster state in a fashion
   resembling [Apache Cassandra](https://cassandra.apache.org/_/index.html). All nodes
   gossip their version of state at a regular interval. This is used to disseminate
   information about cluster membership and node health. This includes reporting
   information about failed or suspected nodes

2. Layer 2 - Uses a Susceptible-Infected-Recovered (SIR) model to propagate key-value
   sets and deletes in an eventually consistent manner. After receiving a set operation,
   the node will gossip the key-value pair to all other nodes until a certain number of
   redundant conversations (i.e. the node already received the update) have occurred.

## 1 - Cluster State Synchronization

Delta aims to provide dynamic cluster membership. This is more difficult to accomplish
if each node is required to know about _all_ other nodes in the cluster before being
initialized. This is the approach taken by [etcd](https://etcd.io/). By using a
gossip-based network, Delta can provide a cluster membership system that is dynamic and
resilient to failure.

This cluster membership and state gossip is considered Layer 1. Layer 1 is implemented
using a Susceptible-Infected (SI) model. In an SI gossip model, nodes never stop
spreading a message. This means quite a bit of network message amplification, but is
useful when it comes to failure detection and membership.

### 1 - Cluster State Data Structure

Layer 1 holds cluster state in a node map `map[NodeID]Node`. `NodeID` is a unique
`int16` identifier for each node. `Node` holds various pieces of identifying information
about the node along with its current state.

```go
package irrelevant

// NodeID is a unique identifier for a node.
type NodeID int16

type Node struct {
    ID NodeID
    // Address is a reachable address for the node.
    Address address.Address
    // Version is software version of the node.
    Version version.MajorMinor
    // Heartbeat is the gossip heartbeat of the node. See [Heartbeat] for more.
    Heartbeat Heartbeat
    // Various additions such as failure detection state, etc. will go here.
}

type NodeMap map[NodeID]Node
```

A node's Heartbeat tracks two values:

```go
package irrelevant

type Heartbeat struct {
    // Version is incremented every time the node gossips information about its state. This is
    //used to merge differing versions of cluster state during gossip.
    Version uint32
    // Generation is incremented every time the node is restarted. This is useful for bringing a node
    // back up to speed after a long period of absence.
    Generation uint16
}
```

### 2 - Anatomy of a Conversation

#### 1 - Sync Message

A node initiates conversation with another node by sending a 'sync' message to another
node. This message contains a list of node digests.

```go
package irrelevant

type Digest struct {
    // NodeID is the node's unique identifier
    ID NodeID
    // Heartbeat is the gossip heartbeat.
    Heartbeat Heartbeat
}

type SyncMessage struct {
    Digests []Digest
}
```

A digest is added to the message for every node in the initiator's state.

#### 2 - Ack Message

After receiving a sync message from the initiator node, the peer node will respond with
an ack message:

```go
package irrelevant

type AckMessage struct {
    // A list of digests for nodes in the peer's state that:
    //
    //    1. The peer node has not seen.
    //    2. Have an older heartbeat than in the sender's Digest.
    //
    Digests []Digest
    // A NodeMap of nodes in the peer's state that:
    //
    //    1. The initiating node has not seen.
    //    2. Have an older heartbeat than in the sender's Digest.
    NodeMap NodeMap
}

```

The peer node makes no updates during this period.

#### 3 - Ack2 Message

After receiving an ack message from the peer, the initiator updates its own state and
responds with a final ack2 message. The initiator compares the heartbeat of every node
in the `AckMessage.NodeMap` with its on state. If the peer sent a new node or a node
with an older heartbeat, the initiator's will replace the node in its state with the
node from the peer. It will them compose a new message:

```go
package irrelevant

type Ack2Message struct {
    // A NodeMap of nodes in the initiator's state that:
    //  1. Are in the peer's ack digests.
    NodeMap NodeMap
}
```

#### 4 - Closing the Conversation

After receiving the final ack2 message, the initiator will update its own state using
the same policy as the peer in the section. It will then close the conversation.

### 2 - Propagation Rate

The propagation rate of cluster state is tuned by the interval at which a node gossips.
Higher propagation rates will result in heavier network traffic, so it's up to the
application to determine the appropriate balance.

## 2 - Adding a Member

Aspen employs a relatively complex process for joining a node to a cluster. This is due
to a desire to identify nodes using a unique `int16` value. The ID of a node is
propagated with almost every message. By using an `int16` vs. `UUID`, we can reduce
overall network traffic by a significant amount. Node IDs are also used far and wide
across the rest of Delta, such as in the key for a channel `NodeID + ChannelID`. This
results in a sample that is 40 percent smaller than with a `UUID`.

The downside of using `int16` id's for nodes is that we need to design a distributed
counter. Fortunately, this is a solved problem. The join process is as follows:

### 1 - Request a Peer to Join

When joining a new node to a cluster, the joining node (known as the **pledge**)
receives a set of one or more peer addresses of other nodes in the cluster. The
**pledge** node will choose a peer at random and send a join request to it. If the peer
acknowledges the request, the joining node will then wait for a second message assigning
it an ID. If the peer rejects the request or doesn't respond, it attempts to send the
request to another peer. This cycle continues until a peer acknowledges the request or a
preset threshold is reached.

The peer that accepts the **pledge** join request is known as the **responsible**. This
node is responsible for safely initiating the **pledge**.

### 2 - Propose an ID

The **responsible** node will begin the initiation process by finding the highest id of
the nodes within its state. It will then select a quorum (>50%) of its peers and send a
proposed id with a value one higher. It will then wait for all peers to approve the
proposed value (these peers are called **jurors**). A juror will approve the value if it
does not have a node in its state with the given ID. A **juror** tracks the results of
all accepted proposals until the state of the accepted **pledge** has been disseminated.
The approval process is serialized by a mutex.

If any node rejects the id, the **responsible** node will reissue the proposal with an
incremented value. This process continues until an ID is accepted. If the
**responsible** node tries to contact an unresponsive peer, it will reselect a quorum of
peers and try again. Once an ID is selected, the **responsible** node will send it to
the **pledge**.

### 3 - Disseminate New Node

Once the **pledge** receives an ID assignment from the **responsible** node, it will
begin to gossip its state to the rest of the cluster. As information about the new node
spreads, **jurors** will remove processed approvals from their state.

### 5 - The First Node

The first node to join the cluster is provided with no peer addresses. It will
automatically assign itself an ID of 1.

### 6 - Implications of Algorithm

Using a quorum based approach to ID assignment means that we get a strong guarantee that
a node will be assigned a unique identifier. It also means that a cluster with less than
half of its nodes available will not be able to add new members. This is an important
property to consider in scenarios with extremely dynamic cluster membership.

## 3 - Key-Value Store

Aspen implements a leased driven key-value store on top of layer 1. The gossip protocol
that disseminates kv updates and tombstones is known as layer 2.

### 1 - Vocabulary

**Host** - The node that is responsible for serving the kv operation to the caller (i.e.
the node where `Get` or `Set` is called). \
**Leaseholder** - The only node that can accept writes for a particular key.

### 2 - Interface

At the simplest level, the key-value store implements the following interface.

```go
package irrelevant

type NodeID int16

type KV interface {
    // Set sets a key to a value. nodeID specifies the node that holds the lease on the key.
    // If nodeID is 0, the lease is assigned to the host node.
    Set(key []byte, leaseholder NodeID, value []byte) error
    // Rest of interface is the same as github.com/synnaxlabs/x/kv.DB.
}
```

### 2 - Life of a Set/Delete

A kv set is processed by the database as follows. It's important to note that deletes
and sets are both propagated using the same steps.

#### 1 - Forward Request to Leaseholder

If the node ID is non-zero, perform a layer 1 lookup for the leaseholder's address.
Forward the request to the leaseholder. If the node ID is zero, allocate the least to
the host node.

#### 2 - Process the Forwarded Set

Add the key-value pair to an update propagation list. This list has the following
structure:

```go
package irrelevant

type UpdateState byte

const (
    // StateInfected means the node is actively gossiping the update to other nodes in the cluster.
    StateInfected UpdateState = iota
    // StateRecovered means the node is no longer gossiping the update.
    StateRecovered
)

type Operation byte

const (
    // OperationSet represents a kv set operation.
    OperationSet Operation = iota
    // OperationDelete represents a kv delete operation.
    OperationDelete
)

type Update struct {
    // Key is the key for the key-value pair.
    Key []byte
    // Value is the value for the key-value pair.
    Value []byte
    // Leaseholder is the ID of the leaseholder node.
    Leaseholder NodeID
    // State is the SIR state of the update.
    State UpdateState
    // Version is incremented every time an existing key is updated.
    Version int32
    // Operation is the operation type of the update.
    Operation Operation
}

type UpdatePropagationList map[interface{}]Update
```

After adding the update to the propagation list, we persist the set to an underlying kv
store, and send a durability acknowledgement to the host node.

#### 3 - Propagate the Update

A node will initiate layer 2 gossip at a set interval (default is 1 second). The gossip
process is as follows:

#### 1 - Initiator Propagates Update (Sync)

The initiating node selects a random peer from layer 1, and set

1. Select a random peer from layer 1, and send a sync message:

```go
package irrelevant

type SyncMessage struct {
    // Updates contains a list of all updates in the nodes current state where:
    // 1. Update.State == StateInfected
    Updates UpdatePropagationList
}
```

#### 2 - Peer Processes Update and Response (Ack)

After receiving a sync message, the peer node [merges](#merging-updates) the updates
into its own state. The node also persists the updates to state. The peer node then
sends the following ack message back to the initiator:

```go
package irrelevant

// Feedback is a struct representing an update that has already been processed by a node.
type Feedback struct {
    Key     []byte
    Version int32
}

type AckMessage struct {
    // Updates contains a list of all updates in the nodes current state that:
    //   1. Update.State == StateInfected
    //   2. Are not already in the peer node's update list.
    //   3. Have a higher version than the peer node's update.
    Updates UpdatePropagationList
    // Feedback is a list of Feedback for the updates a node already has
    // (versions must be identical).
    Feedback []Feedback
}
```

#### 3 - Initiator Processes Update

After receiving an ack message, the initiator [merges ](#merging-updates) the updates
into its own state. Then, for each feedback entry, it flips a coin with a `k`
probability of returning true. If the coin is true, sets the state of the update with
the matching key to `StateRemoved`. End of gossip.

### 3 - Merging Updates

Whenever a node receives an `UpdatePropagationList` from another node, it must merge the
updates into its own state. This process is relevant to steps B and C of the layer 2
gossip algorithm. Each update in the list is merged as follows:

1. If the remote update isn't present in local memory, do a KV lookup to see if we've
   persisted the update already. If we haven't, add to internal state.

2. The remote update is newer than the version in internal state. If this is the case,
   add the update to internal state.

After the updates to merge have been selected, the node must persist them to KV.

### 3 - Life of a Get

Aspen does not support remote get requests. If a key cannot be found in underlying KV,
returns a `ErrNotFound`. This decision was made for two reasons:

1. We maintain a consistent view of storage even when other cluster members cannot be
   reached.
2. We can simply extend the kv interface of an existing store, providing functionality
   such as prefix iteration.

Providing consistent remote reads is an undertaking for future iterations.

### 4 - Garbage Collection

An update only needs to be kept until it has propagated to all cluster members.
Unfortunately, determining the interval of convergence is difficult. Aspen uses the
equations presented in [Gossip](https://www.inf.u-szeged.hu/~jelasity/ddm/gossip.pdf) to
estimate a interval of convergence. The following equation approximates the number of
message (`m`) needed to update all but `s` proportion of the cluster:

<p align="middle">
    <br />
    <img
        src="https://render.githubusercontent.com/render/math?math=m \approx -N\text{log}s"
        height="30px"
        alt="latex eq"
    />
</p>

This equation shows that the convergence interval is directly dependent on:

1. The `k` parameter laid out in step c of the layer 2 gossip algorithm.
2. The number of cluster members (`N`)
3. Our consistency requirements (`s`) i.e. the probability that a node does not receive
   an update.
4. The frequency at which infected nodes can send `m` update messages.

Aspen will tune these parameters to experimentally determine a suitable estimate for the
interval of convergence. Future iterations will likely involve a more complex,
mathematically driven approach.

Once we've determined a suitable interval, we can them move on to the GC process itself.

An update log for a particular key is replaced every time the key is updated ( i.e. the
version is incremented). We can store a timestamp along with the set metadata. If the
duration between the timestamp and GC is greater than the convergence interval, we can
remove the update from persisted storage. The same process applies for both set
operations and delete tombstones.

### 5 - Recovery Constant

The same convergence interval approximation can be used for determining how quickly to
recover an operation from gossip. Aspen will use a variable redundant gossip threshold
to determine whether to recover an operation. This means that a node must gossip and
receive feedback on an operation at least `N` times before being recovered.

For smaller clusters, this threshold can stay fixed, but for larger clusters, we must
use a variable threshold that scales along with the interval of convergence for the
cluster. We're leaving this out of the scope of this RFC, as it isn't necessary for a
minimum viable implementation.

## 4 - Failure Detection

Failure detection (FD) algorithms are essential for building robust distributed systems.
They're also very challenging to implement. While this RFC lays the groundwork for
adding FD, it does not explicitly implement any processes.

One of the most relevant elements of the FD mechanism design is providing an interface
that can notify a consumer when a node fails. Aspen uses an observable copy-on-read
store to accomplish this. When a cluster receives a state change update, Aspen will diff
the old and new states, notifying an subscribers of the change.

As far as future implementation goes, the high level plan follows theory laid out in
["SWIM: Scalable Weakly-consistent Infection-style Process Group Membership Protocol"](https://ieeexplore.ieee.org/document/1028914)
. This involves piggybacking on layer 1 cluster gossip to mark nodes as susceptible or
failed. This is the same strategy that hashicorp's `memberlist` uses. Detailed designs
are left for future RFCs.

## 5 - Failure Recovery

Recovering from failures gracefully follows as a natural extension of detecting them.

### 1 - Key-Value Storage

Failure Recovery is particularly relevant with regard to maintaining a consistent
key-value store. If a node restarts after an extended outage, it will likely miss
key-value operations that have already recovered. As a result, it's important for a
recently revived node to receive all missed operations.

There are a number of approaches to solving the above-mentioned problem. I'll cover a
few of the most natural, and then propose a viable solution.

#### 1 - Forward Entire Store

This is definitely the simplest approach. When a node recovers and begins to gossip its
state once again, a peer detects the recovery, and starts forwarding its entire state
using the following algorithm (we'll call the 'recovering' node the patient and the
forwarding peer the 'doctor'):

1. The doctor detects the patient's recovery, and sends a message to acquire a role as
   the patient's doctor. The patient must acknowledge this request for the doctor to
   continue to the next step. This is done in order to prevent too many nodes in the
   cluster from forwarding their state, clogging network traffic.

2. After the request is approved, the doctor takes a snapshot of the DB and sends it to
   the patient. The patient ingests the snapshot. The ingestion process is blind,
   meaning the patient must be careful NOT to persist any gossiped operations that may
   be overwritten.

This approach has its drawbacks. The most obvious is the large volume of data that needs
to be sent over the network. A large portion of the data we're sending is redundant. The
second is the blind ingestion process; this leaves a node susceptible to bugs that
involve an old operation accidentally overwriting a newer one that has already
propagated.

#### 2 - Forward Entire Store as Gossip

This second approach involves essentially the same process, except that patient
ingestion is done through the normal gossip pipeline. This allows a node to actively
check whether it's overwriting an old version. It also minimizes complexity by reusing
the same software infrastructure to forward recovery operations.

The problem is that this approach involves using the same amount of network traffic AND
is far more inefficient than doing something like a blind `SSTable` ingestion with
`pebble`.

#### 3 - High Water Marking

My proposed solution to the above inefficiencies involves a technique called high water
marking. Each node keeps track of a "high water" mark that represents the highest
version that a node has received from another node. This data structure can be
represented using the following map:

```go
type HighWaterMarks map[node.ID]version.Counter
```

Assigning an accurate high water mark to a node is far more challenging than it first
seems. It's not given that a node will receive operations in sequential order. Recovery
parameters or oddities in the network mean that node A may receive operation
`version.Counter(3)` before it receives `version.Counter(2)` from node B.If Node A
blindly updates its high water mark version for Node B to 3, it's inadvertently saying
it has received operation 2, which is not true. This is a rare case, but must be handled
properly in order to maintain effective eventual consistency.

Node A will need to track the "gaps" in its high water mark using some sort of virtual
WAL:

```go
type HighWaterMarks map[node.ID]struct{
// Mark indicates the most recent version whose history is contiguous (i.e. all previous versions have been received).
Mark version.Counter
// WAL keeps a log of all versions received non-contiguously. This log can be garbage collected after a contiguous
// run of versions have been received.
WAL []version.Counter
}
```

To put our earlier example into practice, this is what the high water mark map would
hold for node A:

```go
nodeAHighWater := HighWaterMarks{B: {Mark: version.Counter(1), WAL: []version.Counter{3}}}
```

After node A receives operation 2, the high water marks can be updated:

```go
nodeAHighWater := HighWaterMarks{B: {Mark: version.Counter(3), WAL: []version.Counter{}}}
```

High water marks can help in making Aspen's key-value failure recovery system more
efficient. After the patient node receives a doctor's request, it can acknowledge the
request with a list of operations that contain it's last remembered high water marks.

The doctor node can now avoid sending all operations before the high water mark,
reducing the amount of network traffic substantially.

#### 4 - Concrete Algorithm

The concrete algorithm for recovering a node's key-value store after failure:

1. The doctor detects the patient's recovery, and sends a message to acquire a role as
   the patient's doctor. The patient either denies the request or forwards its high
   water marks to the doctor.

2. The doctor processes the high water marks, and gossips the missing operations to the
   patient.

3. The patient persists the operations to storage.

The number of doctors can be scaled to reduce the chances of failure during recovery.
The use case for this algorithm could potentially be extended to bringing new nodes up
to speed when they join the cluster (although in this case it might be better to use the
faster SSTable ingestion).

All of these algorithms are designed around the principle that Aspen is intended for
small amounts of data (~100,000 keys MAX).
