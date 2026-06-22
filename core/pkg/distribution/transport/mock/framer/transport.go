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
	"go/types"

	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory framer transports for a cluster of nodes. It aggregates
// the per-operation freighter networks (writer, iterator, relay, deleter) so that a
// single Transport can be provisioned per node, mirroring the gRPC framer.Transport
// that bundles the same operations.
type Network struct {
	// writer backs the streaming writer transport.
	writer *mock.Network[writer.Request, writer.Response]
	// iterator backs the streaming iterator transport.
	iterator *mock.Network[iterator.Request, iterator.Response]
	// relay backs the streaming relay transport.
	relay *mock.Network[relay.Request, relay.Response]
	// deleter backs the unary deleter transport.
	deleter *mock.Network[deleter.Request, types.Nil]
}

// NewNetwork constructs a Network with freshly initialized per-operation networks.
func NewNetwork() *Network {
	return &Network{
		writer:   mock.NewNetwork[writer.Request, writer.Response](),
		iterator: mock.NewNetwork[iterator.Request, iterator.Response](),
		relay:    mock.NewNetwork[relay.Request, relay.Response](),
		deleter:  mock.NewNetwork[deleter.Request, types.Nil](),
	}
}

// New provisions an in-memory framer.Transport for the node at addr. buffers sets the
// channel buffer sizes for the streaming writer, iterator, and relay transports.
func (n *Network) New(addr address.Address, buffers ...int) framer.Transport {
	return transport{
		writer: writerTransport{
			client: n.writer.StreamClient(buffers...),
			server: n.writer.StreamServer(addr, buffers...),
		},
		iterator: iteratorTransport{
			client: n.iterator.StreamClient(buffers...),
			server: n.iterator.StreamServer(addr, buffers...),
		},
		relay: relayTransport{
			client: n.relay.StreamClient(buffers...),
			server: n.relay.StreamServer(addr, buffers...),
		},
		deleter: deleterTransport{
			client: n.deleter.UnaryClient(),
			server: n.deleter.UnaryServer(addr),
		},
	}
}

type transport struct {
	writer   writerTransport
	iterator iteratorTransport
	relay    relayTransport
	deleter  deleterTransport
}

var _ framer.Transport = transport{}

func (t transport) Writer() writer.Transport { return t.writer }

func (t transport) Iterator() iterator.Transport { return t.iterator }

func (t transport) Relay() relay.Transport { return t.relay }

func (t transport) Deleter() deleter.Transport { return t.deleter }

type writerTransport struct {
	client writer.TransportClient
	server writer.TransportServer
}

func (t writerTransport) Client() writer.TransportClient { return t.client }

func (t writerTransport) Server() writer.TransportServer { return t.server }

type iteratorTransport struct {
	client iterator.TransportClient
	server iterator.TransportServer
}

func (t iteratorTransport) Client() iterator.TransportClient { return t.client }

func (t iteratorTransport) Server() iterator.TransportServer { return t.server }

type relayTransport struct {
	client relay.TransportClient
	server relay.TransportServer
}

func (t relayTransport) Client() relay.TransportClient { return t.client }

func (t relayTransport) Server() relay.TransportServer { return t.server }

type deleterTransport struct {
	client deleter.TransportClient
	server deleter.TransportServer
}

func (t deleterTransport) Client() deleter.TransportClient { return t.client }

func (t deleterTransport) Server() deleter.TransportServer { return t.server }
