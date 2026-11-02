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
	"context"
	"sync/atomic"

	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory control transports for a cluster of nodes. It aggregates
// the per-operation freighter networks so that a single Transport can be provisioned
// per node, mirroring the gRPC control.Transport that bundles the same operations.
type Network struct {
	retrieve  *freightertestutil.Network[control.RetrieveRequest, control.RetrieveResponse]
	subscribe *freightertestutil.Network[control.SubscribeRequest, control.SubscribeResponse]
	live      atomic.Int64
}

func NewNetwork() *Network {
	return &Network{
		retrieve: freightertestutil.NewNetwork[control.RetrieveRequest, control.RetrieveResponse](),
		subscribe: freightertestutil.NewNetwork[
			control.SubscribeRequest,
			control.SubscribeResponse,
		](),
	}
}

// New provisions an in-memory control.Transport for the node at addr. buffers sets the
// channel buffer size for the subscription streams.
func (n *Network) New(addr address.Address, buffers ...int) control.Transport {
	return transport{
		retrieveClient: n.retrieve.UnaryClient(),
		retrieveServer: n.retrieve.UnaryServer(addr),
		subscribeClient: countingClient{
			SubscribeClient: n.subscribe.StreamClient(buffers...),
			live:            &n.live,
		},
		subscribeServer: n.subscribe.StreamServer(addr, buffers...),
	}
}

// Subscriptions returns how many subscription streams opened by clients on the network
// have received a response, including streams since closed.
func (n *Network) Subscriptions() int { return int(n.live.Load()) }

type countingClient struct {
	control.SubscribeClient
	live *atomic.Int64
}

func (c countingClient) Stream(
	ctx context.Context,
	target address.Address,
) (control.SubscribeClientStream, error) {
	stream, err := c.SubscribeClient.Stream(ctx, target)
	if err != nil {
		return nil, err
	}
	return &countingStream{SubscribeClientStream: stream, live: c.live}, nil
}

type countingStream struct {
	control.SubscribeClientStream
	live     *atomic.Int64
	received bool
}

func (s *countingStream) Receive() (control.SubscribeResponse, error) {
	res, err := s.SubscribeClientStream.Receive()
	if err == nil && !s.received {
		s.received = true
		s.live.Add(1)
	}
	return res, err
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
