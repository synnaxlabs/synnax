package stream

import (
	"context"
	"github.com/arya-analytics/aspen"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"go/types"
)

type writeSender struct {
	resolver aspen.Resolver
	freightfluence.BatchSwitchSender[[]Sample, WriteRequest]
}

func newWriteSender(
	transport WriteTransport,
	resolver distribcore.Resolver,
	transient confluence.Inlet[error],
) confluence.Sink[[]Sample] {
	w := &writeSender{resolver: resolver}
	w.Senders = freightfluence.ClientTargetedSender[WriteRequest, types.Nil]{
		Transport:         transport,
		MapTargetedSender: make(freightfluence.MapTargetedSender[WriteRequest]),
	}
	w.ApplySwitch = w.Send
	return confluence.InjectTransientSink[[]Sample](transient, w)
}

func (s *writeSender) Send(ctx context.Context, in []Sample, out map[address.Address]WriteRequest) (err error) {
	for _, sample := range in {
		addr, _err := s.resolver.Resolve(sample.ChannelKey.NodeID())
		if _err != nil {
			err = _err
			continue
		}
		_, ok := out[addr]
		if !ok {
			out[addr] = WriteRequest{Samples: make([]Sample, 0, len(in))}
		}
		out[addr] = WriteRequest{Samples: append(out[addr].Samples, sample)}
	}
	return err
}

type remoteLocalSwitch struct {
	host distribcore.NodeID
	confluence.BatchSwitch[[]Sample, []Sample]
}

func newHostSwitch(host distribcore.NodeID) *remoteLocalSwitch {
	hs := &remoteLocalSwitch{host: host}
	hs.ApplySwitch = hs._switch
	return hs
}

func (hs *remoteLocalSwitch) _switch(ctx context.Context, samples []Sample, oReqs map[address.Address][]Sample) error {
	remote := make([]Sample, 0, len(samples))
	local := make([]Sample, 0, len(samples))
	for _, sample := range samples {
		if sample.ChannelKey.NodeID() == hs.host {
			local = append(local, sample)
		} else {
			remote = append(remote, sample)
		}
	}
	oReqs["local"] = local
	oReqs["remote"] = remote
	return nil
}
