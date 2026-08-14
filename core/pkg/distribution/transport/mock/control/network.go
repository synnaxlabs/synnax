// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory control transports for a cluster of nodes. It aggregates
// the per-operation freighter networks so that a single Transport can be provisioned
// per node, mirroring the gRPC control.Transport that bundles the same operations.
type Network struct {
	retrieve  *mock.Network[control.RetrieveRequest, control.RetrieveResponse]
	subscribe *mock.Network[control.SubscribeRequest, control.SubscribeResponse]
}

// NewNetwork constructs a Network with freshly initialized per-operation networks.
func NewNetwork() *Network {
	return &Network{
		retrieve: mock.NewNetwork[control.RetrieveRequest, control.RetrieveResponse](),
		subscribe: mock.NewNetwork[
			control.SubscribeRequest,
			control.SubscribeResponse,
		](),
	}
}

// New provisions an in-memory control.Transport for the node at addr. buffers sets the
// channel buffer size for the subscription streams.
func (n *Network) New(addr address.Address, buffers ...int) control.Transport {
	return transport{
		retrieveClient:  n.retrieve.UnaryClient(),
		retrieveServer:  n.retrieve.UnaryServer(addr),
		subscribeClient: n.subscribe.StreamClient(buffers...),
		subscribeServer: n.subscribe.StreamServer(addr, buffers...),
	}
}

type transport struct {
	retrieveClient  control.RetrieveClient
	retrieveServer  control.RetrieveServer
	subscribeClient control.SubscribeClient
	subscribeServer control.SubscribeServer
}

func (t transport) RetrieveClient() control.RetrieveClient { return t.retrieveClient }

func (t transport) RetrieveServer() control.RetrieveServer { return t.retrieveServer }

func (t transport) SubscribeClient() control.SubscribeClient { return t.subscribeClient }

func (t transport) SubscribeServer() control.SubscribeServer { return t.subscribeServer }
