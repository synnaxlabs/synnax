// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/node"
	aspenv1 "github.com/synnaxlabs/aspen/transport/grpc/v1"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/change"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/version"
)

var (
	_ fgrpc.Translator[pledge.Request, *aspenv1.ClusterPledge]         = pledgeTranslator{}
	_ fgrpc.Translator[gossip.Message, *aspenv1.ClusterGossip]         = clusterGossipTranslator{}
	_ fgrpc.Translator[kv.TxRequest, *aspenv1.TxRequest]               = batchTranslator{}
	_ fgrpc.Translator[kv.FeedbackMessage, *aspenv1.FeedbackMessage]   = feedbackTranslator{}
	_ fgrpc.Translator[kv.RecoveryRequest, *aspenv1.RecoveryRequest]   = recoveryRequestTranslator{}
	_ fgrpc.Translator[kv.RecoveryResponse, *aspenv1.RecoveryResponse] = recoveryResponseTranslator{}
)

type pledgeTranslator struct{}

func (p pledgeTranslator) Forward(_ context.Context, req pledge.Request) (*aspenv1.ClusterPledge, error) {
	return &aspenv1.ClusterPledge{NodeKey: uint32(req.Key), ClusterKey: req.ClusterKey.String()}, nil
}

func (p pledgeTranslator) Backward(_ context.Context, msg *aspenv1.ClusterPledge) (pledge.Request, error) {
	cKey, err := uuid.Parse(msg.ClusterKey)
	if err != nil {
		return pledge.Request{}, err
	}
	return pledge.Request{Key: node.Key(msg.NodeKey), ClusterKey: cKey}, nil
}

type clusterGossipTranslator struct{}

func (c clusterGossipTranslator) Forward(_ context.Context, msg gossip.Message) (*aspenv1.ClusterGossip, error) {
	tMsg := &aspenv1.ClusterGossip{Digests: make(map[uint32]*aspenv1.NodeDigest), Nodes: make(map[uint32]*aspenv1.Node)}
	for _, d := range msg.Digests {
		tMsg.Digests[uint32(d.Key)] = &aspenv1.NodeDigest{
			Id:        uint32(d.Key),
			Heartbeat: &aspenv1.Heartbeat{Version: d.Heartbeat.Version, Generation: d.Heartbeat.Generation},
		}
	}
	for _, n := range msg.Nodes {
		tMsg.Nodes[uint32(n.Key)] = &aspenv1.Node{
			Key:       uint32(n.Key),
			Address:   string(n.Address),
			State:     uint32(n.State),
			Heartbeat: &aspenv1.Heartbeat{Version: n.Heartbeat.Version, Generation: n.Heartbeat.Generation},
		}
	}
	return tMsg, nil
}

func (c clusterGossipTranslator) Backward(_ context.Context, tMsg *aspenv1.ClusterGossip) (gossip.Message, error) {
	var msg gossip.Message
	if len(tMsg.Digests) > 0 {
		msg.Digests = make(map[node.Key]node.Digest)
	}
	if len(tMsg.Nodes) > 0 {
		msg.Nodes = make(map[node.Key]node.Node)
	}
	for _, d := range tMsg.Digests {
		msg.Digests[node.Key(d.Id)] = node.Digest{
			Key:       node.Key(d.Id),
			Heartbeat: version.Heartbeat{Version: d.Heartbeat.Version, Generation: d.Heartbeat.Generation},
		}
	}
	for _, n := range tMsg.Nodes {
		msg.Nodes[node.Key(n.Key)] = node.Node{
			Key:       node.Key(n.Key),
			Address:   address.Address(n.Address),
			State:     node.State(n.State),
			Heartbeat: version.Heartbeat{Version: n.Heartbeat.Version, Generation: n.Heartbeat.Generation},
		}
	}
	return msg, nil
}

type batchTranslator struct{}

func (bt batchTranslator) Forward(_ context.Context, msg kv.TxRequest) (*aspenv1.TxRequest, error) {
	tMsg := &aspenv1.TxRequest{Sender: uint32(msg.Sender), Leaseholder: uint32(msg.Leaseholder)}
	for _, o := range msg.Operations {
		tMsg.Operations = append(tMsg.Operations, translateOpForward(o))
	}
	return tMsg, nil
}

func (bt batchTranslator) Backward(ctx context.Context, tMsg *aspenv1.TxRequest) (kv.TxRequest, error) {
	msg := kv.TxRequest{
		Context:     ctx,
		Sender:      node.Key(tMsg.Sender),
		Leaseholder: node.Key(tMsg.Leaseholder),
		Operations:  make([]kv.Operation, len(tMsg.Operations)),
	}
	for i, o := range tMsg.Operations {
		msg.Operations[i] = translateOpBackward(o)
	}
	return msg, nil
}

func translateOpForward(msg kv.Operation) (tMsg *aspenv1.Operation) {
	return &aspenv1.Operation{
		Key:         msg.Key,
		Value:       msg.Value,
		Variant:     uint32(msg.Variant),
		Leaseholder: uint32(msg.Leaseholder),
		Version:     int64(msg.Version),
	}
}

func translateOpBackward(msg *aspenv1.Operation) (tMsg kv.Operation) {
	return kv.Operation{
		Change: xkv.Change{
			Key:     msg.Key,
			Value:   msg.Value,
			Variant: change.Variant(msg.Variant),
		},
		Leaseholder: node.Key(msg.Leaseholder),
		Version:     version.Counter(msg.Version),
	}
}

type feedbackTranslator struct{}

func (ft feedbackTranslator) Forward(_ context.Context, msg kv.FeedbackMessage) (*aspenv1.FeedbackMessage, error) {
	tMsg := &aspenv1.FeedbackMessage{Sender: uint32(msg.Sender)}
	for _, f := range msg.Digests {
		tMsg.Digests = append(tMsg.Digests, &aspenv1.OperationDigest{
			Key:         f.Key,
			Version:     int64(f.Version),
			Leaseholder: uint32(f.Leaseholder),
		})
	}
	return tMsg, nil
}

func (ft feedbackTranslator) Backward(_ context.Context, tMsg *aspenv1.FeedbackMessage) (kv.FeedbackMessage, error) {
	msg := kv.FeedbackMessage{
		Sender:  node.Key(tMsg.Sender),
		Digests: make([]kv.Digest, len(tMsg.Digests)),
	}
	for i, f := range tMsg.Digests {
		msg.Digests[i] = kv.Digest{
			Key:         f.Key,
			Version:     version.Counter(f.Version),
			Leaseholder: node.Key(f.Leaseholder),
		}
	}
	return msg, nil
}

type recoveryRequestTranslator struct{}

func (r recoveryRequestTranslator) Forward(_ context.Context, msg kv.RecoveryRequest) (*aspenv1.RecoveryRequest, error) {
	return &aspenv1.RecoveryRequest{HighWater: int64(msg.HighWater)}, nil
}

func (r recoveryRequestTranslator) Backward(_ context.Context, tMsg *aspenv1.RecoveryRequest) (kv.RecoveryRequest, error) {
	return kv.RecoveryRequest{HighWater: version.Counter(tMsg.HighWater)}, nil
}

type recoveryResponseTranslator struct{}

func (r recoveryResponseTranslator) Forward(_ context.Context, msg kv.RecoveryResponse) (*aspenv1.RecoveryResponse, error) {
	tMsg := &aspenv1.RecoveryResponse{Operations: make([]*aspenv1.Operation, len(msg.Operations))}
	for i, o := range msg.Operations {
		tMsg.Operations[i] = translateOpForward(o)
	}
	return tMsg, nil
}

func (r recoveryResponseTranslator) Backward(ctx context.Context, tMsg *aspenv1.RecoveryResponse) (kv.RecoveryResponse, error) {
	msg := kv.RecoveryResponse{Operations: make([]kv.Operation, len(tMsg.Operations))}
	for i, o := range tMsg.Operations {
		msg.Operations[i] = translateOpBackward(o)
	}
	return msg, nil
}
