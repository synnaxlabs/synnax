package writer

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/proxy"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"strconv"
)

func openRemoteWriters(
	ctx context.Context,
	targets map[core.NodeID][]channel.Key,
	transient confluence.Inlet[error],
	cfg Config,
) (confluence.Sink[Request], []*freightfluence.Receiver[Response], []address.Address, error) {
	receivers := make([]*freightfluence.Receiver[Response], 0, len(targets))
	addrMap := make(proxy.AddressMap)
	senders := make(map[address.Address]freighter.StreamSenderCloser[Request])
	sender := newRequestSwitchSender(addrMap, transient, senders)
	receiverAddresses := make([]address.Address, 0, len(targets))
	for nodeID, keys := range targets {
		targetAddr, err := cfg.Resolver.Resolve(nodeID)
		if err != nil {
			return sender, receivers, receiverAddresses, err
		}
		addrMap[nodeID] = targetAddr
		client, err := openRemoteClient(ctx, cfg.Transport, targetAddr, keys)
		if err != nil {
			return sender, receivers, receiverAddresses, err
		}
		senders[targetAddr] = client
		receivers = append(receivers, &freightfluence.Receiver[Response]{Receiver: client})
		receiverAddresses = append(receiverAddresses, address.Address("receiver-"+strconv.Itoa(int(nodeID))))
	}
	return sender, receivers, receiverAddresses, nil
}

func openRemoteClient(
	ctx context.Context,
	tran Transport,
	target address.Address,
	keys channel.Keys,
) (Client, error) {
	client, err := tran.Stream(ctx, target)
	if err != nil {
		return nil, err
	}
	return client, client.Send(Request{OpenKeys: keys})
}
