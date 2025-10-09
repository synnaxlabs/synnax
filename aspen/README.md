<br />
<p align="center">
    <a href="https://synnaxlabs.com/">
        <img src="../x/media/static/logo/icon-white-padded.png" width="20%"/>
    </a>
    <br />
    <br />
    <a href="https://app.codecov.io/gh/synnaxlabs/synnax">
        <img src="https://img.shields.io/codecov/c/gh/synnaxlabs/synnax?token=6xqpN1pFt8&color=green&style=for-the-badge&logo=codecov&flag=aspen" />
    </a>
</p>

# Aspen

Aspen is a go-embedded, eventually consistent, heavily read optimized, distributed
key-value store. It uses a distributed counter and two gossip models (SI, SIR) to
establish cluster topology and propagate writes. Aspen has no opinion on which key-value
engine to use, and can be configured to use any store that implements the `KV`
interface.

By default, Aspen uses CockroachDB's [pebble](https://github.com/cockroachdb/pebble) as
its key-value store and [gRPC](https://grpc.io/) for communication between nodes.

Aspen implements an observable which can be used to subscribe to changes in the
database. This is useful for building applications that need to react to new or modified
data, such as search indexes or caches.

Aspen leverages a 'lease' based methodology that allows a node to continue writing
certain keys and reading all values even when completely separated frm the network,
synchronizing changes when it rejoins the cluster.

# Stability and Important Considerations

- Aspen is in active development and is not yet ready for production use. The key-value
  API is stable, but the cluster API will likely change.

- Aspen maintains an entire copy of the key-value store on each node in the cluster.
  This results in excellent read performance, but also means total storage requirements
  scale linearly with cluster size.

- Aspen is eventually consistent, meaning that reads may be stale for some period of
  time.

- The gossip protocol lacks three essential features: failure detection, failure
  recovery, and efficient propagation guarantees. These are features that are currently
  in active development.

- While multi-node writes batched writes are supported, they are not yet transactional.
  Single node batch writes are transactional (if the underlying key-value store supports
  them).

## Installation

```
go get github.com/synnaxlabs/aspen
```

## Usage

```go
package mycoolapp

import (
    "context"
    "time"
    "log"
    "github.com/synnaxlabs/aspen"
)

func main() {
    ctx := context.Background()

    // Open the first database with no peers, telling it to bootstrap a new cluster
    // and listen on port 22626.
    db1, err := aspen.Open(
        ctx,
        "aspen/db1",
        "localhost:22646",
        []aspen.Address{},
        aspen.Bootstrap(),
        aspen.MemBacked(),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer db1.Close()

    // Open the second database, without bootstrapping a new cluster, and tell it to
    // pledge membership to the cluster at localhost:22646.
    db2, err := aspen.Open(
        ctx,
        "aspen/db2",
        "localhost:22647",
        // At least **one address of a peer in the cluster must be provided when opening
        // the database for the first time.  After that, the database will remember
        // the cluster topology and will be able to join the cluster without a peer
        // address.
        []aspen.Address{"localhost:22646"},
        aspen.MemBacked(),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer db2.Close()

    if err := db1.Set(ctx, []byte("key"), []byte("value")); err != nil {
        log.Fatal(err)
    }

    // By default, Aspen propagates operations once a second, so we need to wait here
    // for the gossip to propagate.
    time.Sleep(2 * time.Second)

    // Read the value from the second database.
    v, err := db2.Get(ctx, []byte("key"))
	log.Println(string(v))
    // Output: value
    if err != nil {
        log.Fatal(err)
    }

    // We can also get a map of the nodes in the cluster to their assigned
    // uint16 id.
    nodes := db1.Cluster.Nodes(ctx)

    log.Println(nodes[1].Address)
    // Output: localhost:22646
    log.Println(nodes[2].Address)
    // Output: localhost:22647
}
```
