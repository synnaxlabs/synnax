package kv

import (
	"context"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/confluence"
	"go/types"
)

// |||||| OPERATION ||||||

type (
	BatchTransportClient = freighter.UnaryClient[BatchRequest, BatchRequest]
	BatchTransportServer = freighter.UnaryServer[BatchRequest, BatchRequest]
)

// |||| SENDER ||||

type operationSender struct {
	Config
	confluence.LinearTransform[BatchRequest, BatchRequest]
}

func newOperationSender(cfg Config) segment {
	os := &operationSender{Config: cfg}
	os.TransformFunc.Transform = os.send
	return os
}

func (g *operationSender) send(ctx context.Context, sync BatchRequest) (BatchRequest, bool, error) {
	// If we have no Operations to propagate, it's best to avoid the network chatter.
	if sync.empty() {
		return sync, false, nil
	}
	hostID := g.Cluster.HostID()
	peer := gossip.RandomPeer(g.Cluster.Nodes(), hostID)
	if peer.Address == "" {
		return sync, false, nil
	}
	sync.Sender = hostID
	ack, err := g.BatchTransportClient.Send(ctx, peer.Address, sync)
	if err != nil {
		g.Logger.Errorw("operation gossip failed", "err", err)
	}
	// If we have no Operations to apply, avoid the pipeline overhead.
	return ack, !ack.empty(), nil
}

// |||| RECEIVER ||||

type operationReceiver struct {
	Config
	store store
	confluence.AbstractUnarySource[BatchRequest]
	confluence.EmptyFlow
}

func newOperationReceiver(cfg Config, s store) source {
	or := &operationReceiver{Config: cfg, store: s}
	or.BatchTransportServer.BindHandler(or.handle)
	return or
}

func (g *operationReceiver) handle(ctx context.Context, req BatchRequest) (BatchRequest, error) {
	select {
	case <-ctx.Done():
		return BatchRequest{}, ctx.Err()
	case g.Out.Inlet() <- req:
	}
	br := g.store.PeekState().toBatchRequest()
	br.Sender = g.Cluster.HostID()
	return br, nil
}

// |||||| FEEDBACK ||||||

type FeedbackMessage struct {
	Sender  node.ID
	Digests Digests
}

type (
	FeedbackTransportClient = freighter.UnaryClient[FeedbackMessage, types.Nil]
	FeedbackTransportServer = freighter.UnaryServer[FeedbackMessage, types.Nil]
)

// |||| SENDER ||||

type feedbackSender struct {
	Config
	confluence.UnarySink[BatchRequest]
}

func newFeedbackSender(cfg Config) sink {
	fs := &feedbackSender{Config: cfg}
	fs.Sink = fs.send
	return fs
}

func (f *feedbackSender) send(ctx context.Context, bd BatchRequest) error {
	msg := FeedbackMessage{Sender: f.Cluster.Host().ID, Digests: bd.digests()}
	sender, _ := f.Cluster.Node(bd.Sender)
	if _, err := f.FeedbackTransportClient.Send(context.TODO(), sender.Address, msg); err != nil {
		f.Logger.Errorw("feedback gossip failed", "err", err)
	}
	return nil
}

// |||| RECEIVER ||||

type feedbackReceiver struct {
	Config
	confluence.AbstractUnarySource[BatchRequest]
	confluence.EmptyFlow
}

func newFeedbackReceiver(cfg Config) source {
	fr := &feedbackReceiver{Config: cfg}
	fr.FeedbackTransportServer.BindHandler(fr.handle)
	return fr
}

func (f *feedbackReceiver) handle(ctx context.Context, message FeedbackMessage) (types.Nil, error) {
	f.Out.Inlet() <- message.Digests.toRequest()
	return types.Nil{}, nil
}
