// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory relay transport for a cluster of nodes.
type Network struct {
	net *mock.Network[relay.Request, relay.Response]
}

// NewNetwork constructs a Network with a freshly initialized relay network.
func NewNetwork() *Network {
	return &Network{net: mock.NewNetwork[relay.Request, relay.Response]()}
}

// New provisions an in-memory relay.Transport whose server is hosted at addr. buffers
// sets the channel buffer sizes for the stream.
func (n *Network) New(addr address.Address, buffers ...int) relay.Transport {
	return transport{
		client: n.net.StreamClient(buffers...),
		server: n.net.StreamServer(addr, buffers...),
	}
}

type transport struct {
	client relay.Client
	server relay.Server
}

var _ relay.Transport = transport{}

func (t transport) Client() relay.Client { return t.client }

func (t transport) Server() relay.Server { return t.server }
