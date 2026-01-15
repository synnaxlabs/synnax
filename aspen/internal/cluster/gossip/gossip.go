// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gossip

import (
	"context"
	"time"

	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/rand"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type Gossip struct{ Config }

// New opens a new Gossip that will spread cluster state to and from the given store.
func New(cfgs ...Config) (*Gossip, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	g := &Gossip{Config: cfg}
	g.TransportServer.BindHandler(g.process)
	return g, nil
}

// GoGossip starts a goroutine that gossips at Config.Interval.
func (g *Gossip) GoGossip(ctx signal.Context) {
	g.R.Prod("gossip", g.Config)
	g.L.Info("starting cluster gossip")
	g.L.Debug("config", g.Config.Report().ZapFields()...)
	signal.GoTick(
		ctx,
		g.Interval,
		func(ctx context.Context, t time.Time) error {
			if err := g.GossipOnce(ctx); err != nil {
				g.L.Error("gossip failed", zap.Error(err))
			}
			return nil
		},
		signal.WithKey("gossip"),
	)
}

func (g *Gossip) GossipOnce(ctx context.Context) (err error) {
	snap := g.Store.CopyState()
	peer := RandomPeer(snap.Nodes, snap.HostKey)
	g.incrementHostHeartbeat(ctx)
	if peer.Address != "" {
		err = g.GossipOnceWith(ctx, peer.Address)
	}
	return
}

func (g *Gossip) GossipOnceWith(ctx context.Context, addr address.Address) error {
	sync := Message{Digests: g.Store.CopyState().Nodes.Digests()}
	ack, err := g.TransportClient.Send(ctx, addr, sync)
	if err != nil {
		return err
	}
	ack2 := g.ack(ctx, ack)
	if len(ack2.Nodes) == 0 {
		return nil
	}
	_, err = g.TransportClient.Send(ctx, addr, ack2)
	return err
}

func (g *Gossip) incrementHostHeartbeat(ctx context.Context) {
	host := g.Store.GetHost()
	host.Heartbeat = host.Heartbeat.Increment()
	g.Store.SetNode(ctx, host)
}

func (g *Gossip) process(ctx context.Context, msg Message) (Message, error) {
	ctx, span := g.T.Debug(ctx, "gossip-server")
	defer span.End()
	switch msg.variant() {
	case messageVariantSync:
		return g.sync(msg), nil
	case messageVariantAck2:
		g.ack2(ctx, msg)
		return Message{}, nil
	}
	err := errors.New("[gossip] - received unknown message variant")
	g.L.DPanic(err.Error(), zap.Any("msg", msg))
	return Message{}, span.EndWith(err)
}

func (g *Gossip) sync(sync Message) (ack Message) {
	snap := g.Store.CopyState()
	ack = Message{Nodes: make(node.Group), Digests: make(node.Digests)}
	for _, dig := range sync.Digests {
		n, ok := snap.Nodes[dig.Key]

		// If we have a more recent version of the node, return it to the initiator.
		if ok && n.Heartbeat.OlderThan(dig.Heartbeat) {
			ack.Nodes[dig.Key] = n
		}

		// If we don't have the node or our version is out of date, add it to our digests.
		if !ok || n.Heartbeat.YoungerThan(dig.Heartbeat) {
			ack.Digests[dig.Key] = node.Digest{Key: dig.Key, Heartbeat: n.Heartbeat}
		}
	}

	for _, n := range snap.Nodes {

		// If we have a node that the initiator doesn't have,
		// send it to them.
		if _, ok := sync.Digests[n.Key]; !ok {
			ack.Nodes[n.Key] = n
		}
	}

	return ack
}

func (g *Gossip) ack(ctx context.Context, ack Message) (ack2 Message) {
	// Take a snapshot before we merge the peer's nodes.
	snap := g.Store.CopyState()
	g.Store.Merge(ctx, ack.Nodes)
	ack2 = Message{Nodes: make(node.Group)}
	for _, dig := range ack.Digests {
		// If we have the node, and our version is newer, return it to the
		// peer.
		if n, ok := snap.Nodes[dig.Key]; ok && n.Heartbeat.OlderThan(dig.Heartbeat) {
			ack2.Nodes[dig.Key] = n
		}
	}
	return ack2
}

func (g *Gossip) ack2(ctx context.Context, ack2 Message) { g.Store.Merge(ctx, ack2.Nodes) }

func RandomPeer(nodes node.Group, host node.Key) node.Node {
	return rand.MapValue(nodes.WhereState(node.StateHealthy).WhereNot(host))
}
