// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"go/types"

	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/confluence"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type (
	TxTransportClient = freighter.UnaryClient[TxRequest, TxRequest]
	TxTransportServer = freighter.UnaryServer[TxRequest, TxRequest]
)

type operationClient struct {
	Config
	confluence.LinearTransform[TxRequest, TxRequest]
}

func newOperationClient(cfg Config) segment {
	os := &operationClient{Config: cfg}
	os.Transform = os.send
	return os
}

func (g *operationClient) send(_ context.Context, sync TxRequest) (TxRequest, bool, error) {
	// If we have no NewStreamer to propagate, it's best to avoid the network chatter.
	if sync.empty() {
		return sync, false, nil
	}
	hostID := g.Cluster.HostKey()
	peer := gossip.RandomPeer(g.Cluster.Nodes(), hostID)
	if peer.Address == "" {
		return sync, false, nil
	}
	sync.Sender = hostID
	ack, err := g.BatchTransportClient.Send(sync.Context, peer.Address, sync)
	ack.Context = sync.Context
	if err != nil {
		g.L.Error("operation gossip failed", zap.Error(err))
	}
	// If we have no operations to apply, avoid the pipeline overhead.
	return ack, !ack.empty(), nil
}

type operationServer struct {
	Config
	store store
	confluence.AbstractUnarySource[TxRequest]
	confluence.NopFlow
}

func newOperationServer(cfg Config, s store) source {
	or := &operationServer{Config: cfg, store: s}
	or.BatchTransportServer.BindHandler(or.handle)
	return or
}

func (g *operationServer) handle(ctx context.Context, req TxRequest) (TxRequest, error) {
	// The handler context is cancelled after it returns, so we need to use a separate
	// context for executing the tx.
	req.Context = context.TODO()
	select {
	case <-ctx.Done():
		return TxRequest{}, ctx.Err()
	case g.Out.Inlet() <- req:
	}
	s, release := g.store.PeekState()
	defer release()
	br := s.toBatchRequest(ctx)
	br.Sender = g.Cluster.HostKey()
	return br, nil
}

type FeedbackMessage struct {
	Sender  node.Key
	Digests Digests
}

type (
	FeedbackTransportClient = freighter.UnaryClient[FeedbackMessage, types.Nil]
	FeedbackTransportServer = freighter.UnaryServer[FeedbackMessage, types.Nil]
)

type feedbackSender struct {
	Config
	confluence.UnarySink[TxRequest]
}

func newFeedbackSender(cfg Config) sink {
	fs := &feedbackSender{Config: cfg}
	fs.Sink = fs.send
	return fs
}

func (f *feedbackSender) send(ctx context.Context, bd TxRequest) error {
	msg := FeedbackMessage{Sender: f.Cluster.Host().Key, Digests: bd.digests()}
	sender, _ := f.Cluster.Node(bd.Sender)
	if _, err := f.FeedbackTransportClient.Send(ctx, sender.Address, msg); err != nil {
		f.L.Error("feedback gossip failed", zap.Error(err))
	}
	return nil
}

type feedbackReceiver struct {
	Config
	confluence.AbstractUnarySource[TxRequest]
	confluence.NopFlow
}

func newFeedbackReceiver(cfg Config) source {
	fr := &feedbackReceiver{Config: cfg}
	fr.FeedbackTransportServer.BindHandler(fr.handle)
	return fr
}

func (f *feedbackReceiver) handle(ctx context.Context, msg FeedbackMessage) (types.Nil, error) {
	// The handler context is cancelled after it returns, so we need to use a separate
	// context for passing the feedback to the pipeline.
	return types.Nil{}, signal.SendUnderContext(ctx, f.Out.Inlet(), msg.Digests.toRequest(context.TODO()))
}

type gossipRecoveryTransform struct {
	Config
	confluence.LinearTransform[TxRequest, TxRequest]
	repetitions map[string]int
}

func newGossipRecoveryTransform(cfg Config) segment {
	r := &gossipRecoveryTransform{Config: cfg, repetitions: make(map[string]int)}
	r.Transform = r.transform
	return r
}

func (r *gossipRecoveryTransform) transform(
	_ context.Context,
	in TxRequest,
) (out TxRequest, ok bool, err error) {
	out.Context = in.Context
	for _, op := range in.Operations {
		key := string(lo.Must(xkv.CompositeKey(op.Key, op.Version)))
		if r.repetitions[key] > r.RecoveryThreshold {
			op.state = recovered
			out.Operations = append(out.Operations, op)
			delete(r.repetitions, key)
		}
		r.repetitions[key]++
	}
	return out, true, nil
}
