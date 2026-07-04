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
	indicesByTarget := make(map[node.Key][]int)
	freeIndices := make([]int, 0)
	host := s.cfg.HostResolver.HostKey()
	for i, ch := range channels {
		out[i] = ch
		if out[i].Leaseholder == 0 {
			out[i].Leaseholder = host
		}
		lease := out[i].Lease()
		if lease.IsFree() {
			freeIndices = append(freeIndices, i)
		} else {
			indicesByTarget[lease] = append(indicesByTarget[lease], i)
		}
	}
	for target, indices := range indicesByTarget {
		if target == host {
			if err := s.allocateGateway(ctx, out, indices); err != nil {
				return nil, err
			}
			continue
		}
		if err := s.allocateRemote(ctx, target, out, indices); err != nil {
			return nil, err
		}
	}
	if len(freeIndices) > 0 {
		if host.IsBootstrapper() {
			if err := s.allocateFree(ctx, out, freeIndices); err != nil {
				return nil, err
			}
		} else if err := s.allocateRemote(ctx, node.KeyBootstrapper, out, freeIndices); err != nil {
			return nil, err
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

func pick(channels []Channel, indices []int) []Channel {
	out := make([]Channel, len(indices))
	for j, i := range indices {
		out[j] = channels[i]
	}
	return out
}

func putBack(channels []Channel, indices []int, picked []Channel) {
	for j, i := range indices {
		channels[i] = picked[j]
	}
}

func (s *Service) allocateGateway(ctx context.Context, out []Channel, indices []int) error {
	chs := pick(out, indices)
	if err := assignKeys(ctx, s.leasedCounter, chs); err != nil {
		return err
	}
	if err := s.cfg.TS.CreateChannel(ctx, lo.Map(chs, func(c Channel, _ int) ts.Channel {
		return c.Storage()
	})...); err != nil {
		return err
	}
	putBack(out, indices, chs)
	return nil
}

func (s *Service) allocateFree(
	ctx context.Context, out []Channel, indices []int,
) error {
	chs := pick(out, indices)
	if err := assignKeys(ctx, s.freeCounter, chs); err != nil {
		return err
	}
	// Free channels are non-leased and virtual; they have no persistent storage, so no
	// storage channel is created for them.
	putBack(out, indices, chs)
	return nil
}

func (s *Service) allocateRemote(
	ctx context.Context,
	target node.Key,
	out []Channel,
	indices []int,
) error {
	addr, err := s.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	res, err := s.cfg.Transport.CreateClient().Send(ctx, addr, CreateMessage{Channels: pick(out, indices)})
	if err != nil {
		return err
	}
	if len(res.Channels) != len(indices) {
		return errors.Newf(
			"[channel.Service] - allocator received %d channels from %v, expected %d",
			len(res.Channels), target, len(indices),
		)
	}
	putBack(out, indices, res.Channels)
	return nil
}
