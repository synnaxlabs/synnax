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
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/x/address"
)

type Entry interface {
	Lease() aspen.NodeKey
}

type BatchFactory[E Entry] struct {
	Host aspen.NodeKey
}

type Batch[E Entry] struct {
	Peers   map[cluster.NodeKey][]E
	Gateway []E
	Free    []E
}

func (f BatchFactory[E]) Batch(entries []E) Batch[E] {
	b := Batch[E]{Peers: make(map[cluster.NodeKey][]E)}
	for _, entry := range entries {
		lease := entry.Lease()
		if lease.IsFree() {
			b.Free = append(b.Free, entry)
		} else if lease == f.Host {
			b.Gateway = append(b.Gateway, entry)
		} else {
			b.Peers[lease] = append(b.Peers[lease], entry)
		}
	}
	return b
}

type AddressMap map[cluster.NodeKey]address.Address
