// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package proxy

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/address"
)

// Entry is a value that can be routed to a node in the cluster based on which node
// holds its lease. BatchFactory uses Lease to group entries by their destination node.
type Entry interface {
	// Lease returns the key of the node that holds the lease for this entry. A return
	// of node.KeyFree indicates the entry is not leased to any node.
	Lease() node.Key
}

// BatchFactory partitions Entry values by their leaseholder relative to a fixed host
// node, treating that host as the local node: Batch routes entries leased to it into the
// gateway bucket and all other leased entries to peers. Construct one by converting a
// host node.Key, e.g. BatchFactory[E](host), then call Batch.
type BatchFactory[E Entry] node.Key

// Batch is the result of partitioning a set of Entry values by leaseholder. Every input
// entry appears in exactly one of the three buckets.
type Batch[E Entry] struct {
	// Peers holds entries leased to remote nodes, grouped by leaseholder key.
	Peers map[node.Key][]E
	// Gateway holds entries leased to the local (host) node.
	Gateway []E
	// Free holds entries that are not leased to any node (Lease reports node.KeyFree).
	Free []E
}

// Batch partitions entries into the gateway, peer, and free buckets of the returned
// Batch according to each entry's leaseholder. Entries leased to the factory's host go
// to Gateway, entries with a free lease go to Free, and all remaining entries are
// grouped into Peers by their leaseholder key. The relative order of entries is
// preserved within each bucket.
func (f BatchFactory[E]) Batch(entries []E) Batch[E] {
	host := node.Key(f)
	b := Batch[E]{Peers: make(map[node.Key][]E)}
	for _, entry := range entries {
		lease := entry.Lease()
		if lease.IsFree() {
			b.Free = append(b.Free, entry)
		} else if lease == host {
			b.Gateway = append(b.Gateway, entry)
		} else {
			b.Peers[lease] = append(b.Peers[lease], entry)
		}
	}
	return b
}

// AddressMap maps node keys to their network addresses.
type AddressMap map[node.Key]address.Address
