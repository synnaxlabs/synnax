package writer

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/proxy"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
)

type requestSwitchSender struct {
	freightfluence.BatchSwitchSender[Request, Request]
	addresses proxy.AddressMap
	confluence.TransientProvider
}

func newRequestSwitchSender(
	addresses proxy.AddressMap,
	trans confluence.Inlet[error],
	senders map[address.Address]freighter.StreamSenderCloser[Request],
) confluence.Sink[Request] {
	rs := &requestSwitchSender{addresses: addresses}
	rs.Senders = senders
	rs.BatchSwitchSender.ApplySwitch = rs._switch
	return confluence.InjectTransientSink[Request](trans, rs)
}

func (rs *requestSwitchSender) _switch(ctx context.Context, r Request, oReqs map[address.Address]Request) error {
	for _, seg := range r.Segments {
		addr, ok := rs.addresses[seg.ChannelKey.NodeID()]
		if !ok {
			rs.Transient() <- unspecifiedChannelError(seg.ChannelKey)
			continue
		}
		oReqs[addr] = Request{Segments: append(oReqs[addr].Segments, seg)}
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
	for _, seg := range r.Segments {
		if seg.ChannelKey.NodeID() == rl.host {
			oReqs["localWriter"] = Request{Segments: append(oReqs["local"].Segments, seg)}
		} else {
			oReqs["remoteSender"] = Request{Segments: append(oReqs["remote"].Segments, seg)}
		}
	}
	return nil
}
