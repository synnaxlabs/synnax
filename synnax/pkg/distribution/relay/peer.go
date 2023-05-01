// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"context"

	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/address"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type demand = changex.Change[address.Address, ReadRequest]

type peerReceiver struct {
	Config
	confluence.UnarySink[demand]
	confluence.AbstractUnarySource[Data]
	demands     map[address.Address]channel.Keys
	connections map[core.NodeKey]peerConnection
}

func newPeerReceiver(config Config) confluence.Segment[demand, Data] {
	r := &peerReceiver{
		Config:      config,
		demands:     make(map[address.Address]channel.Keys),
		connections: make(map[core.NodeKey]peerConnection),
	}
	r.Sink = r.sink
	return r
}

func (r peerReceiver) sink(ctx context.Context, ch demand) error {
	// update our demands, so we know what channels we want from what nodes
	nodeKeys := r.updateDemands(ch)
	// make sure we have open connections to all demanded nodes/channels
	r.updateConnections(ctx, nodeKeys)
	return nil
}

func (r peerReceiver) updateDemands(d demand) map[core.NodeKey]channel.Keys {
	if d.Variant == changex.Delete {
		delete(r.demands, d.Key)
	} else {
		r.demands[d.Key] = d.Value.Keys
	}
	nodeDemand := make(map[core.NodeKey]channel.Keys, len(r.connections))
	for _, d := range r.demands {
		for _, k := range d {
			lease := k.Lease()
			if lease != r.HostResolver.HostKey() {
				nodeDemand[lease] = append(nodeDemand[lease], k)
			}
		}
	}
	return nodeDemand
}

func (r peerReceiver) updateConnections(ctx context.Context, nodeKeys map[core.NodeKey]channel.Keys) {
	for node, keys := range nodeKeys {
		if _, ok := r.connections[node]; !ok {
			// We need to open a new connection
			client, err := r.openConnection(ctx, node, keys)
			if err != nil {
				r.L.Error("failed to dial new client")
			}
			r.connections[node] = client
			client.responses.OutTo(r.AbstractUnarySource.Out)
		}
	}

	for addr, client := range r.connections {
		keys, ok := nodeKeys[client.peer]
		if !ok {
			// This will close the stream to the server
			client.requests.Close()
			// If we need this client again, we'll just open an entirely new connection,
			// so we don't need to maintain a reference.
			delete(r.connections, addr)
		} else {
			// If we still need the stream, we'll send the updated key set
			client.requests.Inlet() <- ReadRequest{Keys: keys}
		}
	}
}

func (r peerReceiver) openConnection(ctx context.Context, nodeKey core.NodeKey, keys channel.Keys) (peerConnection, error) {
	addr, err := r.HostResolver.Resolve(nodeKey)
	if err != nil {
		return peerConnection{}, err
	}
	stream, err := r.Transport.Client().Stream(ctx, addr)
	if err != nil {
		return peerConnection{}, err
	}
	return newPeerConnection(ctx, stream), nil
}

type peerConnection struct {
	peer      core.NodeKey
	requests  confluence.Inlet[ReadRequest]
	responses confluence.Source[Data]
}

func newPeerConnection(ctx context.Context, stream ClientStream) (c peerConnection) {
	c.responses = &freightfluence.Receiver[Data]{Receiver: stream}
	sender := &freightfluence.Sender[ReadRequest]{Sender: stream}
	req := confluence.NewStream[ReadRequest](1)
	c.requests = req
	sender.InFrom(req)
	sCtx := signal.Wrap(ctx)
	sender.Flow(sCtx)
	c.responses.Flow(sCtx)
	return c
}
