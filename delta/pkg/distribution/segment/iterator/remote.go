package iterator

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
)

func openRemoteIterators(
	ctx signal.Context,
	tran Transport,
	targets map[core.NodeID][]channel.Key,
	rng telem.TimeRange,
	resolver aspen.HostResolver,
) (*freightfluence.MultiSender[Request], []*freightfluence.Receiver[Response], error) {
	sender := &freightfluence.MultiSender[Request]{}
	receivers := make([]*freightfluence.Receiver[Response], 0, len(targets))
	for nodeID, keys := range targets {
		targetAddr, err := resolver.Resolve(nodeID)
		if err != nil {
			return sender, receivers, err
		}
		client, err := openRemoteClient(ctx, tran, targetAddr, keys, rng)
		if err != nil {
			return sender, receivers, err
		}
		sender.Senders = append(sender.Senders, client)
		receivers = append(receivers, &freightfluence.Receiver[Response]{Receiver: client})
	}
	return sender, receivers, nil
}

func openRemoteClient(
	ctx signal.Context,
	tran Transport,
	target address.Address,
	keys channel.Keys,
	rng telem.TimeRange,
) (Client, error) {
	client, err := tran.Stream(ctx, target)
	if err != nil {
		return nil, err
	}

	// Send an open request to the freighter. This will open a localIterator  on the
	// target node.
	return client, client.Send(Request{
		Command: Open,
		Keys:    keys,
		Range:   rng,
	})
}
