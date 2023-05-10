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
	"github.com/synnaxlabs/x/signal"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

type demand = changex.Change[address.Address, Request]

type receiveCoordinator struct {
	Config
	sGo signal.Context
	confluence.UnarySink[demand]
	confluence.AbstractUnarySource[Response]
	demands   map[address.Address]channel.Keys
	receivers map[core.NodeKey]confluence.Inlet[Request]
}

func newReceiveCoordinator(config Config) confluence.Segment[demand, Response] {
	sCtx, _ := signal.Isolated()
	r := &receiveCoordinator{
		Config:    config,
		demands:   make(map[address.Address]channel.Keys),
		receivers: make(map[core.NodeKey]confluence.Inlet[Request]),
		sGo:       sCtx,
	}
	r.Sink = r.sink
	return r
}

func (c *receiveCoordinator) sink(ctx context.Context, ch demand) error {
	// update our demands, so we know what channels we want from what nodes
	nodeKeys := c.updateDemands(ch)
	// make sure we have open receivers to all demanded nodes/channels
	c.updateConnections(ctx, nodeKeys)
	return nil
}

func (c *receiveCoordinator) updateDemands(d demand) map[core.NodeKey]channel.Keys {
	if d.Variant == changex.Delete {
		delete(c.demands, d.Key)
	} else {
		c.demands[d.Key] = d.Value.Keys
	}
	nodeDemand := make(map[core.NodeKey]channel.Keys, len(c.receivers))
	for _, d := range c.demands {
		for _, k := range d {
			nodeDemand[k.Lease()] = append(nodeDemand[k.Lease()], k)
		}
	}
	return nodeDemand
}

func (c *receiveCoordinator) updateConnections(
	ctx context.Context,
	nodeKeys map[core.NodeKey]channel.Keys,
) {
	// Open any new connections we may need
	for node, keys := range nodeKeys {
		if _, ok := c.receivers[node]; !ok {
			rcv, err := c.openReceiver(ctx, node, keys)
			if err != nil {
				c.L.Error("failed to dial new r")
			}
			requests := confluence.NewStream[Request](1)
			rcv.InFrom(requests)
			rcv.OutTo(c.AbstractUnarySource.Out)
			rcv.Flow(c.sGo, confluence.CloseInletsOnExit())
			c.receivers[node] = requests
		}
	}

	// Update or close any connections we don't need
	for addr, r := range c.receivers {
		keys, ok := nodeKeys[addr]
		if !ok {
			// This will close the stream to the server
			r.Close()
			// If we need this r again, we'll just open an entirely new connection,
			// so we don't need to maintain a reference.
			delete(c.receivers, addr)
		} else {
			// If we still need the stream, we'll send the updated key set
			r.Inlet() <- Request{Keys: keys}
		}
	}
}

func (c *receiveCoordinator) openReceiver(
	ctx context.Context,
	nodeKey core.NodeKey,
	keys channel.Keys,
) (receiver, error) {
	if nodeKey == c.HostResolver.HostKey() {
		sr, err := c.TS.NewStreamReader(ctx, ts.StreamReaderConfig{
			Channels: keys.Storage(),
		})
		if err != nil {
			return nil, err
		}
		return newHostReceiver(sr), nil
	}
	addr, err := c.HostResolver.Resolve(nodeKey)
	if err != nil {
		return nil, err
	}
	stream, err := c.Transport.Client().Stream(ctx, addr)
	if err != nil {
		return nil, err
	}
	return newPeerReceiver(stream), nil
}

// receiver receives written telemetry from peer nodes or from the host's storage layer.
type receiver = confluence.Segment[Request, Response]

func newPeerReceiver(stream ClientStream) receiver {
	receiver := &freightfluence.Receiver[Response]{Receiver: stream}
	sender := &freightfluence.Sender[Request]{Sender: stream}
	p := plumber.New()
	plumber.SetSink[Request](p, "sender", sender)
	plumber.SetSource[Response](p, "receiver", receiver)
	seg := &plumber.Segment[Request, Response]{Pipeline: p}
	lo.Must0(seg.RouteOutletFrom("receiver"))
	lo.Must0(seg.RouteInletTo("sender"))
	return seg
}

func newHostReceiver(reader ts.StreamReader) receiver {
	return confluence.NewTranslator(
		reader,
		reqToStorage,
		resFromStorage,
	)
}
