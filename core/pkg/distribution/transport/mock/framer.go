// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/address"
)

// FramerNetwork aggregates the per-operation framer networks (writer, iterator, relay,
// and deleter) so that a single in-memory framer.Transport can be provisioned per node,
// mirroring the gRPC framer.Transport that bundles the same operations.
type FramerNetwork struct {
	// Writer is the network backing the streaming writer transport.
	Writer *FramerWriterNetwork
	// Iterator is the network backing the streaming iterator transport.
	Iterator *FramerIteratorNetwork
	// Relay is the network backing the streaming relay transport.
	Relay *FramerRelayNetwork
	// Deleter is the network backing the unary deleter transport.
	Deleter *FramerDeleterNetwork
}

// NewFramerNetwork constructs a FramerNetwork with freshly initialized per-operation
// networks.
func NewFramerNetwork() *FramerNetwork {
	return &FramerNetwork{
		Writer:   NewWriterNetwork(),
		Iterator: NewIteratorNetwork(),
		Relay:    NewRelayNetwork(),
		Deleter:  NewDeleterNetwork(),
	}
}

// New provisions an in-memory framer.Transport for the node at addr. buffers sets the
// channel buffer sizes for the streaming writer, iterator, and relay transports.
func (n *FramerNetwork) New(addr address.Address, buffers ...int) framer.Transport {
	return FramerTransport{
		writer:   n.Writer.New(addr, buffers...),
		iterator: n.Iterator.New(addr, buffers...),
		relay:    n.Relay.New(addr, buffers...),
		deleter:  n.Deleter.New(addr),
	}
}

// FramerTransport is an in-memory implementation of the framer.Transport interface.
type FramerTransport struct {
	writer   writer.Transport
	iterator iterator.Transport
	relay    relay.Transport
	deleter  deleter.Transport
}

var _ framer.Transport = FramerTransport{}

// Writer implements the framer.Transport interface.
func (t FramerTransport) Writer() writer.Transport { return t.writer }

// Iterator implements the framer.Transport interface.
func (t FramerTransport) Iterator() iterator.Transport { return t.iterator }

// Relay implements the framer.Transport interface.
func (t FramerTransport) Relay() relay.Transport { return t.relay }

// Deleter implements the framer.Transport interface.
func (t FramerTransport) Deleter() deleter.Transport { return t.deleter }
