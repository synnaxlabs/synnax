package writer

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"go.uber.org/zap"
)

type peerSwitchSender struct {
	freightfluence.BatchSwitchSender[Request, Request]
	addresses proxy.AddressMap
	logger    *zap.Logger
}

func newRequestSwitchSender(
	addresses proxy.AddressMap,
	senders map[address.Address]freighter.StreamSenderCloser[Request],
) confluence.Sink[Request] {
	rs := &peerSwitchSender{addresses: addresses}
	rs.Senders = freightfluence.MapTargetedSender[Request](senders)
	rs.BatchSwitchSender.ApplySwitch = rs._switch
	return rs
}

func (rs *peerSwitchSender) _switch(
	ctx context.Context,
	r Request,
	oReqs map[address.Address]Request,
) error {
	if r.Command == Data {
		for nodeID, frame := range r.Frame.SplitByNodeID() {
			addr, ok := rs.addresses[nodeID]
			if !ok {
				rs.logger.DPanic("missing address for node", zap.Uint32("node", uint32(nodeID)))
			}
			r.Frame = frame
			oReqs[addr] = r
		}
	} else {
		for _, addr := range rs.addresses {
			oReqs[addr] = r
		}
	}

	return nil
}

type peerGatewaySwitch struct {
	confluence.BatchSwitch[Request, Request]
	host core.NodeID
}

func newPeerGatewaySwitch(host core.NodeID) *peerGatewaySwitch {
	rl := &peerGatewaySwitch{host: host}
	rl.ApplySwitch = rl._switch
	return rl
}

func (rl *peerGatewaySwitch) _switch(ctx context.Context, r Request, oReqs map[address.Address]Request) error {
	local, remote := r.Frame.SplitByHost(rl.host)
	pr, gr := r, r
	pr.Frame = remote
	gr.Frame = local
	oReqs[gatewayWriterAddr] = gr
	oReqs[peerSenderAddr] = pr
	return nil
}
