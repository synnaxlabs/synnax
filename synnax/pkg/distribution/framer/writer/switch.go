package writer

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type peerSwitchSender struct {
	freightfluence.BatchSwitchSender[Request, Request]
	confluence.AbstractUnarySource[Response]
	addresses      proxy.AddressMap
	accumulatedErr error
}

func newRequestSwitchSender(
	addresses proxy.AddressMap,
	senders map[address.Address]freighter.StreamSenderCloser[Request],
) confluence.Segment[Request, Response] {
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
	if rs.accumulatedErr != nil {
		if r.Command == Error {
			if err := signal.SendUnderContext(ctx, rs.Out.Inlet(), Response{
				Command: r.Command,
				Ack:     false,
			}); err != nil {
				return err
			}
			rs.accumulatedErr = nil
		}
		return nil
	}
	if r.Command == Data {
		for nodeID, frame := range r.Frame.SplitByNodeID() {
			addr, ok := rs.addresses[nodeID]
			if !ok {
				return signal.SendUnderContext[Response](ctx, rs.Out.Inlet(), Response{
					Command: Error,
					Err:     errors.New("no address found for nodeID"),
				})
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
