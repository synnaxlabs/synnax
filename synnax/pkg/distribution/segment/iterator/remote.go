package iterator

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/telem"
)

func openRemoteIterators(
	ctx context.Context,
	targets map[core.NodeID][]channel.Key,
	cfg Config,
) (*freightfluence.MultiSender[Request], []*freightfluence.Receiver[Response], error) {
	sender := &freightfluence.MultiSender[Request]{}
	receivers := make([]*freightfluence.Receiver[Response], 0, len(targets))
	for nodeID, keys := range targets {
		targetAddr, err := cfg.Resolver.Resolve(nodeID)
		if err != nil {
			return sender, receivers, err
		}
		client, err := openRemoteClient(ctx, cfg.Transport, targetAddr, keys, cfg.TimeRange)
		if err != nil {
			return sender, receivers, err
		}
		sender.Senders = append(sender.Senders, client)
		receivers = append(receivers, &freightfluence.Receiver[Response]{Receiver: client})
	}
	return sender, receivers, nil
}

func openRemoteClient(
	ctx context.Context,
	tran Transport,
	target address.Address,
	keys channel.Keys,
	rng telem.TimeRange,
) (Client, error) {
	client, err := tran.Stream(ctx, target)
	if err != nil {
		return nil, err
	}
	return client, client.Send(Request{Command: Open, Keys: keys, Range: rng})
}
