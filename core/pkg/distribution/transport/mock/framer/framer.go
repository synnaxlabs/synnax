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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	deletermock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/deleter"
	iteratormock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/iterator"
	relaymock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/relay"
	writermock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/writer"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory framer transports for a cluster of nodes. It bundles the
// per-operation networks (deleter, iterator, relay, writer) so that a single Transport
// can be provisioned per node, mirroring the gRPC framer.Transport that bundles the same
// operations.
type Network struct {
	deleter  *deletermock.Network
	iterator *iteratormock.Network
	relay    *relaymock.Network
	writer   *writermock.Network
}

// NewNetwork constructs a Network with freshly initialized per-operation networks.
func NewNetwork() *Network {
	return &Network{
		deleter:  deletermock.NewNetwork(),
		iterator: iteratormock.NewNetwork(),
		relay:    relaymock.NewNetwork(),
		writer:   writermock.NewNetwork(),
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
	deleter  deleter.Transport
	iterator iterator.Transport
	relay    relay.Transport
	writer   writer.Transport
}

var _ framer.Transport = transport{}

// Deleter implements the framer.Transport interface.
func (t transport) Deleter() deleter.Transport { return t.deleter }

// Iterator implements the framer.Transport interface.
func (t transport) Iterator() iterator.Transport { return t.iterator }

// Relay implements the framer.Transport interface.
func (t transport) Relay() relay.Transport { return t.relay }

// Writer implements the framer.Transport interface.
func (t transport) Writer() writer.Transport { return t.writer }
