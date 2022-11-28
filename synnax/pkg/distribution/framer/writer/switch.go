package writer

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
)

type requestSwitchSender struct {
	freightfluence.BatchSwitchSender[Request, Request]
	addresses proxy.AddressMap
	confluence.TransientProvider
	accumulatedErr error
}

func newRequestSwitchSender(
	addresses proxy.AddressMap,
	trans confluence.Inlet[error],
	senders map[address.Address]freighter.StreamSenderCloser[Request],
) confluence.Sink[Request] {
	rs := &requestSwitchSender{addresses: addresses}
	rs.Senders = freightfluence.MapTargetedSender[Request](senders)
	rs.BatchSwitchSender.ApplySwitch = rs._switch
	return confluence.InjectTransientSink[Request](trans, rs)
}

func (rs *requestSwitchSender) _switch(
	ctx context.Context,
	r Request,
	oReqs map[address.Address]Request,
) error {
	if rs.accumulatedErr != nil {
		if r.Command == Error {
			return sendErrorAck(ctx, rs.Transient(), rs.accumulatedErr)
		}
		return nil
	}
	for nodeID, frame := range r.Frame.SplitByNodeID() {
		addr, ok := rs.addresses[nodeID]
		if !ok {
			if err := sendBadAck(ctx, rs.Transient()); err != nil {
				return err
			}
			continue
		}
		oReqs[addr] = Request{Frame: frame}
	}
	return nil
}

type remoteLocalSwitch struct {
	confluence.BatchSwitch[Request, Request]
	host core.NodeID
}

func newRemoteLocalSwitch(host core.NodeID) *remoteLocalSwitch {
	rl := &remoteLocalSwitch{host: host}
	rl.ApplySwitch = rl._switch
	return rl
}

func (rl *remoteLocalSwitch) _switch(ctx context.Context, r Request, oReqs map[address.Address]Request) error {
	local, remote := r.Frame.SplitByHost(rl.host)
	oReqs["localWriter"] = Request{Frame: local}
	oReqs["remoteSender"] = Request{Frame: remote}
	return nil
}
