// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/address"
)

type Entry interface {
	Lease() aspen.NodeKey
}

type BatchFactory[E Entry] interface {
	Batch(entries []E) Batch[E]
}

type batchFactory[E Entry] struct {
	host aspen.NodeKey
}

type Batch[E Entry] struct {
	Gateway []E
	Peers   map[core.NodeKey][]E
}

func NewBatchFactory[E Entry](host aspen.NodeKey) BatchFactory[E] { return batchFactory[E]{host} }

func (p batchFactory[E]) Batch(entries []E) Batch[E] {
	b := Batch[E]{Peers: make(map[core.NodeKey][]E)}
	for _, entry := range entries {
		lease := entry.Lease()
		if lease == p.host {
			b.Gateway = append(b.Gateway, entry)
		} else {
			b.Peers[lease] = append(b.Peers[lease], entry)
		}
	}
	return b
}

type AddressMap map[core.NodeKey]address.Address
