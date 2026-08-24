// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	distdeleter "github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	distiterator "github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	distrelay "github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	distwriter "github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/writer"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory framer transports for a cluster of nodes. It bundles the
// per-operation networks (deleter, iterator, relay, writer) so that a single Transport
// can be provisioned per node, mirroring the gRPC framer.Transport that bundles the
// same operations.
type Network struct {
	deleter  *deleter.Network
	iterator *iterator.Network
	relay    *relay.Network
	writer   *writer.Network
}

// NewNetwork constructs a Network with freshly initialized per-operation networks.
func NewNetwork() *Network {
	return &Network{
		deleter:  deleter.NewNetwork(),
		iterator: iterator.NewNetwork(),
		relay:    relay.NewNetwork(),
		writer:   writer.NewNetwork(),
	}
}

// New provisions an in-memory framer.Transport for the node at addr. buffers sets the
// channel buffer sizes for the streaming iterator, relay, and writer transports.
func (n *Network) New(addr address.Address, buffers ...int) framer.Transport {
	return transport{
		deleter:  n.deleter.New(addr),
		iterator: n.iterator.New(addr, buffers...),
		relay:    n.relay.New(addr, buffers...),
		writer:   n.writer.New(addr, buffers...),
	}
}

type transport struct {
	deleter  distdeleter.Transport
	iterator distiterator.Transport
	relay    distrelay.Transport
	writer   distwriter.Transport
}

func (t transport) Deleter() distdeleter.Transport { return t.deleter }

func (t transport) Iterator() distiterator.Transport { return t.iterator }

func (t transport) Relay() distrelay.Transport { return t.relay }

func (t transport) Writer() distwriter.Transport { return t.writer }
