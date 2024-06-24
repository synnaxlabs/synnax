// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/telem"
)

func (s *Service) openManyPeers(
	ctx context.Context,
	bounds telem.TimeRange,
	chunkSize int64,
	targets map[core.NodeKey][]channel.Key,
) (*freightfluence.MultiSender[Request], []*freightfluence.Receiver[Response], error) {
	var (
		sender    = &freightfluence.MultiSender[Request]{}
		receivers = make([]*freightfluence.Receiver[Response], 0, len(targets))
	)
	for nodeKey, keys := range targets {
		target, err := s.HostResolver.Resolve(nodeKey)
		if err != nil {
			return sender, receivers, err
		}
		client, err := s.openPeerClient(ctx, target, Config{Keys: keys, Bounds: bounds, ChunkSize: chunkSize})
		if err != nil {
			return sender, receivers, err
		}
		sender.Senders = append(sender.Senders, client)
		receivers = append(receivers, &freightfluence.Receiver[Response]{Receiver: client})
	}
	return sender, receivers, nil
}

func (s *Service) openPeerClient(ctx context.Context, target address.Address, cfg Config) (ClientStream, error) {
	client, err := s.Transport.Client().Stream(ctx, target)
	if err != nil {
		return nil, err
	}
	return client, client.Send(Request{Keys: cfg.Keys, ChunkSize: cfg.ChunkSize, Bounds: cfg.Bounds})
}
