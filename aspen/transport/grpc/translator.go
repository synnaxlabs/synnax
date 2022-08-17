package grpc

import (
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/aspen/internal/kv"
	"github.com/arya-analytics/aspen/internal/node"
	aspenv1 "github.com/arya-analytics/aspen/transport/grpc/gen/proto/go/v1"
	"github.com/arya-analytics/freighter/fgrpc"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/version"
)

var (
	_ fgrpc.Translator[node.ID, *aspenv1.ClusterPledge]              = pledgeTranslator{}
	_ fgrpc.Translator[gossip.Message, *aspenv1.ClusterGossip]       = clusterGossipTranslator{}
	_ fgrpc.Translator[kv.BatchRequest, *aspenv1.BatchRequest]       = batchTranslator{}
	_ fgrpc.Translator[kv.FeedbackMessage, *aspenv1.FeedbackMessage] = feedbackTranslator{}
)

type pledgeTranslator struct{}

func (p pledgeTranslator) Forward(id node.ID) (*aspenv1.ClusterPledge, error) {
	return &aspenv1.ClusterPledge{NodeId: uint32(id)}, nil
}

func (p pledgeTranslator) Backward(msg *aspenv1.ClusterPledge) (node.ID, error) {
	return node.ID(msg.NodeId), nil
}

type clusterGossipTranslator struct{}

func (c clusterGossipTranslator) Forward(msg gossip.Message) (*aspenv1.ClusterGossip, error) {
	tMsg := &aspenv1.ClusterGossip{Digests: make(map[uint32]*aspenv1.NodeDigest), Nodes: make(map[uint32]*aspenv1.Node)}
	for _, d := range msg.Digests {
		tMsg.Digests[uint32(d.ID)] = &aspenv1.NodeDigest{
			Id:        uint32(d.ID),
			Heartbeat: &aspenv1.Heartbeat{Version: d.Heartbeat.Version, Generation: d.Heartbeat.Generation},
		}
	}
	for _, n := range msg.Nodes {
		tMsg.Nodes[uint32(n.ID)] = &aspenv1.Node{
			Id:        uint32(n.ID),
			Address:   string(n.Address),
			State:     uint32(n.State),
			Heartbeat: &aspenv1.Heartbeat{Version: n.Heartbeat.Version, Generation: n.Heartbeat.Generation},
		}
	}
	return tMsg, nil
}

func (c clusterGossipTranslator) Backward(tMsg *aspenv1.ClusterGossip) (gossip.Message, error) {
	var msg gossip.Message
	if len(tMsg.Digests) > 0 {
		msg.Digests = make(map[node.ID]node.Digest)
	}
	if len(tMsg.Nodes) > 0 {
		msg.Nodes = make(map[node.ID]node.Node)
	}
	for _, d := range tMsg.Digests {
		msg.Digests[node.ID(d.Id)] = node.Digest{
			ID:        node.ID(d.Id),
			Heartbeat: version.Heartbeat{Version: d.Heartbeat.Version, Generation: d.Heartbeat.Generation},
		}
	}
	for _, n := range tMsg.Nodes {
		msg.Nodes[node.ID(n.Id)] = node.Node{
			ID:        node.ID(n.Id),
			Address:   address.Address(n.Address),
			State:     node.State(n.State),
			Heartbeat: version.Heartbeat{Version: n.Heartbeat.Version, Generation: n.Heartbeat.Generation},
		}
	}
	return msg, nil
}

type batchTranslator struct{}

func (bt batchTranslator) Forward(msg kv.BatchRequest) (*aspenv1.BatchRequest, error) {
	tMsg := &aspenv1.BatchRequest{Sender: uint32(msg.Sender)}
	for _, o := range msg.Operations {
		tMsg.Operations = append(tMsg.Operations, translateOpForward(o))
	}
	return tMsg, nil
}

func (bt batchTranslator) Backward(tMsg *aspenv1.BatchRequest) (kv.BatchRequest, error) {
	msg := kv.BatchRequest{
		Sender:     node.ID(tMsg.Sender),
		Operations: make([]kv.Operation, len(tMsg.Operations)),
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
		Key:         msg.Key,
		Value:       msg.Value,
		Variant:     kv.Variant(msg.Variant),
		Leaseholder: node.ID(msg.Leaseholder),
		Version:     version.Counter(msg.Version),
	}
}

type feedbackTranslator struct{}

func (ft feedbackTranslator) Forward(msg kv.FeedbackMessage) (*aspenv1.FeedbackMessage, error) {
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

func (ft feedbackTranslator) Backward(tMsg *aspenv1.FeedbackMessage) (kv.FeedbackMessage, error) {
	msg := kv.FeedbackMessage{
		Sender:  node.ID(tMsg.Sender),
		Digests: make([]kv.Digest, len(tMsg.Digests)),
	}
	for i, f := range tMsg.Digests {
		msg.Digests[i] = kv.Digest{
			Key:         f.Key,
			Version:     version.Counter(f.Version),
			Leaseholder: node.ID(f.Leaseholder),
		}
	}
	return msg, nil
}
