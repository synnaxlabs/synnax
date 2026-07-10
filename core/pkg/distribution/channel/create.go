// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/errors"
)

// Create assigns local keys to the provided new channels (those with a zero LocalKey)
// by routing to each channel's leaseholder to draw from that node's key counter, and
// creates the corresponding storage channels there. The returned channels carry their
// assigned keys, in the same order as the input. Free-virtual channels draw their keys
// from the bootstrapper. A channel with an unspecified (zero) leaseholder defaults to
// the host node.
func (s *Service) Create(ctx context.Context, channels []Channel) ([]Channel, error) {
	out := make([]Channel, len(channels))
	copy(out, channels)
	host := s.cfg.HostResolver.HostKey()
	for i := range out {
		if out[i].Leaseholder == 0 {
			out[i].Leaseholder = host
		}
	}
	indicesByTarget := lo.GroupBy(
		lo.Range(len(out)),
		func(i int) node.Key { return out[i].Lease() },
	)
	for target, indices := range indicesByTarget {
		chs := lo.Map(indices, func(i int, _ int) Channel { return out[i] })
		var err error
		switch {
		case target == host:
			err = s.allocateGateway(ctx, chs)
		case target.IsFree() && host.IsBootstrapper():
			err = s.allocateFree(ctx, chs)
		case target.IsFree():
			err = s.allocateRemote(ctx, node.KeyBootstrapper, chs)
		default:
			err = s.allocateRemote(ctx, target, chs)
		}
		if err != nil {
			return nil, err
		}
		for j, i := range indices {
			out[i] = chs[j]
		}
	}
	return out, nil
}

func (s *Service) createHandler(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	allocated, err := s.Create(ctx, msg.Channels)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: allocated}, nil
}

// assignKeys draws len(channels with a zero LocalKey) values from the provided counter
// and assigns them sequentially. Index channels have their LocalIndex set to their own
// LocalKey, whether that key was just assigned or already present.
func assignKeys(ctx context.Context, c *counter, channels []Channel) error {
	var toAssign LocalKey
	for _, ch := range channels {
		if ch.LocalKey == 0 {
			toAssign++
		}
	}
	next, err := c.add(ctx, toAssign)
	if err != nil {
		return err
	}
	next -= toAssign
	for i := range channels {
		if channels[i].LocalKey == 0 {
			next++
			channels[i].LocalKey = next
		}
		if channels[i].IsIndex {
			channels[i].LocalIndex = channels[i].LocalKey
		}
	}
	return nil
}

func (s *Service) allocateGateway(ctx context.Context, channels []Channel) error {
	if err := assignKeys(ctx, s.leasedCounter, channels); err != nil {
		return err
	}
	return s.cfg.TS.CreateChannel(ctx, lo.Map(channels, func(c Channel, _ int) ts.Channel {
		return c.Storage()
	})...)
}

func (s *Service) allocateFree(ctx context.Context, channels []Channel) error {
	if err := assignKeys(ctx, s.freeCounter, channels); err != nil {
		return err
	}
	// Free channels are created in the local storage engine on the same path as
	// gateway-leased channels. Every free create executes here on the bootstrapper, so
	// a newly created channel can never collide with an existing storage registration;
	// other nodes register free channels when they scan the channel table at startup
	// (see RegisterVirtualStorage).
	return s.cfg.TS.CreateChannel(ctx, lo.Map(channels, func(c Channel, _ int) ts.Channel {
		return c.Storage()
	})...)
}

func (s *Service) allocateRemote(
	ctx context.Context,
	target node.Key,
	channels []Channel,
) error {
	addr, err := s.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	res, err := s.cfg.Transport.CreateClient().
		Send(ctx, addr, CreateMessage{Channels: channels})
	if err != nil {
		return err
	}
	if len(res.Channels) != len(channels) {
		return errors.Newf(
			"[channel.Service] - allocator received %d channels from %v, expected %d",
			len(res.Channels), target, len(channels),
		)
	}
	copy(channels, res.Channels)
	return nil
}

// RegisterVirtualStorage registers the given virtual channels in this node's local
// storage engine. Virtual channels are never persisted by the storage layer and vanish
// on restart, so each node re-registers the virtual channels it serves writes for once
// at startup; creates after startup register through Create.
func (s *Service) RegisterVirtualStorage(ctx context.Context, channels []Channel) error {
	return s.cfg.TS.CreateChannel(ctx, lo.Map(channels, func(c Channel, _ int) ts.Channel {
		return c.Storage()
	})...)
}
