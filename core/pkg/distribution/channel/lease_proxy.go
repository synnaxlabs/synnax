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
	"go/types"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/errors"
)

func (s *Service) allocateHandler(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	allocated, err := s.Create(ctx, msg.Channels)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: allocated}, nil
}

func (s *Service) deleteHandler(ctx context.Context, msg DeleteRequest) (types.Nil, error) {
	return types.Nil{}, s.cfg.TSChannel.DeleteChannels(msg.Keys.Storage())
}

func (s *Service) renameHandler(ctx context.Context, msg RenameRequest) (types.Nil, error) {
	return types.Nil{}, s.cfg.TSChannel.RenameChannels(ctx, msg.Keys.Storage(), msg.Names)
}

// assignKeys draws len(channels with a zero LocalKey) values from the provided counter
// and assigns them sequentially. Index channels have their LocalIndex set to their own
// newly-assigned LocalKey.
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
	if err := s.cfg.TSChannel.CreateChannel(ctx, toStorage(chs)...); err != nil {
		return err
	}
	putBack(out, indices, chs)
	return nil
}

func (s *Service) allocateFree(ctx context.Context, out []Channel, indices []int) error {
	if s.freeCounter == nil {
		panic("[channel.Service] - tried to assign free virtual keys on non-bootstrapper")
	}
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

func (s *Service) deleteRemote(ctx context.Context, target node.Key, keys Keys) error {
	addr, err := s.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = s.cfg.Transport.DeleteClient().Send(ctx, addr, DeleteRequest{Keys: keys})
	return err
}

func (s *Service) renameRemote(ctx context.Context, target node.Key, keys Keys, names []string) error {
	addr, err := s.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = s.cfg.Transport.RenameClient().Send(ctx, addr, RenameRequest{Keys: keys, Names: names})
	return err
}

type renameBatchEntry struct {
	name string
	key  Key
}

var _ proxy.Entry = renameBatchEntry{}

func (r renameBatchEntry) Lease() node.Key { return r.key.Lease() }

func unzipRenameBatch(entries []renameBatchEntry) (Keys, []string) {
	return lo.UnzipBy2(entries, func(e renameBatchEntry) (Key, string) {
		return e.key, e.name
	})
}

func newRenameBatch(keys Keys, names []string) []renameBatchEntry {
	return lo.ZipBy2(keys, names, func(k Key, n string) renameBatchEntry {
		return renameBatchEntry{key: k, name: n}
	})
}
