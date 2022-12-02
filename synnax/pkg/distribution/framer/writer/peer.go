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

func (s *Service) openManyPeers(
	ctx context.Context,
	targets map[core.NodeID][]channel.Key,
) (confluence.Segment[Request, Response], []*freightfluence.Receiver[Response], []address.Address, error) {
	var (
		receivers         = make([]*freightfluence.Receiver[Response], 0, len(targets))
		addrMap           = make(proxy.AddressMap)
		senders           = make(map[address.Address]freighter.StreamSenderCloser[Request])
		sender            = newRequestSwitchSender(addrMap, senders)
		receiverAddresses = make([]address.Address, 0, len(targets))
	)

	for nodeID, keys := range targets {
		target, err := s.HostResolver.Resolve(nodeID)
		if err != nil {
			return sender, receivers, receiverAddresses, err
		}
		addrMap[nodeID] = target
		client, err := s.openPeerClient(ctx, target, Config{Keys: keys})
		if err != nil {
			return sender, receivers, receiverAddresses, err
		}
		senders[target] = client
		receivers = append(receivers, &freightfluence.Receiver[Response]{Receiver: client})
		receiverAddresses = append(receiverAddresses, address.Address("receiver-"+strconv.Itoa(int(nodeID))))
	}

	return sender, receivers, receiverAddresses, nil
}

func (s *Service) openPeerClient(ctx context.Context,
	target address.Address,
	cfg Config,
) (ClientStream, error) {
	client, err := s.Transport.Client().Stream(ctx, target)
	if err != nil {
		return nil, err
	}
	return client, client.Send(Request{Keys: cfg.Keys})
}
