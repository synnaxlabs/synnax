// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory iterator transport for a cluster of nodes.
type Network struct {
	net *mock.Network[iterator.Request, iterator.Response]
}

// NewNetwork constructs a Network with a freshly initialized iterator network.
func NewNetwork() *Network {
	return &Network{net: mock.NewNetwork[iterator.Request, iterator.Response]()}
}

// New provisions an in-memory iterator.Transport whose server is hosted at addr. buffers
// sets the channel buffer sizes for the stream.
func (n *Network) New(addr address.Address, buffers ...int) iterator.Transport {
	return transport{
		client: n.net.StreamClient(buffers...),
		server: n.net.StreamServer(addr, buffers...),
	}
}

type transport struct {
	client iterator.Client
	server iterator.Server
}

var _ iterator.Transport = transport{}

func (t transport) Client() iterator.Client { return t.client }

func (t transport) Server() iterator.Server { return t.server }
