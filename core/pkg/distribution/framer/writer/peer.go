// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"
	"strconv"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
)

func (s *Service) openManyPeers(
	ctx context.Context,
	cfg Config,
	targets map[cluster.NodeKey][]keyAuthority,
) (confluence.Sink[Request], []*freightfluence.Receiver[Response], []address.Address, error) {
	var (
		receivers         = make([]*freightfluence.Receiver[Response], 0, len(targets))
		addrMap           = make(proxy.AddressMap)
		senders           = make(map[address.Address]freighter.StreamSenderCloser[Request])
		sender            = newRequestSwitchSender(addrMap, senders)
		receiverAddresses = make([]address.Address, 0, len(targets))
	)

	for nodeKey, keys := range targets {
		target, err := s.cfg.HostResolver.Resolve(nodeKey)
		if err != nil {
			return sender, receivers, receiverAddresses, err
		}
		addrMap[nodeKey] = target
		client, err := s.openPeerClient(ctx, target, cfg.setKeyAuthorities(keys))
		if err != nil {
			return sender, receivers, receiverAddresses, err
		}
		senders[target] = client
		receivers = append(receivers, &freightfluence.Receiver[Response]{Receiver: client})
		receiverAddresses = append(receiverAddresses, address.Address("receiver-"+strconv.Itoa(int(nodeKey))))
	}

	return sender, receivers, receiverAddresses, nil
}

func (s *Service) openPeerClient(ctx context.Context,
	target address.Address,
	cfg Config,
) (ClientStream, error) {
	client, err := s.cfg.Transport.Client().Stream(ctx, target)
	if err != nil {
		return nil, err
	}
	return client, client.Send(Request{Config: cfg})
}
