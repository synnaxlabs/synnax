package writer

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
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
		targetAddr, err := cfg.HostResolver.Resolve(nodeID)
		if err != nil {
			return sender, receivers, receiverAddresses, err
		}
		addrMap[nodeID] = targetAddr
		client, err := openRemoteClient(ctx, cfg.Transport.Client(), targetAddr, keys)
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
	tran TransportClient,
	target address.Address,
	keys channel.Keys,
) (ClientStream, error) {
	client, err := tran.Stream(ctx, target)
	if err != nil {
		return nil, err
	}
	return client, client.Send(Request{Keys: keys})
}
